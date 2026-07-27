package minio

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/zoom/darsly/internal/pkg/config"
)

type Client interface {
	Upload(ctx context.Context, objectName, contentType string, reader io.Reader, size int64) (string, error)
	PresignedURL(ctx context.Context, objectName string, expires time.Duration) (string, error)
	Delete(ctx context.Context, objectName string) error
	EnsureBucket(ctx context.Context) error
}

type minioClient struct {
	mc      *minio.Client
	presign *minio.Client // presigned URL'lar uchun (ochiq endpoint); nil → mc ishlatiladi
	bucket  string
}

// nopClient is returned when MinIO is unavailable; all operations return a descriptive error.
type nopClient struct{}

func (nopClient) EnsureBucket(_ context.Context) error { return nil }
func (nopClient) Upload(_ context.Context, _, _ string, _ io.Reader, _ int64) (string, error) {
	return "", fmt.Errorf("minio: not configured")
}
func (nopClient) PresignedURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", fmt.Errorf("minio: not configured")
}
func (nopClient) Delete(_ context.Context, _ string) error {
	return fmt.Errorf("minio: not configured")
}

func NewNop() Client { return nopClient{} }

func New(cfg config.MinioConfig) (Client, error) {
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio: connect: %w", err)
	}
	c := &minioClient{mc: mc, bucket: cfg.Bucket}

	// Presigned URL'lar ochiq endpointga imzolanishi kerak (tashqi klient yeta oladigan).
	// Imzo host header'ni qamrab oladi — shuning uchun alohida client bilan imzolanadi.
	if cfg.PublicEndpoint != "" {
		pc, err := minio.New(cfg.PublicEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
			Secure: cfg.PublicUseSSL,
		})
		if err != nil {
			return nil, fmt.Errorf("minio: public endpoint connect: %w", err)
		}
		c.presign = pc
	}
	return c, nil
}

func (c *minioClient) EnsureBucket(ctx context.Context) error {
	exists, err := c.mc.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("minio: bucket check: %w", err)
	}
	if exists {
		return nil
	}
	if err := c.mc.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("minio: make bucket: %w", err)
	}
	return nil
}

func (c *minioClient) Upload(ctx context.Context, objectName, contentType string, reader io.Reader, size int64) (string, error) {
	_, err := c.mc.PutObject(ctx, c.bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("minio: upload %q: %w", objectName, err)
	}
	return objectName, nil
}

func (c *minioClient) PresignedURL(ctx context.Context, objectName string, expires time.Duration) (string, error) {
	client := c.mc
	if c.presign != nil {
		client = c.presign // ochiq endpointga imzolash
	}
	params := url.Values{}
	u, err := client.PresignedGetObject(ctx, c.bucket, objectName, expires, params)
	if err != nil {
		return "", fmt.Errorf("minio: presign %q: %w", objectName, err)
	}
	return u.String(), nil
}

func (c *minioClient) Delete(ctx context.Context, objectName string) error {
	if err := c.mc.RemoveObject(ctx, c.bucket, objectName, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("minio: delete %q: %w", objectName, err)
	}
	return nil
}
