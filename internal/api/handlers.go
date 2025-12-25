package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/steipete/wacli/internal/service"
	"github.com/steipete/wacli/internal/store"
)

const version = "wasvc/1.0"

// Handlers holds all HTTP handlers and their dependencies.
type Handlers struct {
	manager *service.Manager
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(mgr *service.Manager) *Handlers {
	return &Handlers{manager: mgr}
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes an error response.
func writeError(w http.ResponseWriter, status int, err, code string) {
	writeJSON(w, status, ErrorResponse{Error: err, Code: code})
}

// Health handles GET /health
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	state := h.manager.State()
	status := "ok"
	if !state.State().IsReady() {
		status = "degraded"
	}

	writeJSON(w, http.StatusOK, HealthResponse{
		Status:    status,
		State:     state.State().String(),
		Ready:     state.State().IsReady(),
		Version:   version,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// AuthStatus handles GET /auth/status
func (h *Handlers) AuthStatus(w http.ResponseWriter, r *http.Request) {
	state := h.manager.State()
	info := state.StatusInfo()

	writeJSON(w, http.StatusOK, AuthStatusResponse{
		State:         info.State.String(),
		Authenticated: info.State == service.StateConnected,
		Ready:         info.Ready,
		HasQR:         info.HasQR,
		Error:         info.Error,
	})
}

// AuthQR handles GET /auth/qr
func (h *Handlers) AuthQR(w http.ResponseWriter, r *http.Request) {
	state := h.manager.State()
	currentState := state.State()
	qr := state.QRCode()

	if qr != "" {
		// Generate QR code image server-side
		qrImage, err := generateQRCodeBase64(qr, 256)
		if err != nil {
			writeJSON(w, http.StatusOK, QRCodeResponse{
				QRCode: qr,
				State:  currentState.String(),
				Error:  "failed to generate QR image: " + err.Error(),
			})
			return
		}

		writeJSON(w, http.StatusOK, QRCodeResponse{
			QRCode:  qr,
			QRImage: qrImage,
			State:   currentState.String(),
		})
		return
	}

	// No QR available
	if currentState == service.StateConnected {
		writeJSON(w, http.StatusOK, QRCodeResponse{
			State: currentState.String(),
			Error: "already authenticated",
		})
		return
	}

	// Return current state to help frontend understand what's happening
	writeJSON(w, http.StatusOK, QRCodeResponse{
		State: currentState.String(),
		Error: "QR code not ready yet, please wait...",
	})
}

// AuthInit handles POST /auth/init
func (h *Handlers) AuthInit(w http.ResponseWriter, r *http.Request) {
	state := h.manager.State()

	if state.State() == service.StateConnected {
		writeError(w, http.StatusBadRequest, "already authenticated", "ALREADY_AUTHENTICATED")
		return
	}

	// Start authentication in background with a fresh context (not tied to HTTP request)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		_ = h.manager.InitiateAuth(ctx)
	}()

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"message": "authentication initiated, poll GET /auth/qr for QR code",
		"state":   state.State().String(),
	})
}

// AuthLogout handles POST /auth/logout
func (h *Handlers) AuthLogout(w http.ResponseWriter, r *http.Request) {
	if err := h.manager.Logout(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "LOGOUT_FAILED")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "logged out successfully",
	})
}

// SendText handles POST /messages/text
func (h *Handlers) SendText(w http.ResponseWriter, r *http.Request) {
	var req SendTextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	if strings.TrimSpace(req.To) == "" {
		writeError(w, http.StatusBadRequest, "recipient 'to' is required", "MISSING_TO")
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		writeError(w, http.StatusBadRequest, "message is required", "MISSING_MESSAGE")
		return
	}

	msgID, err := h.manager.SendText(r.Context(), req.To, req.Message)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "SEND_FAILED")
		return
	}

	writeJSON(w, http.StatusOK, SendMessageResponse{
		Success:   true,
		MessageID: msgID,
		To:        req.To,
	})
}

// SendFile handles POST /messages/file
func (h *Handlers) SendFile(w http.ResponseWriter, r *http.Request) {
	var req SendFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	if strings.TrimSpace(req.To) == "" {
		writeError(w, http.StatusBadRequest, "recipient 'to' is required", "MISSING_TO")
		return
	}

	var data []byte
	var err error
	var filename string

	if req.FileData != "" {
		// Decode base64 data
		data, err = decodeBase64(req.FileData)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid base64 file_data", "INVALID_FILE_DATA")
			return
		}
		filename = req.Filename
		if filename == "" {
			filename = "file"
		}
	} else if req.FileURL != "" {
		// Download from URL
		data, filename, err = downloadFile(req.FileURL)
		if err != nil {
			writeError(w, http.StatusBadRequest, "failed to download file: "+err.Error(), "DOWNLOAD_FAILED")
			return
		}
		if req.Filename != "" {
			filename = req.Filename
		}
	} else {
		writeError(w, http.StatusBadRequest, "either file_data or file_url is required", "MISSING_FILE")
		return
	}

	result, err := h.manager.SendFile(r.Context(), req.To, data, filename, req.Caption, req.MimeType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "SEND_FAILED")
		return
	}

	writeJSON(w, http.StatusOK, SendFileResponse{
		Success:   true,
		MessageID: result.MessageID,
		To:        req.To,
		MediaType: result.MediaType,
		Filename:  result.Filename,
		MimeType:  result.MimeType,
	})
}

// Search handles GET /search
func (h *Handlers) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if strings.TrimSpace(query) == "" {
		writeError(w, http.StatusBadRequest, "query parameter 'q' is required", "MISSING_QUERY")
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 200 {
		limit = 200
	}

	messages, err := h.manager.SearchMessages(query, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "SEARCH_FAILED")
		return
	}

	resp := SearchResponse{
		Query:    query,
		Count:    len(messages),
		Messages: make([]MessageResponse, len(messages)),
	}
	for i, m := range messages {
		resp.Messages[i] = messageToResponse(m)
	}

	writeJSON(w, http.StatusOK, resp)
}

// ListChats handles GET /chats
func (h *Handlers) ListChats(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 200 {
		limit = 200
	}

	chats, err := h.manager.ListChats(query, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "LIST_CHATS_FAILED")
		return
	}

	resp := ChatsResponse{
		Count: len(chats),
		Chats: make([]ChatResponse, len(chats)),
	}
	for i, c := range chats {
		resp.Chats[i] = ChatResponse{
			JID:           c.JID,
			Kind:          c.Kind,
			Name:          c.Name,
			LastMessageTS: c.LastMessageTS,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// ListMessages handles GET /chats/{jid}/messages
func (h *Handlers) ListMessages(w http.ResponseWriter, r *http.Request) {
	// Extract chat JID from path
	path := strings.TrimPrefix(r.URL.Path, "/chats/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "messages" {
		writeError(w, http.StatusBadRequest, "invalid path", "INVALID_PATH")
		return
	}
	chatJID := parts[0]

	if strings.TrimSpace(chatJID) == "" {
		writeError(w, http.StatusBadRequest, "chat JID is required", "MISSING_JID")
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 200 {
		limit = 200
	}

	messages, err := h.manager.ListMessages(chatJID, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "LIST_MESSAGES_FAILED")
		return
	}

	resp := MessagesResponse{
		ChatJID:  chatJID,
		Count:    len(messages),
		Messages: make([]MessageResponse, len(messages)),
	}
	for i, m := range messages {
		resp.Messages[i] = messageToResponse(m)
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetMedia handles GET /media/{chat_jid}/{msg_id}
func (h *Handlers) GetMedia(w http.ResponseWriter, r *http.Request) {
	// Extract chat JID and msg ID from path
	path := strings.TrimPrefix(r.URL.Path, "/media/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		writeError(w, http.StatusBadRequest, "chat_jid and msg_id are required", "INVALID_PATH")
		return
	}
	chatJID := parts[0]
	msgID := parts[1]

	info, err := h.manager.GetMediaDownloadInfo(chatJID, msgID)
	if err != nil {
		if store.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "media not found", "NOT_FOUND")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error(), "GET_MEDIA_FAILED")
		return
	}

	// If already downloaded, serve the file
	if strings.TrimSpace(info.LocalPath) != "" {
		http.ServeFile(w, r, info.LocalPath)
		return
	}

	// Return media info for download
	writeJSON(w, http.StatusOK, MediaInfoResponse{
		ChatJID:      info.ChatJID,
		MsgID:        info.MsgID,
		MediaType:    info.MediaType,
		Filename:     info.Filename,
		MimeType:     info.MimeType,
		FileLength:   info.FileLength,
		Downloaded:   info.LocalPath != "",
		LocalPath:    info.LocalPath,
		DownloadedAt: info.DownloadedAt,
	})
}

// Stats handles GET /stats
func (h *Handlers) Stats(w http.ResponseWriter, r *http.Request) {
	a := h.manager.App()
	if a == nil {
		writeError(w, http.StatusServiceUnavailable, "app not initialized", "NOT_INITIALIZED")
		return
	}

	count, _ := a.DB().CountMessages()
	hasFTS := a.DB().HasFTS()

	writeJSON(w, http.StatusOK, StatsResponse{
		MessageCount: count,
		State:        h.manager.State().State().String(),
		HasFTS:       hasFTS,
	})
}

// NotFound handles 404 responses.
func (h *Handlers) NotFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "endpoint not found", "NOT_FOUND")
}

// MethodNotAllowed handles 405 responses.
func (h *Handlers) MethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusMethodNotAllowed, "method not allowed", "METHOD_NOT_ALLOWED")
}

// messageToResponse converts a store.Message to MessageResponse.
func messageToResponse(m store.Message) MessageResponse {
	return MessageResponse{
		ChatJID:   m.ChatJID,
		ChatName:  m.ChatName,
		MsgID:     m.MsgID,
		SenderJID: m.SenderJID,
		Timestamp: m.Timestamp,
		FromMe:    m.FromMe,
		Text:      m.Text,
		MediaType: m.MediaType,
		Snippet:   m.Snippet,
	}
}

// drainBody discards and closes the request body.
func drainBody(r *http.Request) {
	_, _ = io.Copy(io.Discard, r.Body)
	_ = r.Body.Close()
}

// Doctor handles GET /doctor
func (h *Handlers) Doctor(w http.ResponseWriter, r *http.Request) {
	storeDir, lockHeld, authenticated, connected := h.manager.GetDiagnostics()
	messageCount, chatCount, contactCount, groupCount, ftsEnabled, err := h.manager.GetDBStats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "DIAGNOSTICS_FAILED")
		return
	}

	writeJSON(w, http.StatusOK, DoctorResponse{
		StoreDir:      storeDir,
		LockHeld:      lockHeld,
		Authenticated: authenticated,
		Connected:     connected,
		FTSEnabled:    ftsEnabled,
		MessageCount:  messageCount,
		ChatCount:     chatCount,
		ContactCount:  contactCount,
		GroupCount:    groupCount,
	})
}

// SyncStatus handles GET /sync/status
func (h *Handlers) SyncStatus(w http.ResponseWriter, r *http.Request) {
	running, state, startedAt := h.manager.SyncStatus()

	writeJSON(w, http.StatusOK, SyncStatusResponse{
		Running:        running,
		State:          state,
		MessagesSynced: 0, // TODO: track message count
		StartedAt:      startedAt,
	})
}

// DownloadMedia handles POST /media/{chat_jid}/{msg_id}/download
func (h *Handlers) DownloadMedia(w http.ResponseWriter, r *http.Request) {
	// Extract chat JID and msg ID from path: /media/{chat_jid}/{msg_id}/download
	path := strings.TrimPrefix(r.URL.Path, "/media/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 || parts[2] != "download" {
		writeError(w, http.StatusBadRequest, "invalid path", "INVALID_PATH")
		return
	}
	chatJID := parts[0]
	msgID := parts[1]

	result, err := h.manager.DownloadMedia(r.Context(), chatJID, msgID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeError(w, http.StatusNotFound, err.Error(), "NOT_FOUND")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error(), "DOWNLOAD_FAILED")
		return
	}

	writeJSON(w, http.StatusOK, DownloadMediaResponse{
		Success:      true,
		ChatJID:      result.ChatJID,
		MsgID:        result.MsgID,
		MediaType:    result.MediaType,
		MimeType:     result.MimeType,
		LocalPath:    result.LocalPath,
		Bytes:        result.Bytes,
		DownloadedAt: result.DownloadedAt,
	})
}

// decodeBase64 decodes a base64 string, handling data URL prefixes.
func decodeBase64(s string) ([]byte, error) {
	// Strip data URL prefix if present (e.g., "data:image/png;base64,...")
	if idx := strings.Index(s, ","); idx != -1 && strings.Contains(s[:idx], "base64") {
		s = s[idx+1:]
	}
	return base64.StdEncoding.DecodeString(s)
}

// downloadFile downloads a file from a URL and returns its content and filename.
func downloadFile(url string) ([]byte, string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	// Extract filename from URL path
	filename := path.Base(url)
	if filename == "" || filename == "." || filename == "/" {
		filename = "file"
	}

	return data, filename, nil
}

// Backfill handles POST /history/backfill
func (h *Handlers) Backfill(w http.ResponseWriter, r *http.Request) {
	var req BackfillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	if strings.TrimSpace(req.ChatJID) == "" {
		writeError(w, http.StatusBadRequest, "chat_jid is required", "MISSING_CHAT_JID")
		return
	}

	// Set defaults
	count := req.Count
	if count <= 0 {
		count = 50
	}
	requests := req.Requests
	if requests <= 0 {
		requests = 1
	}
	waitSeconds := req.WaitPerRequestSeconds
	if waitSeconds <= 0 {
		waitSeconds = 60
	}

	result, err := h.manager.BackfillHistory(r.Context(), req.ChatJID, count, requests, waitSeconds)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "BACKFILL_FAILED")
		return
	}

	writeJSON(w, http.StatusOK, BackfillResponse{
		Success: true,
		JobID:   "", // Sync operation, no job ID
		Status:  "completed",
		Message: fmt.Sprintf("Added %d messages (%d requests sent)", result.MessagesAdded, result.RequestsSent),
	})
}

// StartSync handles POST /sync/start
func (h *Handlers) StartSync(w http.ResponseWriter, r *http.Request) {
	if err := h.manager.StartSync(r.Context()); err != nil {
		writeError(w, http.StatusConflict, err.Error(), "SYNC_START_FAILED")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "sync started",
	})
}

// StopSync handles POST /sync/stop
func (h *Handlers) StopSync(w http.ResponseWriter, r *http.Request) {
	if err := h.manager.StopSync(); err != nil {
		writeError(w, http.StatusConflict, err.Error(), "SYNC_STOP_FAILED")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "sync stopped",
	})
}
