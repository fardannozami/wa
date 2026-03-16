package domain

import "time"

type TenantPlan string

const (
	TenantPlanFree       TenantPlan = "free"
	TenantPlanPro        TenantPlan = "pro"
	TenantPlanEnterprise TenantPlan = "enterprise"
)

type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusInactive  TenantStatus = "inactive"
	TenantStatusSuspended TenantStatus = "suspended"
)

type Tenant struct {
	ID        string       `json:"id" gorm:"primaryKey;type:uuid"`
	OwnerID   string       `json:"owner_id" gorm:"type:uuid;not null"`
	Plan      TenantPlan   `json:"plan" gorm:"type:varchar(20);default:'free'"`
	Status    TenantStatus `json:"status" gorm:"type:varchar(20);default:'active'"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

type TenantRepository interface {
	Create(tenant *Tenant) error
	FindByID(id string) (*Tenant, error)
	FindByOwnerID(ownerID string) (*Tenant, error)
	Update(tenant *Tenant) error
}
