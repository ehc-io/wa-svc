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

// --- Contact DTOs ---

// ContactResponse represents a contact in API responses.
type ContactResponse struct {
	JID       string    `json:"jid"`
	Phone     string    `json:"phone,omitempty"`
	Name      string    `json:"name,omitempty"`
	Alias     string    `json:"alias,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// ContactsResponse is returned by the contacts listing endpoint.
type ContactsResponse struct {
	Count    int               `json:"count"`
	Contacts []ContactResponse `json:"contacts"`
}

// RefreshContactsResponse is returned after refreshing contacts.
type RefreshContactsResponse struct {
	Success          bool `json:"success"`
	ContactsImported int  `json:"contacts_imported"`
}

// SetAliasRequest is the request body for setting a contact alias.
type SetAliasRequest struct {
	Alias string `json:"alias"`
}

// SetAliasResponse is returned after setting a contact alias.
type SetAliasResponse struct {
	Success bool   `json:"success"`
	JID     string `json:"jid"`
	Alias   string `json:"alias"`
}

// AddTagRequest is the request body for adding a tag to a contact.
type AddTagRequest struct {
	Tag string `json:"tag"`
}

// TagResponse is returned after modifying a contact tag.
type TagResponse struct {
	Success bool   `json:"success"`
	JID     string `json:"jid"`
	Tag     string `json:"tag"`
}

// --- Group DTOs ---

// GroupResponse represents a group in API responses.
type GroupResponse struct {
	JID        string    `json:"jid"`
	Name       string    `json:"name"`
	OwnerJID   string    `json:"owner_jid,omitempty"`
	CreatedAt  time.Time `json:"created_at,omitempty"`
	UpdatedAt  time.Time `json:"updated_at,omitempty"`
}

// GroupsResponse is returned by the groups listing endpoint.
type GroupsResponse struct {
	Count  int             `json:"count"`
	Groups []GroupResponse `json:"groups"`
}

// GroupParticipant represents a group participant.
type GroupParticipant struct {
	JID   string `json:"jid"`
	Role  string `json:"role,omitempty"` // "admin", "superadmin", or empty for regular
	Error string `json:"error,omitempty"`
}

// GroupInfoResponse is returned by the group info endpoint.
type GroupInfoResponse struct {
	JID              string             `json:"jid"`
	Name             string             `json:"name"`
	OwnerJID         string             `json:"owner_jid,omitempty"`
	CreatedAt        time.Time          `json:"created_at,omitempty"`
	ParticipantCount int                `json:"participant_count"`
	Participants     []GroupParticipant `json:"participants,omitempty"`
}

// RefreshGroupsResponse is returned after refreshing groups.
type RefreshGroupsResponse struct {
	Success        bool `json:"success"`
	GroupsImported int  `json:"groups_imported"`
}

// RenameGroupRequest is the request body for renaming a group.
type RenameGroupRequest struct {
	Name string `json:"name"`
}

// RenameGroupResponse is returned after renaming a group.
type RenameGroupResponse struct {
	Success bool   `json:"success"`
	JID     string `json:"jid"`
	Name    string `json:"name"`
}

// UpdateParticipantsRequest is the request body for managing group participants.
type UpdateParticipantsRequest struct {
	Action string   `json:"action"` // "add", "remove", "promote", "demote"
	Users  []string `json:"users"`  // JIDs or phone numbers
}

// UpdateParticipantsResponse is returned after updating group participants.
type UpdateParticipantsResponse struct {
	Success      bool               `json:"success"`
	JID          string             `json:"jid"`
	Action       string             `json:"action"`
	Participants []GroupParticipant `json:"participants"`
}

// InviteLinkResponse is returned when getting a group invite link.
type InviteLinkResponse struct {
	JID  string `json:"jid"`
	Link string `json:"link"`
}

// JoinGroupRequest is the request body for joining a group.
type JoinGroupRequest struct {
	Code string `json:"code"` // Invite code from link
}

// JoinGroupResponse is returned after joining a group.
type JoinGroupResponse struct {
	Success bool   `json:"success"`
	JID     string `json:"jid"`
}

// LeaveGroupResponse is returned after leaving a group.
type LeaveGroupResponse struct {
	Success bool   `json:"success"`
	JID     string `json:"jid"`
}

// --- Media DTOs ---

// DownloadMediaResponse is returned after downloading media.
type DownloadMediaResponse struct {
	Success      bool      `json:"success"`
	ChatJID      string    `json:"chat_jid"`
	MsgID        string    `json:"msg_id"`
	MediaType    string    `json:"media_type"`
	MimeType     string    `json:"mime_type,omitempty"`
	LocalPath    string    `json:"local_path"`
	Bytes        int64     `json:"bytes"`
	DownloadedAt time.Time `json:"downloaded_at"`
}

// SendFileResponse is returned after sending a file.
type SendFileResponse struct {
	Success   bool   `json:"success"`
	MessageID string `json:"message_id"`
	To        string `json:"to"`
	MediaType string `json:"media_type"`
	Filename  string `json:"filename,omitempty"`
	MimeType  string `json:"mime_type,omitempty"`
}

// --- History Backfill DTOs ---

// BackfillRequest is the request body for starting a history backfill.
type BackfillRequest struct {
	ChatJID               string `json:"chat_jid"`
	Count                 int    `json:"count,omitempty"`                    // Messages per request (default: 50)
	Requests              int    `json:"requests,omitempty"`                 // Number of requests (default: 1)
	WaitPerRequestSeconds int    `json:"wait_per_request_seconds,omitempty"` // Wait time between requests
}

// BackfillResponse is returned after starting a backfill job.
type BackfillResponse struct {
	Success bool   `json:"success"`
	JobID   string `json:"job_id"`
	Status  string `json:"status"` // "started", "running", "completed", "failed"
	Message string `json:"message,omitempty"`
}

// BackfillStatusResponse is returned when polling backfill job status.
type BackfillStatusResponse struct {
	JobID         string `json:"job_id"`
	Status        string `json:"status"` // "started", "running", "completed", "failed"
	ChatJID       string `json:"chat_jid"`
	RequestsSent  int    `json:"requests_sent"`
	ResponsesSeen int    `json:"responses_seen"`
	MessagesAdded int    `json:"messages_added"`
	Error         string `json:"error,omitempty"`
}

// --- Sync Control DTOs ---

// SyncStatusResponse is returned by the sync status endpoint.
type SyncStatusResponse struct {
	Running        bool      `json:"running"`
	State          string    `json:"state"`
	MessagesSynced int64     `json:"messages_synced"`
	StartedAt      time.Time `json:"started_at,omitempty"`
}

// StartSyncRequest is the request body for starting sync.
type StartSyncRequest struct {
	RefreshContacts bool `json:"refresh_contacts,omitempty"`
	RefreshGroups   bool `json:"refresh_groups,omitempty"`
	DownloadMedia   bool `json:"download_media,omitempty"`
}

// StartSyncResponse is returned after starting sync.
type StartSyncResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// StopSyncResponse is returned after stopping sync.
type StopSyncResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// --- Doctor/Diagnostics DTOs ---

// DoctorResponse is returned by the doctor endpoint.
type DoctorResponse struct {
	StoreDir      string `json:"store_dir"`
	LockHeld      bool   `json:"lock_held"`
	Authenticated bool   `json:"authenticated"`
	Connected     bool   `json:"connected"`
	FTSEnabled    bool   `json:"fts_enabled"`
	MessageCount  int64  `json:"message_count"`
	ChatCount     int64  `json:"chat_count"`
	ContactCount  int64  `json:"contact_count"`
	GroupCount    int64  `json:"group_count"`
}

// --- Message Context DTOs ---

// MessageContextResponse is returned when getting message context.
type MessageContextResponse struct {
	ChatJID     string            `json:"chat_jid"`
	TargetMsgID string            `json:"target_msg_id"`
	Messages    []MessageResponse `json:"messages"`
}
