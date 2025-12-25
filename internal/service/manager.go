package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/steipete/wacli/internal/app"
	"github.com/steipete/wacli/internal/lock"
	"github.com/steipete/wacli/internal/store"
	"github.com/steipete/wacli/internal/wa"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
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

	// CRITICAL: Register event handler BEFORE Connect() so that pairing events are processed
	// Without this, the key exchange during device linking won't work
	pairSuccess := make(chan struct{}, 1)
	handlerID := m.app.WA().AddEventHandler(func(evt interface{}) {
		switch v := evt.(type) {
		case *events.PairSuccess:
			log.Printf("[Manager] Pair success: %s", v.ID.String())
			select {
			case pairSuccess <- struct{}{}:
			default:
			}
		case *events.Connected:
			log.Println("[Manager] WhatsApp connected event received")
		case *events.Disconnected:
			log.Println("[Manager] WhatsApp disconnected during auth")
		}
	})

	// Connect with QR code generation
	err := m.app.Connect(ctx, true, func(qr string) {
		log.Printf("[Manager] QR code generated (length: %d)", len(qr))
		m.state.SetQRCode(qr)
	})

	// Remove temp handler - sync worker will add its own
	m.app.WA().RemoveEventHandler(handlerID)

	if err != nil {
		log.Printf("[Manager] Authentication failed: %v", err)
		m.state.SetError(err)
		return err
	}

	// Wait a moment for pair success event
	select {
	case <-pairSuccess:
		log.Println("[Manager] Pair success confirmed")
	case <-time.After(5 * time.Second):
		log.Println("[Manager] Pair success event not received, continuing anyway")
	}

	// Verify authentication worked
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
	<-m.ctx.Done()
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
