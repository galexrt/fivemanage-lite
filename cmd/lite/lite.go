package main

import (
	"context"
	"fmt"
	"log/slog"
	nethttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fivemanage/lite/internal/clickhouse"
	"github.com/fivemanage/lite/internal/database"
	"github.com/fivemanage/lite/internal/http"
	"github.com/fivemanage/lite/internal/service/auth"
	"github.com/fivemanage/lite/internal/service/dataset"
	"github.com/fivemanage/lite/internal/service/file"
	"github.com/fivemanage/lite/internal/service/log"
	"github.com/fivemanage/lite/internal/service/member"
	"github.com/fivemanage/lite/internal/service/organization"
	"github.com/fivemanage/lite/internal/service/system"
	"github.com/fivemanage/lite/internal/service/token"
	"github.com/fivemanage/lite/migrate"
	"github.com/fivemanage/lite/pkg/cache"
	"github.com/fivemanage/lite/pkg/logger"
	"github.com/fivemanage/lite/pkg/otel"
	"github.com/fivemanage/lite/pkg/storage"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
)

var Version = "dev"

var rootCmd = &cobra.Command{
	Use:   "fivemanage",
	Short: "Open-source, easy-to-use gaming-community management service.",
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		port := viper.GetInt("port")
		driver := viper.GetString("driver")
		dsn := viper.GetString("dsn")

		logger.New()

		slog.Info("starting Fivemanage application", slog.String("version", Version))

		otelShutdown, err := otel.SetupTracer(viper.GetBool("disable-otel"))
		if err != nil {
			slog.Error("failed to setup OpenTelemetry", slog.Any("error", err))
		}

		defer func() {
			slog.Info("attempting to shutdown OpenTelemetry...")
			otelCtx, otelCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer otelCancel()

			if shutdownErr := otelShutdown(otelCtx); shutdownErr != nil {
				slog.Error("failed to shutdown OpenTelemetry", slog.Any("error", shutdownErr))
			} else {
				slog.Info("OpenTelemetry shutdown successfully.")
			}
		}()

		// this is kinda confusion, 'db' and 'store'
		// maybe we shold call it like...connection and instance?
		// store makes no fucking sense atleast
		db := database.New(driver)
		store := db.Connect(dsn)

		// run migrations for postgres
		// and yes, it's ok to init this here
		migrate.InitMigration(cmd.Context(), store)
		migrate.AutoMigrate(cmd.Context(), store)

		// setup and run migrations for clickhouse
		chConfig := &clickhouse.Config{
			Host:     viper.GetString("clickhouse-host"),
			Username: viper.GetString("clickhouse-username"),
			Password: viper.GetString("clickhouse-password"),
			Database: viper.GetString("clickhouse-database"),
		}

		clickhouseEnabled := shouldInitClickhouse(chConfig.Host)
		if clickhouseEnabled {
			clickhouse.AutoMigrate(cmd.Context(), chConfig)
		}

		clickhouseClient := clickhouse.NewClient(chConfig, clickhouseEnabled)

		s3Provider := viper.GetString("s3-provider")
		storageLayer := storage.New(s3Provider)

		// im not sure if we need to exit here.
		// not all uers might want to use this for file uploads
		// this will also only apply for minio, since I dont even think you can manually create a bucket
		// on their shitty self hosted version
		if s3Provider == "minio" {
			if err := storageLayer.CreateBucket(cmd.Context()); err != nil {
				slog.Warn("failed to create default bucket", slog.Any("error", err))
			}
		}

		uploadLimits := file.UploadLimits{
			ImageBytes: viper.GetInt64("upload-max-image-bytes"),
			AudioBytes: viper.GetInt64("upload-max-audio-bytes"),
			VideoBytes: viper.GetInt64("upload-max-video-bytes"),
			FileBytes:  viper.GetInt64("upload-max-file-bytes"),
		}

		authService := auth.NewService(store)
		tokenService := token.NewService(store)
		fileService := file.NewService(store, storageLayer, uploadLimits)
		organizationService := organization.NewService(store, clickhouseClient)
		memberService := member.NewService(store)
		datsetService := dataset.NewService(store, clickhouseClient)
		logService := log.NewService(store, clickhouseClient, datsetService)
		systemService := system.NewService(Version, viper.GetString("bucket-domain"))

		memcache := cache.NewMemcache(5 * time.Minute)

		server := http.NewServer(
			authService,
			tokenService,
			fileService,
			organizationService,
			memberService,
			logService,
			datsetService,
			systemService,
			memcache,
		)

		err = authService.CreateAdminUser()
		if err != nil {
			slog.Error("failed to create admin user", slog.Any("error", err))
			return
		}

		srv := &nethttp.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: server,
		}

		go func() {
			otelzap.S().Infof("Fivemanage is running on port %d", port)
			if err := srv.ListenAndServe(); err != nil && err != nethttp.ErrServerClosed {
				slog.Error("listen", slog.Any("error", err))
				os.Exit(1)
			}
		}()

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		slog.Info("shutdown Server ...")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			slog.Error("server shutdown", slog.Any("error", err))
			os.Exit(1)
		}

		<-ctx.Done()
		slog.Info("server exiting")
	},
}

func init() {
	slog.Info("initializing Fivemanage...")

	err := godotenv.Load()
	if err != nil {
		slog.Warn("Error loading .env file. Probably becasue we're in production", slog.Any("error", err))
	}

	rootCmd.PersistentFlags().String("driver", "pg", "Database driver")
	rootCmd.Flags().Int("port", 8080, "Port to serve Fivemanage")
	rootCmd.Flags().String("dsn", "", "Database DSN")
	// clickhouse
	rootCmd.Flags().String("clickhouse-host", "localhost:9000", "Clickhouse host")
	rootCmd.Flags().String("clickhouse-username", "default", "Clickhouse username")
	rootCmd.Flags().String("clickhouse-password", "password", "Clickhouse password")
	rootCmd.Flags().String("clickhouse-database", "default", "Clickhouse database")
	// s3
	rootCmd.Flags().String("s3-provider", "minio", "S3 provider")
	rootCmd.Flags().String("bucket-domain", "", "Bucket domain for file storage")
	// otel
	rootCmd.Flags().Bool("disable-otel", false, "Disable OpenTelemetry")
	// upload limits in bytes
	rootCmd.Flags().Int64("upload-max-image-bytes", 10*1024*1024, "Maximum image upload size in bytes")
	rootCmd.Flags().Int64("upload-max-audio-bytes", 50*1024*1024, "Maximum audio upload size in bytes")
	rootCmd.Flags().Int64("upload-max-video-bytes", 100*1024*1024, "Maximum video upload size in bytes")
	rootCmd.Flags().Int64("upload-max-file-bytes", 100*1024*1024, "Maximum generic file upload size in bytes")

	// fuck me
	if err := viper.BindPFlag("port", rootCmd.Flags().Lookup("port")); err != nil {
		bindError(err)
	}
	if err := viper.BindPFlag("dsn", rootCmd.Flags().Lookup("dsn")); err != nil {
		bindError(err)
	}
	if err := viper.BindEnv("driver", "DB_DRIVER"); err != nil {
		bindError(err)
	}
	if err := viper.BindEnv("port", "PORT"); err != nil {
		bindError(err)
	}
	if err := viper.BindEnv("dsn", "DSN"); err != nil {
		bindError(err)
	}
	if err := viper.BindEnv("clickhouse-host", "CLICKHOUSE_HOST"); err != nil {
		bindError(err)
	}
	if err := viper.BindEnv("clickhouse-username", "CLICKHOUSE_USERNAME"); err != nil {
		bindError(err)
	}
	if err := viper.BindEnv("clickhouse-password", "CLICKHOUSE_PASSWORD"); err != nil {
		bindError(err)
	}
	if err := viper.BindEnv("clickhouse-database", "CLICKHOUSE_DATABASE"); err != nil {
		bindError(err)
	}
	if err := viper.BindEnv("s3-provider", "S3_PROVIDER"); err != nil {
		bindError(err)
	}
	if err := viper.BindEnv("bucket-domain", "BUCKET_DOMAIN"); err != nil {
		bindError(err)
	}
	if err := viper.BindEnv("disable-otel", "DISABLE_OTEL"); err != nil {
		bindError(err)
	}
	if err := viper.BindPFlag("upload-max-image-bytes", rootCmd.Flags().Lookup("upload-max-image-bytes")); err != nil {
		bindError(err)
	}
	if err := viper.BindPFlag("upload-max-audio-bytes", rootCmd.Flags().Lookup("upload-max-audio-bytes")); err != nil {
		bindError(err)
	}
	if err := viper.BindPFlag("upload-max-video-bytes", rootCmd.Flags().Lookup("upload-max-video-bytes")); err != nil {
		bindError(err)
	}
	if err := viper.BindPFlag("upload-max-file-bytes", rootCmd.Flags().Lookup("upload-max-file-bytes")); err != nil {
		bindError(err)
	}
	if err := viper.BindEnv("upload-max-image-bytes", "UPLOAD_MAX_IMAGE_BYTES"); err != nil {
		bindError(err)
	}
	if err := viper.BindEnv("upload-max-audio-bytes", "UPLOAD_MAX_AUDIO_BYTES"); err != nil {
		bindError(err)
	}
	if err := viper.BindEnv("upload-max-video-bytes", "UPLOAD_MAX_VIDEO_BYTES"); err != nil {
		bindError(err)
	}
	if err := viper.BindEnv("upload-max-file-bytes", "UPLOAD_MAX_FILE_BYTES"); err != nil {
		bindError(err)
	}

	rootCmd.AddCommand(migrate.RootCmd)
	migrate.RootCmd.AddCommand(
		migrate.InitCmd,
		migrate.MigrateCmd,
		migrate.CreateMigrationCmd,
		migrate.UnlockCmd,
		migrate.LockCmd,
	)
}

func bindError(err error) {
	if err != nil {
		slog.Error("failed to bind env", slog.Any("error", err))
		os.Exit(1)
	}
}

func shouldInitClickhouse(clickhouseHost string) bool {
	return clickhouseHost != ""
}

// @title           Fivemanage Lite API
// @version         1.0
// @description     Open-source, easy-to-use gaming-community management service.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email   support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api
func main() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
