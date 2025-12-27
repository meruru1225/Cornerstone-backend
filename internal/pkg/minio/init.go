package minio

import (
	"Cornerstone/internal/api/config"
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var (
	// Client 是全局的 MinIO 客户端实例
	Client *minio.Client
	// BucketName 是存储桶名称
	BucketName string
)

// Init 初始化 MinIO 客户端
func Init() error {
	cfg := config.Cfg.MinIO

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return fmt.Errorf("failed to initialize minio client: %w", err)
	}

	ctx := context.Background()
	_, err = client.ListBuckets(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to minio server: %w", err)
	}
	Client = client
	BucketName = cfg.Bucket
	return nil
}
