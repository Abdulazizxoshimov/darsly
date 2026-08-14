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
	// Get obyektni o'qish uchun ochadi (qayta kodlash ishchisi yuklab oladi).
	// Chaqiruvchi `Close()` qilishi SHART.
	Get(ctx context.Context, objectName string) (io.ReadCloser, error)
	PresignedURL(ctx context.Context, objectName string, expires time.Duration) (string, error)
	// PresignedPutURL — klient (telefon) to'g'ridan-to'g'ri obyektni YUKLASHi
	// uchun imzolangan PUT havolasi. Katta fayllar (dars yozuvi ~1 GB) uchun
	// server-proxy multipart o'rniga: telefon → MinIO to'g'ridan.
	PresignedPutURL(ctx context.Context, objectName string, expires time.Duration) (string, error)
	// Stat — obyekt mavjudligini va o'lchamini qaytaradi (client upload'ni
	// TASDIQLASH uchun: telefon "yukladim" desa, backend haqiqatan borligini
	// va o'lchamini tekshiradi). Obyekt yo'q → xato.
	Stat(ctx context.Context, objectName string) (int64, error)
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
func (nopClient) Get(_ context.Context, _ string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("minio: not configured")
}
func (nopClient) PresignedURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", fmt.Errorf("minio: not configured")
}
func (nopClient) PresignedPutURL(_ context.Context, _ string, _ time.Duration) (string, error) {
	return "", fmt.Errorf("minio: not configured")
}
func (nopClient) Stat(_ context.Context, _ string) (int64, error) {
	return 0, fmt.Errorf("minio: not configured")
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

func (c *minioClient) Get(ctx context.Context, objectName string) (io.ReadCloser, error) {
	obj, err := c.mc.GetObject(ctx, c.bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("minio: get %q: %w", objectName, err)
	}
	return obj, nil
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

func (c *minioClient) PresignedPutURL(ctx context.Context, objectName string, expires time.Duration) (string, error) {
	client := c.mc
	if c.presign != nil {
		client = c.presign // ochiq endpointga imzolash (telefon yeta oladigan)
	}
	u, err := client.PresignedPutObject(ctx, c.bucket, objectName, expires)
	if err != nil {
		return "", fmt.Errorf("minio: presign put %q: %w", objectName, err)
	}
	return u.String(), nil
}

func (c *minioClient) Stat(ctx context.Context, objectName string) (int64, error) {
	info, err := c.mc.StatObject(ctx, c.bucket, objectName, minio.StatObjectOptions{})
	if err != nil {
		return 0, fmt.Errorf("minio: stat %q: %w", objectName, err)
	}
	return info.Size, nil
}

func (c *minioClient) Delete(ctx context.Context, objectName string) error {
	if err := c.mc.RemoveObject(ctx, c.bucket, objectName, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("minio: delete %q: %w", objectName, err)
	}
	return nil
}
