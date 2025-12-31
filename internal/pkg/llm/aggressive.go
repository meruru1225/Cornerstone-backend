package llm

type TagAggressive struct {
	MainTags  []string `json:"main_tags"`
	Tags      []string `json:"tags"`
	Summaries []string `json:"summaries"`
}
