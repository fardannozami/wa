package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/wa-saas/internal/infrastructure/database"
	"github.com/wa-saas/internal/infrastructure/repository"
	"github.com/wa-saas/internal/infrastructure/scheduler"
	"github.com/wa-saas/internal/infrastructure/whatsapp"
	"github.com/wa-saas/internal/interfaces/http/handlers"
	"github.com/wa-saas/internal/interfaces/http/middleware"
	"github.com/wa-saas/pkg/config"
	"github.com/wa-saas/pkg/logger"
)

func main() {
	godotenv.Load()
	cfg := config.Load()

	log := logger.New(cfg.LogLevel)

	db, err := database.NewPostgresDB(cfg.DatabaseURL, cfg.LogLevel)
	if err != nil {
		log.Fatal("Failed to connect to database", "error", err)
	}

	if err := database.Migrate(db); err != nil {
		log.Fatal("Failed to migrate database", "error", err)
	}

	userRepo := repository.NewUserRepository(db)
	tenantRepo := repository.NewTenantRepository(db)
	deviceRepo := repository.NewDeviceRepository(db)
	contactRepo := repository.NewContactRepository(db)
	groupRepo := repository.NewGroupRepository(db)
	campaignRepo := repository.NewCampaignRepository(db)
	messageRepo := repository.NewMessageRepository(db)

	waService := whatsapp.NewWhatsAppService(deviceRepo, contactRepo, groupRepo, cfg.SessionDir)

	campaignScheduler := scheduler.NewCampaignScheduler(campaignRepo, messageRepo, waService)
	campaignScheduler.Start()

	authHandler := handlers.NewAuthHandler(userRepo, tenantRepo, cfg, log)
	deviceHandler := handlers.NewDeviceHandler(deviceRepo, waService, log)
	wsHandler := handlers.NewWSHandler(waService, cfg.JWTSecret, log)
	messageHandler := handlers.NewMessageHandler(waService, log)
	contactHandler := handlers.NewContactHandler(contactRepo, groupRepo, log)
	groupHandler := handlers.NewGroupHandler(groupRepo, log)
	campaignHandler := handlers.NewCampaignHandler(campaignRepo, contactRepo, messageRepo, log)
	campaignHandler.SetWAService(waService)
	adminHandler := handlers.NewAdminHandler(userRepo, messageRepo, log)

	router := gin.Default()
	router.Use(middleware.CORS())

	api := router.Group("/api/v1")
	{
		api.GET("/auth/google", authHandler.GoogleLogin)
		api.GET("/auth/google/callback", authHandler.GoogleCallback)
		api.POST("/auth/logout", authHandler.Logout)

		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})
	}

	protected := router.Group("/api/v1")
	protected.Use(middleware.AuthMiddleware(cfg.JWTSecret))
	{
		protected.GET("/auth/me", authHandler.Me)

		protected.GET("/device", deviceHandler.Get)
		protected.POST("/device/connect", deviceHandler.Connect)
		protected.POST("/device/disconnect", deviceHandler.Disconnect)
		protected.GET("/device/status", deviceHandler.GetStatus)
		protected.GET("/device/groups", deviceHandler.GetGroups)
		protected.POST("/device/import-group", deviceHandler.ImportGroupContacts)

		protected.POST("/messages", messageHandler.Send)

		protected.GET("/contacts", contactHandler.List)
		protected.POST("/contacts", contactHandler.Create)
		protected.PUT("/contacts/:id", contactHandler.Update)
		protected.DELETE("/contacts/:id", contactHandler.Delete)
		protected.POST("/contacts/import", contactHandler.ImportCSV)

		protected.GET("/groups", groupHandler.List)
		protected.POST("/groups", groupHandler.Create)
		protected.PUT("/groups/:id", groupHandler.Update)
		protected.DELETE("/groups/:id", groupHandler.Delete)

		protected.GET("/campaigns", campaignHandler.List)
		protected.POST("/campaigns", campaignHandler.Create)
		protected.PUT("/campaigns/:id", campaignHandler.Update)
		protected.POST("/campaigns/:id/send", campaignHandler.Send)
		protected.GET("/campaigns/:id", campaignHandler.Get)
		protected.GET("/campaigns/:id/messages", campaignHandler.GetMessages)
		protected.POST("/messages/:messageID/resend", campaignHandler.ResendMessage)
		protected.DELETE("/campaigns/:id", campaignHandler.Delete)

		// Admin routes
		admin := protected.Group("/admin")
		admin.Use(middleware.AdminMiddleware())
		{
			admin.GET("/stats", adminHandler.GetStats)
			admin.GET("/users", adminHandler.ListUsers)
			admin.GET("/metrics", adminHandler.GetMetrics)
		}
	}

	router.GET("/api/v1/device/ws", wsHandler.HandleQR)

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	go func() {
		log.Info("Starting server", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown", "error", err)
	}

	waService.Shutdown()
	campaignScheduler.Stop()

	log.Info("Server exited")
}
