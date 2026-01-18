package handler

import (
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/redis"
	"Cornerstone/internal/pkg/response"
	"Cornerstone/internal/service"
	"context"
	"encoding/base64"
	log "log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

func (s *WsHandler) GetWSTicket(c *gin.Context) {
	userID := c.GetUint64("user_id")
	ticket := uuid.NewString()
	key := consts.WebSocketTicketKey + ticket
	if err := redis.SetWithExpiration(c.Request.Context(), key, userID, time.Minute*5); err != nil {
		response.Error(c, service.UnExpectedError)
		return
	}
	response.Success(c, map[string]interface{}{
		"ticket": ticket,
	})
}

func (s *WsHandler) Connect(c *gin.Context) {
	ticket := c.Query("ticket")
	if ticket == "" {
		response.Error(c, service.UnauthorizedError)
		return
	}

	key := consts.WebSocketTicketKey + ticket
	value, err := redis.GetValue(c.Request.Context(), key)
	if err != nil || value == "" {
		response.Error(c, service.UnauthorizedError)
		return
	}
	userID, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		response.Error(c, service.UnauthorizedError)
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error("WS 协议升级失败", "err", err)
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	userChannel := consts.IMUserKey + strconv.FormatUint(userID, 10)

	// 订阅 Redis 个人总线
	pubsub := redis.Subscribe(context.Background(), userChannel)
	defer func() {
		_ = pubsub.Close() //
	}()

	log.Info("用户 WS 连接已建立并订阅个人频道", "userID", userID, "channel", userChannel)

	stopChan := make(chan struct{})

	// 读循环：监听客户端主动断开或网络异常
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				close(stopChan)
				return
			}
		}
	}()

	// 写循环：从 Redis 接收消息并推送到客户端
	redisCh := pubsub.Channel()
	for {
		select {
		case msg := <-redisCh:
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

			payloadStr := strings.Trim(msg.Payload, "\"")
			var rawData []byte
			var err error

			rawData, err = base64.StdEncoding.DecodeString(payloadStr)
			if err != nil {
				rawData = []byte(payloadStr)
			}

			err = conn.WriteMessage(websocket.TextMessage, rawData)
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
