package ai

type Capabilities struct {
	Service     string      `json:"service"`
	Action      string      `json:"action"`
	Embedding   [][]float64 `json:"embedding"`
	Description []string    `json:"description"`
}

type EmbendingResponse struct {
	Object string          `json:"object"`
	Data   []DataEmbending `json:"data"`
	Model  string          `json:"model"`
	Usage  struct {
		PromptTokens int     `json:"prompt_tokens"`
		TotalTokens  int     `json:"total_tokens"`
		Cost         float32 `json:"cost"`
	} `json:"usage"`
	Provider string `json:"provider"`
	Id       string `json:"id"`
}

type DataEmbending struct {
	Object    string    `json:"object"`
	Index     int       `json:"index"`
	Embedding []float64 `json:"embedding"`
}

func GetCapabilities() []Capabilities {
	return []Capabilities{
		{
			Service:     "jira",
			Action:      "create_project",
			Embedding:   [][]float64{},
			Description: []string{"Создай проект в Jira"},
		},
		{
			Service:     "jira",
			Action:      "create_issue",
			Embedding:   [][]float64{},
			Description: []string{"Создай новую задачу в Jira"},
		},
		{
			Service:     "jira",
			Action:      "create_project_jira",
			Embedding:   [][]float64{},
			Description: []string{"Создай новый проект в Jira", "Сделай проект в jira"},
		},
	}
}
