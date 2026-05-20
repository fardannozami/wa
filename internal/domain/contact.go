package domain

import "time"

type Contact struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid"`
	TenantID  string    `json:"tenant_id" gorm:"type:uuid;not null;index;uniqueIndex:idx_phone_tenant"`
	Name      string    `json:"name" gorm:"type:varchar(255)"`
	Phone     string    `json:"phone" gorm:"type:varchar(50);not null;uniqueIndex:idx_phone_tenant"`
	Prefix    string    `json:"prefix" gorm:"type:varchar(20)"`
	Item1     string    `json:"item1" gorm:"type:varchar(255)"`
	Item2     string    `json:"item2" gorm:"type:varchar(255)"`
	Item3     string    `json:"item3" gorm:"type:varchar(255)"`
	Item4     string    `json:"item4" gorm:"type:varchar(255)"`
	Item5     string    `json:"item5" gorm:"type:varchar(255)"`
	Item6     string    `json:"item6" gorm:"type:varchar(255)"`
	Groups    []Group   `json:"groups" gorm:"many2many:contact_groups;"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Group struct {
	ID        string    `json:"id" gorm:"primaryKey;type:uuid"`
	TenantID  string    `json:"tenant_id" gorm:"type:uuid;not null;index"`
	Name      string    `json:"name" gorm:"type:varchar(255);not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ContactRepository interface {
	Create(contact *Contact) error
	CreateBatch(contacts []Contact) error
	UpsertBatch(contacts []Contact) error
	FindByTenantID(tenantID string, page, limit int) ([]Contact, int64, error)
	FindByTenantIDAndGroupID(tenantID string, groupID string, page, limit int) ([]Contact, int64, error)
	FindByID(id string) (*Contact, error)
	Update(contact *Contact) error
	Delete(id string) error
	FindByPhone(tenantID, phone string) (*Contact, error)
	FindByGroupID(groupID string) ([]Contact, error)
	AddGroup(contactID, groupID string) error
	RemoveGroup(contactID, groupID string) error
	SetGroups(contactID string, groupIDs []string) error
	CountByTenantID(tenantID string) (int64, error)
}

type GroupRepository interface {
	Create(group *Group) error
	FindByTenantID(tenantID string) ([]Group, error)
	FindByTenantIDAndName(tenantID string, name string) (*Group, error)
	FindByID(id string) (*Group, error)
	Update(group *Group) error
	Delete(id string) error
}
