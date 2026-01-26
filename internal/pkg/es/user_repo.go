package es

import (
	"Cornerstone/internal/pkg/util"
	"context"
	"errors"
	"strconv"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/versiontype"
	"github.com/goccy/go-json"
)

type UserRepo interface {
	SearchUser(ctx context.Context, keyword string, from, size int) ([]*UserES, int64, error)
	Exist(ctx context.Context, id uint64) (bool, error)
	IndexUser(ctx context.Context, user *UserES, version int64) error
	DeleteUser(ctx context.Context, id uint64) error
}

type UserRepoImpl struct {
	client *elasticsearch.TypedClient
}

func NewUserRepo(client *elasticsearch.TypedClient) UserRepo {
	return &UserRepoImpl{client: client}
}

func (s *UserRepoImpl) SearchUser(ctx context.Context, keyword string, from, size int) ([]*UserES, int64, error) {
	if keyword == "" {
		return []*UserES{}, 0, nil
	}

	query := &types.Query{
		Bool: &types.BoolQuery{
			Should: []types.Query{
				{
					MultiMatch: &types.MultiMatchQuery{
						Query: keyword,
						Fields: []string{
							"nickname^3",
							"nickname.pinyin^1",
						},
						Boost: util.PtrFloat32(2.0),
					},
				},
				{
					MultiMatch: &types.MultiMatchQuery{
						Query:     keyword,
						Fields:    []string{"nickname"},
						Fuzziness: util.PtrStr("AUTO"),
						Boost:     util.PtrFloat32(0.5),
					},
				},
			},
			MinimumShouldMatch: util.PtrStr("1"),
		},
	}

	res, err := s.client.Search().
		Index(UserIndex).
		Query(query).
		From(from).
		Size(size).
		Do(ctx)

	if err != nil {
		return nil, 0, err
	}

	total := res.Hits.Total.Value
	users := make([]*UserES, 0, len(res.Hits.Hits))

	for _, hit := range res.Hits.Hits {
		var user UserES
		if hit.Source_ == nil {
			continue
		}
		if err := json.Unmarshal(hit.Source_, &user); err != nil {
			continue
		}
		users = append(users, &user)
	}

	return users, total, nil
}

func (s *UserRepoImpl) Exist(ctx context.Context, id uint64) (bool, error) {
	docID := strconv.FormatUint(id, 10)
	res, err := s.client.Exists(UserIndex, docID).Do(ctx)
	if err != nil {
		return false, err
	}
	return res, nil
}

func (s *UserRepoImpl) IndexUser(ctx context.Context, user *UserES, version int64) error {
	docID := strconv.FormatUint(user.ID, 10)

	_, err := s.client.Index(UserIndex).
		Id(docID).
		Document(user).
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

func (s *UserRepoImpl) DeleteUser(ctx context.Context, id uint64) error {
	docID := strconv.FormatUint(id, 10)
	_, err := s.client.Delete(UserIndex, docID).Do(ctx)
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
