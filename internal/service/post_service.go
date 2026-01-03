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
	UpdatePostContent(ctx context.Context, userID uint64, postID uint64, postDTO *dto.PostBaseDTO) error
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
	if err := copier.Copy(&post.MediaList, &postDTO.Medias); err != nil {
		return err
	}

	post.UserID = userID
	post.ID = 0

	return s.postRepo.CreatePost(ctx, post)
}

// GetPostById 获取单个帖子
func (s *postServiceImpl) GetPostById(ctx context.Context, id uint64) (*dto.PostDTO, error) {
	post, err := s.postRepo.GetPost(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.toPostDTO(post)
}

// GetPostByIds 批量获取帖子
func (s *postServiceImpl) GetPostByIds(ctx context.Context, ids []uint64) ([]*dto.PostDTO, error) {
	posts, err := s.postRepo.GetPostByIds(ctx, ids)
	if err != nil {
		return nil, err
	}
	return s.batchToPostDTO(posts)
}

// GetPostByUserId 获取指定用户的公开帖子列表
func (s *postServiceImpl) GetPostByUserId(ctx context.Context, userId uint64, page, pageSize int) ([]*dto.PostDTO, error) {
	posts, err := s.postRepo.GetPostByUserId(ctx, userId, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, err
	}
	return s.batchToPostDTO(posts)
}

// GetPostSelf 获取登录用户自己的帖子列表，含非公开状态
func (s *postServiceImpl) GetPostSelf(ctx context.Context, userId uint64, page, pageSize int) ([]*dto.PostDTO, error) {
	posts, err := s.postRepo.GetPostSelf(ctx, userId, pageSize, (page-1)*pageSize)
	if err != nil {
		return nil, err
	}
	return s.batchToPostDTO(posts)
}

// UpdatePostContent 更新帖子内容及媒体
func (s *postServiceImpl) UpdatePostContent(ctx context.Context, userID uint64, postID uint64, postDTO *dto.PostBaseDTO) error {
	oldPost, err := s.postRepo.GetPost(ctx, postID)
	if err != nil {
		return err
	}
	if oldPost.UserID != userID {
		return UnauthorizedError
	}

	post := &model.Post{}
	if err = copier.Copy(post, postDTO); err != nil {
		return err
	}
	if err = copier.Copy(&post.MediaList, &postDTO.Medias); err != nil {
		return err
	}

	post.ID = postID
	post.UserID = userID

	return s.postRepo.UpdatePostContent(ctx, post)
}

// DeletePost 删除帖子
func (s *postServiceImpl) DeletePost(ctx context.Context, userID uint64, postID uint64) error {
	// 1. 鉴权
	post, err := s.postRepo.GetPost(ctx, postID)
	if err != nil {
		return err
	}
	if post.UserID != userID {
		return UnauthorizedError
	}
	return s.postRepo.DeletePost(ctx, postID)
}

// toPostDTO 将 Model 转换为返回给前端的 DTO
func (s *postServiceImpl) toPostDTO(post *model.Post) (*dto.PostDTO, error) {
	out := &dto.PostDTO{}
	if err := copier.Copy(out, post); err != nil {
		return nil, err
	}
	if err := copier.Copy(&out.Medias, &post.MediaList); err != nil {
		return nil, err
	}

	if post.User.ID > 0 {
		out.UserID = post.User.ID
		if post.User.UserDetail.UserID > 0 {
			out.Nickname = post.User.UserDetail.Nickname
			out.AvatarURL = post.User.UserDetail.AvatarURL
		} else {
			out.Nickname = "用户_" + strconv.FormatUint(post.User.ID, 10)
			out.AvatarURL = "default_avatar.png"
		}
	} else {
		out.Nickname = "未知用户"
		out.AvatarURL = "default_avatar.png"
	}

	return out, nil
}

// batchToPostDTO 批量转换辅助
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
