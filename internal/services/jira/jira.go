package jira

import (
	v3 "github.com/ctreminiom/go-atlassian/v2/jira/v3"
	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
	"github.com/nikaydo/personal-assistant/internal/logg"
)

type Jira struct {
	Email    string
	ApiToken string

	Host string

	Client *v3.Client

	User     []User
	MainUser User

	Projects []*models.NewProjectCreatedScheme

	Logger *logg.Logger
}

func NewJira(email, apiToken, host string, log *logg.Logger) (*Jira, error) {
	j := &Jira{
		Email:    email,
		ApiToken: apiToken,
		Host:     host,
		Logger:   log,
	}
	if err := j.New(); err != nil {
		return nil, err
	}
	log.Info("jira successful auth")

	return j, nil
}

func (j *Jira) New() error {
	atlassian, err := v3.New(nil, j.Host)
	if err != nil {
		return err
	}
	j.Client = atlassian
	j.Client.Auth.SetBasicAuth(j.Email, j.ApiToken)
	return nil
}
