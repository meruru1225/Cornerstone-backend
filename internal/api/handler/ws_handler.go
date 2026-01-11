package handler

import (
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/redis"
	"Cornerstone/internal/pkg/response"
	"Cornerstone/internal/pkg/security"
	"Cornerstone/internal/service"
	"context"
	log "log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type WsHandler struct {
	imService service.IMService
}

func NewWsHandler(im service.IMService) *WsHandler {
	return &WsHandler{imService: im}
}

func (s *WsHandler) Connect(c *gin.Context) {
	// 鉴权
	token := c.Query("token")
	if token == "" {
		response.Error(c, service.UnauthorizedError)
		return
	}
	claims, err := security.ValidateToken(token)
	if err != nil {
		log.Warn("WS 鉴权失败", "err", err)
		response.Error(c, service.UnauthorizedError)
		return
	}
	userID := claims.UserID

	// 升级 Websocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("WS 协议升级失败", "err", err)
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	// 获取用户参与的所有会话，订阅 Redis 频道
	list, err := s.imService.GetConversationList(context.Background(), userID)
	if err != nil {
		log.Error("获取会话列表失败", "userID", userID, "err", err)
		return
	}

	var channels []string
	for _, conv := range list {
		channels = append(channels, consts.IMConversationKey+strconv.FormatUint(conv.ConversationID, 10))
	}

	// 订阅 Redis 总线
	pubsub := redis.Subscribe(context.Background(), channels...)
	defer func() {
		_ = pubsub.Close()
	}()

	log.Info("用户 WS 连接已建立", "userID", userID, "channels", len(channels))

	stopChan := make(chan struct{})

	// 读循环：监听客户端主动断开
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				close(stopChan)
				return
			}
		}
	}()

	// 写循环：监听 Redis 并推送至客户端
	redisCh := pubsub.Channel()
	if err != nil {
		log.Error("WS 设置写超时失败", "userID", userID, "err", err)
		return
	}
	for {
		select {
		case msg := <-redisCh:
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload))
			if err != nil {
				log.Error("WS 推送失败", "userID", userID, "err", err)
				return
			}
		case <-stopChan:
			log.Info("用户 WS 连接已断开", "userID", userID)
			return
		}
	}
}
