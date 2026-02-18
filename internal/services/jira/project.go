package jira

import (
	"context"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"

	_ "github.com/ctreminiom/go-atlassian/v2/jira/v3"
)

func (j *Jira) CreateProject(payload *models.ProjectPayloadScheme) (*models.NewProjectCreatedScheme, *models.ResponseScheme, error) {
	newProject, response, err := j.Client.Project.Create(context.Background(), payload)
	if err != nil {
		return nil, response, err
	}
	j.Projects = append(j.Projects, newProject)
	return newProject, response, nil
}

func (j *Jira) SearchProject(opt *models.ProjectSearchOptionsScheme, startAt, maxResults int) (*models.ProjectSearchScheme, *models.ResponseScheme, error) {
	return j.Client.Project.Search(context.Background(), opt, startAt, maxResults)
}

func (j *Jira) GetProject(projectKeyOrID string) (*models.ProjectScheme, *models.ResponseScheme, error) {
	return j.Client.Project.Get(context.Background(), projectKeyOrID, []string{})
}

func (j *Jira) UpdateProject(projectKeyOrID string, payload *models.ProjectUpdateScheme) (*models.ProjectScheme, *models.ResponseScheme, error) {
	return j.Client.Project.Update(context.Background(), projectKeyOrID, payload)
}

func (j *Jira) DeleteProject(projectKeyOrID string) (*models.ResponseScheme, error) {
	return j.Client.Project.Delete(context.Background(), projectKeyOrID, true)
}

func (j *Jira) DeleteProjectAsynchronously(projectKeyOrID string) (*models.TaskScheme, *models.ResponseScheme, error) {
	return j.Client.Project.DeleteAsynchronously(context.Background(), projectKeyOrID)
}

func (j *Jira) ArchiveProject(projectKeyOrID string) (*models.ResponseScheme, error) {
	return j.Client.Project.Archive(context.Background(), projectKeyOrID)
}

func (j *Jira) RestoreProject(projectKeyOrID string) (*models.ProjectScheme, *models.ResponseScheme, error) {
	return j.Client.Project.Restore(context.Background(), projectKeyOrID)
}

func (j *Jira) Statuses(projectKeyOrID string) ([]*models.ProjectStatusPageScheme, *models.ResponseScheme, error) {
	return j.Client.Project.Statuses(context.Background(), projectKeyOrID)
}

func (j *Jira) GetNotifyScheme(projectKeyOrID string) (*models.NotificationSchemeScheme, *models.ResponseScheme, error) {
	return j.Client.Project.NotificationScheme(context.Background(), projectKeyOrID, []string{})
}
