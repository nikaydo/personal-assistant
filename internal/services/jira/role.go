package jira

import (
	"context"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
)

func (j *Jira) GetRoles() ([]*models.ApplicationRoleScheme, *models.ResponseScheme, error) {
	return j.Client.Role.Gets(context.Background())
}

func (j *Jira) GetRole(roleKey string) (*models.ApplicationRoleScheme, *models.ResponseScheme, error) {
	return j.Client.Role.Get(context.Background(), roleKey)
}
