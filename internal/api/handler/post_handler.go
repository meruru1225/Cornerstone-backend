package handler

import "Cornerstone/internal/service"

type PostHandler struct {
	postSvc service.PostService
}

func NewPostHandler(postSvc service.PostService) *PostHandler {
	return &PostHandler{
		postSvc: postSvc,
	}
}
