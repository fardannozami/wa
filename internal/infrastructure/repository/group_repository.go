package repository

import (
	"github.com/wa-saas/internal/domain"
	"gorm.io/gorm"
)

type GroupRepository struct {
	db *gorm.DB
}

func NewGroupRepository(db *gorm.DB) *GroupRepository {
	return &GroupRepository{db: db}
}

func (r *GroupRepository) Create(group *domain.Group) error {
	return r.db.Create(group).Error
}

func (r *GroupRepository) FindByTenantID(tenantID string) ([]domain.Group, error) {
	var groups []domain.Group
	err := r.db.Where("tenant_id = ?", tenantID).Order("name ASC").Find(&groups).Error
	return groups, err
}

func (r *GroupRepository) FindByTenantIDAndName(tenantID string, name string) (*domain.Group, error) {
	var group domain.Group
	err := r.db.Where("tenant_id = ? AND name = ?", tenantID, name).First(&group).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *GroupRepository) FindByID(id string) (*domain.Group, error) {
	var group domain.Group
	err := r.db.First(&group, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *GroupRepository) Update(group *domain.Group) error {
	return r.db.Save(group).Error
}

func (r *GroupRepository) Delete(id string) error {
	return r.db.Delete(&domain.Group{}, "id = ?", id).Error
}
