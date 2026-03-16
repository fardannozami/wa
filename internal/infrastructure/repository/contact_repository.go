package repository

import (
	"github.com/wa-saas/internal/domain"
	"gorm.io/gorm"
)

type ContactRepository struct {
	db *gorm.DB
}

func NewContactRepository(db *gorm.DB) *ContactRepository {
	return &ContactRepository{db: db}
}

func (r *ContactRepository) Create(contact *domain.Contact) error {
	return r.db.Create(contact).Error
}

func (r *ContactRepository) CreateBatch(contacts []domain.Contact) error {
	return r.db.Create(&contacts).Error
}

func (r *ContactRepository) FindByTenantID(tenantID string, page, limit int) ([]domain.Contact, int64, error) {
	var contacts []domain.Contact
	var total int64

	offset := (page - 1) * limit

	err := r.db.Model(&domain.Contact{}).Where("tenant_id = ?", tenantID).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = r.db.Where("tenant_id = ?", tenantID).Offset(offset).Limit(limit).Find(&contacts).Error
	if err != nil {
		return nil, 0, err
	}

	return contacts, total, nil
}

func (r *ContactRepository) FindByID(id string) (*domain.Contact, error) {
	var contact domain.Contact
	err := r.db.First(&contact, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &contact, nil
}

func (r *ContactRepository) Update(contact *domain.Contact) error {
	return r.db.Save(contact).Error
}

func (r *ContactRepository) Delete(id string) error {
	return r.db.Delete(&domain.Contact{}, "id = ?", id).Error
}

func (r *ContactRepository) FindByPhone(tenantID, phone string) (*domain.Contact, error) {
	var contact domain.Contact
	err := r.db.Where("tenant_id = ? AND phone = ?", tenantID, phone).First(&contact).Error
	if err != nil {
		return nil, err
	}
	return &contact, nil
}
