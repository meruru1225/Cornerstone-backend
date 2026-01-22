package repository

import (
	"Cornerstone/internal/model"
	"context"
	"time"

	"gorm.io/gorm"
)

type ConversationRepo interface {
	CreateConversation(ctx context.Context, conv *model.Conversation, members []*model.ConversationMember) error
	GetConversation(ctx context.Context, convID uint64) (*model.Conversation, error)
	GetConversationByPeerKey(ctx context.Context, peerKey string) (*model.Conversation, error)
	IsMember(ctx context.Context, convID uint64, userID uint64) (bool, error)

	UpdateReadSeq(ctx context.Context, convID, userID, seq uint64) error
	IncrMaxSeq(ctx context.Context, convID uint64, lastMsg string, msgType int8, senderID uint64) (uint64, error)

	GetUserConversationMemList(ctx context.Context, userID uint64) ([]*model.ConversationMember, error)
	GetConvPeersReadSeq(ctx context.Context, convIDs []uint64, peerIDs []uint64) (map[uint64]uint64, error)
	GetTotalUnreadCount(ctx context.Context, userID uint64) (int64, error)
}

type conversationRepoImpl struct {
	db *gorm.DB
}

func NewConversationRepo(db *gorm.DB) ConversationRepo {
	return &conversationRepoImpl{db: db}
}

// CreateConversation 开启事务创建会话及初始成员
func (s *conversationRepoImpl) CreateConversation(ctx context.Context, conv *model.Conversation, members []*model.ConversationMember) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(conv).Error; err != nil {
			return err
		}
		for _, m := range members {
			m.ConversationID = conv.ID
			m.JoinedAt = time.Now()
			if err := tx.Create(m).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetConversation 根据会话 ID 获取会话
func (s *conversationRepoImpl) GetConversation(ctx context.Context, convID uint64) (*model.Conversation, error) {
	var conv model.Conversation
	err := s.db.WithContext(ctx).First(&conv, convID).Error
	return &conv, err
}

// GetConversationByPeerKey 根据会话标识获取会话
func (s *conversationRepoImpl) GetConversationByPeerKey(ctx context.Context, peerKey string) (*model.Conversation, error) {
	var conv model.Conversation
	err := s.db.WithContext(ctx).Where("peer_key = ?", peerKey).First(&conv).Error
	return &conv, err
}

// IsMember 检查用户是否是会话成员
func (s *conversationRepoImpl) IsMember(ctx context.Context, convID uint64, userID uint64) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", convID, userID).
		Count(&count).Error
	return count > 0, err
}

// UpdateReadSeq 更新用户已读进度 (已读回执)
func (s *conversationRepoImpl) UpdateReadSeq(ctx context.Context, convID, userID, seq uint64) error {
	return s.db.WithContext(ctx).Model(&model.ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", convID, userID).
		Update("read_msg_seq", seq).Error
}

// IncrMaxSeq 核心定序逻辑：利用 MySQL 行锁确保 Seq 绝对递增
func (s *conversationRepoImpl) IncrMaxSeq(ctx context.Context, convID uint64, lastMsg string, msgType int8, senderID uint64) (uint64, error) {
	var maxSeq uint64
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 原子更新序列号与预览信息
		err := tx.Model(&model.Conversation{}).Where("id = ?", convID).
			Updates(map[string]interface{}{
				"max_msg_seq":      gorm.Expr("max_msg_seq + 1"),
				"last_msg_content": lastMsg,
				"last_msg_type":    msgType,
				"last_sender_id":   senderID,
				"last_message_at":  time.Now(),
			}).Error
		if err != nil {
			return err
		}
		// 唤醒所有成员会话可见性 (用于“删除会话”后的自动浮现)
		tx.Model(&model.ConversationMember{}).Where("conversation_id = ?", convID).Update("is_visible", 1)

		// 读取并返回自增后的最新 Seq
		return tx.Model(&model.Conversation{}).Select("max_msg_seq").Where("id = ?", convID).Scan(&maxSeq).Error
	})
	return maxSeq, err
}

// GetUserConversationMemList 联表查询，利用嵌套 Model 自动装配
func (s *conversationRepoImpl) GetUserConversationMemList(ctx context.Context, userID uint64) ([]*model.ConversationMember, error) {
	var members []*model.ConversationMember
	// 使用 Conversation__ 别名配合 GORM 的嵌套填充特性
	err := s.db.WithContext(ctx).Table("conversation_members m").
		Select("m.*, "+
			"c.id AS `Conversation__id`, c.type AS `Conversation__type`, "+
			"c.peer_key AS `Conversation__peer_key`, "+
			"c.max_msg_seq AS `Conversation__max_msg_seq`, "+
			"c.last_msg_content AS `Conversation__last_msg_content`, "+
			"c.last_msg_type AS `Conversation__last_msg_type`, "+
			"c.last_sender_id AS `Conversation__last_sender_id`, "+
			"c.last_message_at AS `Conversation__last_message_at`, "+
			"(c.max_msg_seq - m.read_msg_seq) AS unread_count").
		Joins("JOIN conversations c ON m.conversation_id = c.id").
		Where("m.user_id = ? AND m.is_visible = 1", userID).
		Order("m.is_pinned DESC, c.last_message_at DESC").
		Find(&members).Error
	return members, err
}

// GetConvPeersReadSeq 批量获取指定会话中对方的已读进度
func (s *conversationRepoImpl) GetConvPeersReadSeq(ctx context.Context, convIDs []uint64, peerIDs []uint64) (map[uint64]uint64, error) {
	type Result struct {
		ConversationID uint64
		ReadMsgSeq     uint64
	}
	var results []Result
	// 查询条件：会话在列表内，且用户是我们的对手 ID 列表
	err := s.db.WithContext(ctx).Table("conversation_members").
		Select("conversation_id, read_msg_seq").
		Where("conversation_id IN ? AND user_id IN ?", convIDs, peerIDs).
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	resMap := make(map[uint64]uint64)
	for _, r := range results {
		resMap[r.ConversationID] = r.ReadMsgSeq
	}
	return resMap, nil
}

// GetTotalUnreadCount 计算全局未读数
func (s *conversationRepoImpl) GetTotalUnreadCount(ctx context.Context, userID uint64) (int64, error) {
	var total int64
	err := s.db.WithContext(ctx).Table("conversation_members m").
		Joins("JOIN conversations c ON m.conversation_id = c.id").
		Where("m.user_id = ?", userID).
		Select("SUM(CASE WHEN c.max_msg_seq > m.read_msg_seq THEN c.max_msg_seq - m.read_msg_seq ELSE 0 END)").
		Scan(&total).Error
	return total, err
}
