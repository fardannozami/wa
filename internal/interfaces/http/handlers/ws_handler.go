package handlers

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/wa-saas/internal/infrastructure/whatsapp"
	"github.com/wa-saas/internal/interfaces/http/middleware"
	"github.com/wa-saas/pkg/logger"
)

type WSHandler struct {
	waService whatsapp.WAService
	jwtSecret string
	log       *logger.Logger
}

func NewWSHandler(waService whatsapp.WAService, jwtSecret string, log *logger.Logger) *WSHandler {
	return &WSHandler{
		waService: waService,
		jwtSecret: jwtSecret,
		log:       log,
	}
}

func (h *WSHandler) HandleQR(c *gin.Context) {
	h.log.Debug("HandleQR called")
	h.log.Debug("WS Request Details",
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
		"query", c.Request.URL.Query(),
		"upgrade", c.Request.Header.Get("Upgrade"))

	if c.Request.Header.Get("Upgrade") == "websocket" {
		tokenString := c.Query("token")
		if tokenString == "" {
			tokenString = strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		}
		if tokenString == "" {
			c.JSON(401, gin.H{"error": "token required"})
			return
		}

		token, err := jwt.ParseWithClaims(tokenString, &middleware.Claims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(h.jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(401, gin.H{"error": "invalid token"})
			return
		}

		claims, ok := token.Claims.(*middleware.Claims)
		if !ok {
			c.JSON(401, gin.H{"error": "invalid token claims"})
			return
		}

		tenantID := claims.TenantID
		h.log.Debug("Handler: tenantID from claims", "tenantID", tenantID)
		if strings.TrimSpace(tenantID) == "" {
			c.JSON(400, gin.H{"error": "tenant_id not found in token"})
			return
		}

		h.waService.HandleQRWebSocket(tenantID, c.Writer, c.Request)
		return
	}

	c.JSON(400, gin.H{"error": "websocket upgrade required"})
	h.log.Debug("Not a websocket request, returning error")
}
