package kafka

import (
	"Cornerstone/internal/model"
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/es"
	"Cornerstone/internal/pkg/llm"
	"Cornerstone/internal/pkg/minio"
	"Cornerstone/internal/pkg/redis"
	"Cornerstone/internal/pkg/util"
	"Cornerstone/internal/repository"
	"context"
	"fmt"
	"io"
	log "log/slog"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

var ErrAuditDenyTriggered = errors.New("audit deny detected, cancelling other batches")

type Result struct {
	sync.Mutex
	StopChan       chan struct{}
	MainTags       []string
	Tags           []string
	allPendingUrls []string
	maxStatus      int32
}

type PostsHandler struct {
	userDBRepo repository.UserRepo
	postDBRepo repository.PostRepo
	postESRepo es.PostRepo
}

func NewPostsHandler(userDBRepo repository.UserRepo, postDBRepo repository.PostRepo, postESRepo es.PostRepo) *PostsHandler {
	return &PostsHandler{
		userDBRepo: userDBRepo,
		postDBRepo: postDBRepo,
		postESRepo: postESRepo,
	}
}

func (s *PostsHandler) Setup(sarama.ConsumerGroupSession) error {
	log.Info("post consumer setup")
	return nil
}

func (s *PostsHandler) Cleanup(sarama.ConsumerGroupSession) error {
	log.Info("post consumer cleanup")
	return nil
}

func (s *PostsHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	log.Info("topic-post consume claim")
	err := pullMessageBatch(session, claim, s.logic)
	if err != nil {
		log.Error("topic-post process batch error", "err", err)
		return err
	}
	log.Info("topic-post consume claim end")
	return nil
}

func (s *PostsHandler) logic(ctx context.Context, msg *sarama.ConsumerMessage) error {
	canalMsg, err := ToCanalMessage(msg, "posts")
	if err != nil {
		return err
	}

	post, err := s.toESModel(canalMsg)
	if err != nil {
		return err
	}

	if post == nil {
		return s.postESRepo.DeletePost(ctx, StrToUint64(canalMsg.Data[0]["id"]))
	}

	// 查询关联表，获取关联的媒体
	medias, err := s.postDBRepo.GetPostMedias(ctx, post.ID)
	if err != nil {
		return err
	}
	for _, media := range medias {
		post.Media = append(post.Media, es.PostMediaES{
			Type:     media.FileType,
			URL:      media.MediaURL,
			Cover:    media.CoverURL,
			Width:    media.Width,
			Height:   media.Height,
			Duration: media.Duration,
		})
	}

	// 没有内容变更，直接覆写ES
	if !s.checkContentIsChange(canalMsg) {
		getById, err := s.postESRepo.GetPostById(ctx, post.ID)
		if err != nil {
			return err
		}
		if getById != nil {
			post.UserTags = getById.UserTags
			post.AITags = getById.AITags
		}
		return s.getUserDetailAndIndexES(ctx, post, canalMsg.TS)
	}

	// 获取用户设定的标签
	tags := util.ExtractTags(post.Content)
	if len(tags) > 0 {
		post.UserTags = tags
	}

	// LLM 处理内容
	r := &Result{
		StopChan:       make(chan struct{}),
		MainTags:       make([]string, 0),
		Tags:           make([]string, 0),
		allPendingUrls: make([]string, 0),
		maxStatus:      int32(llm.ContentSafePass),
	}
	var closeOnce sync.Once
	safeClose := func() { closeOnce.Do(func() { close(r.StopChan) }) }
	waitDone := make(chan error, 1)
	defer safeClose()
	toLLMContent := &llm.Content{
		Title:   post.Title,
		Content: post.Content,
	}
	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return s.llmProcessContent(gCtx, toLLMContent, r, safeClose)
	})
	g.Go(func() error {
		return s.llmProcessMedia(gCtx, medias, r, safeClose)
	})

	go func() {
		waitDone <- g.Wait()
	}()
	select {
	case err = <-waitDone:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	case <-r.StopChan:
		gCtx.Done()
		return s.postDBRepo.UpdatePostStatus(ctx, post.ID, llm.ContentSafeDeny)
	}

	// LLM 进行语义聚合
	aggress, err := llm.AggressiveTag(ctx, &llm.TagAggressive{
		MainTags: r.MainTags,
		Tags:     r.Tags,
	})
	if err != nil {
		return err
	} else {
		post.AITags = aggress.Tags
		if aggress.MainTag != "" {
			if err = s.postDBRepo.UpsertPostTag(ctx, post.ID, aggress.MainTag); err != nil {
				return err
			}
		}
	}

	post.Status = int(atomic.LoadInt32(&r.maxStatus))
	if err = s.postDBRepo.UpdatePostStatus(ctx, post.ID, post.Status); err != nil {
		return err
	}

	return s.getUserDetailAndIndexES(ctx, post, canalMsg.TS)
}

func (s *PostsHandler) toESModel(message *CanalMessage) (*es.PostES, error) {
	if len(message.Data) == 0 {
		return nil, fmt.Errorf("canal message data is empty")
	}

	row := message.Data[0]

	if row["is_deleted"] == "1" {
		return nil, nil
	}

	return &es.PostES{
		ID:            StrToUint64(row["id"]),
		UserID:        StrToUint64(row["user_id"]),
		Status:        StrToInt(row["status"]),
		Title:         StrToString(row["title"]),
		Content:       StrToString(row["content"]),
		CreatedAt:     StrToDateTime(row["created_at"]),
		UpdatedAt:     StrToDateTime(row["updated_at"]),
		LikesCount:    StrToInt(row["likes_count"]),
		CommentsCount: StrToInt(row["comments_count"]),
		CollectsCount: StrToInt(row["collects_count"]),
	}, nil
}

func (s *PostsHandler) getUserDetailAndIndexES(ctx context.Context, post *es.PostES, timeStamp int64) error {
	// 导入用户信息并覆写ES，此处加锁避免并发一致性问题
	newUUID, err := uuid.NewUUID()
	if err != nil {
		return err
	}
	lockKey := consts.UserDetailLock + strconv.FormatUint(post.UserID, 10)
	_, err = redis.TryLock(ctx, lockKey, newUUID.String(), time.Second*5, -1)
	if err != nil {
		return err
	}
	defer redis.UnLock(ctx, lockKey, newUUID.String())

	users, err := s.userDBRepo.GetUserSimpleInfoByIds(ctx, []uint64{post.UserID})
	if err != nil {
		return err
	}
	if len(users) == 0 {
		return errors.New("user not found")
	}
	post.UserNickname = users[0].Nickname
	post.UserAvatar = users[0].AvatarURL
	return s.postESRepo.IndexPost(ctx, post, timeStamp)
}

func (s *PostsHandler) checkContentIsChange(message *CanalMessage) bool {
	if message.Type == INSERT {
		return true
	}
	row := message.Old[0]
	_, versionChanged := row["content_version"]
	return versionChanged
}

func (s *PostsHandler) llmProcessContent(ctx context.Context, content *llm.Content, res *Result, safeClose func()) error {
	processed, err := llm.ContentProcess(ctx, content)
	if err != nil {
		return err
	}
	s.updateMaxStatus(res, int32(processed.Status), safeClose)
	res.Lock()
	defer res.Unlock()
	if processed.MainTag != "" {
		res.MainTags = append(res.MainTags, processed.MainTag)
	}
	if len(processed.Tags) > 0 {
		res.Tags = append(res.Tags, processed.Tags...)
	}
	return nil
}

func (s *PostsHandler) llmProcessMedia(ctx context.Context, media []*model.PostMedia, res *Result, safeClose func()) error {
	if len(media) == 0 {
		return nil
	}

	// 并发提取所有媒体特征（图片、视频帧、音频转写）
	if err := s.collectAllFeatures(ctx, media, res, safeClose); err != nil {
		return err
	}

	// 如果已发现音频/文本违规，直接返回
	if atomic.LoadInt32(&res.maxStatus) == int32(llm.ContentSafeDeny) {
		return nil
	}

	// 对收集到的图片（原始图+采样帧）进行分批视觉审计
	if err := s.performBatchImageAudit(ctx, res, safeClose); err != nil {
		return err
	}

	return nil
}

func (s *PostsHandler) collectAllFeatures(ctx context.Context, media []*model.PostMedia, res *Result, safeClose func()) error {
	g, gCtx := errgroup.WithContext(ctx)

	for _, m := range media {
		m := m
		g.Go(func() error {
			typePrefix := strings.Split(m.FileType, "/")[0]
			switch typePrefix {
			case consts.MimePrefixImage:
				res.Lock()
				res.allPendingUrls = append(res.allPendingUrls, minio.GetPublicURL(m.MediaURL))
				res.Unlock()
				return nil

			case consts.MimePrefixVideo:
				// 内部启动多个子任务，但不创建新的 errgroup.Wait
				return s.processVideoItem(gCtx, g, m, res, safeClose)

			case consts.MimePrefixAudio:
				return s.processAudioItem(gCtx, g, m, res, safeClose)
			}
			return nil
		})
	}
	return g.Wait()
}

func (s *PostsHandler) processVideoItem(ctx context.Context, g *errgroup.Group, m *model.PostMedia, res *Result, safeClose func()) error {
	// 视觉特征提取任务
	g.Go(func() error {
		url := minio.GetInternalFileURL(m.MediaURL)
		duration, err := util.GetDuration(ctx, url)
		if err != nil {
			return err
		}

		frames, err := util.GetImageFrames(ctx, url, duration)
		if err != nil {
			return err
		}

		for _, frame := range frames {
			f := frame
			g.Go(func() error {
				if closer, ok := f.(io.ReadCloser); ok {
					defer func() {
						_ = closer.Close()
					}()
				}
				objName := fmt.Sprintf("%s.jpg", uuid.NewString())
				fileName, err := minio.UploadTempFile(ctx, objName, f, "image/jpeg")
				if err != nil {
					return err
				}

				res.Lock()
				res.allPendingUrls = append(res.allPendingUrls, minio.GetTempFileURL(fileName, true))
				res.Unlock()
				return nil
			})
		}
		return nil
	})

	// 音频特征提取与即时审计任务
	g.Go(func() error {
		url := minio.GetInternalFileURL(m.MediaURL)
		audio, err := util.GetAudioStream(ctx, url)
		if err != nil {
			return err
		}
		defer func() {
			_ = audio.Close()
		}()

		objName := fmt.Sprintf("%s.wav", uuid.NewString())
		fileName, err := minio.UploadTempFile(ctx, objName, audio, "audio/wav")
		if err != nil {
			return err
		}

		text, err := util.AudioStreamToText(ctx, minio.GetTempFileURL(fileName, false))
		if err != nil {
			return err
		}

		return s.llmProcessContent(ctx, &llm.Content{Content: text}, res, safeClose)
	})
	return nil
}

func (s *PostsHandler) processAudioItem(ctx context.Context, g *errgroup.Group, m *model.PostMedia, res *Result, safeClose func()) error {
	g.Go(func() error {
		url := minio.GetInternalFileURL(m.MediaURL)
		text, err := util.AudioStreamToText(ctx, url)
		if err != nil {
			return err
		}
		return s.llmProcessContent(ctx, &llm.Content{Content: text}, res, safeClose)
	})
	return nil
}

func (s *PostsHandler) performBatchImageAudit(ctx context.Context, res *Result, safeClose func()) error {
	urls := res.allPendingUrls
	if len(urls) == 0 {
		return nil
	}

	const batchSize = 5
	g, gCtx := errgroup.WithContext(ctx)

	for i := 0; i < len(urls); i += batchSize {
		end := i + batchSize
		if end > len(urls) {
			end = len(urls)
		}
		batch := urls[i:end]

		g.Go(func() error {
			processed, err := llm.ImageProcess(gCtx, batch)
			if err != nil {
				return err
			}

			s.updateMaxStatus(res, int32(processed.Status), safeClose)
			res.Lock()
			defer res.Unlock()
			if processed.MainTag != "" {
				res.MainTags = append(res.MainTags, processed.MainTag)
			}
			if processed.Tags != nil {
				res.Tags = append(res.Tags, processed.Tags...)
			}
			return nil
		})
	}

	err := g.Wait()
	if err != nil && !errors.Is(err, ErrAuditDenyTriggered) {
		return err
	}
	return nil
}

// updateMaxStatus 使用原子操作确保并发环境下状态更新的安全性
// 遵循优先级：Deny(3) > Warn(2) > Pass(1)
func (s *PostsHandler) updateMaxStatus(res *Result, val int32, safeClose func()) {
	for {
		if val == int32(llm.ContentSafeDeny) {
			log.Info("审核触发拒绝策略")
			safeClose()
			return
		}
		old := atomic.LoadInt32(&res.maxStatus)
		if val <= old {
			return
		}
		// 使用 CAS 保证并发更新的原子性
		if atomic.CompareAndSwapInt32(&res.maxStatus, old, val) {
			return
		}
	}
}
