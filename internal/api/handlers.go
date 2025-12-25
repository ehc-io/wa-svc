package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
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
