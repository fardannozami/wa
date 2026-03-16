package domain

import "time"

type Contact struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid"`
	TenantID  string    `json:"tenant_id" gorm:"type:uuid;not null;index"`
	Name      string    `json:"name" gorm:"type:varchar(255)"`
	Phone     string    `json:"phone" gorm:"type:varchar(50);not null"`
	Tags      string    `json:"tags" gorm:"type:text"` // JSON array of tags
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ContactRepository interface {
	Create(contact *Contact) error
	CreateBatch(contacts []Contact) error
	FindByTenantID(tenantID string, page, limit int) ([]Contact, int64, error)
	FindByID(id string) (*Contact, error)
	Update(contact *Contact) error
	Delete(id string) error
	FindByPhone(tenantID, phone string) (*Contact, error)
}
