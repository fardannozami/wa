package repository

import (
	"time"

	"github.com/wa-saas/internal/domain"
	"gorm.io/gorm"
)

type MessageRepository struct {
	db *gorm.DB
}

func NewMessageRepository(db *gorm.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (r *MessageRepository) Create(message *domain.Message) error {
	return r.db.Create(message).Error
}

func (r *MessageRepository) CreateBatch(messages []domain.Message) error {
	return r.db.Create(&messages).Error
}

func (r *MessageRepository) FindByCampaignID(campaignID string) ([]domain.Message, error) {
	var messages []domain.Message
	err := r.db.Where("campaign_id = ?", campaignID).Order("created_at ASC").Find(&messages).Error
	return messages, err
}

func (r *MessageRepository) FindByID(id string) (*domain.Message, error) {
	var message domain.Message
	err := r.db.First(&message, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &message, nil
}

func (r *MessageRepository) Update(message *domain.Message) error {
	return r.db.Save(message).Error
}

func (r *MessageRepository) FindPendingByTenantID(tenantID string, limit int) ([]domain.Message, error) {
	var messages []domain.Message
	err := r.db.Where("tenant_id = ? AND status IN ?", tenantID, []string{"pending", "queued"}).Limit(limit).Find(&messages).Error
	return messages, err
}

func (r *MessageRepository) CountByCampaignID(campaignID string) (int64, int64, int64, error) {
	var total, success, failed int64

	err := r.db.Model(&domain.Message{}).Where("campaign_id = ?", campaignID).Count(&total).Error
	if err != nil {
		return 0, 0, 0, err
	}

	err = r.db.Model(&domain.Message{}).Where("campaign_id = ? AND status = ?", campaignID, domain.MessageStatusSent).Count(&success).Error
	if err != nil {
		return 0, 0, 0, err
	}

	err = r.db.Model(&domain.Message{}).Where("campaign_id = ? AND status = ?", campaignID, domain.MessageStatusFailed).Count(&failed).Error
	if err != nil {
		return 0, 0, 0, err
	}

	return total, success, failed, nil
}

func (r *MessageRepository) MarkAsSent(id string) error {
	now := time.Now()
	return r.db.Model(&domain.Message{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":     domain.MessageStatusSent,
		"sent_at":    now,
		"updated_at": now,
	}).Error
}

func (r *MessageRepository) MarkAsFailed(id, errMsg string) error {
	now := time.Now()
	return r.db.Model(&domain.Message{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":      domain.MessageStatusFailed,
		"error":       errMsg,
		"retry_count": gorm.Expr("retry_count + 1"),
		"updated_at":  now,
	}).Error
}

func (r *MessageRepository) CountAllSent() (int64, error) {
	var count int64
	err := r.db.Model(&domain.Message{}).Where("status = ?", domain.MessageStatusSent).Count(&count).Error
	return count, err
}

func (r *MessageRepository) CountSentByTenantID(tenantID string) (int64, error) {
	var count int64
	err := r.db.Model(&domain.Message{}).Where("tenant_id = ? AND status = ?", tenantID, domain.MessageStatusSent).Count(&count).Error
	return count, err
}
