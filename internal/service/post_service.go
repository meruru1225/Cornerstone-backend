package service

import (
	"Cornerstone/internal/api/dto"
	"Cornerstone/internal/model"
	"Cornerstone/internal/repository"
	"context"
	"strconv"

	"github.com/jinzhu/copier"
)

type PostService interface {
	CreatePost(ctx context.Context, userID uint64, postDTO *dto.PostBaseDTO) error
	GetPostById(ctx context.Context, id uint64) (*dto.PostDTO, error)
	GetPostByIds(ctx context.Context, ids []uint64) ([]*dto.PostDTO, error)
	GetPostByUserId(ctx context.Context, userId uint64, page, pageSize int) ([]*dto.PostDTO, error)
	GetPostSelf(ctx context.Context, userId uint64, page, pageSize int) ([]*dto.PostDTO, error)
	UpdatePost(ctx context.Context, userID uint64, postID uint64, postDTO *dto.PostBaseDTO) error
	DeletePost(ctx context.Context, userID uint64, postID uint64) error
}

type postServiceImpl struct {
	postRepo repository.PostRepo
}

func NewPostService(postRepo repository.PostRepo) PostService {
	return &postServiceImpl{
		postRepo: postRepo,
	}
}

// CreatePost 创建帖子
func (s *postServiceImpl) CreatePost(ctx context.Context, userID uint64, postDTO *dto.PostBaseDTO) error {
	post := &model.Post{}
	if err := copier.Copy(post, postDTO); err != nil {
		return err
	}
	post.UserID = userID
	post.ID = 0

	postMedias := s.mapMedias(postDTO.Medias)
	return s.postRepo.CreatePost(ctx, post, postMedias)
}

// GetPostById 获取单个帖子
func (s *postServiceImpl) GetPostById(ctx context.Context, id uint64) (*dto.PostDTO, error) {
	post, err := s.postRepo.GetPost(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.toPostDTO(post)
}

// GetPostByIds 批量获取
func (s *postServiceImpl) GetPostByIds(ctx context.Context, ids []uint64) ([]*dto.PostDTO, error) {
	posts, err := s.postRepo.GetPostByIds(ctx, ids)
	if err != nil {
		return nil, err
	}
	return s.batchToPostDTO(posts)
}

// GetPostByUserId 获取某人的帖子列表
func (s *postServiceImpl) GetPostByUserId(ctx context.Context, userId uint64, page, pageSize int) ([]*dto.PostDTO, error) {
	posts, err := s.postRepo.GetPostByUserId(ctx, userId, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, err
	}

	return s.batchToPostDTO(posts)
}

// GetPostSelf 获取自己的帖子列表
func (s *postServiceImpl) GetPostSelf(ctx context.Context, userId uint64, page, pageSize int) ([]*dto.PostDTO, error) {
	posts, err := s.postRepo.GetPostSelf(ctx, userId, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, err
	}
	return s.batchToPostDTO(posts)
}

// UpdatePost 更新帖子
func (s *postServiceImpl) UpdatePost(ctx context.Context, userID uint64, postID uint64, postDTO *dto.PostBaseDTO) error {
	oldPost, err := s.postRepo.GetPost(ctx, postID)
	if err != nil {
		return err
	}

	if oldPost.UserID != userID {
		return UnauthorizedError
	}

	post := &model.Post{}
	if err := copier.Copy(post, postDTO); err != nil {
		return err
	}
	post.ID = postID
	post.UserID = userID
	post.Status = 0

	postMedias := s.mapMedias(postDTO.Medias)
	return s.postRepo.UpdatePost(ctx, post, postMedias)
}

// DeletePost 删除帖子 (包含鉴权)
func (s *postServiceImpl) DeletePost(ctx context.Context, userID uint64, postID uint64) error {
	post, err := s.postRepo.GetPost(ctx, postID)
	if err != nil {
		return err
	}

	if post.UserID != userID {
		return UnauthorizedError
	}

	return s.postRepo.DeletePost(ctx, postID)
}

// toPostDTO 转换 DTO
func (s *postServiceImpl) toPostDTO(post *model.Post) (*dto.PostDTO, error) {
	out := &dto.PostDTO{}
	if err := copier.Copy(out, post); err != nil {
		return nil, err
	}

	if len(post.Media) > 0 {
		if err := copier.Copy(&out.Medias, post.Media); err != nil {
			return nil, err
		}
	}

	if post.User.ID > 0 {
		out.UserID = post.User.ID
		if post.User.UserDetail.UserID > 0 {
			out.Nickname = post.User.UserDetail.Nickname
			out.AvatarURL = post.User.UserDetail.AvatarURL
		} else {
			out.Nickname = "用户" + strconv.FormatUint(post.User.ID, 10)
			out.AvatarURL = "default_avatar.png"
		}
	} else {
		out.UserID = 0
		out.Nickname = "未知用户"
		out.AvatarURL = "default_avatar.png"
	}
	return out, nil
}

// batchToPostDTO 批量转换 DTO
func (s *postServiceImpl) batchToPostDTO(posts []*model.Post) ([]*dto.PostDTO, error) {
	out := make([]*dto.PostDTO, len(posts))
	for i, post := range posts {
		item, err := s.toPostDTO(post)
		if err != nil {
			return nil, err
		}
		out[i] = item
	}
	return out, nil
}

// mapMedias 转换 Medias
func (s *postServiceImpl) mapMedias(dtos []dto.Medias) []*model.PostMedia {
	postMedias := make([]*model.PostMedia, len(dtos))
	for i, object := range dtos {
		postMedias[i] = &model.PostMedia{
			MediaURL:  object.MediaName,
			FileType:  object.MimeType,
			SortOrder: int8(i),
			Width:     object.Width,
			Height:    object.Height,
			CoverURL:  object.CoverURL,
		}
	}
	return postMedias
}
