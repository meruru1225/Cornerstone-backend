package dto

// CommentCreateDTO 创建评论请求
type CommentCreateDTO struct {
	PostID        uint64         `json:"post_id" binding:"required"`
	Content       string         `json:"content" binding:"required,max=1000"`
	MediaInfo     *MediasBaseDTO `json:"media_info"` // 复用之前的媒体 DTO
	RootID        uint64         `json:"root_id"`    // 0 表示一级评论
	ParentID      uint64         `json:"parent_id"`  // 父评论 ID
	ReplyToUserID uint64         `json:"reply_to_user_id"`
}

// CommentDTO 评论返回详情
type CommentDTO struct {
	ID              uint64           `json:"id"`
	PostID          uint64           `json:"post_id"`
	UserID          uint64           `json:"user_id"`
	Nickname        string           `json:"nickname"`
	AvatarURL       string           `json:"avatar_url"`
	Content         string           `json:"content"`
	MediaInfo       []*MediasBaseDTO `json:"media_info"`
	RootID          uint64           `json:"root_id"`
	ParentID        uint64           `json:"parent_id"`
	ReplyToUserID   uint64           `json:"reply_to_user_id"`
	ReplyToNickname string           `json:"reply_to_nickname"`
	LikesCount      int              `json:"likes_count"`
	CreatedAt       string           `json:"created_at"`

	SubComments     []*CommentDTO `json:"sub_comments"`
	SubCommentCount int64         `json:"sub_comment_count"`
}

// PostActionStateDTO 帖子交互状态数据
type PostActionStateDTO struct {
	LikeCount    int64 `json:"like_count"`
	CollectCount int64 `json:"collect_count"`
	CommentCount int64 `json:"comment_count"`
	ViewCount    int64 `json:"view_count"`
	IsLiked      bool  `json:"is_liked"`
	IsCollected  bool  `json:"is_collected"`
}

// PostBatchLikesReq 批量获取点赞数请求
type PostBatchLikesReq struct {
	PostIDs []uint64 `json:"post_ids" binding:"required,min=1,max=100"`
}

// PostBatchLikesDTO 批量获取点赞数响应
type PostBatchLikesDTO struct {
	Likes map[uint64]int64 `json:"likes"`
}

// PostActionReq 点赞/收藏通用请求
type PostActionReq struct {
	PostID uint64 `json:"post_id" binding:"required"`
	Action int    `json:"action" binding:"required,oneof=1 2"` // 1:执行, 2:取消
}
