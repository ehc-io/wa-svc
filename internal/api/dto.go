package api

import (
	"time"
)

// --- Request DTOs ---

// SendTextRequest is the request body for sending a text message.
type SendTextRequest struct {
	To      string `json:"to"`
	Message string `json:"message"`
}

// SendFileRequest is the request body for sending a file.
type SendFileRequest struct {
	To       string `json:"to"`
	FileURL  string `json:"file_url,omitempty"`
	FileData string `json:"file_data,omitempty"` // base64 encoded
	Filename string `json:"filename,omitempty"`
	Caption  string `json:"caption,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
}

// --- Response DTOs ---

// ErrorResponse is returned when an error occurs.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// HealthResponse is returned by the health check endpoint.
type HealthResponse struct {
	Status    string `json:"status"`
	State     string `json:"state"`
	Ready     bool   `json:"ready"`
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
}

// AuthStatusResponse is returned by the auth status endpoint.
type AuthStatusResponse struct {
	State         string `json:"state"`
	Authenticated bool   `json:"authenticated"`
	Ready         bool   `json:"ready"`
	HasQR         bool   `json:"has_qr"`
	Error         string `json:"error,omitempty"`
}

// QRCodeResponse is returned when requesting a QR code.
type QRCodeResponse struct {
	QRCode  string `json:"qr_code,omitempty"`
	QRImage string `json:"qr_image,omitempty"` // Base64 PNG data URL
	State   string `json:"state"`
	Error   string `json:"error,omitempty"`
}

// SendMessageResponse is returned after sending a message.
type SendMessageResponse struct {
	Success   bool   `json:"success"`
	MessageID string `json:"message_id"`
	To        string `json:"to"`
}

// MessageResponse represents a message in API responses.
type MessageResponse struct {
	ChatJID   string    `json:"chat_jid"`
	ChatName  string    `json:"chat_name"`
	MsgID     string    `json:"msg_id"`
	SenderJID string    `json:"sender_jid,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	FromMe    bool      `json:"from_me"`
	Text      string    `json:"text,omitempty"`
	MediaType string    `json:"media_type,omitempty"`
	Snippet   string    `json:"snippet,omitempty"`
}

// SearchResponse is returned by the search endpoint.
type SearchResponse struct {
	Query    string            `json:"query"`
	Count    int               `json:"count"`
	Messages []MessageResponse `json:"messages"`
}

// ChatResponse represents a chat in API responses.
type ChatResponse struct {
	JID           string    `json:"jid"`
	Kind          string    `json:"kind"`
	Name          string    `json:"name"`
	LastMessageTS time.Time `json:"last_message_ts,omitempty"`
}

// ChatsResponse is returned by the chats listing endpoint.
type ChatsResponse struct {
	Count int            `json:"count"`
	Chats []ChatResponse `json:"chats"`
}

// MessagesResponse is returned by the messages listing endpoint.
type MessagesResponse struct {
	ChatJID  string            `json:"chat_jid"`
	Count    int               `json:"count"`
	Messages []MessageResponse `json:"messages"`
}

// MediaInfoResponse is returned by the media info endpoint.
type MediaInfoResponse struct {
	ChatJID      string    `json:"chat_jid"`
	MsgID        string    `json:"msg_id"`
	MediaType    string    `json:"media_type"`
	Filename     string    `json:"filename,omitempty"`
	MimeType     string    `json:"mime_type,omitempty"`
	FileLength   uint64    `json:"file_length,omitempty"`
	Downloaded   bool      `json:"downloaded"`
	LocalPath    string    `json:"local_path,omitempty"`
	DownloadedAt time.Time `json:"downloaded_at,omitempty"`
}

// StatsResponse is returned by the stats endpoint.
type StatsResponse struct {
	MessageCount int64  `json:"message_count"`
	State        string `json:"state"`
	HasFTS       bool   `json:"has_fts"`
}
