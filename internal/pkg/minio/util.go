package minio

import (
	"Cornerstone/internal/api/config"
	"context"
	"fmt"
	"io"
	"net/url"

	"github.com/minio/minio-go/v7"
)

// UploadFile 上传文件到MinIO
func UploadFile(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) (string, error) {
	if Client == nil {
		return "", fmt.Errorf("minio client is not initialized")
	}

	uploadInfo, err := Client.PutObject(ctx, MainBucket, objectName, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	return uploadInfo.Key, nil
}

// UploadTempFile 上传文件到MinIO，临时存储，默认24h过期
func UploadTempFile(ctx context.Context, objectName string, reader io.Reader, contentType string) (string, error) {
	if Client == nil {
		return "", fmt.Errorf("minio client is not initialized")
	}

	uploadInfo, err := Client.PutObject(ctx, TempBucket, objectName, reader, -1, minio.PutObjectOptions{
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

	err := Client.RemoveObject(ctx, MainBucket, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// GetPublicURL 获取文件的公共访问URL
func GetPublicURL(objectName string) string {
	cfg := config.Cfg.MinIO
	// 构造公共URL
	safeObjectName := url.PathEscape(objectName)
	publicURL := fmt.Sprintf("https://%s/%s/%s", cfg.ExternalEndpoint, MainBucket, safeObjectName)
	return publicURL
}

func GetInternalFileURL(objectName string) string {
	cfg := config.Cfg.MinIO
	safeObjectName := url.PathEscape(objectName)
	internalURL := fmt.Sprintf("http://%s/%s/%s", cfg.InternalEndpoint, MainBucket, safeObjectName)
	return internalURL
}

// GetTempFileURL 获取临时文件的公共访问URL
func GetTempFileURL(objectName string, external bool) string {
	cfg := config.Cfg.MinIO

	var endpoint string
	var useSSL bool
	if external {
		endpoint = cfg.ExternalEndpoint
		useSSL = true
	} else {
		endpoint = cfg.InternalEndpoint
		useSSL = cfg.InternalUseSSL
	}

	protocol := "http"
	if useSSL {
		protocol = "https"
	}
	safeObjectName := url.PathEscape(objectName)
	tempURL := fmt.Sprintf("%s://%s/%s/%s", protocol, endpoint, TempBucket, safeObjectName)
	return tempURL
}
