package scheduler

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/wa-saas/internal/domain"
	"github.com/wa-saas/internal/infrastructure/repository"
	"github.com/wa-saas/internal/infrastructure/whatsapp"
	"github.com/wa-saas/pkg/logger"
)

type CampaignScheduler struct {
	campaignRepo *repository.CampaignRepository
	messageRepo  *repository.MessageRepository
	waService    whatsapp.WAService
	stopChan     chan bool
	log          *logger.Logger
}

func NewCampaignScheduler(campaignRepo *repository.CampaignRepository, messageRepo *repository.MessageRepository, waService whatsapp.WAService, log *logger.Logger) *CampaignScheduler {
	return &CampaignScheduler{
		campaignRepo: campaignRepo,
		messageRepo:  messageRepo,
		waService:    waService,
		stopChan:     make(chan bool),
		log:          log,
	}
}

func (s *CampaignScheduler) Start() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.processScheduledCampaigns()
			case <-s.stopChan:
				s.log.Info("Scheduler stopped")
				return
			}
		}
	}()
	s.log.Info("Scheduler started")
}

func (s *CampaignScheduler) Stop() {
	s.stopChan <- true
}

func (s *CampaignScheduler) processScheduledCampaigns() {
	campaigns, err := s.campaignRepo.FindScheduled()
	if err != nil {
		s.log.Error("Error finding scheduled campaigns", "error", err)
		return
	}

	for _, campaign := range campaigns {
		if campaign.ScheduledAt != nil && time.Now().After(*campaign.ScheduledAt) {
			s.log.Info("Attempting to run scheduled campaign", "campaignID", campaign.ID)
			
			// Atomically move from Scheduled to Running
			updated, err := s.campaignRepo.UpdateStatusAtomic(campaign.ID, []domain.CampaignStatus{domain.CampaignStatusScheduled}, domain.CampaignStatusRunning)
			if err != nil {
				s.log.Error("Error updating campaign status", "campaignID", campaign.ID, "error", err)
				continue
			}
			
			if updated {
				s.log.Info("Running scheduled campaign", "campaignID", campaign.ID)
				s.runCampaign(&campaign)
			} else {
				s.log.Info("Campaign already started or cancelled, skipping", "campaignID", campaign.ID)
			}
		}
	}
}

func (s *CampaignScheduler) runCampaign(campaign *domain.Campaign) {
	messages, err := s.messageRepo.FindByCampaignID(campaign.ID)
	if err != nil || len(messages) == 0 {
		s.log.Warn("No messages found for campaign", "campaignID", campaign.ID)
		return
	}

	s.waService.PushCampaignUpdate(campaign.TenantID, map[string]interface{}{
		"campaign_id":   campaign.ID,
		"status":        "running",
		"success_count": 0,
		"failed_count":  0,
	})

	successCount := 0
	failedCount := 0

	for i, msg := range messages {
		s.waService.SendTypingIndicator(campaign.TenantID, msg.Phone)
		time.Sleep(2 * time.Second)

		whatsappID, sendErr := s.waService.SendMessage(campaign.TenantID, msg.Phone, msg.Message, msg.ImageURL)
		if sendErr != nil {
			msg.Status = domain.MessageStatusFailed
			msg.Error = sendErr.Error()
			failedCount++
			fmt.Printf("[Scheduler] Failed to send to %s: %v\n", msg.Phone, sendErr)
			_ = s.messageRepo.Update(&msg)
		} else {
			successCount++
			_ = s.messageRepo.MarkAsSent(msg.ID, whatsappID)
		}

		campaign.SuccessCount = successCount
		campaign.FailedCount = failedCount

		s.waService.PushCampaignUpdate(campaign.TenantID, map[string]interface{}{
			"campaign_id":   campaign.ID,
			"status":        "running",
			"success_count": successCount,
			"failed_count":  failedCount,
		})

		if i < len(messages)-1 {
			delay := time.Duration(25+rand.IntN(36)) * time.Second
			time.Sleep(delay)
		}
	}

	campaign.Status = domain.CampaignStatusCompleted
	campaign.SuccessCount = successCount
	campaign.FailedCount = failedCount
	s.campaignRepo.Update(campaign)

	s.waService.PushCampaignUpdate(campaign.TenantID, map[string]interface{}{
		"campaign_id":   campaign.ID,
		"status":        "completed",
		"success_count": successCount,
		"failed_count":  failedCount,
	})

	s.log.Info("Campaign completed", "campaignID", campaign.ID, "success", successCount, "failed", failedCount)
}
