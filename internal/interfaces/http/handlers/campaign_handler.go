package handlers

import (
	"fmt"
	"math/rand/v2"
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
	deviceRepo   *repository.DeviceRepository
	waService    whatsapp.WAService
	log          *logger.Logger
}

func NewCampaignHandler(campaignRepo *repository.CampaignRepository, contactRepo *repository.ContactRepository, messageRepo *repository.MessageRepository, deviceRepo *repository.DeviceRepository, log *logger.Logger) *CampaignHandler {
	return &CampaignHandler{
		campaignRepo: campaignRepo,
		contactRepo:  contactRepo,
		messageRepo:  messageRepo,
		deviceRepo:   deviceRepo,
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
		ImageURL    string   `json:"image_url"`
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
		ImageURL: input.ImageURL,
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
		ImageURL    string   `json:"image_url"`
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
	if input.ImageURL != "" {
		campaign.ImageURL = input.ImageURL
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

func (h *CampaignHandler) GetMessages(c *gin.Context) {
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

	messages, err := h.messageRepo.FindByCampaignID(campaignID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": messages})
}

func (h *CampaignHandler) ResendMessage(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	messageID := c.Param("messageID")

	message, err := h.messageRepo.FindByID(messageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
		return
	}

	if message.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized"})
		return
	}

	whatsappID, err := h.waService.SendMessage(tenantID, message.Phone, message.Message, message.ImageURL)
	if err != nil {
		message.Status = domain.MessageStatusFailed
		h.log.Error("Failed to resend message", "error", err, "phone", message.Phone)
	} else {
		_ = h.messageRepo.MarkAsSent(message.ID, whatsappID)
	}

	h.messageRepo.Update(message)

	c.JSON(http.StatusOK, gin.H{"message": "Message resent"})
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

	// Verify remaining daily message limit
	device, err := h.deviceRepo.FindByTenantID(tenantID)
	dailyLimit := 100
	if err == nil {
		dailyLimit = device.DailyLimit
	}

	sentToday, err := h.messageRepo.CountSentTodayByTenantID(tenantID)
	if err == nil {
		remaining := int64(dailyLimit) - sentToday
		if remaining <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Daily message limit reached (%d/%d). You cannot send any more messages today.", sentToday, dailyLimit),
			})
			return
		}
		if int64(len(messages)) > remaining {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Campaign size (%d messages) exceeds your remaining daily quota (%d messages left out of %d). Please reduce the campaign contacts, wait until tomorrow, or use a warmer WhatsApp number.", len(messages), remaining, dailyLimit),
			})
			return
		}
	}

	updated, err := h.campaignRepo.UpdateStatusAtomic(campaignID, []domain.CampaignStatus{domain.CampaignStatusDraft, domain.CampaignStatusScheduled}, domain.CampaignStatusRunning)
	if err != nil {
		h.log.Error("Failed to update campaign status", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update campaign status"})
		return
	}

	if !updated {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Campaign cannot be started (already running or cancelled)"})
		return
	}

	go func() {
		h.processCampaignMessages(campaign, messages)
	}()

	c.JSON(http.StatusOK, gin.H{"message": "Campaign started"})
}

func (h *CampaignHandler) processCampaignMessages(campaign *domain.Campaign, messages []domain.Message) {
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
		h.waService.SendTypingIndicator(campaign.TenantID, msg.Phone)

		time.Sleep(500 * time.Millisecond)

		whatsappID, sendErr := h.waService.SendMessage(campaign.TenantID, msg.Phone, msg.Message, msg.ImageURL)
		if sendErr != nil {
			msg.Status = domain.MessageStatusFailed
			failedCount++
			h.log.Error("Failed to send message", "error", sendErr, "phone", msg.Phone)
			_ = h.messageRepo.Update(&msg)
		} else {
			_ = h.messageRepo.MarkAsSent(msg.ID, whatsappID)
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

		if i < len(messages)-1 {
			// Setiap kelipatan 50 pesan terkirim, berikan jeda istirahat panjang (5-10 menit)
			if (i+1)%50 == 0 {
				longDelay := time.Duration(300+rand.IntN(301)) * time.Second // 300 hingga 600 detik (5-10 menit)
				h.log.Info("Sent 50 messages, applying long delay to avoid ban", "messagesSent", i+1, "duration", longDelay.String(), "campaign_id", campaign.ID)
				time.Sleep(longDelay)
			} else {
				// Jeda antar pesan standar (diperlama: 45 hingga 90 detik)
				delay := time.Duration(45+rand.IntN(46)) * time.Second // 45 hingga 90 detik
				time.Sleep(delay)
			}
		}
	}
}

func (h *CampaignHandler) createMessagesForCampaign(campaign *domain.Campaign, contactIDs []string, template string) []domain.Message {
	_ = h.messageRepo.DeleteByCampaignID(campaign.ID)
	var messages []domain.Message

	for _, contactID := range contactIDs {
		contact, err := h.contactRepo.FindByID(contactID)
		if err != nil {
			continue
		}

		message := h.replaceTemplate(template, contact)

		messages = append(messages, domain.Message{
			ID:         uuid.New().String(),
			CampaignID: campaign.ID,
			ContactID:  contactID,
			TenantID:   campaign.TenantID,
			Phone:      contact.Phone,
			Message:    message,
			ImageURL:   campaign.ImageURL,
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

func (h *CampaignHandler) replaceTemplate(template string, contact *domain.Contact) string {
	res := strings.ReplaceAll(template, "{{name}}", contact.Name)
	res = strings.ReplaceAll(res, "{{prefix}}", contact.Prefix)
	res = strings.ReplaceAll(res, "{{item1}}", contact.Item1)
	res = strings.ReplaceAll(res, "{{item2}}", contact.Item2)
	res = strings.ReplaceAll(res, "{{item3}}", contact.Item3)
	res = strings.ReplaceAll(res, "{{item4}}", contact.Item4)
	res = strings.ReplaceAll(res, "{{item5}}", contact.Item5)
	res = strings.ReplaceAll(res, "{{item6}}", contact.Item6)
	return res
}
