package whatsapp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/coder/websocket"
	_ "github.com/mattn/go-sqlite3"
	"github.com/wa-saas/internal/domain"

	"go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	"github.com/skip2/go-qrcode"
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
	SendMessage(tenantID, phone, message, mediaURL string) error
	SendTypingIndicator(tenantID, phone string)
	HandleQRWebSocket(tenantID string, w http.ResponseWriter, r *http.Request)
	PushCampaignUpdate(tenantID string, data map[string]interface{})
	GetJoinedGroups(tenantID string) ([]map[string]interface{}, error)
	ImportGroupContacts(tenantID, groupJID string) (int, error)
	Shutdown()
}

type WhatsAppService struct {
	deviceRepo  domain.DeviceRepository
	contactRepo domain.ContactRepository
	groupRepo   domain.GroupRepository
	sessionDir  string

	mu      sync.RWMutex
	clients map[string]*WhatsMeowClient

	wsMu    sync.RWMutex
	wsConns map[string]*websocket.Conn
}

type WhatsMeowClient struct {
	DeviceID string
	TenantID string
	Client   *whatsmeow.Client
	Status   domain.DeviceStatus
	Phone    string
}

func NewWhatsAppService(deviceRepo domain.DeviceRepository, contactRepo domain.ContactRepository, groupRepo domain.GroupRepository, sessionDir string) *WhatsAppService {
	return &WhatsAppService{
		deviceRepo:  deviceRepo,
		contactRepo: contactRepo,
		groupRepo:   groupRepo,
		sessionDir:  sessionDir,
		clients:     make(map[string]*WhatsMeowClient),
		wsConns:     make(map[string]*websocket.Conn),
	}
}

func (s *WhatsAppService) GenerateQR(tenantID string) (QRCode, error) {
	fmt.Printf("[WhatsMeow] GenerateQR called for tenant: %s\n", tenantID)
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

	dbLog := waLog.Stdout("Database", "ERROR", true)
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
	} else if deviceStore.ID != nil {
		// If the store already has an ID, we can't generate a QR code for it.
		// This happens if a previous disconnect didn't fully wipe the SQLite DB.
		// We explicitly delete the old session and create a fresh one.
		_ = deviceStore.Delete(context.Background())
		deviceStore = container.NewDevice()
	}

	clientLog := waLog.Stdout("Client", "ERROR", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	client.AddEventHandler(func(evt interface{}) {
		s.handleEvent(tenantID, evt)
	})

	qrChan, err := client.GetQRChannel(context.Background())
	if err != nil {
		return QRCode{}, fmt.Errorf("failed to get QR channel: %w", err)
	}

	err = client.Connect()
	if err != nil {
		return QRCode{}, fmt.Errorf("failed to connect: %w", err)
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

		qrImage := ""
		png, err := qrcode.Encode(evt.Code, qrcode.Medium, 256)
		if err == nil {
			qrImage = base64.StdEncoding.EncodeToString(png)
		}

		qrPayload := map[string]interface{}{
			"type":  "qr",
			"code":  evt.Code,
			"image": qrImage,
		}
		s.pushToWebSocket(tenantID, qrPayload)

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

		s.pushToWebSocket(tenantID, map[string]interface{}{
			"type":   "connected",
			"status": domain.DeviceStatusConnected,
		})

		fmt.Printf("[WhatsMeow] Device connected for tenant %s\n", tenantID)

	case "failed":
		device.Status = domain.DeviceStatusDisconnected
		_ = s.deviceRepo.Update(device)

		s.mu.Lock()
		delete(s.clients, tenantID)
		s.mu.Unlock()

		s.pushToWebSocket(tenantID, map[string]interface{}{
			"type":   "failed",
			"status": domain.DeviceStatusDisconnected,
		})

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
	case *events.LoggedOut:
		fmt.Printf("[WhatsMeow] Logged Out for tenant %s\n", tenantID)
		
		device, err := s.deviceRepo.FindByTenantID(tenantID)
		if err == nil {
			device.Status = domain.DeviceStatusDisconnected
			device.JID = ""
			_ = s.deviceRepo.Update(device)
		}

		s.mu.Lock()
		delete(s.clients, tenantID)
		s.mu.Unlock()

		s.pushToWebSocket(tenantID, map[string]interface{}{
			"type":   "logged_out",
			"status": domain.DeviceStatusDisconnected,
		})
	}
}

func (s *WhatsAppService) GetStatus(tenantID string) (domain.DeviceStatus, string, error) {
	s.mu.RLock()
	client, exists := s.clients[tenantID]
	s.mu.RUnlock()

	// 1. Check from existing whatsmeow client in memory
	if exists && client != nil && client.Client != nil {
		if client.Client.IsConnected() {
			return domain.DeviceStatusConnected, client.Phone, nil
		}
		if client.Status == domain.DeviceStatusQRGenerated {
			return domain.DeviceStatusQRGenerated, client.Phone, nil
		}
		return domain.DeviceStatusDisconnected, client.Phone, nil
	}

	// 2. Not in memory, check from DB.
	device, err := s.deviceRepo.FindByTenantID(tenantID)
	if err != nil {
		return domain.DeviceStatusDisconnected, "", err
	}

	// 3. If the DB thinks it's connected, but it's not in memory, try to spin it up.
	// This ensures that after a server restart, the API returns the actual status from Whatsmeow.
	if device.Status == domain.DeviceStatusConnected {
		fmt.Printf("[WhatsMeow] Instance not in memory, attempting to restore connection for tenant %s\n", tenantID)
		err := s.Connect(tenantID)
		if err == nil {
			s.mu.RLock()
			newClient := s.clients[tenantID]
			s.mu.RUnlock()
			if newClient != nil {
				return domain.DeviceStatusConnected, newClient.Phone, nil
			}
			return domain.DeviceStatusConnected, device.PhoneNumber, nil
		}

		fmt.Printf("[WhatsMeow] Failed to restore connection: %v\n", err)
		// Update DB so it reflects WhatsApp's actual disconnected state
		device.Status = domain.DeviceStatusDisconnected
		_ = s.deviceRepo.Update(device)

		return domain.DeviceStatusDisconnected, device.PhoneNumber, nil
	}

	return device.Status, device.PhoneNumber, nil
}

func (s *WhatsAppService) Connect(tenantID string) error {
	device, err := s.deviceRepo.FindByTenantID(tenantID)
	if err != nil {
		return fmt.Errorf("device not found: %w", err)
	}

	dbLog := waLog.Stdout("Database", "ERROR", true)
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

	clientLog := waLog.Stdout("Client", "ERROR", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	client.AddEventHandler(func(evt interface{}) {
		s.handleEvent(tenantID, evt)
	})

	if client.Store.ID == nil {
		return fmt.Errorf("no session found, please generate QR code first")
	}

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
			_ = client.Client.Logout(context.Background())
			client.Client.Disconnect()
			if client.Client.Store != nil {
				_ = client.Client.Store.Delete(context.Background())
			}
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

func (s *WhatsAppService) SendMessage(tenantID, phone, message, mediaURL string) error {
	fmt.Printf("[WhatsMeow] SendMessage called: tenant=%s, phone=%s, message=%s, mediaURL=%s\n", tenantID, phone, message, mediaURL)

	s.mu.RLock()
	client, exists := s.clients[tenantID]
	s.mu.RUnlock()

	if !exists || client.Client == nil || (client.Status != domain.DeviceStatusConnected && client.Status != domain.DeviceStatusActive) {
		fmt.Printf("[WhatsMeow] Client not connected, attempting to reconnect for tenant %s\n", tenantID)
		if err := s.Connect(tenantID); err != nil {
			fmt.Printf("[WhatsMeow] Reconnect failed: %v\n", err)
			return fmt.Errorf("device not connected: %w", err)
		}

		s.mu.RLock()
		client, exists = s.clients[tenantID]
		s.mu.RUnlock()

		if !exists || client == nil {
			return fmt.Errorf("failed to reconnect")
		}
	}

	jid := types.NewJID(phone, "s.whatsapp.net")
	fmt.Printf("[WhatsMeow] Sending to JID: %s\n", jid.String())

	var waMsg waE2E.Message

	if mediaURL != "" {
		fmt.Printf("[WhatsMeow] Handling media message: %s\n", mediaURL)
		
		var imageData []byte
		var err error
		
		// Handle both local paths and URLs
		if strings.HasPrefix(mediaURL, "http") {
			resp, err := http.Get(mediaURL)
			if err != nil {
				return fmt.Errorf("failed to download image: %w", err)
			}
			defer resp.Body.Close()
			imageData, err = io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read downloaded image: %w", err)
			}
		} else {
			// Assume it's a local file path relative to project root
			// Strip leading slash if present
			path := strings.TrimPrefix(mediaURL, "/")
			imageData, err = os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read local image file: %w", err)
			}
		}

		// Determine mimetype
		mimetype := http.DetectContentType(imageData)
		
		// Upload to WA
		uploadResp, err := client.Client.Upload(context.Background(), imageData, whatsmeow.MediaImage)
		if err != nil {
			return fmt.Errorf("failed to upload image to WhatsApp: %w", err)
		}

		waMsg.ImageMessage = &waE2E.ImageMessage{
			Caption:       proto.String(message),
			Mimetype:      proto.String(mimetype),
			URL:           proto.String(uploadResp.URL),
			DirectPath:    proto.String(uploadResp.DirectPath),
			MediaKey:      uploadResp.MediaKey,
			FileEncSHA256: uploadResp.FileEncSHA256,
			FileSHA256:    uploadResp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(imageData))),
		}
	} else {
		waMsg.Conversation = proto.String(message)
	}

	resp, err := client.Client.SendMessage(context.Background(), jid, &waMsg)
	if err != nil {
		fmt.Printf("[WhatsMeow] SendMessage error: %v\n", err)
		return fmt.Errorf("failed to send message: %w", err)
	}

	fmt.Printf("[WhatsMeow] Message sent to %s (ID: %s)\n", phone, resp.ID)
	return nil
}

func (s *WhatsAppService) SendTypingIndicator(tenantID, phone string) {
	s.mu.RLock()
	client, exists := s.clients[tenantID]
	s.mu.RUnlock()

	if !exists || client.Client == nil {
		return
	}

	jid := types.NewJID(phone, "s.whatsapp.net")
	err := client.Client.SendChatPresence(context.Background(), jid, types.ChatPresenceComposing, "")
	if err != nil {
		fmt.Printf("[WhatsMeow] Failed to send typing indicator: %v\n", err)
	}
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
	return uuid.New().String()
}

func (s *WhatsAppService) PushCampaignUpdate(tenantID string, data map[string]interface{}) {
	data["type"] = "campaign_update"
	fmt.Printf("[PushCampaignUpdate] tenantID=%s, data=%+v\n", tenantID, data)
	s.pushToWebSocket(tenantID, data)
}

func (s *WhatsAppService) pushToWebSocket(tenantID string, data map[string]interface{}) {
	s.wsMu.RLock()
	conn, ok := s.wsConns[tenantID]
	s.wsMu.RUnlock()

	fmt.Printf("[WS] pushToWebSocket: tenantID=%s, hasConn=%v, conn=%v\n", tenantID, ok, conn)

	if !ok || conn == nil {
		return
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("[WS] Failed to marshal JSON: %v\n", err)
		return
	}

	err = conn.Write(context.Background(), websocket.MessageText, jsonData)
	if err != nil {
		fmt.Printf("[WS] Failed to write message: %v\n", err)
		s.wsMu.Lock()
		delete(s.wsConns, tenantID)
		s.wsMu.Unlock()
	}
}

func (s *WhatsAppService) HandleQRWebSocket(tenantID string, w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[WS] HandleQRWebSocket called with tenantID: %s\n", tenantID)
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		CompressionMode:    websocket.CompressionContextTakeover,
		InsecureSkipVerify: true,
	})
	if err != nil {
		fmt.Printf("[WS] Failed to accept WebSocket: %v\n", err)
		return
	}

	s.wsMu.Lock()
	s.wsConns[tenantID] = conn
	s.wsMu.Unlock()

	fmt.Printf("[WS] WebSocket connected for tenant %s\n", tenantID)

	ctx := r.Context()
	<-ctx.Done()

	s.wsMu.Lock()
	delete(s.wsConns, tenantID)
	s.wsMu.Unlock()

	conn.Close(websocket.StatusNormalClosure, "")
	fmt.Printf("[WS] WebSocket disconnected for tenant %s\n", tenantID)
}

func (s *WhatsAppService) GetJoinedGroups(tenantID string) ([]map[string]interface{}, error) {
	s.mu.RLock()
	client, ok := s.clients[tenantID]
	s.mu.RUnlock()

	isConnected := client != nil && client.Client != nil && client.Client.IsConnected()
	fmt.Printf("[GetJoinedGroups] tenantID=%s, inMap=%v, clientNil=%v, IsConnected=%v\n", tenantID, ok, client == nil, isConnected)

	if !isConnected {
		device, err := s.deviceRepo.FindByTenantID(tenantID)
		fmt.Printf("[GetJoinedGroups] device status from DB: %s, err: %v\n", device.Status, err)
		if err == nil && device.Status == domain.DeviceStatusConnected {
			fmt.Printf("[GetJoinedGroups] Attempting to reconnect...\n")
			if err := s.Connect(tenantID); err != nil {
				return nil, fmt.Errorf("device not connected: %w", err)
			}
			s.mu.RLock()
			client, ok = s.clients[tenantID]
			s.mu.RUnlock()
			if !ok || client == nil || client.Client == nil || !client.Client.IsConnected() {
				return nil, fmt.Errorf("device not connected after reconnect")
			}
		} else {
			return nil, fmt.Errorf("device not connected")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	groups, err := client.Client.GetJoinedGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %w", err)
	}

	var result []map[string]interface{}
	for _, group := range groups {
		result = append(result, map[string]interface{}{
			"jid":   group.JID.String(),
			"group": group.Name,
			"name":  group.Name,
			"phone": group.JID.User,
		})
	}

	return result, nil
}

func (s *WhatsAppService) ImportGroupContacts(tenantID, groupJID string) (int, error) {
	s.mu.RLock()
	client, ok := s.clients[tenantID]
	s.mu.RUnlock()

	if !ok || client == nil || client.Client == nil || !client.Client.IsConnected() {
		device, err := s.deviceRepo.FindByTenantID(tenantID)
		if err == nil && device.Status == domain.DeviceStatusConnected {
			if err := s.Connect(tenantID); err != nil {
				return 0, fmt.Errorf("device not connected: %w", err)
			}
			s.mu.RLock()
			client, ok = s.clients[tenantID]
			s.mu.RUnlock()
			if !ok || client == nil || client.Client == nil || !client.Client.IsConnected() {
				return 0, fmt.Errorf("device not connected")
			}
		} else {
			return 0, fmt.Errorf("device not connected")
		}
	}

	jid, err := types.ParseJID(groupJID)
	if err != nil {
		return 0, fmt.Errorf("invalid group JID: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := client.Client.GetGroupInfo(ctx, jid)
	if err != nil {
		if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline") {
			return 0, fmt.Errorf("group not accessible - you may no longer be a member of this group, or WhatsApp servers are slow. Try refreshing the group list")
		}
		return 0, fmt.Errorf("failed to get group info: %w", err)
	}

	var participantJIDs []types.JID
		// Try to get participants from linked groups if it's a community
	linkedParticipants, err := client.Client.GetLinkedGroupsParticipants(ctx, jid)
	if err == nil && len(linkedParticipants) > 0 {
		participantJIDs = linkedParticipants
	} else {
		// Fallback to regular group participants
		for _, p := range resp.Participants {
			participantJIDs = append(participantJIDs, p.JID)
		}
	}

	// Find or create group in DB
	groupName := resp.Name
	if groupName == "" {
		groupName = "Imported Group"
	}
	dbGroup, err := s.groupRepo.FindByTenantIDAndName(tenantID, groupName)
	if err != nil {
		dbGroup = &domain.Group{
			ID:       uuid.New().String(),
			TenantID: tenantID,
			Name:     groupName,
		}
		if err := s.groupRepo.Create(dbGroup); err != nil {
			fmt.Printf("[WhatsMeow] Failed to create group record: %v\n", err)
		}
	}

	var contacts []domain.Contact
	for _, pJID := range participantJIDs {
		if pJID.Server == "lid" {
			pn, err := client.Client.Store.LIDs.GetPNForLID(ctx, pJID)
			if err == nil && !pn.IsEmpty() {
				pJID = pn
			} else {
				// Cannot resolve LID to a real phone number right now
				continue
			}
		}

		phone := pJID.User
		if phone == "" {
			continue
		}

		// Try to get contact name from store
		contactName := ""
		if contact, err := client.Client.Store.Contacts.GetContact(ctx, pJID); err == nil {
			if contact.PushName != "" {
				contactName = contact.PushName
			} else if contact.FullName != "" {
				contactName = contact.FullName
			}
		}

		contacts = append(contacts, domain.Contact{
			ID:       uuid.New().String(),
			TenantID: tenantID,
			Phone:    phone,
			Name:     contactName,
			Groups:   []domain.Group{*dbGroup},
		})
	}

	if err := s.contactRepo.UpsertBatch(contacts); err != nil {
		return 0, fmt.Errorf("failed to save contacts: %w", err)
	}

	return len(contacts), nil
}
