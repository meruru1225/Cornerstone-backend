package service

import (
	"Cornerstone/internal/api/dto"
	"Cornerstone/internal/model"
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/minio"
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
	userRepo    repository.UserRepo
	convRepo    repository.ConversationRepo
	messageRepo mongo.MessageRepo
	retryChan   chan *mongo.Message
	wg          sync.WaitGroup
	stopChan    chan struct{}
}

// NewIMService 构造函数：初始化服务并启动异步校准工作池
func NewIMService(userRepo repository.UserRepo, convRepo repository.ConversationRepo, messageRepo mongo.MessageRepo) IMService {
	s := &imServiceImpl{
		userRepo:    userRepo,
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

	// 确定会话 ID 与 目标用户 ID
	if convID == 0 {
		id, err := s.GetOrCreateConversation(ctx, senderID, targetID, 1)
		if err != nil {
			return nil, err
		}
		convID = id
	} else {
		conv, err := s.convRepo.GetConversation(ctx, convID)
		if err != nil {
			return nil, err
		}
		targetID, _ = s.parsePeerID(conv.PeerKey, senderID)
	}

	// MySQL 原子定序
	newSeq, err := s.convRepo.IncrMaxSeq(ctx, convID, req.Content, int8(req.MsgType), senderID)
	if err != nil {
		return nil, err
	}

	_ = s.convRepo.UpdateReadSeq(ctx, convID, senderID, newSeq)

	var mongoPayload []mongo.Payload
	if len(req.Payload) > 0 {
		mongoPayload = make([]mongo.Payload, 0, len(req.Payload))
		for _, p := range req.Payload {
			mb := mongo.Payload{
				MimeType: p.MimeType,
				MediaURL: p.MediaURL,
				Width:    p.Width,
				Height:   p.Height,
				Duration: p.Duration,
			}
			if p.CoverURL != nil {
				mb.CoverURL = *p.CoverURL
			}
			mongoPayload = append(mongoPayload, mb)
		}
	}

	msgModel := &mongo.Message{
		ConversationID: convID,
		SenderID:       senderID,
		MsgType:        req.MsgType,
		Content:        req.Content,
		Payload:        mongoPayload,
		Seq:            newSeq,
		CreatedAt:      time.Now(),
	}

	writeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.messageRepo.SaveMessage(writeCtx, msgModel); err != nil {
		select {
		case s.retryChan <- msgModel:
		default:
		}
	}

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

func (s *imServiceImpl) GetConversationList(ctx context.Context, userID uint64) ([]*dto.ConversationDTO, error) {
	members, err := s.convRepo.GetUserConversationMemList(ctx, userID)
	if err != nil {
		return nil, err
	}

	res := make([]*dto.ConversationDTO, 0, len(members))
	peerIDs := make([]uint64, 0)
	convIDs := make([]uint64, 0)

	for _, m := range members {
		d := &dto.ConversationDTO{
			ConversationID: m.ConversationID,
			Type:           m.Conversation.Type,
			MyReadSeq:      m.ReadMsgSeq,
			MaxSeq:         m.Conversation.MaxMsgSeq,
			LastMsgContent: m.Conversation.LastMsgContent,
			LastMsgType:    m.Conversation.LastMsgType,
			LastSenderID:   m.Conversation.LastSenderID,
			LastMessageAt:  m.Conversation.LastMessageAt,
			UnreadCount:    m.UnreadCount,
			IsMuted:        m.IsMuted == 1,
			IsPinned:       m.IsPinned == 1,
		}

		if m.Conversation.Type == 1 {
			pID, _ := s.parsePeerID(m.Conversation.PeerKey, userID)
			d.PeerID = pID
			peerIDs = append(peerIDs, pID)
			convIDs = append(convIDs, m.ConversationID)
		}
		res = append(res, d)
	}

	if len(peerIDs) > 0 {
		userMap := make(map[uint64]*model.UserDetail)
		if users, err := s.userRepo.GetUserSimpleInfoByIds(ctx, peerIDs); err == nil {
			for _, u := range users {
				userMap[u.UserID] = u
			}
		}

		peerReadMap, _ := s.convRepo.GetConvPeersReadSeq(ctx, convIDs, peerIDs)

		for _, d := range res {
			if u, ok := userMap[d.PeerID]; ok {
				d.CoverURL = minio.GetPublicURL(u.AvatarURL)
				d.Title = u.Nickname
			}
			// 填对方已读进度 (仅单聊)
			if d.Type == 1 {
				if seq, ok := peerReadMap[d.ConversationID]; ok {
					d.PeerReadSeq = seq
				}
			}
		}
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
	dtoPayload := make([]dto.MediasBaseDTO, 0, len(m.Payload))
	for _, p := range m.Payload {
		item := dto.MediasBaseDTO{
			MimeType: p.MimeType,
			MediaURL: minio.GetPublicURL(p.MediaURL),
			Width:    p.Width,
			Height:   p.Height,
			Duration: p.Duration,
		}
		if p.CoverURL != "" {
			url := minio.GetPublicURL(p.CoverURL)
			item.CoverURL = &url
		}
		dtoPayload = append(dtoPayload, item)
	}

	return &dto.MessageDTO{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		SenderID:       m.SenderID,
		MsgType:        m.MsgType,
		Content:        m.Content,
		Payload:        dtoPayload,
		Seq:            m.Seq,
		CreatedAt:      m.CreatedAt,
	}
}
