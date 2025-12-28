package es

import (
	"context"
	"errors"
	log "log/slog"
	"strconv"

	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/versiontype"
	"github.com/goccy/go-json"
)

type PostRepo interface {
	GetPostById(ctx context.Context, id uint64) (*PostES, error)
	IndexPost(ctx context.Context, post *PostES, version int64) error
	DeletePost(ctx context.Context, id uint64) error
}

type PostRepoImpl struct {
}

func NewPostRepo() PostRepo {
	return &PostRepoImpl{}
}

func (s *PostRepoImpl) GetPostById(ctx context.Context, id uint64) (*PostES, error) {
	docID := strconv.FormatUint(id, 10)
	result, err := Client.Get(PostIndex, docID).Do(ctx)
	if err != nil {
		var e *types.ElasticsearchError
		if errors.As(err, &e) {
			if e.Status == NotFoundCode {
				log.Warn("Post not found in ES", "id", id)
				return nil, nil
			}
		}
		return nil, err
	}
	var post PostES
	if err = json.Unmarshal(result.Source_, &post); err != nil {
		return nil, err
	}
	if post.UserTags == nil {
		post.UserTags = make([]string, 0)
	}
	if post.AITags == nil {
		post.AITags = make([]string, 0)
	}
	if post.Media == nil {
		post.Media = make([]PostMediaES, 0)
	}
	return &post, nil
}

func (s *PostRepoImpl) IndexPost(ctx context.Context, post *PostES, version int64) error {
	docID := strconv.FormatUint(post.ID, 10)

	log.Info("ES Index Post", "post", post, "version", version)

	_, err := Client.Index(PostIndex).
		Id(docID).
		Document(post).
		Version(strconv.FormatInt(version, 10)).
		VersionType(versiontype.External).
		Do(ctx)

	if err != nil {
		var e *types.ElasticsearchError
		if errors.As(err, &e) {
			if e.Status == ConflictCode {
				log.Warn("Post Version Conflict", "id", post.ID, "version", version)
				return nil
			}
		}
		return err
	}

	return nil
}

func (s *PostRepoImpl) DeletePost(ctx context.Context, id uint64) error {
	docID := strconv.FormatUint(id, 10)

	_, err := Client.Delete(PostIndex, docID).Do(ctx)

	if err != nil {
		var e *types.ElasticsearchError
		if errors.As(err, &e) {
			if e.Status == NotFoundCode {
				log.Warn("Post already deleted or not found in ES", "id", id)
				return nil
			}
		}
		return err
	}

	log.Info("ES Delete success", "id", id)
	return nil
}
