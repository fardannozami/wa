package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wa-saas/internal/infrastructure/whatsapp"
	"github.com/wa-saas/pkg/logger"
)

type MessageHandler struct {
	waService whatsapp.WAService
	log       *logger.Logger
}

func NewMessageHandler(waService whatsapp.WAService, log *logger.Logger) *MessageHandler {
	return &MessageHandler{
		waService: waService,
		log:       log,
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

	if _, err := h.waService.SendMessage(tenantID, req.Phone, req.Message, req.MediaURL); err != nil {
		h.log.Error("Failed to send message", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message sent successfully"})
}
