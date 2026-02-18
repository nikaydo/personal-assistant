package database

type Capability struct {
	Service     string    `json:"service"`
	Action      string    `json:"action"`
	Embedding   []float32 `json:"embedding"`
	Description []string  `json:"description"`
}

type DataEmbending struct {
	Object    string    `json:"object"`
	Index     int       `json:"index"`
	Embedding []float32 `json:"embedding"`
}

func GetCapabilities() []Capability {
	return []Capability{
		{
			Service:     "jira",
			Action:      "create_project",
			Embedding:   []float32{},
			Description: []string{"Создай проект в Jira"},
		},
		{
			Service:     "jira",
			Action:      "create_issue",
			Embedding:   []float32{},
			Description: []string{"Создай новую задачу в Jira"},
		},
		{
			Service:     "jira",
			Action:      "create_project_jira",
			Embedding:   []float32{},
			Description: []string{"Создай новый проект в Jira", "Сделай проект в jira"},
		},
	}
}
