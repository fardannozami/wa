package domain

import "time"

type MessageStatus string

const (
	MessageStatusPending   MessageStatus = "pending"
	MessageStatusQueued    MessageStatus = "queued"
	MessageStatusSending   MessageStatus = "sending"
	MessageStatusSent      MessageStatus = "sent"
	MessageStatusDelivered MessageStatus = "delivered"
	MessageStatusRead      MessageStatus = "read"
	MessageStatusFailed    MessageStatus = "failed"
)

type Message struct {
	ID          string        `json:"id" gorm:"primaryKey;type:uuid"`
	CampaignID  string        `json:"campaign_id" gorm:"type:uuid;not null;index"`
	ContactID   string        `json:"contact_id" gorm:"type:uuid;not null"`
	TenantID    string        `json:"tenant_id" gorm:"type:uuid;not null;index"`
	WhatsAppID  string        `json:"whatsapp_id" gorm:"type:varchar(255);index"`
	Phone       string        `json:"phone" gorm:"type:varchar(50);not null"`
	Message     string        `json:"message" gorm:"type:text;not null"`
	Status      MessageStatus `json:"status" gorm:"type:varchar(20);default:'pending'"`
	SentAt      *time.Time    `json:"sent_at"`
	DeliveredAt *time.Time    `json:"delivered_at"`
	ReadAt      *time.Time    `json:"read_at"`
	Error       string        `json:"error" gorm:"type:text"`
	RetryCount  int           `json:"retry_count" gorm:"default:0"`
	ImageURL    string        `json:"image_url" gorm:"type:text"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

type MessageRepository interface {
	Create(message *Message) error
	CreateBatch(messages []Message) error
	FindByCampaignID(campaignID string) ([]Message, error)
	FindByID(id string) (*Message, error)
	Update(message *Message) error
	FindPendingByTenantID(tenantID string, limit int) ([]Message, error)
	CountByCampaignID(campaignID string) (int64, int64, int64, error)
	CountAllSent() (int64, error)
	CountSentByTenantID(tenantID string) (int64, error)
	FindByWhatsAppID(whatsappID string) (*Message, error)
	MarkAsDelivered(whatsappID string) error
	MarkAsRead(whatsappID string) error
}
