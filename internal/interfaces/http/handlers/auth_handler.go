package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/wa-saas/internal/domain"
	"github.com/wa-saas/internal/infrastructure/repository"
	httpmiddleware "github.com/wa-saas/internal/interfaces/http/middleware"
	"github.com/wa-saas/pkg/config"
	"github.com/wa-saas/pkg/logger"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type AuthHandler struct {
	userRepo     *repository.UserRepository
	tenantRepo   *repository.TenantRepository
	cfg          *config.Config
	log          *logger.Logger
	oauth2Config *oauth2.Config
}

func NewAuthHandler(userRepo *repository.UserRepository, tenantRepo *repository.TenantRepository, cfg *config.Config, log *logger.Logger) *AuthHandler {
	oauth2Config := &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		Scopes:       []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile", "openid"},
		Endpoint:     google.Endpoint,
	}

	return &AuthHandler{
		userRepo:     userRepo,
		tenantRepo:   tenantRepo,
		cfg:          cfg,
		log:          log,
		oauth2Config: oauth2Config,
	}
}

func (h *AuthHandler) GoogleLogin(c *gin.Context) {
	if h.cfg.GoogleClientID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Google OAuth is not configured on the server",
		})
		return
	}

	state := uuid.New().String()
	c.Set("oauth_state", state)

	url := h.oauth2Config.AuthCodeURL(state)

	c.JSON(http.StatusOK, gin.H{
		"url": url,
	})
}

func (h *AuthHandler) GoogleCallback(c *gin.Context) {
	state := c.Query("state")

	if state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No state parameter"})
		return
	}

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No code provided"})
		return
	}

	token, err := h.oauth2Config.Exchange(context.Background(), code)
	if err != nil {
		h.log.Error("Failed to exchange token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange token"})
		return
	}

	userInfo, err := h.getUserInfo(token.AccessToken)
	if err != nil {
		h.log.Error("Failed to get user info", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
		return
	}

	user, err := h.findOrCreateUser(userInfo)
	if err != nil {
		h.log.Error("Failed to find or create user", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	tenant, err := h.findOrCreateTenant(user.ID)
	if err != nil {
		h.log.Error("Failed to find or create tenant", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tenant"})
		return
	}

	jwtToken, err := h.generateToken(user.ID, tenant.ID, user.Email, user.IsAdmin)
	if err != nil {
		h.log.Error("Failed to generate token", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Build frontend redirect URL using the configured FrontendURL
	redirectURL := h.cfg.FrontendURL + "/oauth/callback?token=" + jwtToken
	c.Redirect(http.StatusFound, redirectURL)
}

type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	VerifiedEmail bool   `json:"verified_email"`
}

func (h *AuthHandler) getUserInfo(accessToken string) (*GoogleUserInfo, error) {
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + accessToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var userInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

func (h *AuthHandler) findOrCreateUser(info *GoogleUserInfo) (*domain.User, error) {
	user, err := h.userRepo.FindByGoogleID(info.ID)
	if err == nil {
		return user, nil
	}

	user = &domain.User{
		ID:       uuid.New().String(),
		GoogleID: info.ID,
		Email:    info.Email,
		Name:     info.Name,
	}

	if err := h.userRepo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

func (h *AuthHandler) findOrCreateTenant(ownerID string) (*domain.Tenant, error) {
	tenant, err := h.tenantRepo.FindByOwnerID(ownerID)
	if err == nil {
		return tenant, nil
	}

	tenant = &domain.Tenant{
		ID:      uuid.New().String(),
		OwnerID: ownerID,
		Plan:    domain.TenantPlanFree,
		Status:  domain.TenantStatusActive,
	}

	if err := h.tenantRepo.Create(tenant); err != nil {
		return nil, err
	}

	return tenant, nil
}

func (h *AuthHandler) generateToken(userID, tenantID, email string, isAdmin bool) (string, error) {
	claims := &httpmiddleware.Claims{
		UserID:   userID,
		TenantID: tenantID,
		Email:    email,
		IsAdmin:  isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.cfg.JWTSecret))
}


func (h *AuthHandler) Logout(c *gin.Context) {
	c.SetCookie("token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetString("user_id")
	tenantID := c.GetString("tenant_id")
	email := c.GetString("email")

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":        userID,
			"email":     email,
			"tenant_id": tenantID,
			"is_admin":  c.GetBool("is_admin"),
		},
	})
}
