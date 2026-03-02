package jira

import (
	"context"

	"github.com/ctreminiom/go-atlassian/v2/pkg/infra/models"
)

func (j *Jira) GetBanner() (*models.AnnouncementBannerScheme, *models.ResponseScheme, error) {
	return j.Client.Banner.Get(context.Background())
}

func (j *Jira) UpdateBanner(payload *models.AnnouncementBannerPayloadScheme) (*models.ResponseScheme, error) {
	return j.Client.Banner.Update(context.Background(), payload)
}
