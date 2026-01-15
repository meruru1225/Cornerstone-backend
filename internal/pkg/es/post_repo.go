package es

import (
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/util"
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/conflicts"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/sortorder"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/versiontype"
	"github.com/goccy/go-json"
)

type PostRepo interface {
	HybridSearch(ctx context.Context, queryText string, queryVector []float32, from, size int) ([]*PostES, error)
	GetPostById(ctx context.Context, id uint64) (*PostES, error)
	GetPostByMainTag(ctx context.Context, tag string, isMain bool, from, size int) ([]*PostES, error)
	GetLatestPosts(ctx context.Context, from, size int) ([]*PostES, error)
	IndexPost(ctx context.Context, post *PostES, version int64) error
	DeletePost(ctx context.Context, id uint64) error
	UpdatePostUserDetail(ctx context.Context, userID uint64, newNickname string, newAvatar string) error
}

type PostRepoImpl struct {
}

func NewPostRepo() PostRepo {
	return &PostRepoImpl{}
}

func (s *PostRepoImpl) HybridSearch(ctx context.Context, queryText string, queryVector []float32, from, size int) ([]*PostES, error) {
	statusFilter := types.Query{
		Term: map[string]types.TermQuery{
			"status": {Value: consts.PostStatusNormal},
		},
	}

	searchReq := Client.Search().
		Index(PostIndex).
		// 配置 k-NN 搜索
		Knn(types.KnnSearch{
			Field:         "content_vector",
			QueryVector:   queryVector,
			K:             util.PtrInt(10),
			NumCandidates: util.PtrInt(100),
			Similarity:    util.PtrFloat32(0.6),
			Filter:        []types.Query{statusFilter},
		}).
		// 配置传统文本搜索（增强精准度）
		Query(&types.Query{
			Bool: &types.BoolQuery{
				Must: []types.Query{
					{
						MultiMatch: &types.MultiMatchQuery{
							Query:  queryText,
							Fields: []string{"title^2", "content"},
						},
					},
				},
				Filter: []types.Query{statusFilter},
			},
		}).
		Source_(&types.SourceFilter{
			Excludes: []string{"content_vector"},
		}).
		From(from).
		Size(size)

	return s.executeSearch(ctx, searchReq)
}

func (s *PostRepoImpl) GetPostById(ctx context.Context, id uint64) (*PostES, error) {
	docID := strconv.FormatUint(id, 10)
	result, err := Client.Get(PostIndex, docID).Do(ctx)
	if err != nil {
		var e *types.ElasticsearchError
		if errors.As(err, &e) {
			if e.Status == NotFoundCode {
				return nil, nil
			}
		}
		return nil, err
	}
	if result.Source_ == nil {
		return nil, nil
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

func (s *PostRepoImpl) GetPostByMainTag(ctx context.Context, tag string, isMain bool, from, size int) ([]*PostES, error) {
	searchField := "user_tags"
	if isMain {
		searchField = "main_tag"
	}

	searchReq := Client.Search().
		Index(PostIndex).
		Query(&types.Query{
			Bool: &types.BoolQuery{
				Must: []types.Query{
					{
						Term: map[string]types.TermQuery{
							searchField: {Value: tag},
						},
					},
				},
				Filter: []types.Query{
					{
						Term: map[string]types.TermQuery{
							"status": {Value: consts.PostStatusNormal},
						},
					},
				},
			},
		}).
		Source_(&types.SourceFilter{
			Excludes: []string{"content_vector"},
		}).
		Sort(types.SortOptions{
			SortOptions: map[string]types.FieldSort{
				"created_at": {Order: &sortorder.Desc},
			},
		}).
		From(from).
		Size(size)

	return s.executeSearch(ctx, searchReq)
}

// GetLatestPosts 降级逻辑：获取最新的帖子列表
func (s *PostRepoImpl) GetLatestPosts(ctx context.Context, from, size int) ([]*PostES, error) {
	searchReq := Client.Search().
		Index(PostIndex).
		Query(&types.Query{
			Term: map[string]types.TermQuery{
				"status": {Value: consts.PostStatusNormal},
			},
		}).
		Sort(types.SortOptions{SortOptions: map[string]types.FieldSort{
			"created_at": {Order: &sortorder.Desc},
		}}).
		Source_(&types.SourceFilter{Excludes: []string{"content_vector"}}).
		From(from).
		Size(size)

	return s.executeSearch(ctx, searchReq)
}

func (s *PostRepoImpl) IndexPost(ctx context.Context, post *PostES, version int64) error {
	docID := strconv.FormatUint(post.ID, 10)

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
				return nil
			}
		}
		return err
	}

	return nil
}

// UpdatePostUserDetail 同步更新 post_index 中冗余的用户信息
func (s *PostRepoImpl) UpdatePostUserDetail(ctx context.Context, userID uint64, newNickname string, newAvatar string) error {
	nicknameJSON, _ := json.Marshal(newNickname)
	avatarJSON, _ := json.Marshal(newAvatar)

	params := map[string]json.RawMessage{
		"new_nickname": json.RawMessage(nicknameJSON),
		"new_avatar":   json.RawMessage(avatarJSON),
	}

	scriptSource := "ctx._source.user_nickname = params.new_nickname; ctx._source.user_avatar = params.new_avatar;"

	req := Client.UpdateByQuery(PostIndex).
		Query(&types.Query{
			Term: map[string]types.TermQuery{
				"user_id": {Value: userID},
			},
		}).
		Script(&types.Script{
			Source: &scriptSource,
			Params: params,
		}).
		Conflicts(conflicts.Proceed)

	resp, err := req.Do(ctx)
	if err != nil {
		return errors.New(fmt.Sprintf("Post Index: Update User Detail Failed: %s", err.Error()))
	}

	if len(resp.Failures) != 0 {
		return errors.New(fmt.Sprintf("Post Index: Update User Detail Has Failures, count: %d", len(resp.Failures)))
	}

	return nil
}

func (s *PostRepoImpl) executeSearch(ctx context.Context, req *search.Search) ([]*PostES, error) {
	resp, err := req.Do(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]*PostES, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		var post PostES
		if err = json.Unmarshal(hit.Source_, &post); err != nil {
			continue
		}
		results = append(results, &post)
	}
	return results, nil
}
