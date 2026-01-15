package handler

import (
	"Cornerstone/internal/pkg/consts"
	"Cornerstone/internal/pkg/redis"
	"Cornerstone/internal/pkg/response"
	"Cornerstone/internal/pkg/security"
	"Cornerstone/internal/service"
	"context"
	"encoding/base64"
	log "log/slog"
	"net/http"
	"strconv"
	"strings"
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
