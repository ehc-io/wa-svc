package service

import (
	"context"
	"fmt"
	"log"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/steipete/wacli/internal/app"
	"github.com/steipete/wacli/internal/lock"
	"github.com/steipete/wacli/internal/store"
	"github.com/steipete/wacli/internal/wa"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// MessageHandler is called when a new message is received.
type MessageHandler func(msg *ReceivedMessage)

// ReceivedMessage represents a message received from WhatsApp.
type ReceivedMessage struct {
	ChatJID    string    `json:"chat_jid"`
	ChatName   string    `json:"chat_name"`
	MsgID      string    `json:"msg_id"`
	SenderJID  string    `json:"sender_jid,omitempty"`
	SenderName string    `json:"sender_name,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	FromMe     bool      `json:"from_me"`
	Text       string    `json:"text,omitempty"`
	MediaType  string    `json:"media_type,omitempty"`
	Caption    string    `json:"caption,omitempty"`
}

// Manager is the central service that manages the WhatsApp connection lifecycle.
type Manager struct {
	config Config
	state  *StateMachine
	app    *app.App
	lock   *lock.Lock

	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	syncRunning    bool
	syncCtx        context.Context
	syncCancel     context.CancelFunc
	eventHandlerID uint32

	messageHandlers []MessageHandler
	handlersMu      sync.RWMutex
}

// NewManager creates a new service manager.
func NewManager(cfg Config) (*Manager, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &Manager{
		config: cfg,
		state:  NewStateMachine(),
	}, nil
}

// Start initializes the service, acquires the lock, and starts the WhatsApp connection.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.app != nil {
		return fmt.Errorf("manager already started")
	}

	// Acquire file lock
	lk, err := lock.Acquire(m.config.DataDir)
	if err != nil {
		m.state.SetError(err)
		return fmt.Errorf("acquire lock: %w", err)
	}
	m.lock = lk

	// Initialize app
	a, err := app.New(app.Options{
		StoreDir: m.config.DataDir,
		Version:  "wasvc/1.0",
	})
	if err != nil {
		_ = lk.Release()
		m.state.SetError(err)
		return fmt.Errorf("initialize app: %w", err)
	}
	m.app = a

	// Create cancellable context for background tasks
	m.ctx, m.cancel = context.WithCancel(ctx)

	// Try to connect
	go m.connectAndSync()

	return nil
}

// Stop gracefully shuts down the service.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel != nil {
		m.cancel()
	}

	if m.app != nil {
		if m.eventHandlerID != 0 && m.app.WA() != nil {
			m.app.WA().RemoveEventHandler(m.eventHandlerID)
		}
		m.app.Close()
		m.app = nil
	}

	if m.lock != nil {
		_ = m.lock.Release()
		m.lock = nil
	}

	m.state.SetState(StateDisconnected)
	return nil
}

// State returns the current state machine.
func (m *Manager) State() *StateMachine {
	return m.state
}

// Config returns the current configuration.
func (m *Manager) Config() Config {
	return m.config
}

// App returns the underlying app instance (for direct access to DB, etc.).
func (m *Manager) App() *app.App {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.app
}

// OnMessage registers a handler for incoming messages.
func (m *Manager) OnMessage(handler MessageHandler) {
	m.handlersMu.Lock()
	defer m.handlersMu.Unlock()
	m.messageHandlers = append(m.messageHandlers, handler)
}

// notifyMessageHandlers calls all registered message handlers.
func (m *Manager) notifyMessageHandlers(msg *ReceivedMessage) {
	m.handlersMu.RLock()
	handlers := m.messageHandlers
	m.handlersMu.RUnlock()

	for _, h := range handlers {
		go h(msg)
	}
}

// connectAndSync handles the initial connection and starts the sync worker.
func (m *Manager) connectAndSync() {
	m.state.SetState(StateConnecting)

	if err := m.app.OpenWA(); err != nil {
		log.Printf("[Manager] Failed to open WA client: %v", err)
		m.state.SetError(err)
		return
	}

	// Check if already authenticated
	if m.app.WA().IsAuthed() {
		log.Println("[Manager] Already authenticated, connecting...")
		if err := m.app.Connect(m.ctx, false, nil); err != nil {
			log.Printf("[Manager] Failed to connect: %v", err)
			m.state.SetError(err)
			return
		}
		m.state.SetState(StateConnected)
		m.startSyncWorker()
	} else {
		log.Println("[Manager] Not authenticated, waiting for QR scan...")
		m.state.SetState(StateUnauthenticated)
	}
}

// InitiateAuth starts the QR code authentication flow.
func (m *Manager) InitiateAuth(ctx context.Context) error {
	m.mu.Lock()
	if m.app == nil {
		m.mu.Unlock()
		return fmt.Errorf("manager not started")
	}
	m.mu.Unlock()

	// Open WA client if not already open
	if err := m.app.OpenWA(); err != nil {
		log.Printf("[Manager] Failed to open WA: %v", err)
		m.state.SetError(err)
		return err
	}

	if m.app.WA() != nil && m.app.WA().IsAuthed() {
		log.Println("[Manager] Already authenticated")
		return fmt.Errorf("already authenticated")
	}

	log.Println("[Manager] Starting authentication flow...")
	m.state.SetState(StateConnecting)

	// Channels to track auth completion
	connected := make(chan struct{}, 1)
	authFailed := make(chan error, 1)

	// CRITICAL: Register event handler BEFORE Connect() so that pairing events are processed
	// Without this, the key exchange during device linking won't work.
	// The handler must stay active through the entire pairing process, not just until Connect() returns.
	handlerID := m.app.WA().AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.PairSuccess:
			log.Printf("[Manager] Pair success: %s", v.ID.String())
		case *events.PairError:
			log.Printf("[Manager] Pair error: %v", v.Error)
			select {
			case authFailed <- fmt.Errorf("pairing failed: %v", v.Error):
			default:
			}
		case *events.Connected:
			log.Println("[Manager] WhatsApp connected event received")
			select {
			case connected <- struct{}{}:
			default:
			}
		case *events.Disconnected:
			log.Println("[Manager] WhatsApp disconnected during auth")
		}
	})

	// Connect with QR code generation - this blocks until QR flow completes
	err := m.app.Connect(ctx, true, func(qr string) {
		log.Printf("[Manager] QR code generated (length: %d)", len(qr))
		m.state.SetQRCode(qr)
	})

	if err != nil {
		m.app.WA().RemoveEventHandler(handlerID)
		log.Printf("[Manager] Authentication failed: %v", err)
		m.state.SetError(err)
		return err
	}

	// Wait for Connected event with timeout - the encryption handshake may take a few seconds
	// after Connect() returns. DO NOT remove the event handler until we confirm auth is complete.
	select {
	case <-connected:
		log.Println("[Manager] Authentication confirmed via Connected event")
	case err := <-authFailed:
		m.app.WA().RemoveEventHandler(handlerID)
		log.Printf("[Manager] Authentication failed: %v", err)
		m.state.SetError(err)
		return err
	case <-time.After(30 * time.Second):
		// Timeout waiting for Connected event - check if actually authed anyway
		if !m.app.WA().IsAuthed() {
			err := fmt.Errorf("authentication timed out waiting for connection")
			m.app.WA().RemoveEventHandler(handlerID)
			log.Printf("[Manager] %v", err)
			m.state.SetError(err)
			return err
		}
		log.Println("[Manager] Auth appears complete despite no Connected event")
	case <-ctx.Done():
		m.app.WA().RemoveEventHandler(handlerID)
		return ctx.Err()
	}

	// NOW it's safe to remove the temp handler - sync worker will add its own
	m.app.WA().RemoveEventHandler(handlerID)

	// Final verification that authentication worked
	if !m.app.WA().IsAuthed() {
		err := fmt.Errorf("authentication did not complete properly")
		log.Printf("[Manager] %v", err)
		m.state.SetError(err)
		return err
	}

	m.state.ClearQRCode()
	m.state.SetState(StateConnected)
	log.Println("[Manager] Authentication successful")

	// Start sync worker after successful auth
	m.startSyncWorker()
	return nil
}

// startSyncWorker starts the background sync process.
func (m *Manager) startSyncWorker() {
	m.mu.Lock()
	if m.syncRunning {
		m.mu.Unlock()
		return
	}
	m.syncRunning = true
	m.syncCtx, m.syncCancel = context.WithCancel(m.ctx)
	m.mu.Unlock()

	go m.runSyncWorker()
}

// runSyncWorker runs the sync loop.
func (m *Manager) runSyncWorker() {
	defer func() {
		m.mu.Lock()
		m.syncRunning = false
		m.mu.Unlock()
	}()

	log.Println("[Manager] Starting sync worker...")

	// Register event handler for messages
	m.eventHandlerID = m.app.WA().AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			m.handleIncomingMessage(v)
		case *events.Connected:
			log.Println("[Manager] WhatsApp connected")
			m.state.SetState(StateConnected)
		case *events.Disconnected:
			log.Println("[Manager] WhatsApp disconnected")
			m.state.SetState(StateDisconnected)
		case *events.HistorySync:
			m.handleHistorySync(v)
		}
	})

	// Keep the worker running until context is cancelled
	<-m.syncCtx.Done()
	log.Println("[Manager] Sync worker stopped")
}

// handleIncomingMessage processes an incoming message.
func (m *Manager) handleIncomingMessage(evt *events.Message) {
	pm := wa.ParseLiveMessage(evt)
	if pm.ID == "" {
		return
	}

	// Store the message
	a := m.App()
	if a == nil {
		return
	}

	chatName := ""
	if a.WA() != nil {
		chatName = a.WA().ResolveChatName(m.ctx, pm.Chat, pm.PushName)
	}

	_ = a.DB().UpsertChat(pm.Chat.String(), chatKind(pm.Chat), chatName, pm.Timestamp)

	var mediaType, caption string
	if pm.Media != nil {
		mediaType = pm.Media.Type
		caption = pm.Media.Caption
	}

	_ = a.DB().UpsertMessage(store.UpsertMessageParams{
		ChatJID:    pm.Chat.String(),
		ChatName:   chatName,
		MsgID:      pm.ID,
		SenderJID:  pm.SenderJID,
		SenderName: pm.PushName,
		Timestamp:  pm.Timestamp,
		FromMe:     pm.FromMe,
		Text:       pm.Text,
		MediaType:  mediaType,
	})

	// Notify handlers
	msg := &ReceivedMessage{
		ChatJID:    pm.Chat.String(),
		ChatName:   chatName,
		MsgID:      pm.ID,
		SenderJID:  pm.SenderJID,
		SenderName: pm.PushName,
		Timestamp:  pm.Timestamp,
		FromMe:     pm.FromMe,
		Text:       pm.Text,
		MediaType:  mediaType,
		Caption:    caption,
	}
	m.notifyMessageHandlers(msg)
}

// handleHistorySync processes history sync events.
func (m *Manager) handleHistorySync(evt *events.HistorySync) {
	log.Printf("[Manager] Processing history sync (%d conversations)", len(evt.Data.Conversations))

	a := m.App()
	if a == nil {
		return
	}

	for _, conv := range evt.Data.Conversations {
		chatID := conv.GetID()
		if chatID == "" {
			continue
		}
		for _, msg := range conv.Messages {
			if msg.Message == nil {
				continue
			}
			pm := wa.ParseHistoryMessage(chatID, msg.Message)
			if pm.ID == "" {
				continue
			}

			chatName := ""
			if a.WA() != nil {
				chatName = a.WA().ResolveChatName(m.ctx, pm.Chat, pm.PushName)
			}

			_ = a.DB().UpsertChat(pm.Chat.String(), chatKind(pm.Chat), chatName, pm.Timestamp)
			_ = a.DB().UpsertMessage(store.UpsertMessageParams{
				ChatJID:    pm.Chat.String(),
				ChatName:   chatName,
				MsgID:      pm.ID,
				SenderJID:  pm.SenderJID,
				SenderName: pm.PushName,
				Timestamp:  pm.Timestamp,
				FromMe:     pm.FromMe,
				Text:       pm.Text,
			})
		}
	}
}

// SendText sends a text message to the specified recipient.
func (m *Manager) SendText(ctx context.Context, to, text string) (string, error) {
	if !m.state.State().IsReady() {
		return "", fmt.Errorf("service not ready (state: %s)", m.state.State())
	}

	a := m.App()
	if a == nil || a.WA() == nil {
		return "", fmt.Errorf("WhatsApp client not available")
	}

	toJID, err := wa.ParseUserOrJID(to)
	if err != nil {
		return "", fmt.Errorf("invalid recipient: %w", err)
	}

	msgID, err := a.WA().SendText(ctx, toJID, text)
	if err != nil {
		return "", err
	}

	// Store sent message
	now := time.Now().UTC()
	chatName := a.WA().ResolveChatName(ctx, toJID, "")
	_ = a.DB().UpsertChat(toJID.String(), chatKind(toJID), chatName, now)
	_ = a.DB().UpsertMessage(store.UpsertMessageParams{
		ChatJID:    toJID.String(),
		ChatName:   chatName,
		MsgID:      string(msgID),
		SenderName: "me",
		Timestamp:  now,
		FromMe:     true,
		Text:       text,
	})

	return string(msgID), nil
}

// SendFileResult contains the result of sending a file.
type SendFileResult struct {
	MessageID string
	MediaType string
	Filename  string
	MimeType  string
}

// SendFile sends a file/media to the specified recipient.
func (m *Manager) SendFile(ctx context.Context, to string, data []byte, filename, caption, mimeType string) (*SendFileResult, error) {
	if !m.state.State().IsReady() {
		return nil, fmt.Errorf("service not ready (state: %s)", m.state.State())
	}

	a := m.App()
	if a == nil || a.WA() == nil {
		return nil, fmt.Errorf("WhatsApp client not available")
	}

	toJID, err := wa.ParseUserOrJID(to)
	if err != nil {
		return nil, fmt.Errorf("invalid recipient: %w", err)
	}

	// Detect mime type if not provided
	if mimeType == "" {
		mimeType = detectMimeType(filename, data)
	}

	// Determine media type and upload type
	mediaType := "document"
	uploadType, _ := wa.MediaTypeFromString("document")
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		mediaType = "image"
		uploadType, _ = wa.MediaTypeFromString("image")
	case strings.HasPrefix(mimeType, "video/"):
		mediaType = "video"
		uploadType, _ = wa.MediaTypeFromString("video")
	case strings.HasPrefix(mimeType, "audio/"):
		mediaType = "audio"
		uploadType, _ = wa.MediaTypeFromString("audio")
	}

	// Upload the file
	up, err := a.WA().Upload(ctx, data, uploadType)
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}

	// Build the message
	msg := buildMediaMessage(mediaType, mimeType, filename, caption, up)

	// Send the message
	msgID, err := a.WA().SendProtoMessage(ctx, toJID, msg)
	if err != nil {
		return nil, fmt.Errorf("send failed: %w", err)
	}

	// Store sent message
	now := time.Now().UTC()
	chatName := a.WA().ResolveChatName(ctx, toJID, "")
	_ = a.DB().UpsertChat(toJID.String(), chatKind(toJID), chatName, now)
	_ = a.DB().UpsertMessage(store.UpsertMessageParams{
		ChatJID:       toJID.String(),
		ChatName:      chatName,
		MsgID:         msgID,
		SenderName:    "me",
		Timestamp:     now,
		FromMe:        true,
		Text:          caption,
		MediaType:     mediaType,
		MediaCaption:  caption,
		Filename:      filename,
		MimeType:      mimeType,
		DirectPath:    up.DirectPath,
		MediaKey:      up.MediaKey,
		FileSHA256:    up.FileSHA256,
		FileEncSHA256: up.FileEncSHA256,
		FileLength:    up.FileLength,
	})

	return &SendFileResult{
		MessageID: msgID,
		MediaType: mediaType,
		Filename:  filename,
		MimeType:  mimeType,
	}, nil
}

// SearchMessages searches messages in the database.
func (m *Manager) SearchMessages(query string, limit int) ([]store.Message, error) {
	a := m.App()
	if a == nil {
		return nil, fmt.Errorf("app not initialized")
	}

	return a.DB().SearchMessages(store.SearchMessagesParams{
		Query: query,
		Limit: limit,
	})
}

// ListChats returns recent chats.
func (m *Manager) ListChats(query string, limit int) ([]store.Chat, error) {
	a := m.App()
	if a == nil {
		return nil, fmt.Errorf("app not initialized")
	}

	return a.DB().ListChats(query, limit)
}

// ListMessages returns messages from a chat.
func (m *Manager) ListMessages(chatJID string, limit int) ([]store.Message, error) {
	a := m.App()
	if a == nil {
		return nil, fmt.Errorf("app not initialized")
	}

	return a.DB().ListMessages(store.ListMessagesParams{
		ChatJID: chatJID,
		Limit:   limit,
	})
}

// GetMediaDownloadInfo returns media info for a message.
func (m *Manager) GetMediaDownloadInfo(chatJID, msgID string) (store.MediaDownloadInfo, error) {
	a := m.App()
	if a == nil {
		return store.MediaDownloadInfo{}, fmt.Errorf("app not initialized")
	}

	return a.DB().GetMediaDownloadInfo(chatJID, msgID)
}

// DownloadMediaResult contains the result of downloading media.
type DownloadMediaResult struct {
	ChatJID      string
	MsgID        string
	MediaType    string
	MimeType     string
	LocalPath    string
	Bytes        int64
	DownloadedAt time.Time
}

// DownloadMedia downloads media for a message and saves it to the store.
func (m *Manager) DownloadMedia(ctx context.Context, chatJID, msgID string) (*DownloadMediaResult, error) {
	if !m.state.State().IsReady() {
		return nil, fmt.Errorf("service not ready (state: %s)", m.state.State())
	}

	a := m.App()
	if a == nil || a.WA() == nil {
		return nil, fmt.Errorf("app not initialized")
	}

	// Get media info from database
	info, err := a.DB().GetMediaDownloadInfo(chatJID, msgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get media info: %w", err)
	}

	if info.MediaType == "" || info.DirectPath == "" || len(info.MediaKey) == 0 {
		return nil, fmt.Errorf("message has no downloadable media metadata")
	}

	// If already downloaded, return existing info
	if info.LocalPath != "" {
		return &DownloadMediaResult{
			ChatJID:      info.ChatJID,
			MsgID:        info.MsgID,
			MediaType:    info.MediaType,
			MimeType:     info.MimeType,
			LocalPath:    info.LocalPath,
			Bytes:        int64(info.FileLength),
			DownloadedAt: info.DownloadedAt,
		}, nil
	}

	// Resolve output path
	targetPath, err := a.ResolveMediaOutputPath(info, "")
	if err != nil {
		return nil, fmt.Errorf("failed to resolve output path: %w", err)
	}

	// Download the media
	bytes, err := a.WA().DownloadMediaToFile(ctx, info.DirectPath, info.FileEncSHA256, info.FileSHA256, info.MediaKey, info.FileLength, info.MediaType, "", targetPath)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}

	// Mark as downloaded in database
	now := time.Now().UTC()
	_ = a.DB().MarkMediaDownloaded(info.ChatJID, info.MsgID, targetPath, now)

	return &DownloadMediaResult{
		ChatJID:      info.ChatJID,
		MsgID:        info.MsgID,
		MediaType:    info.MediaType,
		MimeType:     info.MimeType,
		LocalPath:    targetPath,
		Bytes:        bytes,
		DownloadedAt: now,
	}, nil
}

// Logout disconnects and clears the session.
func (m *Manager) Logout(ctx context.Context) error {
	a := m.App()
	if a == nil || a.WA() == nil {
		return fmt.Errorf("not initialized")
	}

	if err := a.WA().Logout(ctx); err != nil {
		return err
	}

	m.state.SetState(StateUnauthenticated)
	return nil
}

func chatKind(chat types.JID) string {
	if chat.Server == types.GroupServer {
		return "group"
	}
	if chat.IsBroadcastList() {
		return "broadcast"
	}
	if chat.Server == types.DefaultUserServer {
		return "dm"
	}
	return "unknown"
}

// --- Contact Methods ---

// SearchContacts searches contacts in the local database.
func (m *Manager) SearchContacts(query string, limit int) ([]store.Contact, error) {
	a := m.App()
	if a == nil {
		return nil, fmt.Errorf("app not initialized")
	}
	return a.DB().SearchContacts(query, limit)
}

// GetContact returns a single contact from the local database.
func (m *Manager) GetContact(jid string) (store.Contact, error) {
	a := m.App()
	if a == nil {
		return store.Contact{}, fmt.Errorf("app not initialized")
	}
	return a.DB().GetContact(jid)
}

// RefreshContacts imports contacts from WhatsApp to the local database.
func (m *Manager) RefreshContacts(ctx context.Context) (int, error) {
	a := m.App()
	if a == nil || a.WA() == nil {
		return 0, fmt.Errorf("app not initialized")
	}

	contacts, err := a.WA().GetAllContacts(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get contacts: %w", err)
	}

	count := 0
	for jid, contact := range contacts {
		err := a.DB().UpsertContact(jid.String(), jid.User, contact.PushName, contact.FullName, contact.FirstName, contact.BusinessName)
		if err != nil {
			log.Printf("[Manager] Failed to upsert contact %s: %v", jid.String(), err)
			continue
		}
		count++
	}

	return count, nil
}

// SetContactAlias sets an alias for a contact.
func (m *Manager) SetContactAlias(jid, alias string) error {
	a := m.App()
	if a == nil {
		return fmt.Errorf("app not initialized")
	}
	return a.DB().SetAlias(jid, alias)
}

// RemoveContactAlias removes the alias from a contact.
func (m *Manager) RemoveContactAlias(jid string) error {
	a := m.App()
	if a == nil {
		return fmt.Errorf("app not initialized")
	}
	return a.DB().RemoveAlias(jid)
}

// AddContactTag adds a tag to a contact.
func (m *Manager) AddContactTag(jid, tag string) error {
	a := m.App()
	if a == nil {
		return fmt.Errorf("app not initialized")
	}
	return a.DB().AddTag(jid, tag)
}

// RemoveContactTag removes a tag from a contact.
func (m *Manager) RemoveContactTag(jid, tag string) error {
	a := m.App()
	if a == nil {
		return fmt.Errorf("app not initialized")
	}
	return a.DB().RemoveTag(jid, tag)
}

// --- Group Methods ---

// ListGroups returns groups from the local database.
func (m *Manager) ListGroups(query string, limit int) ([]store.Group, error) {
	a := m.App()
	if a == nil {
		return nil, fmt.Errorf("app not initialized")
	}
	return a.DB().ListGroups(query, limit)
}

// GetGroupInfo fetches live group info from WhatsApp.
func (m *Manager) GetGroupInfo(ctx context.Context, jidStr string) (*types.GroupInfo, error) {
	a := m.App()
	if a == nil || a.WA() == nil {
		return nil, fmt.Errorf("app not initialized")
	}

	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return nil, fmt.Errorf("invalid JID: %w", err)
	}

	return a.WA().GetGroupInfo(ctx, jid)
}

// RefreshGroups imports joined groups from WhatsApp to the local database.
func (m *Manager) RefreshGroups(ctx context.Context) (int, error) {
	a := m.App()
	if a == nil || a.WA() == nil {
		return 0, fmt.Errorf("app not initialized")
	}

	groups, err := a.WA().GetJoinedGroups(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get groups: %w", err)
	}

	count := 0
	for _, g := range groups {
		ownerJID := ""
		if g.OwnerJID.User != "" {
			ownerJID = g.OwnerJID.String()
		}
		err := a.DB().UpsertGroup(g.JID.String(), g.Name, ownerJID, g.GroupCreated)
		if err != nil {
			log.Printf("[Manager] Failed to upsert group %s: %v", g.JID.String(), err)
			continue
		}
		count++
	}

	return count, nil
}

// RenameGroup changes the name of a group.
func (m *Manager) RenameGroup(ctx context.Context, jidStr, name string) error {
	a := m.App()
	if a == nil || a.WA() == nil {
		return fmt.Errorf("app not initialized")
	}

	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return fmt.Errorf("invalid JID: %w", err)
	}

	return a.WA().SetGroupName(ctx, jid, name)
}

// UpdateGroupParticipants modifies group participants.
func (m *Manager) UpdateGroupParticipants(ctx context.Context, groupJIDStr string, users []string, action string) ([]types.GroupParticipant, error) {
	a := m.App()
	if a == nil || a.WA() == nil {
		return nil, fmt.Errorf("app not initialized")
	}

	groupJID, err := types.ParseJID(groupJIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid group JID: %w", err)
	}

	var userJIDs []types.JID
	for _, user := range users {
		jid, err := wa.ParseUserOrJID(user)
		if err != nil {
			return nil, fmt.Errorf("invalid user %s: %w", user, err)
		}
		userJIDs = append(userJIDs, jid)
	}

	var waAction wa.GroupParticipantAction
	switch action {
	case "add":
		waAction = wa.GroupParticipantAdd
	case "remove":
		waAction = wa.GroupParticipantRemove
	case "promote":
		waAction = wa.GroupParticipantPromote
	case "demote":
		waAction = wa.GroupParticipantDemote
	default:
		return nil, fmt.Errorf("invalid action: %s", action)
	}

	return a.WA().UpdateGroupParticipants(ctx, groupJID, userJIDs, waAction)
}

// GetGroupInviteLink returns the invite link for a group.
func (m *Manager) GetGroupInviteLink(ctx context.Context, jidStr string) (string, error) {
	a := m.App()
	if a == nil || a.WA() == nil {
		return "", fmt.Errorf("app not initialized")
	}

	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return "", fmt.Errorf("invalid JID: %w", err)
	}

	return a.WA().GetGroupInviteLink(ctx, jid, false)
}

// RevokeGroupInviteLink revokes and returns a new invite link.
func (m *Manager) RevokeGroupInviteLink(ctx context.Context, jidStr string) (string, error) {
	a := m.App()
	if a == nil || a.WA() == nil {
		return "", fmt.Errorf("app not initialized")
	}

	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return "", fmt.Errorf("invalid JID: %w", err)
	}

	return a.WA().GetGroupInviteLink(ctx, jid, true)
}

// JoinGroup joins a group using an invite code.
func (m *Manager) JoinGroup(ctx context.Context, code string) (string, error) {
	a := m.App()
	if a == nil || a.WA() == nil {
		return "", fmt.Errorf("app not initialized")
	}

	jid, err := a.WA().JoinGroupWithLink(ctx, code)
	if err != nil {
		return "", err
	}

	return jid.String(), nil
}

// LeaveGroup leaves a group.
func (m *Manager) LeaveGroup(ctx context.Context, jidStr string) error {
	a := m.App()
	if a == nil || a.WA() == nil {
		return fmt.Errorf("app not initialized")
	}

	jid, err := types.ParseJID(jidStr)
	if err != nil {
		return fmt.Errorf("invalid JID: %w", err)
	}

	return a.WA().LeaveGroup(ctx, jid)
}

// --- Sync Control Methods ---

// SyncStatus returns the current sync worker status.
func (m *Manager) SyncStatus() (running bool, state string, startedAt time.Time) {
	m.mu.RLock()
	running = m.syncRunning
	m.mu.RUnlock()
	state = string(m.state.State())
	// Note: startedAt tracking could be added if needed
	return
}

// IsSyncRunning returns whether the sync worker is running.
func (m *Manager) IsSyncRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.syncRunning
}

// --- Diagnostics ---

// GetDiagnostics returns diagnostic information about the service.
func (m *Manager) GetDiagnostics() (storeDir string, lockHeld bool, authenticated bool, connected bool) {
	storeDir = m.config.DataDir
	lockHeld = m.lock != nil

	a := m.App()
	if a != nil && a.WA() != nil {
		authenticated = a.WA().IsAuthed()
		connected = a.WA().IsConnected()
	}

	return
}

// GetDBStats returns database statistics.
func (m *Manager) GetDBStats() (messageCount, chatCount, contactCount, groupCount int64, ftsEnabled bool, err error) {
	a := m.App()
	if a == nil {
		err = fmt.Errorf("app not initialized")
		return
	}

	messageCount, err = a.DB().CountMessages()
	if err != nil {
		return
	}

	chatCount, err = a.DB().CountChats()
	if err != nil {
		return
	}

	contactCount, err = a.DB().CountContacts()
	if err != nil {
		return
	}

	groupCount, err = a.DB().CountGroups()
	if err != nil {
		return
	}

	ftsEnabled = a.DB().HasFTS()
	return
}

// detectMimeType detects the MIME type from filename extension or content.
func detectMimeType(filename string, data []byte) string {
	// Try extension first
	ext := strings.ToLower(filepath.Ext(filename))
	if mimeType := mime.TypeByExtension(ext); mimeType != "" {
		return mimeType
	}

	// Fall back to content sniffing
	sniff := data
	if len(sniff) > 512 {
		sniff = sniff[:512]
	}
	return http.DetectContentType(sniff)
}

// buildMediaMessage builds a WhatsApp media message.
func buildMediaMessage(mediaType, mimeType, filename, caption string, up whatsmeow.UploadResponse) *waProto.Message {
	msg := &waProto.Message{}

	switch mediaType {
	case "image":
		msg.ImageMessage = &waProto.ImageMessage{
			URL:           proto.String(up.URL),
			DirectPath:    proto.String(up.DirectPath),
			MediaKey:      up.MediaKey,
			FileEncSHA256: up.FileEncSHA256,
			FileSHA256:    up.FileSHA256,
			FileLength:    proto.Uint64(up.FileLength),
			Mimetype:      proto.String(mimeType),
			Caption:       proto.String(caption),
		}
	case "video":
		msg.VideoMessage = &waProto.VideoMessage{
			URL:           proto.String(up.URL),
			DirectPath:    proto.String(up.DirectPath),
			MediaKey:      up.MediaKey,
			FileEncSHA256: up.FileEncSHA256,
			FileSHA256:    up.FileSHA256,
			FileLength:    proto.Uint64(up.FileLength),
			Mimetype:      proto.String(mimeType),
			Caption:       proto.String(caption),
		}
	case "audio":
		msg.AudioMessage = &waProto.AudioMessage{
			URL:           proto.String(up.URL),
			DirectPath:    proto.String(up.DirectPath),
			MediaKey:      up.MediaKey,
			FileEncSHA256: up.FileEncSHA256,
			FileSHA256:    up.FileSHA256,
			FileLength:    proto.Uint64(up.FileLength),
			Mimetype:      proto.String(mimeType),
			PTT:           proto.Bool(false),
		}
	default:
		msg.DocumentMessage = &waProto.DocumentMessage{
			URL:           proto.String(up.URL),
			DirectPath:    proto.String(up.DirectPath),
			MediaKey:      up.MediaKey,
			FileEncSHA256: up.FileEncSHA256,
			FileSHA256:    up.FileSHA256,
			FileLength:    proto.Uint64(up.FileLength),
			Mimetype:      proto.String(mimeType),
			FileName:      proto.String(filename),
			Caption:       proto.String(caption),
			Title:         proto.String(filename),
		}
	}

	return msg
}

// BackfillResult represents the result of a history backfill operation.
type BackfillResult struct {
	ChatJID        string
	RequestsSent   int
	ResponsesSeen  int
	MessagesAdded  int64
	MessagesSynced int64
}

// BackfillHistory requests older messages for a chat from the primary device.
func (m *Manager) BackfillHistory(ctx context.Context, chatJID string, count, requests, waitSeconds int) (*BackfillResult, error) {
	a := m.App()
	if a == nil {
		return nil, fmt.Errorf("app not initialized")
	}

	waitDuration := time.Duration(waitSeconds) * time.Second
	if waitDuration <= 0 {
		waitDuration = 60 * time.Second
	}

	result, err := a.BackfillHistory(ctx, app.BackfillOptions{
		ChatJID:        chatJID,
		Count:          count,
		Requests:       requests,
		WaitPerRequest: waitDuration,
		IdleExit:       5 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	return &BackfillResult{
		ChatJID:        result.ChatJID,
		RequestsSent:   result.RequestsSent,
		ResponsesSeen:  result.ResponsesSeen,
		MessagesAdded:  result.MessagesAdded,
		MessagesSynced: result.MessagesSynced,
	}, nil
}

// StartSync starts the sync worker if not already running.
func (m *Manager) StartSync(ctx context.Context) error {
	m.mu.Lock()
	if m.syncRunning {
		m.mu.Unlock()
		return fmt.Errorf("sync is already running")
	}
	m.mu.Unlock()

	// Start sync in a goroutine
	go m.startSyncWorker()
	return nil
}

// StopSync stops the sync worker if running.
func (m *Manager) StopSync() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.syncRunning {
		return fmt.Errorf("sync is not running")
	}

	if m.syncCancel != nil {
		m.syncCancel()
	}
	return nil
}
