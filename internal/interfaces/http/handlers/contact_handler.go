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
	log         *logger.Logger
}

func NewContactHandler(contactRepo *repository.ContactRepository, log *logger.Logger) *ContactHandler {
	return &ContactHandler{
		contactRepo: contactRepo,
		log:         log,
	}
}

func (h *ContactHandler) List(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	contacts, total, err := h.contactRepo.FindByTenantID(tenantID, page, limit)
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
		Name  string `json:"name" binding:"required"`
		Phone string `json:"phone" binding:"required"`
		Tags  string `json:"tags"`
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
		Tags:     input.Tags,
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
		Name  string `json:"name"`
		Phone string `json:"phone"`
		Tags  string `json:"tags"`
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
	if input.Tags != "" {
		contact.Tags = input.Tags
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

	var contacts []domain.Contact
	for i, record := range records {
		if i == 0 {
			continue
		}
		if len(record) < 2 {
			continue
		}

		phone := h.sanitizePhone(record[1])

		contacts = append(contacts, domain.Contact{
			ID:       uuid.New().String(),
			TenantID: tenantID,
			Name:     strings.TrimSpace(record[0]),
			Phone:    phone,
		})
	}

	if len(contacts) > 0 {
		if err := h.contactRepo.CreateBatch(contacts); err != nil {
			h.log.Error("Failed to import contacts", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to import contacts"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Contacts imported successfully",
		"imported": len(contacts),
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
