package http

import (
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"github.com/fivemanage/lite/internal/http/internalapi"
	internalmiddleware "github.com/fivemanage/lite/internal/http/middleware"
	"github.com/fivemanage/lite/internal/http/publicapi"
	_validator "github.com/fivemanage/lite/internal/http/validator"
	"github.com/fivemanage/lite/internal/service/auth"
	"github.com/fivemanage/lite/internal/service/dataset"
	"github.com/fivemanage/lite/internal/service/file"
	"github.com/fivemanage/lite/internal/service/log"
	"github.com/fivemanage/lite/internal/service/member"
	"github.com/fivemanage/lite/internal/service/organization"
	"github.com/fivemanage/lite/internal/service/system"
	"github.com/fivemanage/lite/internal/service/token"
	"github.com/fivemanage/lite/pkg/cache"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4/middleware"
)

//go:embed dist/*
var webContent embed.FS

type Server struct {
	Engine *echo.Echo
}

// TODO: Add sentry for monitoring. There should be an opt-out option.
func NewServer(
	authService *auth.Service,
	tokenService *token.Service,
	fileService *file.Service,
	organizationService *organization.Service,
	memberService *member.Service,
	logService *log.Service,
	datasetService *dataset.Service,
	systemService *system.Service,
	memcache *cache.Cache,
) *echo.Echo {
	app := echo.New()
	app.Debug = true

	app.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOriginFunc: func(origin string) (bool, error) {
			return true, nil
		},
		AllowHeaders:     []string{"*"},
		AllowCredentials: true,
	}))

	app.Use(otelecho.Middleware("lite-api"))
	app.Use(middleware.Recover())
	app.Use(internalmiddleware.AppContext)
	app.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(20)))

	app.Validator = &_validator.CustomValidator{Validator: validator.New()}

	app.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		Filesystem: getFileSystem("dist"),
		HTML5:      true,
		Skipper: func(c echo.Context) bool {
			path := c.Request().URL.Path
			return strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/file/")
		},
	}))

	app.GET("/file/:organizationId/:fileName", publicapi.ProxyFileHandler(fileService))

	apiGroup := app.Group("/api")
	dashApi := apiGroup.Group("/dash")

	internalapi.Add(
		dashApi,
		authService,
		tokenService,
		organizationService,
		memberService,
		fileService,
		datasetService,
		systemService,
	)
	publicapi.Add(apiGroup, fileService, tokenService, logService, memcache)

	return app
}

func getFileSystem(path string) http.FileSystem {
	fs, err := fs.Sub(webContent, path)
	if err != nil {
		panic(err)
	}

	slog.Info(fmt.Sprintf("serving static files from %s", path))

	return http.FS(fs)
}
