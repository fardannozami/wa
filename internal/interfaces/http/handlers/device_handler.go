package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wa-saas/internal/domain"
	"github.com/wa-saas/internal/infrastructure/repository"
	"github.com/wa-saas/internal/infrastructure/whatsapp"
	"github.com/wa-saas/pkg/logger"
)

type DeviceHandler struct {
	deviceRepo *repository.DeviceRepository
	waService  whatsapp.WAService
	log        *logger.Logger
}

func NewDeviceHandler(deviceRepo *repository.DeviceRepository, waService whatsapp.WAService, log *logger.Logger) *DeviceHandler {
	return &DeviceHandler{
		deviceRepo: deviceRepo,
		waService:  waService,
		log:        log,
	}
}

func (h *DeviceHandler) Get(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	device, err := h.deviceRepo.FindByTenantID(tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"device": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"device": device})
}

func (h *DeviceHandler) Connect(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	device, err := h.deviceRepo.FindByTenantID(tenantID)
	if err != nil {
		device = &domain.Device{
			ID:       uuid.New().String(),
			TenantID: tenantID,
			Status:   domain.DeviceStatusDisconnected,
		}
		h.deviceRepo.Create(device)
	}

	qr, err := h.waService.GenerateQR(tenantID)
	if err != nil {
		h.log.Error("Failed to generate QR", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"qr_code": qr.Code,
		"image":   qr.ImageBase64,
		"status":  device.Status,
	})
}

func (h *DeviceHandler) Disconnect(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	if err := h.waService.Disconnect(tenantID); err != nil {
		h.log.Error("Failed to disconnect", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Disconnected successfully"})
}

func (h *DeviceHandler) GetStatus(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	status, phoneNumber, err := h.waService.GetStatus(tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"status": domain.DeviceStatusDisconnected,
			"phone":  "",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": status,
		"phone":  phoneNumber,
	})
}

func (h *DeviceHandler) GetGroups(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	groups, err := h.waService.GetJoinedGroups(tenantID)
	if err != nil {
		h.log.Error("Failed to get groups", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": groups})
}

func (h *DeviceHandler) ImportGroupContacts(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var input struct {
		GroupJID string `json:"group_jid" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	contacts, err := h.waService.ImportGroupContacts(tenantID, input.GroupJID)
	if err != nil {
		h.log.Error("Failed to import group contacts", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": contacts})
}
