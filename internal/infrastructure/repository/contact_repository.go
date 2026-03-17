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

	err = r.db.Preload("Groups").Where("tenant_id = ?", tenantID).Offset(offset).Limit(limit).Find(&contacts).Error
	if err != nil {
		return nil, 0, err
	}

	return contacts, total, nil
}

func (r *ContactRepository) FindByTenantIDAndGroupID(tenantID string, groupID string, page, limit int) ([]domain.Contact, int64, error) {
	var contacts []domain.Contact
	var total int64

	offset := (page - 1) * limit

	query := r.db.Model(&domain.Contact{}).Where("tenant_id = ?", tenantID)

	if groupID != "" {
		query = query.Joins("JOIN contact_groups ON contact_groups.contact_id = contacts.id").Where("contact_groups.group_id = ?", groupID)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Preload("Groups").Offset(offset).Limit(limit).Find(&contacts).Error
	if err != nil {
		return nil, 0, err
	}

	return contacts, total, nil
}

func (r *ContactRepository) FindByID(id string) (*domain.Contact, error) {
	var contact domain.Contact
	err := r.db.Preload("Groups").First(&contact, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &contact, nil
}

func (r *ContactRepository) Update(contact *domain.Contact) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var existing domain.Contact
		if err := tx.Preload("Groups").First(&existing, "id = ?", contact.ID).Error; err != nil {
			return err
		}

		existing.Name = contact.Name
		existing.Phone = contact.Phone
		existing.Prefix = contact.Prefix

		if err := tx.Save(&existing).Error; err != nil {
			return err
		}

		if contact.Groups != nil && len(contact.Groups) > 0 {
			if err := tx.Exec("DELETE FROM contact_groups WHERE contact_id = ?", contact.ID).Error; err != nil {
				return err
			}

			for _, g := range contact.Groups {
				if g.ID != "" {
					if err := tx.Exec("INSERT INTO contact_groups (contact_id, group_id) VALUES (?, ?)", contact.ID, g.ID).Error; err != nil {
						return err
					}
				}
			}
		}

		return nil
	})
}

func (r *ContactRepository) Delete(id string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("contact_id = ?", id).Delete(&domain.Contact{}).Error; err != nil {
			return err
		}
		return tx.Delete(&domain.Contact{}, "id = ?", id).Error
	})
}

func (r *ContactRepository) FindByPhone(tenantID, phone string) (*domain.Contact, error) {
	var contact domain.Contact
	err := r.db.Preload("Groups").Where("tenant_id = ? AND phone = ?", tenantID, phone).First(&contact).Error
	if err != nil {
		return nil, err
	}
	return &contact, nil
}

func (r *ContactRepository) FindByGroupID(groupID string) ([]domain.Contact, error) {
	var contacts []domain.Contact
	err := r.db.Joins("JOIN contact_groups ON contact_groups.contact_id = contacts.id").Where("contact_groups.group_id = ?", groupID).Find(&contacts).Error
	return contacts, err
}

func (r *ContactRepository) AddGroup(contactID, groupID string) error {
	return r.db.Model(&domain.Contact{}).Where("id = ?", contactID).Association("Groups").Append(&domain.Group{ID: groupID})
}

func (r *ContactRepository) RemoveGroup(contactID, groupID string) error {
	return r.db.Model(&domain.Contact{}).Where("id = ?", contactID).Association("Groups").Delete(&domain.Group{ID: groupID})
}

func (r *ContactRepository) SetGroups(contactID string, groupIDs []string) error {
	var groups []domain.Group
	for _, id := range groupIDs {
		groups = append(groups, domain.Group{ID: id})
	}
	return r.db.Model(&domain.Contact{}).Where("id = ?", contactID).Association("Groups").Replace(groups)
}
