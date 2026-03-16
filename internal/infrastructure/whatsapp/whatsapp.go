package whatsapp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/wa-saas/internal/domain"
	"go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

type QRCode struct {
	Code        string `json:"code"`
	ImageBase64 string `json:"image_base64"`
}

type WAService interface {
	GenerateQR(tenantID string) (QRCode, error)
	GetStatus(tenantID string) (domain.DeviceStatus, string, error)
	Connect(tenantID string) error
	Disconnect(tenantID string) error
	SendMessage(tenantID, phone, message string) error
	Shutdown()
}

type WhatsAppService struct {
	deviceRepo domain.DeviceRepository
	sessionDir string

	mu      sync.RWMutex
	clients map[string]*WhatsMeowClient
}

type WhatsMeowClient struct {
	DeviceID string
	TenantID string
	Client   *whatsmeow.Client
	Status   domain.DeviceStatus
	Phone    string
}

func NewWhatsAppService(deviceRepo domain.DeviceRepository, sessionDir string) *WhatsAppService {
	return &WhatsAppService{
		deviceRepo: deviceRepo,
		sessionDir: sessionDir,
		clients:    make(map[string]*WhatsMeowClient),
	}
}

func (s *WhatsAppService) GenerateQR(tenantID string) (QRCode, error) {
	device, err := s.deviceRepo.FindByTenantID(tenantID)
	if err != nil {
		device = &domain.Device{
			ID:       s.generateID(),
			TenantID: tenantID,
			Status:   domain.DeviceStatusDisconnected,
		}
		if err := s.deviceRepo.Create(device); err != nil {
			return QRCode{}, fmt.Errorf("failed to create device: %w", err)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if client, exists := s.clients[tenantID]; exists && client.Status == domain.DeviceStatusConnected {
		return QRCode{}, fmt.Errorf("device already connected")
	}

	dbLog := waLog.Stdout("Database", "DEBUG", true)
	dbPath := filepath.Join(s.sessionDir, "wa_"+tenantID+".db") + "?_foreign_keys=on"

	if err := os.MkdirAll(s.sessionDir, 0755); err != nil {
		return QRCode{}, fmt.Errorf("failed to create session directory: %w", err)
	}

	container, err := sqlstore.New(context.Background(), "sqlite3", dbPath, dbLog)
	if err != nil {
		return QRCode{}, fmt.Errorf("failed to create store: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		deviceStore = container.NewDevice()
	}

	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	client.AddEventHandler(func(evt interface{}) {
		s.handleEvent(tenantID, evt)
	})

	err = client.Connect()
	if err != nil {
		return QRCode{}, fmt.Errorf("failed to connect: %w", err)
	}

	qrChan, err := client.GetQRChannel(context.Background())
	if err != nil {
		return QRCode{}, fmt.Errorf("failed to get QR channel: %w", err)
	}

	s.clients[tenantID] = &WhatsMeowClient{
		DeviceID: device.ID,
		TenantID: tenantID,
		Client:   client,
		Status:   domain.DeviceStatusQRGenerated,
	}

	device.Status = domain.DeviceStatusQRGenerated
	_ = s.deviceRepo.Update(device)

	go func() {
		for evt := range qrChan {
			s.handleQRChannel(tenantID, device, client, evt)
		}
	}()

	qrCode := QRCode{
		Code:        "waiting_for_scan",
		ImageBase64: "",
	}

	return qrCode, nil
}

func (s *WhatsAppService) handleQRChannel(tenantID string, device *domain.Device, client *whatsmeow.Client, evt whatsmeow.QRChannelItem) {
	switch evt.Event {
	case "code":
		fmt.Printf("[WhatsMeow] QR Code received for tenant %s\n", tenantID)

	case "success":
		device.Status = domain.DeviceStatusConnected
		if client.Store.ID != nil {
			device.JID = client.Store.ID.ToNonAD().String()
		}
		device.LastSeen = time.Now()
		_ = s.deviceRepo.Update(device)

		s.mu.Lock()
		if c, ok := s.clients[tenantID]; ok {
			c.Status = domain.DeviceStatusConnected
		}
		s.mu.Unlock()

		fmt.Printf("[WhatsMeow] Device connected for tenant %s\n", tenantID)

	case "failed":
		device.Status = domain.DeviceStatusDisconnected
		_ = s.deviceRepo.Update(device)

		s.mu.Lock()
		delete(s.clients, tenantID)
		s.mu.Unlock()

		fmt.Printf("[WhatsMeow] QR scan failed for tenant %s\n", tenantID)
	}
}

func (s *WhatsAppService) handleEvent(tenantID string, evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		fmt.Printf("[WhatsMeow] Received message: %s\n", v.Message.GetConversation())
	case *events.Connected:
		fmt.Printf("[WhatsMeow] Connected for tenant %s\n", tenantID)
	case *events.Disconnected:
		fmt.Printf("[WhatsMeow] Disconnected for tenant %s\n", tenantID)
		s.mu.Lock()
		if c, ok := s.clients[tenantID]; ok {
			c.Status = domain.DeviceStatusDisconnected
		}
		s.mu.Unlock()
	}
}

func (s *WhatsAppService) GetStatus(tenantID string) (domain.DeviceStatus, string, error) {
	device, err := s.deviceRepo.FindByTenantID(tenantID)
	if err != nil {
		return domain.DeviceStatusDisconnected, "", err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if client, exists := s.clients[tenantID]; exists {
		return client.Status, client.Phone, nil
	}

	return device.Status, device.PhoneNumber, nil
}

func (s *WhatsAppService) Connect(tenantID string) error {
	device, err := s.deviceRepo.FindByTenantID(tenantID)
	if err != nil {
		return fmt.Errorf("device not found: %w", err)
	}

	dbLog := waLog.Stdout("Database", "DEBUG", true)
	dbPath := filepath.Join(s.sessionDir, "wa_"+tenantID+".db") + "?_foreign_keys=on"

	if err := os.MkdirAll(s.sessionDir, 0755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	container, err := sqlstore.New(context.Background(), "sqlite3", dbPath, dbLog)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	client.AddEventHandler(func(evt interface{}) {
		s.handleEvent(tenantID, evt)
	})

	err = client.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	phone := ""
	if client.Store.ID != nil {
		phone = client.Store.ID.User
	}

	s.mu.Lock()
	s.clients[tenantID] = &WhatsMeowClient{
		DeviceID: device.ID,
		TenantID: tenantID,
		Client:   client,
		Status:   domain.DeviceStatusConnected,
		Phone:    phone,
	}
	s.mu.Unlock()

	device.Status = domain.DeviceStatusConnected
	if client.Store.ID != nil {
		device.JID = client.Store.ID.ToNonAD().String()
	}
	device.PhoneNumber = phone
	device.LastSeen = time.Now()
	_ = s.deviceRepo.Update(device)

	return nil
}

func (s *WhatsAppService) Disconnect(tenantID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if client, exists := s.clients[tenantID]; exists {
		if client.Client != nil {
			client.Client.Disconnect()
		}
		delete(s.clients, tenantID)
	}

	device, err := s.deviceRepo.FindByTenantID(tenantID)
	if err != nil {
		return fmt.Errorf("device not found: %w", err)
	}

	device.Status = domain.DeviceStatusDisconnected
	device.SessionData = ""
	device.JID = ""

	return s.deviceRepo.Update(device)
}

func (s *WhatsAppService) SendMessage(tenantID, phone, message string) error {
	s.mu.RLock()
	client, exists := s.clients[tenantID]
	s.mu.RUnlock()

	if !exists || client.Client == nil || (client.Status != domain.DeviceStatusConnected && client.Status != domain.DeviceStatusActive) {
		return fmt.Errorf("device not connected")
	}

	jid := types.NewJID(phone, "c.us")

	resp, err := client.Client.SendMessage(context.Background(), jid, &waE2E.Message{
		Conversation: proto.String(message),
	})
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	fmt.Printf("[WhatsMeow] Message sent to %s: %s (ID: %s)\n", phone, message, resp.ID)
	return nil
}

func (s *WhatsAppService) Shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for tenantID, client := range s.clients {
		if client.Client != nil {
			client.Client.Disconnect()
		}
		delete(s.clients, tenantID)
	}
}

func (s *WhatsAppService) generateID() string {
	return fmt.Sprintf("dev_%d", time.Now().UnixNano())
}
