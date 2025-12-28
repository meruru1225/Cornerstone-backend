package minio

import (
	"Cornerstone/internal/api/config"
	"context"
	"fmt"
	log "log/slog"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
)

var (
	// Client 全局 MinIO 客户端实例
	Client *minio.Client
	// MainBucket 主要存储桶
	MainBucket string
	// TempBucket 临时存储桶
	TempBucket string
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
	MainBucket = cfg.MainBucket
	TempBucket = cfg.TempBucket
	return EnsureTempBucketLifecycle(ctx)
}

func EnsureTempBucketLifecycle(ctx context.Context) error {
	lcConfig, err := Client.GetBucketLifecycle(ctx, TempBucket)
	if err != nil {
		lcConfig = lifecycle.NewConfiguration()
	}

	const targetDays = 1
	hasTargetRule := false
	for _, rule := range lcConfig.Rules {
		// 判定条件：状态开启 + 全桶匹配(无Prefix) + 过期天数为1
		if rule.Status == "Enabled" &&
			rule.Expiration.Days == targetDays &&
			rule.RuleFilter.Prefix == "" {
			hasTargetRule = true
			log.Info("检测到已存在兼容的过期策略", "ruleID", rule.ID)
			break
		}
	}

	// 如果没找到符合要求的规则，则添加一条带有固定 ID 的规则
	if !hasTargetRule {
		newRule := lifecycle.Rule{
			ID:     "SystemAutoDeleteRule",
			Status: "Enabled",
			Expiration: lifecycle.Expiration{
				Days: targetDays,
			},
		}
		lcConfig.Rules = append(lcConfig.Rules, newRule)

		err = Client.SetBucketLifecycle(ctx, TempBucket, lcConfig)
		if err != nil {
			return fmt.Errorf("设置生命周期失败: %w", err)
		}
		log.Info("已自动补全 TempBucket 的 1 天过期策略")
	}

	return nil
}
