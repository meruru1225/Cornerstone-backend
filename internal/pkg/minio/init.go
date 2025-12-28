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

	var endpoint string
	var useSSL bool
	if cfg.InternalEndpoint != "" {
		endpoint = cfg.InternalEndpoint
		useSSL = cfg.InternalUseSSL
	} else {
		endpoint = cfg.ExternalEndpoint
		useSSL = true
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: useSSL,
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
