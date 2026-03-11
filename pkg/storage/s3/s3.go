package s3

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/sirupsen/logrus"
)

const (
	MinIO = "minio"
	R2    = "r2"
	S3    = "s3"
)

type Storage struct {
	client *s3.Client
	bucket string
}

func New(s3Provider string) *Storage {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(os.Getenv("AWS_REGION")),
	)
	if err != nil {
		log.Fatalf("failed to load default config: %v", err)
	}

	// s3 will automatically use the following environment variables:
	// AWS_ACCESS_KEY_ID
	// AWS_SECRET_ACCESS_KEY
	// these are custom:
	// AWS_ENDPOINT
	// AWS_BUCKET
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(os.Getenv("AWS_ENDPOINT"))

		// does this only apply if minio is self hosted on the same server?
		if s3Provider == MinIO {
			o.UsePathStyle = true
		}
	})

	bucket := os.Getenv("AWS_BUCKET")
	if bucket == "" {
		logrus.Fatalln("environment variable: AWS_BUCKET is not set")
	}

	return &Storage{
		client: client,
		bucket: bucket,
	}
}

// UploadFile will both upload and replace as long as the key is the same
func (r *Storage) UploadFile(ctx context.Context, file io.Reader, key, contentType string) error {
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		fmt.Println("Error uploading file to S3 bucket", err)
		return err
	}

	return nil
}

func (r *Storage) GetFile(ctx context.Context, key string) (io.ReadCloser, string, int64, error) {
	output, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var noSuchKey *types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			return nil, "", 0, os.ErrNotExist
		}

		return nil, "", 0, err
	}

	contentType := ""
	if output.ContentType != nil {
		contentType = *output.ContentType
	}

	var contentLength int64
	if output.ContentLength != nil {
		contentLength = *output.ContentLength
	}

	return output.Body, contentType, contentLength, nil
}

func (r *Storage) DeleteFile() error {
	_, err := r.client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{})
	if err != nil {
		return err
	}

	return nil
}

func (r *Storage) CreateBucket(ctx context.Context) error {
	_, err := r.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(r.bucket),
	})
	if err == nil {
		slog.Info(fmt.Sprintf("bucket [%s] already exists. skipping creation and policy application", r.bucket))
		return nil // Bucket exists, nothing more to do.
	}

	var nfe *types.NotFound
	if !errors.As(err, &nfe) {
		return fmt.Errorf("failed to check for bucket: %w", err)
	}

	_, err = r.client.CreateBucket(context.TODO(), &s3.CreateBucketInput{
		Bucket: aws.String(r.bucket),
	})
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	slog.Info(fmt.Sprintf("Created bucket: %s", r.bucket))

	// i hate this bro
	bucketPolicy := map[string]any{
		"Version": "2012-10-17",
		"Statement": []map[string]any{
			{
				"Sid":       "PublicReadGetObject",
				"Effect":    "Allow",
				"Principal": "*",
				"Action":    "s3:GetObject",
				"Resource":  fmt.Sprintf("arn:aws:s3:::%s/*", r.bucket),
			},
		},
	}

	policyBytes, err := json.Marshal(bucketPolicy)
	if err != nil {
		return fmt.Errorf("failed to marshal bucket policy: %w", err)
	}

	_, err = r.client.PutBucketPolicy(context.TODO(), &s3.PutBucketPolicyInput{
		Bucket: aws.String(r.bucket),
		Policy: aws.String(string(policyBytes)),
	})
	if err != nil {
		return fmt.Errorf("failed to apply bucket policy: %w", err)
	}
	logrus.Info("Successfully applied public-read bucket policy.")

	return nil
}
