package scheduler

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/wa-saas/internal/domain"
	"github.com/wa-saas/internal/infrastructure/repository"
	"github.com/wa-saas/internal/infrastructure/whatsapp"
)

type CampaignScheduler struct {
	campaignRepo *repository.CampaignRepository
	messageRepo  *repository.MessageRepository
	waService    whatsapp.WAService
	stopChan     chan bool
}

func NewCampaignScheduler(campaignRepo *repository.CampaignRepository, messageRepo *repository.MessageRepository, waService whatsapp.WAService) *CampaignScheduler {
	return &CampaignScheduler{
		campaignRepo: campaignRepo,
		messageRepo:  messageRepo,
		waService:    waService,
		stopChan:     make(chan bool),
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
				fmt.Println("[Scheduler] Stopped")
				return
			}
		}
	}()
	fmt.Println("[Scheduler] Started")
}

func (s *CampaignScheduler) Stop() {
	s.stopChan <- true
}

func (s *CampaignScheduler) processScheduledCampaigns() {
	campaigns, err := s.campaignRepo.FindScheduled()
	if err != nil {
		fmt.Printf("[Scheduler] Error finding scheduled campaigns: %v\n", err)
		return
	}

	for _, campaign := range campaigns {
		if campaign.ScheduledAt != nil && time.Now().After(*campaign.ScheduledAt) {
			fmt.Printf("[Scheduler] Attempting to run scheduled campaign: %s\n", campaign.ID)
			
			// Atomically move from Scheduled to Running
			updated, err := s.campaignRepo.UpdateStatusAtomic(campaign.ID, []domain.CampaignStatus{domain.CampaignStatusScheduled}, domain.CampaignStatusRunning)
			if err != nil {
				fmt.Printf("[Scheduler] Error updating campaign status: %v\n", err)
				continue
			}
			
			if updated {
				fmt.Printf("[Scheduler] Running scheduled campaign: %s\n", campaign.ID)
				s.runCampaign(&campaign)
			} else {
				fmt.Printf("[Scheduler] Campaign %s already started or cancelled, skipping\n", campaign.ID)
			}
		}
	}
}

func (s *CampaignScheduler) runCampaign(campaign *domain.Campaign) {
	messages, err := s.messageRepo.FindByCampaignID(campaign.ID)
	if err != nil || len(messages) == 0 {
		fmt.Printf("[Scheduler] No messages found for campaign: %s\n", campaign.ID)
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

	fmt.Printf("[Scheduler] Campaign %s completed: success=%d, failed=%d\n", campaign.ID, successCount, failedCount)
}
