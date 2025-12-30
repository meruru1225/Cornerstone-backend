package api

import "Cornerstone/internal/api/handler"

// HandlersGroup 封装了所有已初始化的 Handler 实例
type HandlersGroup struct {
	UserHandler       *handler.UserHandler
	UserFollowHandler *handler.UserFollowHandler
	PostHandler       *handler.PostHandler
}
