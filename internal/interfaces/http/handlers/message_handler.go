package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wa-saas/internal/domain"
	"github.com/wa-saas/internal/infrastructure/whatsapp"
	"github.com/wa-saas/pkg/logger"
)

type MessageHandler struct {
	waService   whatsapp.WAService
	messageRepo domain.MessageRepository
	log         *logger.Logger
}

func NewMessageHandler(waService whatsapp.WAService, messageRepo domain.MessageRepository, log *logger.Logger) *MessageHandler {
	return &MessageHandler{
		waService:   waService,
		messageRepo: messageRepo,
		log:         log,
	}
}

type SendMessageRequest struct {
	Phone    string `json:"phone" binding:"required"`
	Message  string `json:"message" binding:"required"`
	MediaURL string `json:"media_url"`
}

func (h *MessageHandler) Send(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create message record for tracking
	msg := &domain.Message{
		ID:         uuid.New().String(),
		TenantID:   tenantID,
		Phone:      req.Phone,
		Message:    req.Message,
		ImageURL:   req.MediaURL,
		Status:     domain.MessageStatusPending,
		CampaignID: "", // Individual message
	}

	if err := h.messageRepo.Create(msg); err != nil {
		h.log.Error("Failed to create message record", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
		return
	}

	whatsappID, err := h.waService.SendMessage(tenantID, req.Phone, req.Message, req.MediaURL)
	if err != nil {
		h.log.Error("Failed to send message", "error", err)
		msg.Status = domain.MessageStatusFailed
		msg.Error = err.Error()
		h.messageRepo.Update(msg)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Mark as sent and update with whatsapp ID
	if err := h.messageRepo.MarkAsSent(msg.ID, whatsappID); err != nil {
		h.log.Error("Failed to mark message as sent", "error", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message sent successfully", "whatsapp_id": whatsappID})
}
