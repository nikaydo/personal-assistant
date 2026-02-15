package jira

type ProjectSearchOptions struct {
	Options    ProjectSearchOptionsScheme `json:"options"`
	StartAt    int                        `json:"startAt"`
	MaxResults int                        `json:"maxResults"`
}

type ProjectSearchOptionsScheme struct {
	OrderBy    string   `json:"orderBy,omitempty"`
	IDs        []int    `json:"ids,omitempty"`
	Keys       []string `json:"keys,omitempty"`
	Query      string   `json:"query,omitempty"`
	TypeKeys   []string `json:"typeKeys,omitempty"`
	CategoryID int      `json:"categoryID,omitempty"`
	Action     string   `json:"action,omitempty"`
}
