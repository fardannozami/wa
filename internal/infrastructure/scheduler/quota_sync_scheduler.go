package scheduler

import (
	"time"

	"github.com/wa-saas/internal/infrastructure/whatsapp"
	"github.com/wa-saas/pkg/logger"
)

// QuotaSyncScheduler periodically syncs contact counts and recalculates
// the daily message quota (trust level) for all active connected devices.
type QuotaSyncScheduler struct {
	waService whatsapp.WAService
	interval  time.Duration
	stopChan  chan bool
	log       *logger.Logger
}

func NewQuotaSyncScheduler(waService whatsapp.WAService, interval time.Duration, log *logger.Logger) *QuotaSyncScheduler {
	return &QuotaSyncScheduler{
		waService: waService,
		interval:  interval,
		stopChan:  make(chan bool),
		log:       log,
	}
}

func (s *QuotaSyncScheduler) Start() {
	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		s.log.Info("QuotaSyncScheduler started", "interval", s.interval.String())

		for {
			select {
			case <-ticker.C:
				s.log.Info("Running periodic quota sync for all active tenants")
				s.waService.SyncAllQuotas()
			case <-s.stopChan:
				s.log.Info("QuotaSyncScheduler stopped")
				return
			}
		}
	}()
}

func (s *QuotaSyncScheduler) Stop() {
	s.stopChan <- true
}
