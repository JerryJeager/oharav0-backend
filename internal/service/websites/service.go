package websites

import (
	"context"

	"github.com/JerryJeager/raglearn/internal/models"
)

type WebsiteSv interface {
	CreateWebsite(ctx context.Context, website *models.Website) error
	GetWebsites(ctx context.Context) (*models.WebsiteList, error)
	GetWebsite(ctx context.Context, websiteID int) (*models.Website, error)
	DeleteWebsite(ctx context.Context, websiteID int) error
}

type WebsiteServ struct {
	repo WebsiteStore
}

func NewWebsiteService(repo WebsiteStore) *WebsiteServ {
	return &WebsiteServ{repo: repo}
}

func (s *WebsiteServ) CreateWebsite(ctx context.Context, website *models.Website) error {
	return s.repo.CreateWebsite(ctx, website)
}

func (s *WebsiteServ) GetWebsites(ctx context.Context) (*models.WebsiteList, error) {
	return s.repo.GetWebsites(ctx)
}

func (s *WebsiteServ) GetWebsite(ctx context.Context, websiteID int) (*models.Website, error) {
	return s.repo.GetWebsite(ctx, websiteID)
}

func (s *WebsiteServ) DeleteWebsite(ctx context.Context, websiteID int) error {
	return s.repo.DeleteWebsite(ctx, websiteID)
}
