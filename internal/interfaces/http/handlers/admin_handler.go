package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wa-saas/internal/domain"
	"github.com/wa-saas/pkg/logger"
)

type AdminHandler struct {
	userRepo    domain.UserRepository
	messageRepo domain.MessageRepository
	log         *logger.Logger
}

func NewAdminHandler(userRepo domain.UserRepository, messageRepo domain.MessageRepository, log *logger.Logger) *AdminHandler {
	return &AdminHandler{
		userRepo:    userRepo,
		messageRepo: messageRepo,
		log:         log,
	}
}

func (h *AdminHandler) GetStats(c *gin.Context) {
	totalUsers, err := h.userRepo.Count()
	if err != nil {
		h.log.Error("Failed to count users", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stats"})
		return
	}

	totalMessages, err := h.messageRepo.CountAllSent()
	if err != nil {
		h.log.Error("Failed to count sent messages", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stats"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_users":    totalUsers,
		"total_messages": totalMessages,
	})
}

func (h *AdminHandler) ListUsers(c *gin.Context) {
	users, err := h.userRepo.FindAll()
	if err != nil {
		h.log.Error("Failed to list users", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	c.JSON(http.StatusOK, users)
}
