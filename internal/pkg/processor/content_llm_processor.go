package processor

import (
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/es"
	"Cornerstone/internal/pkg/llm"
	"Cornerstone/internal/pkg/minio"
	"Cornerstone/internal/pkg/util"
	"context"
	"errors"
	"fmt"
	"io"
	log "log/slog"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

var ErrAuditDenyTriggered = errors.New("audit deny detected, cancelling other batches")

type Result struct {
	sync.Mutex
	StopChan       chan struct{}
	MaxStatus      int32
	MainTags       []string
	Tags           []string
	Summaries      []string
	allPendingUrls []string
}

type ContentLLMProcessor interface {
	Process(ctx context.Context, title, content string, media []*es.PostMediaES, auditOnly bool) (*Result, error)
}

type contentLLMProcessorImpl struct{}

func NewContentLLMProcessor() ContentLLMProcessor {
	return &contentLLMProcessorImpl{}
}

func (s *contentLLMProcessorImpl) Process(ctx context.Context, title, content string, media []*es.PostMediaES, auditOnly bool) (*Result, error) {
	res := &Result{
		StopChan: make(chan struct{}),
	}

	log.InfoContext(ctx, "ContentLLMProcessor started", "media_count", len(media), "audit_only", auditOnly)

	gCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	g, gCtx := errgroup.WithContext(gCtx)

	// 文本处理
	g.Go(func() error {
		return s.handleText(gCtx, title, content, res, cancel, auditOnly)
	})

	// 媒体处理
	if len(media) > 0 {
		g.Go(func() error {
			if err := s.collectAllFeatures(gCtx, media, res, cancel, auditOnly); err != nil {
				return err
			}
			if atomic.LoadInt32(&res.MaxStatus) == int32(llm.ContentSafeDeny) {
				return nil
			}
			return s.performBatchImageAudit(gCtx, res, cancel, auditOnly)
		})
	}

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- g.Wait()
	}()

	select {
	case err := <-waitDone:
		if err != nil {
			if errors.Is(err, ErrAuditDenyTriggered) {
				log.InfoContext(ctx, "process finished with audit deny")
				return res, nil
			}
			log.ErrorContext(ctx, "ContentLLMProcessor wait error", "err", err)
			return nil, err
		}
	case <-res.StopChan:
		cancel()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	return res, nil
}

func (s *contentLLMProcessorImpl) handleText(ctx context.Context, title, content string, res *Result, cancel context.CancelFunc, auditOnly bool) error {
	if title == "" && content == "" {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	input := &llm.Content{Title: &title, Content: content}
	processed, err := llm.ContentProcess(ctx, input, auditOnly)
	if err != nil {
		return err
	}

	s.updateMaxStatus(res, int32(processed.Status), cancel)
	s.updateResultSlice(res, processed)
	return nil
}

func (s *contentLLMProcessorImpl) collectAllFeatures(ctx context.Context, media []*es.PostMediaES, res *Result, cancel context.CancelFunc, auditOnly bool) error {
	g, gCtx := errgroup.WithContext(ctx)

	for _, m := range media {
		m := m
		g.Go(func() error {
			select {
			case <-gCtx.Done():
				return gCtx.Err()
			default:
			}

			typePrefix := strings.Split(m.Type, "/")[0]
			switch typePrefix {
			case consts.MimePrefixImage:
				res.Lock()
				res.allPendingUrls = append(res.allPendingUrls, minio.GetForcePublicURL(m.URL))
				res.Unlock()
				return nil
			case consts.MimePrefixVideo:
				if m.Cover != nil && *m.Cover != "" {
					res.Lock()
					res.allPendingUrls = append(res.allPendingUrls, minio.GetForcePublicURL(*m.Cover))
					res.Unlock()
				}
				return s.processVideoItem(gCtx, g, m, res, cancel, auditOnly)
			case consts.MimePrefixAudio:
				return s.processAudioItem(gCtx, g, m, res, cancel, auditOnly)
			}
			return nil
		})
	}
	return g.Wait()
}

func (s *contentLLMProcessorImpl) processVideoItem(ctx context.Context, g *errgroup.Group, m *es.PostMediaES, res *Result, cancel context.CancelFunc, auditOnly bool) error {
	g.Go(func() error {
		url := minio.GetInternalFileURL(m.URL)
		duration, err := util.GetDuration(ctx, url)
		if err != nil {
			return err
		}
		frames, err := util.GetImageFrames(ctx, url, duration)
		log.InfoContext(ctx, "get video image frames successfully")
		if err != nil {
			return err
		}
		for _, frame := range frames {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			g.Go(func() error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				f, err := util.ResizeImage(frame, 768, 0, 85)
				if err != nil {
					return err
				}
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

	g.Go(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		url := minio.GetInternalFileURL(m.URL)
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
		return s.handleText(ctx, "", text, res, cancel, auditOnly)
	})
	return nil
}

func (s *contentLLMProcessorImpl) processAudioItem(ctx context.Context, g *errgroup.Group, m *es.PostMediaES, res *Result, cancel context.CancelFunc, auditOnly bool) error {
	g.Go(func() error {
		url := minio.GetInternalFileURL(m.URL)
		text, err := util.AudioStreamToText(ctx, url)
		if err != nil {
			return err
		}
		return s.handleText(ctx, "", text, res, cancel, auditOnly)
	})
	return nil
}

func (s *contentLLMProcessorImpl) performBatchImageAudit(ctx context.Context, res *Result, cancel context.CancelFunc, auditOnly bool) error {
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
			select {
			case <-gCtx.Done():
				return gCtx.Err()
			default:
			}

			processed, err := llm.ImageProcess(gCtx, batch, auditOnly)
			if err != nil {
				return err
			}

			s.updateMaxStatus(res, int32(processed.Status), cancel)
			s.updateResultSlice(res, processed)

			return nil
		})
	}
	return g.Wait()
}

func (s *contentLLMProcessorImpl) updateResultSlice(res *Result, processed *llm.ContentResponse) {
	res.Lock()
	defer res.Unlock()
	if processed.MainTag != "" {
		res.MainTags = append(res.MainTags, processed.MainTag)
	}
	if len(processed.Tags) > 0 {
		res.Tags = append(res.Tags, processed.Tags...)
	}
	if processed.Summary != "" {
		res.Summaries = append(res.Summaries, processed.Summary)
	}
}

func (s *contentLLMProcessorImpl) updateMaxStatus(res *Result, val int32, cancel context.CancelFunc) {
	for {
		old := atomic.LoadInt32(&res.MaxStatus)
		if val <= old && old != 0 {
			return
		}
		if val == int32(llm.ContentSafeDeny) {
			atomic.StoreInt32(&res.MaxStatus, val)
			select {
			case <-res.StopChan:
			default:
				close(res.StopChan)
			}
			cancel()
			return
		}
		if atomic.CompareAndSwapInt32(&res.MaxStatus, old, val) {
			return
		}
	}
}
