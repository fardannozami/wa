package repository

import (
	"time"

	"github.com/wa-saas/internal/domain"
	"gorm.io/gorm"
)

type CampaignRepository struct {
	db *gorm.DB
}

func NewCampaignRepository(db *gorm.DB) *CampaignRepository {
	return &CampaignRepository{db: db}
}

func (r *CampaignRepository) Create(campaign *domain.Campaign) error {
	return r.db.Create(campaign).Error
}

func (r *CampaignRepository) FindByTenantID(tenantID string, page, limit int) ([]domain.Campaign, int64, error) {
	var campaigns []domain.Campaign
	var total int64

	offset := (page - 1) * limit

	err := r.db.Model(&domain.Campaign{}).Where("tenant_id = ?", tenantID).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = r.db.Where("tenant_id = ?", tenantID).Offset(offset).Limit(limit).Order("created_at DESC").Find(&campaigns).Error
	if err != nil {
		return nil, 0, err
	}

	return campaigns, total, nil
}

func (r *CampaignRepository) FindByID(id string) (*domain.Campaign, error) {
	var campaign domain.Campaign
	err := r.db.First(&campaign, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &campaign, nil
}

func (r *CampaignRepository) Update(campaign *domain.Campaign) error {
	return r.db.Save(campaign).Error
}

func (r *CampaignRepository) Delete(id string) error {
	return r.db.Delete(&domain.Campaign{}, "id = ?", id).Error
}

func (r *CampaignRepository) FindScheduled() ([]domain.Campaign, error) {
	var campaigns []domain.Campaign
	err := r.db.Where("status = ? AND scheduled_at IS NOT NULL AND scheduled_at <= ?",
		domain.CampaignStatusScheduled, time.Now()).Find(&campaigns).Error
	return campaigns, err
}
