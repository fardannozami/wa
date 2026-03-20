package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wa-saas/internal/domain"
	"github.com/wa-saas/pkg/logger"
)

type StatsHandler struct {
	campaignRepo domain.CampaignRepository
	contactRepo  domain.ContactRepository
	messageRepo  domain.MessageRepository
	log          *logger.Logger
}

func NewStatsHandler(
	campaignRepo domain.CampaignRepository,
	contactRepo domain.ContactRepository,
	messageRepo domain.MessageRepository,
	log *logger.Logger,
) *StatsHandler {
	return &StatsHandler{
		campaignRepo: campaignRepo,
		contactRepo:  contactRepo,
		messageRepo:  messageRepo,
		log:          log,
	}
}

func (h *StatsHandler) GetStats(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	totalContacts, err := h.contactRepo.CountByTenantID(tenantID)
	if err != nil {
		h.log.Error("Failed to count contacts", "error", err, "tenant_id", tenantID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stats"})
		return
	}

	totalCampaigns, err := h.campaignRepo.CountByTenantID(tenantID)
	if err != nil {
		h.log.Error("Failed to count campaigns", "error", err, "tenant_id", tenantID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stats"})
		return
	}

	totalMessages, err := h.messageRepo.CountSentByTenantID(tenantID)
	if err != nil {
		h.log.Error("Failed to count sent messages", "error", err, "tenant_id", tenantID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"contacts":  totalContacts,
		"campaigns": totalCampaigns,
		"sent":      totalMessages,
	})
}
