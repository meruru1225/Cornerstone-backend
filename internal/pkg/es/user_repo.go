package es

import (
	"context"
	"errors"
	"strconv"

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
}

func NewUserRepo() UserRepo {
	return &UserRepoImpl{}
}

func (s *UserRepoImpl) SearchUser(ctx context.Context, keyword string, from, size int) ([]*UserES, int64, error) {
	query := &types.Query{
		Match: map[string]types.MatchQuery{
			"nickname": {
				Query: keyword,
			},
		},
	}

	res, err := Client.Search().
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
		if err := json.Unmarshal(hit.Source_, &user); err != nil {
			continue
		}
		users = append(users, &user)
	}

	return users, total, nil
}

func (s *UserRepoImpl) Exist(ctx context.Context, id uint64) (bool, error) {
	docID := strconv.FormatUint(id, 10)
	res, err := Client.Exists(UserIndex, docID).Do(ctx)
	if err != nil {
		return false, err
	}
	return res, nil
}

func (s *UserRepoImpl) IndexUser(ctx context.Context, user *UserES, version int64) error {
	docID := strconv.FormatUint(user.ID, 10)

	_, err := Client.Index(UserIndex).
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
	_, err := Client.Delete(UserIndex, docID).Do(ctx)
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
