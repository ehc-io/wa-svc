package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// ListGroups handles GET /groups
func (h *Handlers) ListGroups(w http.ResponseWriter, r *http.Request) {
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

	groups, err := h.manager.ListGroups(query, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "LIST_GROUPS_FAILED")
		return
	}

	resp := GroupsResponse{
		Count:  len(groups),
		Groups: make([]GroupResponse, len(groups)),
	}
	for i, g := range groups {
		resp.Groups[i] = GroupResponse{
			JID:       g.JID,
			Name:      g.Name,
			OwnerJID:  g.OwnerJID,
			CreatedAt: g.CreatedAt,
			UpdatedAt: g.UpdatedAt,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// GetGroupInfo handles GET /groups/{jid}
func (h *Handlers) GetGroupInfo(w http.ResponseWriter, r *http.Request) {
	jid := strings.TrimPrefix(r.URL.Path, "/groups/")
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

	info, err := h.manager.GetGroupInfo(r.Context(), jid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "GET_GROUP_INFO_FAILED")
		return
	}

	participants := make([]GroupParticipant, len(info.Participants))
	for i, p := range info.Participants {
		role := ""
		if p.IsAdmin {
			role = "admin"
		}
		if p.IsSuperAdmin {
			role = "superadmin"
		}
		participants[i] = GroupParticipant{
			JID:  p.JID.String(),
			Role: role,
		}
	}

	ownerJID := ""
	if info.OwnerJID.User != "" {
		ownerJID = info.OwnerJID.String()
	}

	writeJSON(w, http.StatusOK, GroupInfoResponse{
		JID:              info.JID.String(),
		Name:             info.Name,
		OwnerJID:         ownerJID,
		CreatedAt:        info.GroupCreated,
		ParticipantCount: len(info.Participants),
		Participants:     participants,
	})
}

// RefreshGroups handles POST /groups/refresh
func (h *Handlers) RefreshGroups(w http.ResponseWriter, r *http.Request) {
	count, err := h.manager.RefreshGroups(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "REFRESH_GROUPS_FAILED")
		return
	}

	writeJSON(w, http.StatusOK, RefreshGroupsResponse{
		Success:        true,
		GroupsImported: count,
	})
}

// RenameGroup handles PUT /groups/{jid}/name
func (h *Handlers) RenameGroup(w http.ResponseWriter, r *http.Request) {
	// Extract JID from path: /groups/{jid}/name
	path := strings.TrimPrefix(r.URL.Path, "/groups/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "name" {
		writeError(w, http.StatusBadRequest, "invalid path", "INVALID_PATH")
		return
	}
	jid := parts[0]

	var req RenameGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required", "MISSING_NAME")
		return
	}

	if err := h.manager.RenameGroup(r.Context(), jid, req.Name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "RENAME_GROUP_FAILED")
		return
	}

	writeJSON(w, http.StatusOK, RenameGroupResponse{
		Success: true,
		JID:     jid,
		Name:    req.Name,
	})
}

// UpdateGroupParticipants handles POST /groups/{jid}/participants
func (h *Handlers) UpdateGroupParticipants(w http.ResponseWriter, r *http.Request) {
	// Extract JID from path: /groups/{jid}/participants
	path := strings.TrimPrefix(r.URL.Path, "/groups/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "participants" {
		writeError(w, http.StatusBadRequest, "invalid path", "INVALID_PATH")
		return
	}
	jid := parts[0]

	var req UpdateParticipantsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	if req.Action == "" {
		writeError(w, http.StatusBadRequest, "action is required", "MISSING_ACTION")
		return
	}
	if len(req.Users) == 0 {
		writeError(w, http.StatusBadRequest, "users are required", "MISSING_USERS")
		return
	}

	result, err := h.manager.UpdateGroupParticipants(r.Context(), jid, req.Users, req.Action)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "UPDATE_PARTICIPANTS_FAILED")
		return
	}

	participants := make([]GroupParticipant, len(result))
	for i, p := range result {
		errStr := ""
		if p.Error != 0 {
			errStr = strconv.Itoa(int(p.Error))
		}
		participants[i] = GroupParticipant{
			JID:   p.JID.String(),
			Error: errStr,
		}
	}

	writeJSON(w, http.StatusOK, UpdateParticipantsResponse{
		Success:      true,
		JID:          jid,
		Action:       req.Action,
		Participants: participants,
	})
}

// GetGroupInviteLink handles GET /groups/{jid}/invite
func (h *Handlers) GetGroupInviteLink(w http.ResponseWriter, r *http.Request) {
	// Extract JID from path: /groups/{jid}/invite
	path := strings.TrimPrefix(r.URL.Path, "/groups/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "invite" {
		writeError(w, http.StatusBadRequest, "invalid path", "INVALID_PATH")
		return
	}
	jid := parts[0]

	link, err := h.manager.GetGroupInviteLink(r.Context(), jid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "GET_INVITE_LINK_FAILED")
		return
	}

	writeJSON(w, http.StatusOK, InviteLinkResponse{
		JID:  jid,
		Link: "https://chat.whatsapp.com/" + link,
	})
}

// RevokeGroupInviteLink handles POST /groups/{jid}/invite/revoke
func (h *Handlers) RevokeGroupInviteLink(w http.ResponseWriter, r *http.Request) {
	// Extract JID from path: /groups/{jid}/invite/revoke
	path := strings.TrimPrefix(r.URL.Path, "/groups/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 || parts[1] != "invite" || parts[2] != "revoke" {
		writeError(w, http.StatusBadRequest, "invalid path", "INVALID_PATH")
		return
	}
	jid := parts[0]

	link, err := h.manager.RevokeGroupInviteLink(r.Context(), jid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "REVOKE_INVITE_LINK_FAILED")
		return
	}

	writeJSON(w, http.StatusOK, InviteLinkResponse{
		JID:  jid,
		Link: "https://chat.whatsapp.com/" + link,
	})
}

// JoinGroup handles POST /groups/join
func (h *Handlers) JoinGroup(w http.ResponseWriter, r *http.Request) {
	var req JoinGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST")
		return
	}

	if strings.TrimSpace(req.Code) == "" {
		writeError(w, http.StatusBadRequest, "code is required", "MISSING_CODE")
		return
	}

	jid, err := h.manager.JoinGroup(r.Context(), req.Code)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "JOIN_GROUP_FAILED")
		return
	}

	writeJSON(w, http.StatusOK, JoinGroupResponse{
		Success: true,
		JID:     jid,
	})
}

// LeaveGroup handles POST /groups/{jid}/leave
func (h *Handlers) LeaveGroup(w http.ResponseWriter, r *http.Request) {
	// Extract JID from path: /groups/{jid}/leave
	path := strings.TrimPrefix(r.URL.Path, "/groups/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "leave" {
		writeError(w, http.StatusBadRequest, "invalid path", "INVALID_PATH")
		return
	}
	jid := parts[0]

	if err := h.manager.LeaveGroup(r.Context(), jid); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), "LEAVE_GROUP_FAILED")
		return
	}

	writeJSON(w, http.StatusOK, LeaveGroupResponse{
		Success: true,
		JID:     jid,
	})
}
