package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wa-saas/internal/domain"
	"github.com/wa-saas/internal/infrastructure/repository"
	"github.com/wa-saas/pkg/logger"
)

type CampaignHandler struct {
	campaignRepo *repository.CampaignRepository
	contactRepo  *repository.ContactRepository
	messageRepo  *repository.MessageRepository
	log          *logger.Logger
}

func NewCampaignHandler(campaignRepo *repository.CampaignRepository, contactRepo *repository.ContactRepository, messageRepo *repository.MessageRepository, log *logger.Logger) *CampaignHandler {
	return &CampaignHandler{
		campaignRepo: campaignRepo,
		contactRepo:  contactRepo,
		messageRepo:  messageRepo,
		log:          log,
	}
}

func (h *CampaignHandler) List(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	campaigns, total, err := h.campaignRepo.FindByTenantID(tenantID, page, limit)
	if err != nil {
		h.log.Error("Failed to list campaigns", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  campaigns,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *CampaignHandler) Create(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var input struct {
		Name        string   `json:"name" binding:"required"`
		Template    string   `json:"template" binding:"required"`
		ContactIDs  []string `json:"contact_ids"`
		ScheduledAt *string  `json:"scheduled_at"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	campaign := &domain.Campaign{
		ID:       uuid.New().String(),
		TenantID: tenantID,
		Name:     input.Name,
		Template: input.Template,
		Status:   domain.CampaignStatusDraft,
	}

	if input.ScheduledAt != nil {
		scheduledTime, err := time.Parse(time.RFC3339, *input.ScheduledAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scheduled_at format"})
			return
		}
		campaign.ScheduledAt = &scheduledTime
		campaign.Status = domain.CampaignStatusScheduled
	}

	if err := h.campaignRepo.Create(campaign); err != nil {
		h.log.Error("Failed to create campaign", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(input.ContactIDs) > 0 {
		h.createMessagesForCampaign(campaign, input.ContactIDs, input.Template)
	}

	c.JSON(http.StatusCreated, campaign)
}

func (h *CampaignHandler) Get(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	campaignID := c.Param("id")

	campaign, err := h.campaignRepo.FindByID(campaignID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Campaign not found"})
		return
	}

	if campaign.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized"})
		return
	}

	total, success, failed, _ := h.messageRepo.CountByCampaignID(campaignID)

	c.JSON(http.StatusOK, gin.H{
		"campaign": campaign,
		"total":    total,
		"success":  success,
		"failed":   failed,
	})
}

func (h *CampaignHandler) Delete(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	campaignID := c.Param("id")

	campaign, err := h.campaignRepo.FindByID(campaignID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Campaign not found"})
		return
	}

	if campaign.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized"})
		return
	}

	if err := h.campaignRepo.Delete(campaignID); err != nil {
		h.log.Error("Failed to delete campaign", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Campaign deleted"})
}

func (h *CampaignHandler) createMessagesForCampaign(campaign *domain.Campaign, contactIDs []string, template string) {
	var messages []domain.Message

	for _, contactID := range contactIDs {
		contact, err := h.contactRepo.FindByID(contactID)
		if err != nil {
			continue
		}

		message := h.replaceTemplate(template, contact.Name)

		messages = append(messages, domain.Message{
			ID:         uuid.New().String(),
			CampaignID: campaign.ID,
			ContactID:  contactID,
			TenantID:   campaign.TenantID,
			Phone:      contact.Phone,
			Message:    message,
			Status:     domain.MessageStatusPending,
		})
	}

	if len(messages) > 0 {
		h.messageRepo.CreateBatch(messages)

		campaign.TotalCount = len(messages)
		campaign.Status = domain.CampaignStatusRunning
		h.campaignRepo.Update(campaign)
	}
}

func (h *CampaignHandler) replaceTemplate(template, name string) string {
	return strings.ReplaceAll(template, "{{name}}", name)
}
