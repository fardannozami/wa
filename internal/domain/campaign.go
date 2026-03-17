package domain

import "time"

type CampaignStatus string

const (
	CampaignStatusDraft     CampaignStatus = "draft"
	CampaignStatusScheduled CampaignStatus = "scheduled"
	CampaignStatusRunning   CampaignStatus = "running"
	CampaignStatusCompleted CampaignStatus = "completed"
	CampaignStatusCancelled CampaignStatus = "cancelled"
	CampaignStatusFailed    CampaignStatus = "failed"
)

type Campaign struct {
	ID           string         `json:"id" gorm:"primaryKey;type:uuid"`
	TenantID     string         `json:"tenant_id" gorm:"type:uuid;not null;index"`
	Name         string         `json:"name" gorm:"type:varchar(255);not null"`
	Template     string         `json:"template" gorm:"type:text;not null"` // Message template with {{name}} placeholder
	Status       CampaignStatus `json:"status" gorm:"type:varchar(20);default:'draft'"`
	ScheduledAt  *time.Time     `json:"scheduled_at"`
	StartedAt    *time.Time     `json:"started_at"`
	CompletedAt  *time.Time     `json:"completed_at"`
	TotalCount   int            `json:"total_count" gorm:"default:0"`
	SuccessCount int            `json:"success_count" gorm:"default:0"`
	FailedCount  int            `json:"failed_count" gorm:"default:0"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type CampaignRepository interface {
	Create(campaign *Campaign) error
	FindByTenantID(tenantID string, page, limit int) ([]Campaign, int64, error)
	FindByID(id string) (*Campaign, error)
	Update(campaign *Campaign) error
	Delete(id string) error
	FindScheduled() ([]Campaign, error)
}
