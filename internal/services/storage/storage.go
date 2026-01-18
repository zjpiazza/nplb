package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.uber.org/zap"
)

// Storage is an interface for storage operations.
type Storage interface {
	Upload(ctx context.Context, key string, content []byte, contentType string) error
	UploadFile(ctx context.Context, localPath, key string) error
	Download(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) ([]string, error)
	GetPublicURL(key string) string
}

// R2Storage provides methods for interacting with Cloudflare R2.
type R2Storage struct {
	client    *s3.Client
	bucket    string
	publicURL string
	logger    *zap.Logger
}

// NewR2Storage creates a new R2 storage client.
func NewR2Storage(accountID, accessKeyID, secretAccessKey, bucket, publicURL string, logger *zap.Logger) (*R2Storage, error) {
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: endpoint,
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithEndpointResolverWithOptions(r2Resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKeyID,
			secretAccessKey,
			"",
		)),
		config.WithRegion("auto"), // R2 uses "auto" region
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(cfg)

	return &R2Storage{
		client:    client,
		bucket:    bucket,
		publicURL: publicURL,
		logger:    logger,
	}, nil
}

// Upload uploads content to R2.
func (r *R2Storage) Upload(ctx context.Context, key string, content []byte, contentType string) error {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(r.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(content),
		ContentType: aws.String(contentType),
	}

	_, err := r.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to upload to R2: %w", err)
	}

	r.logger.Debug("uploaded file to R2",
		zap.String("key", key),
		zap.Int("size", len(content)),
		zap.String("content_type", contentType),
	)

	return nil
}

// UploadFile uploads a local file to R2.
func (r *R2Storage) UploadFile(ctx context.Context, localPath, key string) error {
	content, err := io.ReadAll(strings.NewReader(localPath))
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	contentType := getContentType(key)
	return r.Upload(ctx, key, content, contentType)
}

// Download downloads content from R2.
func (r *R2Storage) Download(ctx context.Context, key string) ([]byte, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	}

	result, err := r.client.GetObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to download from R2: %w", err)
	}
	defer result.Body.Close()

	content, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return content, nil
}

// Delete deletes an object from R2.
func (r *R2Storage) Delete(ctx context.Context, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucket),
		Key:    aws.String(key),
	}

	_, err := r.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete from R2: %w", err)
	}

	return nil
}

// List lists objects with a given prefix.
func (r *R2Storage) List(ctx context.Context, prefix string) ([]string, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(r.bucket),
		Prefix: aws.String(prefix),
	}

	var keys []string
	paginator := s3.NewListObjectsV2Paginator(r.client, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}

		for _, obj := range page.Contents {
			keys = append(keys, *obj.Key)
		}
	}

	return keys, nil
}

// GetPublicURL returns the public URL for an object.
func (r *R2Storage) GetPublicURL(key string) string {
	if r.publicURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimSuffix(r.publicURL, "/"), key)
	}
	return fmt.Sprintf("https://%s.r2.dev/%s", r.bucket, key)
}

// getContentType returns the content type based on file extension.
func getContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".deb":
		return "application/vnd.debian.binary-package"
	case ".gz":
		return "application/gzip"
	case ".xz":
		return "application/x-xz"
	case ".gpg", ".sig":
		return "application/pgp-signature"
	case ".asc":
		return "application/pgp-keys"
	default:
		return "text/plain"
	}
}
