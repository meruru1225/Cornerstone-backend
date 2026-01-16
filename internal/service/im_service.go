package service

import (
	"Cornerstone/internal/api/dto"
	"Cornerstone/internal/model"
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/mongo"
	"Cornerstone/internal/pkg/redis"
	"Cornerstone/internal/repository"
	"context"
	"fmt"
	log "log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/goccy/go-json"
)

// IMService 即时通讯服务接口定义
type IMService interface {
	SendMessage(ctx context.Context, senderID uint64, req *dto.SendMessageReq) (*dto.MessageDTO, error)
	GetOrCreateConversation(ctx context.Context, userID, targetUserID uint64, convType int8) (uint64, error)
	GetChatHistory(ctx context.Context, userID uint64, convID uint64, lastSeq uint64, pageSize int) ([]*dto.MessageDTO, error)
	SyncMessages(ctx context.Context, userID uint64, convID uint64, lastSeq uint64, pageSize int) ([]*dto.MessageDTO, error)
	GetConversationList(ctx context.Context, userID uint64) ([]*dto.ConversationDTO, error)
	MarkAsRead(ctx context.Context, userID uint64, convID uint64, seq uint64) error
	Close()
}

type imServiceImpl struct {
	convRepo    repository.ConversationRepo
	messageRepo mongo.MessageRepo
	retryChan   chan *mongo.Message
	wg          sync.WaitGroup
	stopChan    chan struct{}
}

// NewIMService 构造函数：初始化服务并启动异步校准工作池
func NewIMService(convRepo repository.ConversationRepo, messageRepo mongo.MessageRepo) IMService {
	s := &imServiceImpl{
		convRepo:    convRepo,
		messageRepo: messageRepo,
		retryChan:   make(chan *mongo.Message, 2048),
		stopChan:    make(chan struct{}),
	}

	workerCount := 5
	s.wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go s.calibrationWorker()
	}

	return s
}

// SendMessage 发送消息
func (s *imServiceImpl) SendMessage(ctx context.Context, senderID uint64, req *dto.SendMessageReq) (*dto.MessageDTO, error) {
	var convID = req.ConversationID
	var targetID = req.TargetUserID

	// 1确定会话 ID 与 目标用户 ID
	if convID == 0 {
		if targetID == 0 {
			return nil, fmt.Errorf("target_user_id is required for new conversation")
		}
		id, err := s.GetOrCreateConversation(ctx, senderID, targetID, 1)
		if err != nil {
			return nil, err
		}
		convID = id
	} else {
		// 校验成员权限并解析 targetID
		conv, err := s.convRepo.GetConversation(ctx, convID)
		if err != nil {
			return nil, err
		}
		isMember, _ := s.convRepo.IsMember(ctx, convID, senderID)
		if !isMember {
			return nil, fmt.Errorf("not a member of this conversation")
		}
		targetID, _ = s.parsePeerID(conv.PeerKey, senderID)
	}

	// MySQL 原子定序
	newSeq, err := s.convRepo.IncrMaxSeq(ctx, convID, req.Content, int8(req.MsgType), senderID)
	if err != nil {
		return nil, err
	}

	// 构造并存入 MongoDB
	msgModel := &mongo.Message{
		ConversationID: convID,
		SenderID:       senderID,
		MsgType:        req.MsgType,
		Content:        req.Content,
		Payload:        mongo.MMap(req.Payload),
		Seq:            newSeq,
		CreatedAt:      time.Now(),
	}

	writeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := s.messageRepo.SaveMessage(writeCtx, msgModel); err != nil {
		select {
		case s.retryChan <- msgModel:
		default:
		}
	}

	// 推送到接收者的【用户个人频道】
	_ = s.publishMessageToRedis(context.Background(), msgModel, targetID)

	return s.toMessageDTO(msgModel), nil
}

// GetOrCreateConversation 针对单聊：获取或创建会话
func (s *imServiceImpl) GetOrCreateConversation(ctx context.Context, userID, targetUserID uint64, convType int8) (uint64, error) {
	// 生成单聊唯一的 PeerKey
	var peerKey string
	if userID < targetUserID {
		peerKey = fmt.Sprintf("%d_%d", userID, targetUserID)
	} else {
		peerKey = fmt.Sprintf("%d_%d", targetUserID, userID)
	}

	conv, err := s.convRepo.GetConversationByPeerKey(ctx, peerKey)
	if err == nil {
		return conv.ID, nil
	}

	newConv := &model.Conversation{
		Type:          convType,
		PeerKey:       peerKey,
		LastMessageAt: time.Now(),
	}
	members := []*model.ConversationMember{
		{UserID: userID, IsVisible: 1, JoinedAt: time.Now()},
		{UserID: targetUserID, IsVisible: 1, JoinedAt: time.Now()},
	}

	if err := s.convRepo.CreateConversation(ctx, newConv, members); err != nil {
		return 0, err
	}
	return newConv.ID, nil
}

// GetChatHistory 拉取历史，包含空洞自愈
func (s *imServiceImpl) GetChatHistory(ctx context.Context, userID uint64, convID uint64, lastSeq uint64, pageSize int) ([]*dto.MessageDTO, error) {
	isMember, err := s.convRepo.IsMember(ctx, convID, userID)
	if err != nil || !isMember {
		return nil, UnauthorizedError
	}

	models, err := s.messageRepo.GetHistory(ctx, convID, lastSeq, pageSize)
	if err != nil {
		return nil, err
	}

	if lastSeq == 0 {
		conv, err := s.convRepo.GetConversation(ctx, convID)
		if err == nil {
			hasGap := (len(models) == 0 && conv.MaxMsgSeq > 0) || (len(models) > 0 && models[0].Seq < conv.MaxMsgSeq)
			if hasGap {
				stub := &dto.MessageDTO{
					ConversationID: conv.ID,
					Content:        conv.LastMsgContent,
					MsgType:        int(conv.LastMsgType),
					SenderID:       conv.LastSenderID,
					Seq:            conv.MaxMsgSeq,
					CreatedAt:      conv.LastMessageAt,
				}
				res := []*dto.MessageDTO{stub}
				for _, m := range models {
					res = append(res, s.toMessageDTO(m))
				}
				return res, nil
			}
		}
	}

	res := make([]*dto.MessageDTO, 0, len(models))
	for _, m := range models {
		res = append(res, s.toMessageDTO(m))
	}
	return res, nil
}

func (s *imServiceImpl) SyncMessages(ctx context.Context, userID uint64, convID uint64, lastSeq uint64, pageSize int) ([]*dto.MessageDTO, error) {
	isMember, err := s.convRepo.IsMember(ctx, convID, userID)
	if err != nil || !isMember {
		return nil, UnauthorizedError
	}

	models, err := s.messageRepo.SyncMessages(ctx, convID, lastSeq, pageSize)
	if err != nil {
		return nil, err
	}
	res := make([]*dto.MessageDTO, 0, len(models))
	for _, m := range models {
		res = append(res, s.toMessageDTO(m))
	}
	return res, nil
}

// GetConversationList 获取会话列表
func (s *imServiceImpl) GetConversationList(ctx context.Context, userID uint64) ([]*dto.ConversationDTO, error) {
	members, err := s.convRepo.GetUserConversationMemList(ctx, userID)
	if err != nil {
		return nil, err
	}

	res := make([]*dto.ConversationDTO, 0, len(members))
	for _, m := range members {
		d := &dto.ConversationDTO{
			ConversationID: m.ConversationID,
			Type:           m.Conversation.Type,
			LastMsgContent: m.Conversation.LastMsgContent,
			LastMsgType:    m.Conversation.LastMsgType,
			LastSenderID:   m.Conversation.LastSenderID,
			LastMessageAt:  m.Conversation.LastMessageAt,
			UnreadCount:    m.UnreadCount,
			IsMuted:        m.IsMuted == 1,
			IsPinned:       m.IsPinned == 1,
		}

		if m.Conversation.Type == 1 {
			peerID, _ := s.parsePeerID(m.Conversation.PeerKey, userID)
			d.PeerID = peerID
		}
		res = append(res, d)
	}
	return res, nil
}

// MarkAsRead 标记已读
func (s *imServiceImpl) MarkAsRead(ctx context.Context, userID uint64, convID uint64, seq uint64) error {
	isMember, err := s.convRepo.IsMember(ctx, convID, userID)
	if err != nil || !isMember {
		return UnauthorizedError
	}

	conv, err := s.convRepo.GetConversation(ctx, convID)
	if err != nil {
		return err
	}

	targetSeq := seq
	if targetSeq > conv.MaxMsgSeq {
		targetSeq = conv.MaxMsgSeq
	}

	if err = s.convRepo.UpdateReadSeq(ctx, convID, userID, targetSeq); err != nil {
		return err
	}

	peerID, err := s.parsePeerID(conv.PeerKey, userID)
	if err != nil {
		return err
	}
	go func() {
		err = s.publishReadReceipt(convID, userID, peerID, targetSeq)
		if err != nil {
			log.Error("Failed to publish read receipt", "err", err)
		}
	}()

	return nil
}

// publishMessageToRedis 发布消息到接收者的用户频道
func (s *imServiceImpl) publishMessageToRedis(ctx context.Context, msg *mongo.Message, targetUserID uint64) error {
	data, err := json.Marshal(s.toMessageDTO(msg))
	if err != nil {
		return err
	}
	channel := consts.IMUserKey + strconv.FormatUint(targetUserID, 10)
	return redis.Publish(ctx, channel, data)
}

// publishReadReceipt 发布已读回执到对方频道
func (s *imServiceImpl) publishReadReceipt(convID, fromUID, toPeerID, seq uint64) error {
	receipt := &dto.ReadReceiptDTO{
		ConversationID: convID,
		UserID:         fromUID,
		ReadSeq:        seq,
		Type:           "READ_RECEIPT",
	}
	data, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	channel := consts.IMUserKey + strconv.FormatUint(toPeerID, 10)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return redis.Publish(ctx, channel, data)
}

func (s *imServiceImpl) Close() {
	close(s.stopChan)
	s.wg.Wait()
	log.Info("IMService shut down gracefully")
}

func (s *imServiceImpl) calibrationWorker() {
	defer s.wg.Done()
	for {
		select {
		case msg := <-s.retryChan:
			backoff := time.Second
			for i := 0; i < 3; i++ {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				err := s.messageRepo.SaveMessage(ctx, msg)
				cancel()
				if err == nil {
					break
				}
				time.Sleep(backoff)
				backoff *= 2
			}
		case <-s.stopChan:
			return
		}
	}
}

func (s *imServiceImpl) parsePeerID(peerKey string, currentUserID uint64) (uint64, error) {
	var u1, u2 uint64
	_, err := fmt.Sscanf(peerKey, "%d_%d", &u1, &u2)
	if err != nil {
		return 0, err
	}
	if u1 == currentUserID {
		return u2, nil
	}
	return u1, nil
}

func (s *imServiceImpl) toMessageDTO(m *mongo.Message) *dto.MessageDTO {
	return &dto.MessageDTO{
		ID: m.ID, ConversationID: m.ConversationID, SenderID: m.SenderID,
		MsgType: m.MsgType, Content: m.Content, Payload: m.Payload,
		Seq: m.Seq, CreatedAt: m.CreatedAt,
	}
}
