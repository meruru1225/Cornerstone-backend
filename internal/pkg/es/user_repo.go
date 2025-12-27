package es

import (
	"context"
	"errors"
	log "log/slog"
	"strconv"

	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/versiontype"
)

type UserRepo interface {
	IndexUser(ctx context.Context, user *UserES, version int64) error
	DeleteUser(ctx context.Context, id uint64) error
}

type UserRepoImpl struct {
}

func NewUserRepo() UserRepo {
	return &UserRepoImpl{}
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
				log.Warn("Version conflict detected, skipping old data",
					"user_id", user.ID,
					"version", version)
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
				log.Warn("User already deleted or not found in ES", "id", id)
				return nil
			}
		}
		return err
	}
	return nil
}
