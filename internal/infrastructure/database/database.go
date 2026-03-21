package database

import (
	"fmt"
	"log"

	"github.com/wa-saas/internal/domain"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewPostgresDB(dsn string, logLevel string) (*gorm.DB, error) {
	lvl := logger.Error
	if logLevel == "debug" {
		lvl = logger.Info
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(lvl),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	log.Println("Connected to PostgreSQL database")
	return db, nil
}

func Migrate(db *gorm.DB) error {
	err := db.AutoMigrate(
		&domain.User{},
		&domain.Tenant{},
		&domain.Device{},
		&domain.Contact{},
		&domain.Group{},
		&domain.Campaign{},
		&domain.Message{},
	)
	if err != nil {
		return err
	}

	// Ensure composite unique index exists for contact upsert ON CONFLICT
	db.Exec("DROP INDEX IF EXISTS idx_phone_tenant")
	db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_phone_tenant ON contacts (phone, tenant_id)")

	// Manually ensure whatsapp_id column exists in messages table if AutoMigrate missed it
	return db.Exec("ALTER TABLE messages ADD COLUMN IF NOT EXISTS whatsapp_id VARCHAR(255)").Error
}
