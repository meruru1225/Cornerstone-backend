package service

import (
	"Cornerstone/internal/api/dto"
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

// IMService 即时通讯服务
type IMService interface {
	SendMessage(ctx context.Context, senderID uint64, req *dto.SendMessageReq) (*dto.MessageDTO, error)
	GetChatHistory(ctx context.Context, convID uint64, lastSeq uint64, pageSize int) ([]*dto.MessageDTO, error)
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
		retryChan:   make(chan *mongo.Message, 2048), // 充足的缓冲区
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
	newSeq, err := s.convRepo.IncrMaxSeq(ctx, req.ConversationID, req.Content, int8(req.MsgType), senderID)
	if err != nil {
		return nil, fmt.Errorf("failed to increment sequence: %w", err)
	}

	msgModel := &mongo.Message{
		ConversationID: req.ConversationID,
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
		log.Warn("MongoDB save failed, message sent to retry channel", "seq", newSeq, "err", err)
		select {
		case s.retryChan <- msgModel:
		default:
			log.Error("Retry channel overflow, message persistence risk", "seq", newSeq)
		}
	}

	go s.publishMessageToRedis(context.Background(), msgModel)

	return s.toMessageDTO(msgModel), nil
}

// GetChatHistory 拉取历史
func (s *imServiceImpl) GetChatHistory(ctx context.Context, convID uint64, lastSeq uint64, pageSize int) ([]*dto.MessageDTO, error) {
	models, err := s.messageRepo.GetHistory(ctx, convID, lastSeq, pageSize)
	if err != nil {
		return nil, err
	}

	// 空洞自愈：若第一页数据发现 Mongo 落后于 MySQL 存根，则动态补足
	if lastSeq == 0 {
		conv, err := s.convRepo.GetConversation(ctx, convID)
		if err == nil {
			// 检测是否存在未同步到 Mongo 的最新消息
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
				// 将补救的存根消息置于结果集首位
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

// GetConversationList 获取会话
func (s *imServiceImpl) GetConversationList(ctx context.Context, userID uint64) ([]*dto.ConversationDTO, error) {
	members, err := s.convRepo.GetUserConversationList(ctx, userID)
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

		// 单聊场景：解析 PeerID (对方ID)
		if m.Conversation.Type == 1 {
			peerID, err := s.parsePeerID(m.Conversation.PeerKey, userID)
			if err == nil {
				d.PeerID = peerID
			}
		}
		res = append(res, d)
	}
	return res, nil
}

func (s *imServiceImpl) MarkAsRead(ctx context.Context, userID uint64, convID uint64, seq uint64) error {
	conv, err := s.convRepo.GetConversation(ctx, convID)
	if err != nil {
		return err
	}

	targetSeq := seq
	if targetSeq > conv.MaxMsgSeq {
		targetSeq = conv.MaxMsgSeq
	}

	err = s.convRepo.UpdateReadSeq(ctx, convID, userID, targetSeq)
	if err != nil {
		return err
	}

	go s.publishReadReceipt(convID, userID, targetSeq)

	return nil
}

// Close 优雅关闭服务
func (s *imServiceImpl) Close() {
	close(s.stopChan)
	s.wg.Wait() // 等待所有 Worker 协程安全退出
	log.Info("IMService shut down gracefully")
}

// calibrationWorker 后台异步校准协程
func (s *imServiceImpl) calibrationWorker() {
	defer s.wg.Done()
	for {
		select {
		case msg := <-s.retryChan:
			backoff := time.Second
			for i := 0; i < 3; i++ {
				// 为重试请求设置独立的上下文超时
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				err := s.messageRepo.SaveMessage(ctx, msg)
				cancel()

				if err == nil {
					log.Debug("Calibration success", "seq", msg.Seq)
					break
				}
				log.Warn("Calibration attempt failed", "seq", msg.Seq, "attempt", i+1)
				time.Sleep(backoff)
				backoff *= 2 // 指数退避
			}
		case <-s.stopChan:
			return
		}
	}
}

// parsePeerID 解析 PeerID
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

// publishMessageToRedis 将消息推送到 Redis 消息总线
func (s *imServiceImpl) publishMessageToRedis(ctx context.Context, msg *mongo.Message) {
	channel := consts.IMConversationKey + strconv.FormatUint(msg.ConversationID, 10)
	data, _ := json.Marshal(s.toMessageDTO(msg))

	// 使用 Redis Publish
	if err := redis.Publish(ctx, channel, data); err != nil {
		log.Error("Redis Publish failed", "channel", channel, "err", err)
	}
}

// publishReadReceipt 发布已读信号到 Redis
func (s *imServiceImpl) publishReadReceipt(convID uint64, userID uint64, readSeq uint64) {
	receipt := &dto.ReadReceiptDTO{
		ConversationID: convID,
		UserID:         userID,
		ReadSeq:        readSeq,
		Type:           "READ_RECEIPT",
	}

	channel := consts.IMConversationKey + strconv.FormatUint(convID, 10)

	// 序列化并发布
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := redis.Publish(ctx, channel, receipt); err != nil {
		log.Error("发布已读回执失败", "convID", convID, "err", err)
	}
}

// toMessageDTO 转换为 DTO
func (s *imServiceImpl) toMessageDTO(m *mongo.Message) *dto.MessageDTO {
	return &dto.MessageDTO{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		SenderID:       m.SenderID,
		MsgType:        m.MsgType,
		Content:        m.Content,
		Payload:        m.Payload,
		Seq:            m.Seq,
		CreatedAt:      m.CreatedAt,
	}
}
