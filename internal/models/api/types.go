package apimodels

type Query struct {
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
}
