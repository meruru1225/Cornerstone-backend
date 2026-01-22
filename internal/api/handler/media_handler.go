package handler

import (
	"Cornerstone/internal/api/dto"
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/minio"
	"Cornerstone/internal/pkg/redis"
	"Cornerstone/internal/pkg/response"
	"Cornerstone/internal/pkg/util"
	"Cornerstone/internal/service"
	log "log/slog"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
	"github.com/google/uuid"
)

type MediaHandler struct{}

func NewMediaHandler() *MediaHandler {
	return &MediaHandler{}
}

func (s *MediaHandler) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, service.ErrParamInvalid)
		return
	}

	reader, err := file.Open()
	if err != nil {
		response.Error(c, service.ErrParamInvalid)
		return
	}
	defer func() { _ = reader.Close() }()

	contentType, err := util.GetSafeContentType(reader)
	log.InfoContext(c.Request.Context(), "contentType", contentType)

	isImage := strings.HasPrefix(contentType, consts.MimePrefixImage)
	isVideo := strings.HasPrefix(contentType, consts.MimePrefixVideo)
	isAudio := strings.HasPrefix(contentType, consts.MimePrefixAudio)
	if !isImage && !isVideo && !isAudio {
		response.Error(c, service.ErrFileNotSupported)
		return
	}

	ext := path.Ext(file.Filename)
	objectName := time.Now().Format("2006/01/02/") + uuid.NewString() + ext

	fileKey, err := minio.UploadFile(c.Request.Context(), objectName, reader, file.Size, contentType)
	if err != nil {
		log.ErrorContext(c, "MinIO upload failed", "err", err)
		response.Error(c, service.UnExpectedError)
		return
	}

	publicUrl := minio.GetPublicURL(fileKey)
	var width, height int
	var duration float64

	if isImage || isVideo {
		w, h, err := util.GetDimensions(c.Request.Context(), publicUrl)
		if err == nil {
			width, height = w, h
		} else {
			log.WarnContext(c, "failed to get dimensions via ffprobe", "url", publicUrl, "err", err)
		}
	}

	if isVideo || isAudio {
		dur, err := util.GetDuration(c.Request.Context(), publicUrl)
		if err == nil {
			duration = dur
		} else {
			log.WarnContext(c, "failed to get duration via ffprobe", "url", publicUrl, "err", err)
		}
	}

	meta := dto.MediaTempMetadata{
		MimeType:  contentType,
		Width:     width,
		Height:    height,
		Duration:  duration,
		CreatedAt: time.Now().Unix(),
	}
	metaBytes, _ := json.Marshal(meta)
	_ = redis.HSet(c.Request.Context(), consts.MediaTempKey, fileKey, string(metaBytes))

	res := map[string]interface{}{
		"url":      fileKey,
		"mime":     contentType,
		"width":    width,
		"height":   height,
		"duration": duration,
		"size":     file.Size,
		"original": file.Filename,
	}

	log.InfoContext(c, "media upload success and metadata cached", "fileKey", fileKey, "type", contentType)
	response.Success(c, res)
}
