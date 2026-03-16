package repository

import (
	"github.com/wa-saas/internal/domain"
	"gorm.io/gorm"
)

type DeviceRepository struct {
	db *gorm.DB
}

func NewDeviceRepository(db *gorm.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

func (r *DeviceRepository) Create(device *domain.Device) error {
	return r.db.Create(device).Error
}

func (r *DeviceRepository) FindByTenantID(tenantID string) (*domain.Device, error) {
	var device domain.Device
	err := r.db.Where("tenant_id = ?", tenantID).First(&device).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

func (r *DeviceRepository) FindByID(id string) (*domain.Device, error) {
	var device domain.Device
	err := r.db.First(&device, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &device, nil
}

func (r *DeviceRepository) Update(device *domain.Device) error {
	return r.db.Save(device).Error
}

func (r *DeviceRepository) Delete(id string) error {
	return r.db.Delete(&domain.Device{}, "id = ?", id).Error
}
