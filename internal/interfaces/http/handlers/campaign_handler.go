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
	"github.com/wa-saas/internal/infrastructure/whatsapp"
	"github.com/wa-saas/pkg/logger"
)

type CampaignHandler struct {
	campaignRepo *repository.CampaignRepository
	contactRepo  *repository.ContactRepository
	messageRepo  *repository.MessageRepository
	waService    whatsapp.WAService
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

func (h *CampaignHandler) SetWAService(waService whatsapp.WAService) {
	h.waService = waService
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

func (h *CampaignHandler) Update(c *gin.Context) {
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

	if campaign.Status == domain.CampaignStatusRunning {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot edit running campaign"})
		return
	}

	var input struct {
		Name        string   `json:"name"`
		Template    string   `json:"template"`
		ContactIDs  []string `json:"contact_ids"`
		ScheduledAt *string  `json:"scheduled_at"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Name != "" {
		campaign.Name = input.Name
	}
	if input.Template != "" {
		campaign.Template = input.Template
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

	if err := h.campaignRepo.Update(campaign); err != nil {
		h.log.Error("Failed to update campaign", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(input.ContactIDs) > 0 {
		h.createMessagesForCampaign(campaign, input.ContactIDs, campaign.Template)
	}

	c.JSON(http.StatusOK, campaign)
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

	messages, _ := h.messageRepo.FindByCampaignID(campaignID)
	var contactIDs []string
	for _, msg := range messages {
		contactIDs = append(contactIDs, msg.ContactID)
	}

	c.JSON(http.StatusOK, gin.H{
		"campaign":    campaign,
		"total":       total,
		"success":     success,
		"failed":      failed,
		"contact_ids": contactIDs,
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

func (h *CampaignHandler) Send(c *gin.Context) {
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

	if campaign.Status == domain.CampaignStatusRunning {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Campaign already running"})
		return
	}

	var input struct {
		ScheduledAt *string `json:"scheduled_at"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.ScheduledAt != nil {
		scheduledTime, err := time.Parse(time.RFC3339, *input.ScheduledAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scheduled_at format"})
			return
		}
		campaign.ScheduledAt = &scheduledTime
		campaign.Status = domain.CampaignStatusScheduled
		h.campaignRepo.Update(campaign)
		c.JSON(http.StatusOK, gin.H{"message": "Campaign scheduled", "scheduled_at": campaign.ScheduledAt})
		return
	}

	messages, err := h.messageRepo.FindByCampaignID(campaignID)
	if err != nil || len(messages) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No messages in campaign"})
		return
	}

	campaign.Status = domain.CampaignStatusRunning
	h.campaignRepo.Update(campaign)

	go func() {
		h.processCampaignMessages(campaign, messages)
	}()

	c.JSON(http.StatusOK, gin.H{"message": "Campaign started"})
}

func (h *CampaignHandler) processCampaignMessages(campaign *domain.Campaign, messages []domain.Message) {
	campaign.Status = domain.CampaignStatusRunning
	h.campaignRepo.Update(campaign)

	h.waService.PushCampaignUpdate(campaign.TenantID, map[string]interface{}{
		"campaign_id":   campaign.ID,
		"status":        "running",
		"success_count": 0,
		"failed_count":  0,
	})

	defer func() {
		campaign.Status = domain.CampaignStatusCompleted
		h.campaignRepo.Update(campaign)

		h.waService.PushCampaignUpdate(campaign.TenantID, map[string]interface{}{
			"campaign_id":   campaign.ID,
			"status":        campaign.Status,
			"success_count": campaign.SuccessCount,
			"failed_count":  campaign.FailedCount,
		})
	}()

	successCount := 0
	failedCount := 0

	for i, msg := range messages {
		if err := h.waService.SendMessage(campaign.TenantID, msg.Phone, msg.Message); err != nil {
			msg.Status = domain.MessageStatusFailed
			failedCount++
			h.log.Error("Failed to send message", "error", err, "phone", msg.Phone)
		} else {
			msg.Status = domain.MessageStatusSent
			successCount++
		}

		now := time.Now()
		msg.SentAt = &now
		h.messageRepo.Update(&msg)

		campaign.SuccessCount = successCount
		campaign.FailedCount = failedCount

		h.waService.PushCampaignUpdate(campaign.TenantID, map[string]interface{}{
			"campaign_id":   campaign.ID,
			"status":        "running",
			"success_count": successCount,
			"failed_count":  failedCount,
		})

		if i > 0 && i%10 == 0 {
			time.Sleep(2 * time.Second)
		}
	}
}

func (h *CampaignHandler) createMessagesForCampaign(campaign *domain.Campaign, contactIDs []string, template string) []domain.Message {
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
		h.campaignRepo.Update(campaign)
	}

	return messages
}

func (h *CampaignHandler) replaceTemplate(template, name string) string {
	return strings.ReplaceAll(template, "{{name}}", name)
}
