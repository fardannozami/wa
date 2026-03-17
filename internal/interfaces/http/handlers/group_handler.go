package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wa-saas/internal/domain"
	"github.com/wa-saas/internal/infrastructure/repository"
	"github.com/wa-saas/pkg/logger"
)

type GroupHandler struct {
	groupRepo *repository.GroupRepository
	log       *logger.Logger
}

func NewGroupHandler(groupRepo *repository.GroupRepository, log *logger.Logger) *GroupHandler {
	return &GroupHandler{
		groupRepo: groupRepo,
		log:       log,
	}
}

func (h *GroupHandler) List(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	groups, err := h.groupRepo.FindByTenantID(tenantID)
	if err != nil {
		h.log.Error("Failed to list groups", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": groups})
}

func (h *GroupHandler) Create(c *gin.Context) {
	tenantID := c.GetString("tenant_id")

	var input struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group := &domain.Group{
		ID:       uuid.New().String(),
		TenantID: tenantID,
		Name:     input.Name,
	}

	if err := h.groupRepo.Create(group); err != nil {
		h.log.Error("Failed to create group", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, group)
}

func (h *GroupHandler) Update(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	groupID := c.Param("id")

	var input struct {
		Name string `json:"name"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group, err := h.groupRepo.FindByID(groupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	if group.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized"})
		return
	}

	if input.Name != "" {
		group.Name = input.Name
	}

	if err := h.groupRepo.Update(group); err != nil {
		h.log.Error("Failed to update group", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, group)
}

func (h *GroupHandler) Delete(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	groupID := c.Param("id")

	group, err := h.groupRepo.FindByID(groupID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	if group.TenantID != tenantID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized"})
		return
	}

	if err := h.groupRepo.Delete(groupID); err != nil {
		h.log.Error("Failed to delete group", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Group deleted"})
}
