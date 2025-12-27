package minio

import (
	"Cornerstone/internal/api/config"
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
)

// UploadFile 上传文件到MinIO
func UploadFile(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) (string, error) {
	if Client == nil {
		return "", fmt.Errorf("minio client is not initialized")
	}
	bucket := BucketName

	uploadInfo, err := Client.PutObject(ctx, bucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	return uploadInfo.Key, nil
}

// DeleteFile 删除MinIO中的文件
func DeleteFile(ctx context.Context, objectName string) error {
	if Client == nil {
		return fmt.Errorf("minio client is not initialized")
	}
	bucket := BucketName

	err := Client.RemoveObject(ctx, bucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// GetPublicURL 获取文件的公共访问URL
func GetPublicURL(objectName string) string {
	cfg := config.Cfg.MinIO
	endpoint := cfg.Endpoint

	// 构造公共URL
	protocol := "http"
	if cfg.UseSSL {
		protocol = "https"
	}

	publicURL := fmt.Sprintf("%s://%s/%s/%s", protocol, endpoint, BucketName, objectName)
	return publicURL
}
