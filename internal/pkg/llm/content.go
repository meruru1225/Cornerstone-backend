package llm

type Content struct {
	Title   *string `json:"title,omitempty"`
	Content string  `json:"content"`
}
