package es

import (
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/util"
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/core/search"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/conflicts"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/functionboostmode"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/sortorder"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/versiontype"
	"github.com/goccy/go-json"
	"golang.org/x/sync/errgroup"
)

const MaxSearchDepth = 400

type PostRepo interface {
	HybridSearch(ctx context.Context, queryText string, queryVector []float32, from, size int) ([]*PostES, error)
	HybridSearchMe(ctx context.Context, userID uint64, queryText string, queryVector []float32, from, size int) ([]*PostES, error)
	RecommendPosts(ctx context.Context, queryText string, queryVector []float32, lastSortValues []interface{}, size int, seed int64) ([]*PostES, error)
	GetSuggestions(ctx context.Context, keyword string) ([]string, error)
	GetPostById(ctx context.Context, id uint64) (*PostES, error)
	GetPostByTag(ctx context.Context, tag string, isMain bool, from, size int) ([]*PostES, error)
	GetLatestPosts(ctx context.Context, from, size int) ([]*PostES, error)
	GetLatestPostsByCursor(ctx context.Context, lastSortValues []interface{}, size int) ([]*PostES, error)
	IndexPost(ctx context.Context, post *PostES, version int64) error
	DeletePost(ctx context.Context, id uint64) error
	UpdatePostUserDetail(ctx context.Context, userID uint64, newNickname string, newAvatar string) error
}

type PostRepoImpl struct {
	client *elasticsearch.TypedClient
}

func NewPostRepo(client *elasticsearch.TypedClient) PostRepo {
	return &PostRepoImpl{client: client}
}

func (s *PostRepoImpl) HybridSearch(ctx context.Context, queryText string, queryVector []float32, from, size int) ([]*PostES, error) {
	if from >= MaxSearchDepth {
		return []*PostES{}, nil
	}

	requestedDepth := from + size
	candidateLimit := s.calculateCandidateLimit(requestedDepth)

	statusFilter := []types.Query{{
		Term: map[string]types.TermQuery{
			"status": {Value: consts.PostStatusNormal},
		},
	}}

	return s.executeHybridFusion(ctx, queryText, queryVector, statusFilter, candidateLimit, from, size, nil, nil)
}

func (s *PostRepoImpl) HybridSearchMe(ctx context.Context, userID uint64, queryText string, queryVector []float32, from, size int) ([]*PostES, error) {
	if from >= MaxSearchDepth {
		return []*PostES{}, nil
	}

	requestedDepth := from + size
	candidateLimit := s.calculateCandidateLimit(requestedDepth)

	userFilter := []types.Query{{
		Term: map[string]types.TermQuery{
			"user_id": {Value: userID},
		},
	}}

	return s.executeHybridFusion(ctx, queryText, queryVector, userFilter, candidateLimit, from, size, nil, nil)
}

// RecommendPosts 推荐流：混合检索 + 随机种子 + SearchAfter
func (s *PostRepoImpl) RecommendPosts(ctx context.Context, queryText string, queryVector []float32, lastSortValues []interface{}, size int, seed int64) ([]*PostES, error) {
	req := s.client.Search().Index(PostIndex).Size(size)

	boolQuery := &types.BoolQuery{
		Filter: []types.Query{
			{Term: map[string]types.TermQuery{"status": {Value: consts.PostStatusNormal}}},
		},
		Should: []types.Query{},
	}

	if queryText != "" {
		boolQuery.Should = append(boolQuery.Should, types.Query{
			MultiMatch: &types.MultiMatchQuery{
				Query:  queryText,
				Fields: []string{"title^2", "plain_content", "user_tags"},
				Boost:  util.PtrFloat32(2.0),
			},
		})
	}

	seedStr := strconv.FormatInt(seed, 10)
	weightVal := types.Float64(1.0)
	boolQuery.Should = append(boolQuery.Should, types.Query{
		FunctionScore: &types.FunctionScoreQuery{
			Functions: []types.FunctionScore{
				{
					RandomScore: &types.RandomScoreFunction{
						Seed:  &seedStr,
						Field: util.PtrStr("_seq_no"),
					},
					Weight: &weightVal,
				},
			},
			BoostMode: &functionboostmode.Sum,
		},
	})

	req.Query(&types.Query{Bool: boolQuery})

	if len(queryVector) > 0 {
		req.Knn(types.KnnSearch{
			Field:         "content_vector",
			QueryVector:   queryVector,
			K:             util.PtrInt(size),
			NumCandidates: util.PtrInt(size * 5),
			Boost:         util.PtrFloat32(20.0),
		})
	}

	if len(lastSortValues) > 0 {
		searchAfterValues := make([]types.FieldValue, len(lastSortValues))
		for i, v := range lastSortValues {
			searchAfterValues[i] = v
		}
		req.SearchAfter(searchAfterValues...)
	}

	req.Source_(&types.SourceFilter{Excludes: []string{"content_vector"}})

	return s.executeSearch(ctx, req)
}

func (s *PostRepoImpl) GetSuggestions(ctx context.Context, keyword string) ([]string, error) {
	suggestKey := "post-suggest"

	suggester := types.NewSuggester()
	suggester.Suggesters[suggestKey] = types.FieldSuggester{
		Prefix: &keyword,
		Completion: &types.CompletionSuggester{
			Field: "title.suggestion",
			Fuzzy: &types.SuggestFuzziness{
				Fuzziness: util.PtrStr("AUTO"),
			},
			Size: util.PtrInt(5),
		},
	}

	res, err := s.client.Search().
		Index(PostIndex).
		Suggest(suggester).
		Size(0).
		Do(ctx)
	if err != nil {
		return nil, err
	}

	suggestions := make([]string, 0)
	if results, ok := res.Suggest[suggestKey]; ok {
		for _, r := range results {
			if cs, ok := r.(*types.CompletionSuggest); ok {
				for _, opt := range cs.Options {
					suggestions = append(suggestions, opt.Text)
				}
			}
		}
	}
	return suggestions, nil
}

func (s *PostRepoImpl) GetPostById(ctx context.Context, id uint64) (*PostES, error) {
	docID := strconv.FormatUint(id, 10)
	result, err := s.client.Get(PostIndex, docID).Do(ctx)
	if err != nil {
		var e *types.ElasticsearchError
		if errors.As(err, &e) {
			if e.Status == 404 {
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

func (s *PostRepoImpl) GetPostByTag(ctx context.Context, tag string, isMain bool, from, size int) ([]*PostES, error) {
	searchField := "user_tags"
	if isMain {
		searchField = "main_tag"
	}

	searchReq := s.client.Search().
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

// GetLatestPosts 获取最新的帖子列表
func (s *PostRepoImpl) GetLatestPosts(ctx context.Context, from, size int) ([]*PostES, error) {
	searchReq := s.client.Search().
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

func (s *PostRepoImpl) GetLatestPostsByCursor(ctx context.Context, lastSortValues []interface{}, size int) ([]*PostES, error) {
	req := s.client.Search().
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
		Size(size)

	// 注入游标
	if len(lastSortValues) > 0 {
		searchAfterValues := make([]types.FieldValue, len(lastSortValues))
		for i, v := range lastSortValues {
			searchAfterValues[i] = v
		}
		req.SearchAfter(searchAfterValues...)
	}

	return s.executeSearch(ctx, req)
}

func (s *PostRepoImpl) IndexPost(ctx context.Context, post *PostES, version int64) error {
	docID := strconv.FormatUint(post.ID, 10)

	_, err := s.client.Index(PostIndex).
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

	_, err := s.client.Delete(PostIndex, docID).Do(ctx)

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

func (s *PostRepoImpl) UpdatePostUserDetail(ctx context.Context, userID uint64, newNickname string, newAvatar string) error {
	nicknameJSON, _ := json.Marshal(newNickname)
	avatarJSON, _ := json.Marshal(newAvatar)

	params := map[string]json.RawMessage{
		"new_nickname": json.RawMessage(nicknameJSON),
		"new_avatar":   json.RawMessage(avatarJSON),
	}

	scriptSource := "ctx._source.user_nickname = params.new_nickname; ctx._source.user_avatar = params.new_avatar;"

	req := s.client.UpdateByQuery(PostIndex).
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

func (s *PostRepoImpl) executeHybridFusion(ctx context.Context, queryText string, queryVector []float32, filters []types.Query, limit, from, size int, queryDecorator func(*types.Query) *types.Query, lastSortValues []interface{}) ([]*PostES, error) {
	var (
		vectorResults []*PostES
		textResults   []*PostES
	)

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		vectorResults, err = s.vectorSearch(ctx, queryVector, limit, filters, lastSortValues)
		return err
	})

	g.Go(func() error {
		var err error
		textResults, err = s.textSearch(ctx, queryText, limit, filters, queryDecorator, lastSortValues)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	merged := s.manualRRF(vectorResults, textResults)

	start := from
	if start > len(merged) {
		return []*PostES{}, nil
	}
	end := start + size
	if end > len(merged) {
		end = len(merged)
	}

	return merged[start:end], nil
}

func (s *PostRepoImpl) vectorSearch(ctx context.Context, vector []float32, limit int, filters []types.Query, lastSortValues []interface{}) ([]*PostES, error) {
	if len(vector) == 0 {
		return []*PostES{}, nil
	}
	req := s.client.Search().Index(PostIndex).
		Knn(types.KnnSearch{
			Field:         "content_vector",
			QueryVector:   vector,
			K:             util.PtrInt(limit),
			NumCandidates: util.PtrInt(limit * 2),
			Filter:        filters,
		}).
		Source_(&types.SourceFilter{Excludes: []string{"content_vector"}}).
		Size(limit)

	if len(lastSortValues) > 0 {
		searchAfterValues := make([]types.FieldValue, len(lastSortValues))
		for i, v := range lastSortValues {
			searchAfterValues[i] = v
		}
		req.SearchAfter(searchAfterValues...)
	}

	return s.executeSearch(ctx, req)
}

func (s *PostRepoImpl) textSearch(ctx context.Context, text string, limit int, filters []types.Query, decorator func(*types.Query) *types.Query, lastSortValues []interface{}) ([]*PostES, error) {
	if text == "" {
		return []*PostES{}, nil
	}

	baseQuery := &types.Query{
		Bool: &types.BoolQuery{
			Should: []types.Query{
				{
					MultiMatch: &types.MultiMatchQuery{
						Query:  text,
						Fields: []string{"title^3", "title.pinyin^1", "plain_content^1", "ai_summary^1", "user_tags^3"},
						Boost:  util.PtrFloat32(2.0),
					},
				},
				{
					MultiMatch: &types.MultiMatchQuery{
						Query:     text,
						Fields:    []string{"title", "plain_content"},
						Fuzziness: util.PtrStr("AUTO"),
						Boost:     util.PtrFloat32(0.5),
					},
				},
			},
			Filter: filters,
		},
	}

	finalQuery := baseQuery
	if decorator != nil {
		finalQuery = decorator(baseQuery)
	}

	req := s.client.Search().Index(PostIndex).
		Query(finalQuery).
		Source_(&types.SourceFilter{Excludes: []string{"content_vector"}}).
		Size(limit)

	if len(lastSortValues) > 0 {
		searchAfterValues := make([]types.FieldValue, len(lastSortValues))
		for i, v := range lastSortValues {
			searchAfterValues[i] = v
		}
		req.SearchAfter(searchAfterValues...)
	}

	return s.executeSearch(ctx, req)
}

func (s *PostRepoImpl) manualRRF(ranks ...[]*PostES) []*PostES {
	const k = 60
	scoreMap := make(map[uint64]float64)
	postMap := make(map[uint64]*PostES)

	for _, resultList := range ranks {
		for rank, post := range resultList {
			scoreMap[post.ID] += 1.0 / float64(k+rank+1)
			postMap[post.ID] = post
		}
	}

	merged := make([]*PostES, 0, len(postMap))
	for id := range postMap {
		merged = append(merged, postMap[id])
	}

	sort.Slice(merged, func(i, j int) bool {
		return scoreMap[merged[i].ID] > scoreMap[merged[j].ID]
	})

	return merged
}

func (s *PostRepoImpl) calculateCandidateLimit(depth int) int {
	limit := depth * 2

	if limit < depth {
		limit = depth
	}

	if limit > MaxSearchDepth {
		limit = MaxSearchDepth
	}

	return limit
}

func (s *PostRepoImpl) executeSearch(ctx context.Context, req *search.Search) ([]*PostES, error) {
	resp, err := req.Do(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]*PostES, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		var post PostES
		if hit.Source_ == nil {
			continue
		}
		if err = json.Unmarshal(hit.Source_, &post); err != nil {
			continue
		}
		if len(hit.Sort) > 0 {
			post.Sort = make([]interface{}, len(hit.Sort))
			for i, v := range hit.Sort {
				post.Sort[i] = v
			}
		}
		results = append(results, &post)
	}
	return results, nil
}
