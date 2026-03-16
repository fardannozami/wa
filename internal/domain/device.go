package domain

import "time"

type DeviceStatus string

const (
	DeviceStatusDisconnected DeviceStatus = "disconnected"
	DeviceStatusQRGenerated  DeviceStatus = "qr_generated"
	DeviceStatusConnecting   DeviceStatus = "connecting"
	DeviceStatusConnected    DeviceStatus = "connected"
	DeviceStatusActive       DeviceStatus = "active"
)

type Device struct {
	ID          string       `json:"id" gorm:"primaryKey;type:uuid"`
	TenantID    string       `json:"tenant_id" gorm:"type:uuid;uniqueIndex;not null"`
	JID         string       `json:"jid" gorm:"type:varchar(100)"`
	SessionData string       `json:"session_data" gorm:"type:text"`
	Status      DeviceStatus `json:"status" gorm:"type:varchar(20);default:'disconnected'"`
	LastSeen    time.Time    `json:"last_seen"`
	PhoneNumber string       `json:"phone_number"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

type DeviceRepository interface {
	Create(device *Device) error
	FindByTenantID(tenantID string) (*Device, error)
	FindByID(id string) (*Device, error)
	Update(device *Device) error
	Delete(id string) error
}
