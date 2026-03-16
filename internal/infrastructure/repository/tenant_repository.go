package repository

import (
	"github.com/wa-saas/internal/domain"
	"gorm.io/gorm"
)

type TenantRepository struct {
	db *gorm.DB
}

func NewTenantRepository(db *gorm.DB) *TenantRepository {
	return &TenantRepository{db: db}
}

func (r *TenantRepository) Create(tenant *domain.Tenant) error {
	return r.db.Create(tenant).Error
}

func (r *TenantRepository) FindByID(id string) (*domain.Tenant, error) {
	var tenant domain.Tenant
	err := r.db.First(&tenant, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *TenantRepository) FindByOwnerID(ownerID string) (*domain.Tenant, error) {
	var tenant domain.Tenant
	err := r.db.Where("owner_id = ?", ownerID).First(&tenant).Error
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *TenantRepository) Update(tenant *domain.Tenant) error {
	return r.db.Save(tenant).Error
}
