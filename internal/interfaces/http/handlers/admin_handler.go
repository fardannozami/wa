package handlers

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wa-saas/internal/domain"
	"github.com/wa-saas/pkg/logger"
)

type AdminHandler struct {
	userRepo    domain.UserRepository
	messageRepo domain.MessageRepository
	log         *logger.Logger
	startTime   time.Time
}

func NewAdminHandler(userRepo domain.UserRepository, messageRepo domain.MessageRepository, log *logger.Logger) *AdminHandler {
	return &AdminHandler{
		userRepo:    userRepo,
		messageRepo: messageRepo,
		log:         log,
		startTime:   time.Now(),
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

func (h *AdminHandler) GetMetrics(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics := gin.H{
		"memory": gin.H{
			"alloc":       m.Alloc / 1024 / 1024,
			"total_alloc": m.TotalAlloc / 1024 / 1024,
			"sys":         m.Sys / 1024 / 1024,
			"num_gc":      m.NumGC,
		},
		"goroutines": runtime.NumGoroutine(),
		"cpus":       runtime.NumCPU(),
		"uptime":     time.Since(h.startTime).Seconds(),
	}

	c.JSON(http.StatusOK, metrics)
}
