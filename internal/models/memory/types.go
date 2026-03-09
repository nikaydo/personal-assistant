package memory

import openroutermodels "github.com/nikaydo/personal-assistant/internal/models/openrouter"

type SystemSettings struct {
	Tone               string `json:"tone,omitempty"`               // neutral, polite, rude, sarcastic
	Verbosity          string `json:"verbosity,omitempty"`          // short, normal, detailed
	Formality          string `json:"formality,omitempty"`          // casual, neutral, formal
	Language           string `json:"language,omitempty"`           // auto, english, russian
	Emotion            string `json:"emotion,omitempty"`            // neutral, happy, angry, sarcastic
	HumorLevel         string `json:"humorLevel,omitempty"`         // none, low, medium, high
	DetailLevel        string `json:"detailLevel,omitempty"`        // short, normal, detailed
	KnowledgeFocus     string `json:"knowledgeFocus,omitempty"`     // technical, creative, general, personal
	ConfidenceMode     string `json:"confidenceMode,omitempty"`     // safe, assertive, speculative
	PolitenessLevel    string `json:"politenessLevel,omitempty"`    // low, medium, high
	PersonalityProfile string `json:"personalityProfile,omitempty"` // techer, developer, actor
}

type History struct {
	Question ShotTermQuestion `json:"question"`
	Answer   ShotTermAnswer   `json:"answer"`
	Model    string           `json:"model,omitempty"`
	Id       string           `json:"id,omitempty"`
	Created  int64            `json:"created,omitempty"`
}

type ToolsHistory struct {
	Model     string `json:"model,omitempty"`
	Id        string `json:"id,omitempty"`
	Created   int64  `json:"created,omitempty"`
	Role      string `json:"role,omitempty"`
	Type      string `json:"type,omitempty"`
	CallId    string `json:"callId,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
	Output    string `json:"output,omitempty"`
	Content   string `json:"content,omitempty"`
}
type ShotTermQuestion struct {
	Text string
}

type ShotTermAnswer struct {
	Text  string
	Usage openroutermodels.Usage
}
