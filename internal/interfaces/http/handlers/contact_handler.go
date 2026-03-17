package handlers

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wa-saas/internal/domain"
	"github.com/wa-saas/internal/infrastructure/repository"
	"github.com/wa-saas/pkg/logger"
)

type ContactHandler struct {
	contactRepo *repository.ContactRepository
	groupRepo   *repository.GroupRepository
	log         *logger.Logger
}

func NewContactHandler(contactRepo *repository.ContactRepository, groupRepo *repository.GroupRepository, log *logger.Logger) *ContactHandler {
	return &ContactHandler{
		contactRepo: contactRepo,
		groupRepo:   groupRepo,
		log:         log,
	}
}

func (h *ContactHandler) List(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	groupID := c.Query("group_id")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var contacts []domain.Contact
	var total int64
	var err error

	contacts, total, err = h.contactRepo.FindByTenantIDAndGroupID(tenantID, groupID, page, limit)

	if err != nil {
		h.log.Error("Failed to list contacts", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  contacts,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *ContactHandler) Create(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var input struct {
		Name    string `json:"name" binding:"required"`
		Phone   string `json:"phone" binding:"required"`
		GroupID string `json:"group_id"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.log.Info("Creating contact", "name", input.Name, "phone", input.Phone, "tenant", tenantID)

	contact := &domain.Contact{
		ID:       uuid.New().String(),
		TenantID: tenantID,
		Name:     input.Name,
		Phone:    h.sanitizePhone(input.Phone),
	}
	if input.GroupID != "" {
		contact.GroupID = &input.GroupID
	}

	h.log.Info("Contact to create", "contact", contact)

	if err := h.contactRepo.Create(contact); err != nil {
		h.log.Error("Failed to create contact", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.log.Info("Contact created successfully", "id", contact.ID)

	c.JSON(http.StatusCreated, contact)
}

func (h *ContactHandler) Update(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	contactID := c.Param("id")

	var input struct {
		Name    string `json:"name"`
		Phone   string `json:"phone"`
		GroupID string `json:"group_id"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	contact, err := h.contactRepo.FindByID(contactID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contact not found"})
		return
	}

	if contact.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized"})
		return
	}

	if input.Name != "" {
		contact.Name = input.Name
	}
	if input.Phone != "" {
		contact.Phone = h.sanitizePhone(input.Phone)
	}
	if input.GroupID != "" {
		contact.GroupID = &input.GroupID
	} else {
		contact.GroupID = nil
	}

	if err := h.contactRepo.Update(contact); err != nil {
		h.log.Error("Failed to update contact", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, contact)
}

func (h *ContactHandler) Delete(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	contactID := c.Param("id")

	contact, err := h.contactRepo.FindByID(contactID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Contact not found"})
		return
	}

	if contact.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized"})
		return
	}

	if err := h.contactRepo.Delete(contactID); err != nil {
		h.log.Error("Failed to delete contact", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Contact deleted"})
}

func (h *ContactHandler) ImportCSV(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	openedFile, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer openedFile.Close()

	reader := csv.NewReader(openedFile)

	records, err := reader.ReadAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read CSV"})
		return
	}

	groupMap := make(map[string]string)
	groups, _ := h.groupRepo.FindByTenantID(tenantID)
	for _, g := range groups {
		groupMap[strings.ToLower(g.Name)] = g.ID
	}
	h.log.Info("Group map", "groupMap", groupMap)

	updatedCount := 0
	createdCount := 0

	for i, record := range records {
		if i == 0 {
			continue
		}
		if len(record) < 2 {
			continue
		}

		phone := h.sanitizePhone(record[1])
		name := strings.TrimSpace(record[0])

		existingContact, err := h.contactRepo.FindByPhone(tenantID, phone)
		if err == nil && existingContact != nil {
			existingContact.Name = name
			if len(record) >= 3 && strings.TrimSpace(record[2]) != "" {
				groupName := strings.TrimSpace(record[2])
				h.log.Info("Processing group", "groupName", groupName, "groupMap", groupMap)
				if groupID, ok := groupMap[strings.ToLower(groupName)]; ok {
					h.log.Info("Found group", "groupID", groupID)
					existingContact.GroupID = &groupID
				} else {
					h.log.Info("Group not found, setting to nil")
					existingContact.GroupID = nil
				}
			}
			h.contactRepo.Update(existingContact)
			updatedCount++
		} else {
			contact := domain.Contact{
				ID:       uuid.New().String(),
				TenantID: tenantID,
				Name:     name,
				Phone:    phone,
			}

			if len(record) >= 3 && strings.TrimSpace(record[2]) != "" {
				groupName := strings.TrimSpace(record[2])
				if groupID, ok := groupMap[strings.ToLower(groupName)]; ok {
					contact.GroupID = &groupID
				}
			}

			h.contactRepo.Create(&contact)
			createdCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Contacts imported",
		"created": createdCount,
		"updated": updatedCount,
	})
}

func (h *ContactHandler) sanitizePhone(phone string) string {
	phone = strings.TrimSpace(phone)

	phone = strings.ReplaceAll(phone, " ", "")
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, "(", "")
	phone = strings.ReplaceAll(phone, ")", "")
	phone = strings.ReplaceAll(phone, "+", "")

	if strings.HasPrefix(phone, "0") {
		phone = "62" + phone[1:]
	}

	return phone
}
