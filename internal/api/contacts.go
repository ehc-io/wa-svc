package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// SearchContacts handles GET /contacts
func (h *Handlers) SearchContacts(w http.ResponseWriter, r *http.Request) {
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

	contacts, err := h.manager.SearchContacts(query, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "SEARCH_CONTACTS_FAILED")
		return
	}

	resp := ContactsResponse{
		Count:    len(contacts),
		Contacts: make([]ContactResponse, len(contacts)),
	}
	for i, c := range contacts {
		resp.Contacts[i] = ContactResponse{
			JID:       c.JID,
			Phone:     c.Phone,
			Name:      c.Name,
			Alias:     c.Alias,
			Tags:      c.Tags,
			UpdatedAt: c.UpdatedAt,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetContact handles GET /contacts/{jid}
func (h *Handlers) GetContact(w http.ResponseWriter, r *http.Request) {
	jid := strings.TrimPrefix(r.URL.Path, "/contacts/")
	jid = strings.TrimSuffix(jid, "/")

	if strings.TrimSpace(jid) == "" {
		writeError(w, http.StatusBadRequest, "JID is required", "MISSING_JID")
		return
	}

	// Check if this is a sub-resource path
	if strings.Contains(jid, "/") {
		writeError(w, http.StatusNotFound, "endpoint not found", "NOT_FOUND")
		return
	}

	contact, err := h.manager.GetContact(jid)
	if err != nil {
		writeError(w, http.StatusNotFound, "contact not found", "NOT_FOUND")
		return
	}

	writeJSON(w, http.StatusOK, ContactResponse{
		JID:       contact.JID,
		Phone:     contact.Phone,
		Name:      contact.Name,
		Alias:     contact.Alias,
		Tags:      contact.Tags,
		UpdatedAt: contact.UpdatedAt,
	})
}

// RefreshContacts handles POST /contacts/refresh
func (h *Handlers) RefreshContacts(w http.ResponseWriter, r *http.Request) {
	count, err := h.manager.RefreshContacts(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "REFRESH_CONTACTS_FAILED")
		return
	}

	writeJSON(w, http.StatusOK, RefreshContactsResponse{
		Success:          true,
		ContactsImported: count,
	})
}

// SetContactAlias handles PUT /contacts/{jid}/alias
func (h *Handlers) SetContactAlias(w http.ResponseWriter, r *http.Request) {
	// Extract JID from path: /contacts/{jid}/alias
	path := strings.TrimPrefix(r.URL.Path, "/contacts/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "alias" {
		writeError(w, http.StatusBadRequest, "invalid path", "INVALID_PATH")
		return
	}
	jid := parts[0]

	var req SetAliasRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	if strings.TrimSpace(req.Alias) == "" {
		writeError(w, http.StatusBadRequest, "alias is required", "MISSING_ALIAS")
		return
	}

	if err := h.manager.SetContactAlias(jid, req.Alias); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "SET_ALIAS_FAILED")
		return
	}

	writeJSON(w, http.StatusOK, SetAliasResponse{
		Success: true,
		JID:     jid,
		Alias:   req.Alias,
	})
}

// DeleteContactAlias handles DELETE /contacts/{jid}/alias
func (h *Handlers) DeleteContactAlias(w http.ResponseWriter, r *http.Request) {
	// Extract JID from path: /contacts/{jid}/alias
	path := strings.TrimPrefix(r.URL.Path, "/contacts/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "alias" {
		writeError(w, http.StatusBadRequest, "invalid path", "INVALID_PATH")
		return
	}
	jid := parts[0]

	if err := h.manager.RemoveContactAlias(jid); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "DELETE_ALIAS_FAILED")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"jid":     jid,
	})
}

// AddContactTag handles POST /contacts/{jid}/tags
func (h *Handlers) AddContactTag(w http.ResponseWriter, r *http.Request) {
	// Extract JID from path: /contacts/{jid}/tags
	path := strings.TrimPrefix(r.URL.Path, "/contacts/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "tags" {
		writeError(w, http.StatusBadRequest, "invalid path", "INVALID_PATH")
		return
	}
	jid := parts[0]

	var req AddTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	if strings.TrimSpace(req.Tag) == "" {
		writeError(w, http.StatusBadRequest, "tag is required", "MISSING_TAG")
		return
	}

	if err := h.manager.AddContactTag(jid, req.Tag); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "ADD_TAG_FAILED")
		return
	}

	writeJSON(w, http.StatusOK, TagResponse{
		Success: true,
		JID:     jid,
		Tag:     req.Tag,
	})
}

// DeleteContactTag handles DELETE /contacts/{jid}/tags/{tag}
func (h *Handlers) DeleteContactTag(w http.ResponseWriter, r *http.Request) {
	// Extract JID and tag from path: /contacts/{jid}/tags/{tag}
	path := strings.TrimPrefix(r.URL.Path, "/contacts/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 || parts[1] != "tags" {
		writeError(w, http.StatusBadRequest, "invalid path", "INVALID_PATH")
		return
	}
	jid := parts[0]
	tag := parts[2]

	if strings.TrimSpace(tag) == "" {
		writeError(w, http.StatusBadRequest, "tag is required", "MISSING_TAG")
		return
	}

	if err := h.manager.RemoveContactTag(jid, tag); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "DELETE_TAG_FAILED")
		return
	}

	writeJSON(w, http.StatusOK, TagResponse{
		Success: true,
		JID:     jid,
		Tag:     tag,
	})
}
