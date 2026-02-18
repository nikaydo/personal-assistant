package services

import (
	"github.com/nikaydo/personal-assistant/internal/logg"
	jira "github.com/nikaydo/personal-assistant/internal/services/jira"
)

type JiraService = jira.Jira
type JiraServiceUser = jira.User
type JiraServiceAvatarURLs = jira.AvatarURLs

func NewJira(email, apiToken, host string, log *logg.Logger) (*JiraService, error) {
	return jira.NewJira(email, apiToken, host, log)
}
