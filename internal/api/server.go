package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/steipete/wacli/internal/service"
)

// Server is the HTTP API server.
type Server struct {
	config   service.Config
	manager  *service.Manager
	handlers *Handlers
	server   *http.Server
}

// NewServer creates a new API server.
func NewServer(cfg service.Config, mgr *service.Manager) *Server {
	handlers := NewHandlers(mgr)

	mux := http.NewServeMux()

	// Web UI for authentication (no auth required)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Only serve auth page for exact root path
		if r.URL.Path != "/" {
			handlers.NotFound(w, r)
			return
		}
		handlers.AuthPage(w, r)
	})

	// Health endpoints (no auth required)
	mux.HandleFunc("/health", handlers.Health)
	mux.HandleFunc("/healthz", handlers.Health)

	// Auth endpoints
	mux.HandleFunc("/auth/status", handlers.AuthStatus)
	mux.HandleFunc("/auth/qr", handlers.AuthQR)
	mux.HandleFunc("/auth/init", methodHandler(http.MethodPost, handlers.AuthInit))
	mux.HandleFunc("/auth/logout", methodHandler(http.MethodPost, handlers.AuthLogout))

	// Message endpoints
	mux.HandleFunc("/messages/text", methodHandler(http.MethodPost, handlers.SendText))
	mux.HandleFunc("/messages/file", methodHandler(http.MethodPost, handlers.SendFile))

	// Search endpoint
	mux.HandleFunc("/search", methodHandler(http.MethodGet, handlers.Search))

	// Chats endpoints
	mux.HandleFunc("/chats", methodHandler(http.MethodGet, handlers.ListChats))
	mux.HandleFunc("/chats/", chatMessagesHandler(handlers))

	// Media endpoint
	mux.HandleFunc("/media/", mediaHandler(handlers))

	// Stats endpoint
	mux.HandleFunc("/stats", methodHandler(http.MethodGet, handlers.Stats))

	// Contacts endpoints
	mux.HandleFunc("/contacts", methodHandler(http.MethodGet, handlers.SearchContacts))
	mux.HandleFunc("/contacts/refresh", methodHandler(http.MethodPost, handlers.RefreshContacts))
	mux.HandleFunc("/contacts/", contactsHandler(handlers))

	// Groups endpoints
	mux.HandleFunc("/groups", methodHandler(http.MethodGet, handlers.ListGroups))
	mux.HandleFunc("/groups/refresh", methodHandler(http.MethodPost, handlers.RefreshGroups))
	mux.HandleFunc("/groups/join", methodHandler(http.MethodPost, handlers.JoinGroup))
	mux.HandleFunc("/groups/", groupsHandler(handlers))

	// Sync control endpoints
	mux.HandleFunc("/sync/status", methodHandler(http.MethodGet, handlers.SyncStatus))
	mux.HandleFunc("/sync/start", methodHandler(http.MethodPost, handlers.StartSync))
	mux.HandleFunc("/sync/stop", methodHandler(http.MethodPost, handlers.StopSync))

	// History backfill endpoint
	mux.HandleFunc("/history/backfill", methodHandler(http.MethodPost, handlers.Backfill))

	// Doctor/diagnostics endpoint
	mux.HandleFunc("/doctor", methodHandler(http.MethodGet, handlers.Doctor))

	// Apply middleware
	handler := ChainMiddleware(
		mux,
		LoggingMiddleware,
		RecoveryMiddleware,
		CORSMiddleware,
		ContentTypeMiddleware,
		func(next http.Handler) http.Handler {
			return APIKeyMiddleware(cfg.APIKey, next)
		},
	)

	server := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return &Server{
		config:   cfg,
		manager:  mgr,
		handlers: handlers,
		server:   server,
	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	log.Printf("[API] Starting server on %s", s.config.Addr())
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("[API] Shutting down server...")
	return s.server.Shutdown(ctx)
}

// Addr returns the server address.
func (s *Server) Addr() string {
	return s.config.Addr()
}

// methodHandler restricts an endpoint to a specific HTTP method.
func methodHandler(method string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method != method {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed", "METHOD_NOT_ALLOWED")
			return
		}
		handler(w, r)
	}
}

// chatMessagesHandler handles /chats/{jid}/messages routes.
func chatMessagesHandler(h *Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/chats/")
		if strings.Contains(path, "/messages") {
			if r.Method != http.MethodGet {
				writeError(w, http.StatusMethodNotAllowed, "method not allowed", "METHOD_NOT_ALLOWED")
				return
			}
			h.ListMessages(w, r)
			return
		}
		writeError(w, http.StatusNotFound, "endpoint not found", "NOT_FOUND")
	}
}

// contactsHandler handles /contacts/{jid}/* routes.
func contactsHandler(h *Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/contacts/")
		parts := strings.Split(path, "/")

		// /contacts/{jid}
		if len(parts) == 1 && parts[0] != "" {
			if r.Method != http.MethodGet {
				writeError(w, http.StatusMethodNotAllowed, "method not allowed", "METHOD_NOT_ALLOWED")
				return
			}
			h.GetContact(w, r)
			return
		}

		// /contacts/{jid}/alias
		if len(parts) >= 2 && parts[1] == "alias" {
			switch r.Method {
			case http.MethodPut:
				h.SetContactAlias(w, r)
			case http.MethodDelete:
				h.DeleteContactAlias(w, r)
			default:
				writeError(w, http.StatusMethodNotAllowed, "method not allowed", "METHOD_NOT_ALLOWED")
			}
			return
		}

		// /contacts/{jid}/tags or /contacts/{jid}/tags/{tag}
		if len(parts) >= 2 && parts[1] == "tags" {
			switch r.Method {
			case http.MethodPost:
				h.AddContactTag(w, r)
			case http.MethodDelete:
				h.DeleteContactTag(w, r)
			default:
				writeError(w, http.StatusMethodNotAllowed, "method not allowed", "METHOD_NOT_ALLOWED")
			}
			return
		}

		writeError(w, http.StatusNotFound, "endpoint not found", "NOT_FOUND")
	}
}

// groupsHandler handles /groups/{jid}/* routes.
func groupsHandler(h *Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/groups/")
		parts := strings.Split(path, "/")

		// /groups/{jid}
		if len(parts) == 1 && parts[0] != "" {
			if r.Method != http.MethodGet {
				writeError(w, http.StatusMethodNotAllowed, "method not allowed", "METHOD_NOT_ALLOWED")
				return
			}
			h.GetGroupInfo(w, r)
			return
		}

		// /groups/{jid}/name
		if len(parts) >= 2 && parts[1] == "name" {
			if r.Method != http.MethodPut {
				writeError(w, http.StatusMethodNotAllowed, "method not allowed", "METHOD_NOT_ALLOWED")
				return
			}
			h.RenameGroup(w, r)
			return
		}

		// /groups/{jid}/participants
		if len(parts) >= 2 && parts[1] == "participants" {
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "method not allowed", "METHOD_NOT_ALLOWED")
				return
			}
			h.UpdateGroupParticipants(w, r)
			return
		}

		// /groups/{jid}/invite or /groups/{jid}/invite/revoke
		if len(parts) >= 2 && parts[1] == "invite" {
			if len(parts) >= 3 && parts[2] == "revoke" {
				if r.Method != http.MethodPost {
					writeError(w, http.StatusMethodNotAllowed, "method not allowed", "METHOD_NOT_ALLOWED")
					return
				}
				h.RevokeGroupInviteLink(w, r)
				return
			}
			if r.Method != http.MethodGet {
				writeError(w, http.StatusMethodNotAllowed, "method not allowed", "METHOD_NOT_ALLOWED")
				return
			}
			h.GetGroupInviteLink(w, r)
			return
		}

		// /groups/{jid}/leave
		if len(parts) >= 2 && parts[1] == "leave" {
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "method not allowed", "METHOD_NOT_ALLOWED")
				return
			}
			h.LeaveGroup(w, r)
			return
		}

		writeError(w, http.StatusNotFound, "endpoint not found", "NOT_FOUND")
	}
}

// mediaHandler handles /media/{chat_jid}/{msg_id}/* routes.
func mediaHandler(h *Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/media/")
		parts := strings.Split(path, "/")

		// /media/{chat_jid}/{msg_id}/download
		if len(parts) >= 3 && parts[2] == "download" {
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "method not allowed", "METHOD_NOT_ALLOWED")
				return
			}
			h.DownloadMedia(w, r)
			return
		}

		// /media/{chat_jid}/{msg_id} - GET media info or serve file
		if len(parts) >= 2 {
			if r.Method != http.MethodGet {
				writeError(w, http.StatusMethodNotAllowed, "method not allowed", "METHOD_NOT_ALLOWED")
				return
			}
			h.GetMedia(w, r)
			return
		}

		writeError(w, http.StatusNotFound, "endpoint not found", "NOT_FOUND")
	}
}
