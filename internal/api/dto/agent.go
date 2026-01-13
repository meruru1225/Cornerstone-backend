package dto

type AgentConverseRequest struct {
	ChatID   string `json:"chat_id"`
	Question string `json:"question"`
}
