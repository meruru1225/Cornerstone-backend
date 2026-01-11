package job

import (
	"Cornerstone/internal/api/dto"
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/minio"
	"Cornerstone/internal/pkg/redis"
	"context"
	"encoding/json"
	log "log/slog"
	"time"
)

type MediaCleanupJob struct{}

func NewMediaCleanupJob() *MediaCleanupJob {
	return &MediaCleanupJob{}
}

func (s *MediaCleanupJob) Run() {
	ctx := context.Background()
	log.Info("start media cleanup job")

	allMedia, err := redis.HGetAll(ctx, consts.MediaTempKey)
	if err != nil {
		log.Error("failed to get media temp hash", "err", err)
		return
	}

	now := time.Now().Unix()
	expiration := int64(24 * 60 * 60)
	count := 0

	for fileKey, val := range allMedia {
		var meta dto.MediaTempMetadata
		if err := json.Unmarshal([]byte(val), &meta); err != nil {
			log.Warn("invalid media meta format", "fileKey", fileKey)
			continue
		}

		if now-meta.CreatedAt > expiration {
			if err = minio.DeleteFile(ctx, fileKey); err != nil {
				log.Error("failed to delete expired file from minio", "fileKey", fileKey, "err", err)
				continue
			}

			if err = redis.HDel(ctx, consts.MediaTempKey, fileKey); err != nil {
				log.Error("failed to remove media token from redis", "fileKey", fileKey, "err", err)
			}

			count++
			log.Info("cleanup expired media resource", "fileKey", fileKey, "mime", meta.MimeType)
		}
	}

	if count > 0 {
		log.Info("media cleanup job finished", "cleaned_count", count)
	}
}
