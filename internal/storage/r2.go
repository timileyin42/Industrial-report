// Package storage wraps Cloudflare R2 (S3-compatible) for storing
// generated export files. R2 exposes the standard S3 API against a
// per-account endpoint, so this is the AWS SDK v2 S3 client pointed at
// that endpoint rather than a bespoke HTTP client.
package storage

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Client struct {
	s3      *s3.Client
	presign *s3.PresignClient
	bucket  string
}

// NewFromEnv returns nil (not an error) when R2 isn't configured — the
// caller (the export job worker) treats a nil client as "storage isn't
// set up yet" and fails jobs with a clear message rather than the
// process refusing to start. Object storage is additive infrastructure,
// unlike DATABASE_URL/JWT_SECRET which the whole API depends on.
func NewFromEnv(ctx context.Context) (*Client, error) {
	accountID := os.Getenv("R2_ACCOUNT_ID")
	accessKey := os.Getenv("R2_ACCESS_KEY_ID")
	secretKey := os.Getenv("R2_SECRET_ACCESS_KEY")
	bucket := os.Getenv("R2_BUCKET")
	if accountID == "" || accessKey == "" || secretKey == "" || bucket == "" {
		return nil, nil
	}

	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("auto"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		return nil, err
	}

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true
	})

	return &Client{s3: s3Client, presign: s3.NewPresignClient(s3Client), bucket: bucket}, nil
}

func (c *Client) Upload(ctx context.Context, key string, data []byte, contentType string) error {
	_, err := c.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	return err
}

// PresignGet returns a time-limited download URL — generated export
// files are never served directly through the API process.
func (c *Client) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	req, err := c.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", err
	}
	return req.URL, nil
}
