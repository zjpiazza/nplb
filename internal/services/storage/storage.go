package storage

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	// "github.com/zjpiazza/nplb/internal/config"
)

// R2Storage provides methods for interacting with Cloudflare R2.
type R2Storage struct {
	client *s3.Client
	bucket string
}

// NewR2Storage creates a new R2 storage client.
func NewR2Storage(accountID, accessKeyID, secretAccessKey, bucket string) (*R2Storage, error) {
	r2Resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: "https://<CLOUDFLARE_ACCOUNT_ID>.r2.cloudflarestorage.com",
		}, nil
	})

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithEndpointResolverWithOptions(r2Resolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKeyID,
			secretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg)

	return &R2Storage{
		client: client,
		bucket: bucket,
	}, nil
}

// UploadFile uploads a file to R2.
func (r *R2Storage) UploadFile(ctx context.Context, filePath, objectKey string) error {
	// TODO: Implement file upload logic.
	return nil
}
