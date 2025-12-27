package llm

type ClassifyMessage struct {
	MainTag string   `json:"main_tag"`
	Tags    []string `json:"tags"`
}
