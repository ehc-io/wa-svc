# WhatsApp Service - API Reference

Complete REST API documentation for the `wasvc` HTTP server.

## Table of Contents

- [Overview](#overview)
- [Authentication](#authentication)
- [Common Patterns](#common-patterns)
- [Health & Status](#health--status)
- [Authentication Endpoints](#authentication-endpoints)
- [Messaging Endpoints](#messaging-endpoints)
- [Search & Query](#search--query)
- [Chat Management](#chat-management)
- [Contact Management](#contact-management)
- [Group Management](#group-management)
- [Media Handling](#media-handling)
- [History & Sync](#history--sync)
- [Diagnostics](#diagnostics)
- [Error Codes](#error-codes)
- [Webhook Events](#webhook-events)

---

## Overview

### Base URL

```
http://localhost:8080
```

Configure via environment variables:
- `WASVC_HOST`: Host to bind (default: `0.0.0.0`)
- `WASVC_PORT`: Port to listen (default: `8080`)

### API Version

Current version: `wasvc/1.0`

All endpoints return JSON responses.

---

## Authentication

### API Key Authentication

**Two methods supported:**

1. **Bearer Token** (Recommended):
   ```http
   Authorization: Bearer your-secret-api-key
   ```

2. **Custom Header**:
   ```http
   X-API-Key: your-secret-api-key
   ```

### Configuration

Set via environment variable:
```bash
WASVC_API_KEY=your-secret-api-key
```

### Exempted Endpoints

No authentication required for:
- `GET /` (Web UI)
- `GET /health`
- `GET /healthz`

All other endpoints require authentication if `WASVC_API_KEY` is set.

---

## Common Patterns

### Response Format

**Success Response:**
```json
{
  "success": true,
  "data": { ... },
  "message": "optional message"
}
```

**Error Response:**
```json
{
  "error": "Human-readable error message",
  "code": "ERROR_CODE",
  "details": "Optional additional details"
}
```

### HTTP Status Codes

- `200 OK`: Successful request
- `201 Created`: Resource created
- `202 Accepted`: Request accepted for async processing
- `400 Bad Request`: Invalid request parameters
- `401 Unauthorized`: Missing or invalid API key
- `404 Not Found`: Resource not found
- `405 Method Not Allowed`: HTTP method not supported
- `409 Conflict`: Resource conflict (e.g., already authenticated)
- `500 Internal Server Error`: Server error

### Date/Time Format

All timestamps use ISO 8601 format with UTC timezone:
```
2025-12-26T10:30:00Z
```

### JID Format

WhatsApp Jabber IDs follow these formats:
- **User**: `1234567890@s.whatsapp.net`
- **Group**: `1234567890-1640000000@g.us`

Most endpoints accept either full JID or just the phone number (auto-converted to JID).

---

## Health & Status

### GET /health

Check service health and readiness.

**Request:**
```http
GET /health
```

**Response:** `200 OK`
```json
{
  "status": "ok",
  "state": "connected",
  "ready": true,
  "version": "wasvc/1.0",
  "timestamp": "2025-12-26T10:30:00Z"
}
```

**Fields:**
- `status`: `ok` (ready) or `degraded` (not ready)
- `state`: Connection state (see [States](#connection-states))
- `ready`: Boolean indicating if service can handle requests
- `version`: Service version
- `timestamp`: Current server time

**Connection States:**
- `disconnected`: Not connected to WhatsApp
- `connecting`: Attempting to connect
- `connected`: Connected and ready
- `unauthenticated`: Not authenticated (need QR scan)
- `error`: Error state

---

## Authentication Endpoints

### POST /auth/init

Initiate QR code authentication flow.

**Request:**
```http
POST /auth/init
Authorization: Bearer your-api-key
```

**Response:** `202 Accepted`
```json
{
  "message": "authentication initiated, poll GET /auth/qr for QR code",
  "state": "connecting"
}
```

**Error Responses:**
- `400 Bad Request`: Already authenticated
- `500 Internal Server Error`: Failed to start auth

**Flow:**
1. Call `POST /auth/init`
2. Poll `GET /auth/qr` until QR code available
3. Display QR code to user
4. User scans with WhatsApp mobile app
5. Poll `GET /auth/status` until `authenticated: true`

---

### GET /auth/qr

Retrieve the current QR code for scanning.

**Request:**
```http
GET /auth/qr
Authorization: Bearer your-api-key
```

**Response:** `200 OK`

**When QR Available:**
```json
{
  "qr_code": "1@ABC123XYZ...",
  "qr_image": "data:image/png;base64,iVBORw0KGgo...",
  "state": "connecting"
}
```

**When Already Authenticated:**
```json
{
  "state": "connected",
  "error": "already authenticated"
}
```

**When QR Not Ready:**
```json
{
  "state": "connecting",
  "error": "QR code not ready yet, please wait..."
}
```

**Fields:**
- `qr_code`: Raw QR code string (for custom rendering)
- `qr_image`: Base64-encoded PNG data URL (ready for `<img src="">`)
- `state`: Current connection state
- `error`: Error or status message if QR unavailable

**Usage Example (HTML):**
```html
<img src="{{ qr_image }}" alt="WhatsApp QR Code" />
```

---

### GET /auth/status

Check current authentication and connection status.

**Request:**
```http
GET /auth/status
Authorization: Bearer your-api-key
```

**Response:** `200 OK`
```json
{
  "state": "connected",
  "authenticated": true,
  "ready": true,
  "has_qr": false,
  "error": ""
}
```

**Fields:**
- `state`: Connection state string
- `authenticated`: True if authenticated with WhatsApp
- `ready`: True if service can handle requests
- `has_qr`: True if QR code is available
- `error`: Error message if any

**Polling Example:**
```javascript
const checkAuth = async () => {
  const res = await fetch('/auth/status', {
    headers: { 'Authorization': 'Bearer your-api-key' }
  });
  const data = await res.json();
  return data.authenticated;
};

// Poll every 2 seconds
const interval = setInterval(async () => {
  if (await checkAuth()) {
    clearInterval(interval);
    console.log('Authenticated!');
  }
}, 2000);
```

---

### POST /auth/logout

Disconnect and clear the current session.

**Request:**
```http
POST /auth/logout
Authorization: Bearer your-api-key
```

**Response:** `200 OK`
```json
{
  "message": "logged out successfully"
}
```

**Effect:**
- Disconnects from WhatsApp
- Clears session data
- Unlinks device from WhatsApp account
- Requires re-authentication via QR code

---

## Messaging Endpoints

### POST /messages/text

Send a text message.

**Request:**
```http
POST /messages/text
Authorization: Bearer your-api-key
Content-Type: application/json

{
  "to": "1234567890",
  "message": "Hello, World!"
}
```

**Request Body:**
```json
{
  "to": "1234567890",              // Phone number or full JID
  "message": "Your message here"   // Text content
}
```

**Response:** `200 OK`
```json
{
  "success": true,
  "message_id": "3EB0C6C6F7F75F9C5B8E",
  "to": "1234567890@s.whatsapp.net"
}
```

**Error Responses:**
- `400 Bad Request`: Missing `to` or `message`
- `500 Internal Server Error`: Send failed

**Notes:**
- Automatically stores sent message in database
- `message_id` can be used to track delivery (not implemented in v1)
- Supports Unicode and emojis

---

### POST /messages/file

Send a file/media message.

**Request:**
```http
POST /messages/file
Authorization: Bearer your-api-key
Content-Type: application/json

{
  "to": "1234567890",
  "file_data": "iVBORw0KGgo...",
  "filename": "photo.jpg",
  "caption": "Check out this photo!",
  "mime_type": "image/jpeg"
}
```

**Request Body:**
```json
{
  "to": "1234567890",                    // Required: recipient
  "file_data": "base64EncodedData",      // Option 1: Base64 file data
  "file_url": "https://example.com/file",// Option 2: URL to download
  "filename": "file.jpg",                 // Optional: filename
  "caption": "Optional caption",          // Optional: message caption
  "mime_type": "image/jpeg"               // Optional: MIME type
}
```

**Either `file_data` OR `file_url` must be provided.**

**Response:** `200 OK`
```json
{
  "success": true,
  "message_id": "3EB0C6C6F7F75F9C5B8E",
  "to": "1234567890@s.whatsapp.net",
  "media_type": "image",
  "filename": "photo.jpg",
  "mime_type": "image/jpeg"
}
```

**Media Types:**
- **image**: `image/*` MIME types
- **video**: `video/*` MIME types
- **audio**: `audio/*` MIME types
- **document**: Everything else

**Base64 Data URL Support:**
```json
{
  "file_data": "data:image/png;base64,iVBORw0KGgo..."
}
```
The prefix is automatically stripped.

**Size Limits:**
- Images: ~16 MB
- Videos: ~100 MB
- Documents: ~100 MB
- Audio: ~16 MB

**Notes:**
- Large files may take time to upload
- HTTP timeout is 5 minutes
- Files are encrypted by WhatsApp protocol

---

## Search & Query

### GET /search

Full-text search across all messages.

**Request:**
```http
GET /search?q=hello&limit=50
Authorization: Bearer your-api-key
```

**Query Parameters:**
- `q` (required): Search query
- `limit` (optional): Max results (default: 50, max: 200)

**Response:** `200 OK`
```json
{
  "query": "hello",
  "count": 2,
  "messages": [
    {
      "chat_jid": "1234567890@s.whatsapp.net",
      "chat_name": "John Doe",
      "msg_id": "3EB0C6C6F7F75F9C5B8E",
      "sender_jid": "1234567890@s.whatsapp.net",
      "timestamp": "2025-12-26T10:30:00Z",
      "from_me": false,
      "text": "Hello, how are you?",
      "media_type": "",
      "snippet": "... [Hello], how are you? ..."
    }
  ]
}
```

**Search Features:**

1. **FTS5 Full-Text Search** (if available):
   - Searches: message text, captions, filenames, chat names, sender names
   - BM25 ranking (relevance scoring)
   - Snippet generation with highlights `[...]`
   - Exact phrase: `"hello world"`
   - Boolean: `hello AND world`, `hello OR world`
   - Prefix: `hel*`

2. **Fallback LIKE Search** (if FTS5 unavailable):
   - Case-insensitive LIKE queries
   - Slower but functional
   - Warning issued on first use

**Search Query Examples:**
```
hello                  // Simple word search
"hello world"          // Exact phrase
hello world            // Both words (AND)
hello OR world         // Either word
hel*                   // Prefix search
```

**Response Fields:**
- `snippet`: Text excerpt with search term highlighted (FTS5 only)
- `chat_name`: Resolved chat/contact name
- `from_me`: True if sent by you

---

## Chat Management

### GET /chats

List recent chats.

**Request:**
```http
GET /chats?q=john&limit=50
Authorization: Bearer your-api-key
```

**Query Parameters:**
- `q` (optional): Filter by chat name or JID
- `limit` (optional): Max results (default: 50, max: 200)

**Response:** `200 OK`
```json
{
  "count": 2,
  "chats": [
    {
      "jid": "1234567890@s.whatsapp.net",
      "kind": "dm",
      "name": "John Doe",
      "last_message_ts": "2025-12-26T10:30:00Z"
    },
    {
      "jid": "1234567890-1640000000@g.us",
      "kind": "group",
      "name": "Project Team",
      "last_message_ts": "2025-12-26T09:15:00Z"
    }
  ]
}
```

**Chat Kinds:**
- `dm`: Direct message (1-on-1)
- `group`: Group chat
- `broadcast`: Broadcast list
- `unknown`: Unknown type

**Sorting:**
Chats are sorted by `last_message_ts` descending (most recent first).

---

### GET /chats/{jid}/messages

List messages from a specific chat.

**Request:**
```http
GET /chats/1234567890@s.whatsapp.net/messages?limit=100
Authorization: Bearer your-api-key
```

**Path Parameters:**
- `jid`: Chat JID (URL-encoded if contains special characters)

**Query Parameters:**
- `limit` (optional): Max results (default: 50, max: 200)

**Response:** `200 OK`
```json
{
  "chat_jid": "1234567890@s.whatsapp.net",
  "count": 2,
  "messages": [
    {
      "chat_jid": "1234567890@s.whatsapp.net",
      "chat_name": "John Doe",
      "msg_id": "3EB0C6C6F7F75F9C5B8E",
      "sender_jid": "1234567890@s.whatsapp.net",
      "timestamp": "2025-12-26T10:30:00Z",
      "from_me": false,
      "text": "Hello!",
      "media_type": "",
      "snippet": ""
    }
  ]
}
```

**Sorting:**
Messages are sorted by timestamp descending (most recent first).

---

## Contact Management

### GET /contacts

Search contacts in the local database.

**Request:**
```http
GET /contacts?q=john&limit=50
Authorization: Bearer your-api-key
```

**Query Parameters:**
- `q` (required): Search query (name, phone, JID)
- `limit` (optional): Max results (default: 50, max: 200)

**Response:** `200 OK`
```json
{
  "count": 1,
  "contacts": [
    {
      "jid": "1234567890@s.whatsapp.net",
      "phone": "1234567890",
      "name": "John Doe",
      "alias": "",
      "tags": ["client", "vip"],
      "updated_at": "2025-12-26T10:00:00Z"
    }
  ]
}
```

**Contact Name Priority:**
1. Local alias (if set)
2. Full name
3. Push name
4. Business name
5. First name
6. JID

---

### GET /contacts/{jid}

Get a single contact.

**Request:**
```http
GET /contacts/1234567890@s.whatsapp.net
Authorization: Bearer your-api-key
```

**Response:** `200 OK`
```json
{
  "jid": "1234567890@s.whatsapp.net",
  "phone": "1234567890",
  "name": "John Doe",
  "alias": "Johnny",
  "tags": ["client", "vip"],
  "updated_at": "2025-12-26T10:00:00Z"
}
```

**Error Responses:**
- `404 Not Found`: Contact not in database

---

### POST /contacts/refresh

Import contacts from WhatsApp.

**Request:**
```http
POST /contacts/refresh
Authorization: Bearer your-api-key
```

**Response:** `200 OK`
```json
{
  "success": true,
  "contacts_imported": 247
}
```

**Notes:**
- Fetches all contacts from WhatsApp
- Updates local database
- Preserves local aliases and tags
- Can be slow for many contacts (1-2 seconds)

---

### PUT /contacts/{jid}/alias

Set a local alias for a contact.

**Request:**
```http
PUT /contacts/1234567890@s.whatsapp.net/alias
Authorization: Bearer your-api-key
Content-Type: application/json

{
  "alias": "Johnny"
}
```

**Response:** `200 OK`
```json
{
  "success": true,
  "jid": "1234567890@s.whatsapp.net",
  "alias": "Johnny"
}
```

**Notes:**
- Local only (not synced to WhatsApp)
- Overrides display name in search results
- Empty string not allowed (use DELETE to remove)

---

### DELETE /contacts/{jid}/alias

Remove the local alias from a contact.

**Request:**
```http
DELETE /contacts/1234567890@s.whatsapp.net/alias
Authorization: Bearer your-api-key
```

**Response:** `200 OK`
```json
{
  "success": true,
  "jid": "1234567890@s.whatsapp.net"
}
```

---

### POST /contacts/{jid}/tags

Add a tag to a contact.

**Request:**
```http
POST /contacts/1234567890@s.whatsapp.net/tags
Authorization: Bearer your-api-key
Content-Type: application/json

{
  "tag": "vip"
}
```

**Response:** `200 OK`
```json
{
  "success": true,
  "jid": "1234567890@s.whatsapp.net",
  "tag": "vip"
}
```

**Notes:**
- Tags are local only
- Case-sensitive
- No spaces allowed
- Idempotent (adding existing tag is OK)

---

### DELETE /contacts/{jid}/tags

Remove a tag from a contact.

**Request:**
```http
DELETE /contacts/1234567890@s.whatsapp.net/tags
Authorization: Bearer your-api-key
Content-Type: application/json

{
  "tag": "vip"
}
```

**Response:** `200 OK`
```json
{
  "success": true,
  "jid": "1234567890@s.whatsapp.net",
  "tag": "vip"
}
```

---

## Group Management

### GET /groups

List groups from local database.

**Request:**
```http
GET /groups?q=project&limit=50
Authorization: Bearer your-api-key
```

**Query Parameters:**
- `q` (optional): Filter by group name or JID
- `limit` (optional): Max results (default: 50, max: 200)

**Response:** `200 OK`
```json
{
  "count": 1,
  "groups": [
    {
      "jid": "1234567890-1640000000@g.us",
      "name": "Project Team",
      "owner_jid": "9876543210@s.whatsapp.net",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2025-12-26T10:00:00Z"
    }
  ]
}
```

**Sorting:**
Groups sorted by creation date descending.

---

### POST /groups/refresh

Import joined groups from WhatsApp.

**Request:**
```http
POST /groups/refresh
Authorization: Bearer your-api-key
```

**Response:** `200 OK`
```json
{
  "success": true,
  "groups_imported": 12
}
```

**Notes:**
- Only imports groups you're a member of
- Updates local database
- Also updates participant lists

---

### GET /groups/{jid}

Get detailed group information (live from WhatsApp).

**Request:**
```http
GET /groups/1234567890-1640000000@g.us
Authorization: Bearer your-api-key
```

**Response:** `200 OK`
```json
{
  "jid": "1234567890-1640000000@g.us",
  "name": "Project Team",
  "owner_jid": "9876543210@s.whatsapp.net",
  "created_at": "2024-01-01T00:00:00Z",
  "participant_count": 15,
  "participants": [
    {
      "jid": "1234567890@s.whatsapp.net",
      "role": "admin",
      "error": ""
    },
    {
      "jid": "9876543210@s.whatsapp.net",
      "role": "superadmin",
      "error": ""
    }
  ]
}
```

**Participant Roles:**
- `` (empty): Regular member
- `admin`: Group admin
- `superadmin`: Group owner

---

### PUT /groups/{jid}/name

Rename a group.

**Request:**
```http
PUT /groups/1234567890-1640000000@g.us/name
Authorization: Bearer your-api-key
Content-Type: application/json

{
  "name": "New Project Name"
}
```

**Response:** `200 OK`
```json
{
  "success": true,
  "jid": "1234567890-1640000000@g.us",
  "name": "New Project Name"
}
```

**Error Responses:**
- `403 Forbidden`: Not an admin
- `404 Not Found`: Group not found

---

### POST /groups/{jid}/participants

Manage group participants (add, remove, promote, demote).

**Request:**
```http
POST /groups/1234567890-1640000000@g.us/participants
Authorization: Bearer your-api-key
Content-Type: application/json

{
  "action": "add",
  "users": ["1111111111", "2222222222"]
}
```

**Request Body:**
```json
{
  "action": "add|remove|promote|demote",
  "users": ["phone1", "phone2", "jid@s.whatsapp.net"]
}
```

**Actions:**
- `add`: Add users to group
- `remove`: Remove users from group
- `promote`: Make users admins
- `demote`: Remove admin privileges

**Response:** `200 OK`
```json
{
  "success": true,
  "jid": "1234567890-1640000000@g.us",
  "action": "add",
  "participants": [
    {
      "jid": "1111111111@s.whatsapp.net",
      "role": "",
      "error": ""
    },
    {
      "jid": "2222222222@s.whatsapp.net",
      "role": "",
      "error": "not in contacts"
    }
  ]
}
```

**Notes:**
- Requires admin privileges for all actions
- Errors per participant included in response
- Some users may fail while others succeed

---

### GET /groups/{jid}/invite

Get the group invite link.

**Request:**
```http
GET /groups/1234567890-1640000000@g.us/invite
Authorization: Bearer your-api-key
```

**Response:** `200 OK`
```json
{
  "jid": "1234567890-1640000000@g.us",
  "link": "https://chat.whatsapp.com/ABC123XYZ"
}
```

**Error Responses:**
- `403 Forbidden`: Not an admin

---

### POST /groups/{jid}/invite/revoke

Revoke and generate a new invite link.

**Request:**
```http
POST /groups/1234567890-1640000000@g.us/invite/revoke
Authorization: Bearer your-api-key
```

**Response:** `200 OK`
```json
{
  "jid": "1234567890-1640000000@g.us",
  "link": "https://chat.whatsapp.com/NEW123XYZ"
}
```

**Notes:**
- Old link becomes invalid
- Requires admin privileges

---

### POST /groups/join

Join a group via invite code.

**Request:**
```http
POST /groups/join
Authorization: Bearer your-api-key
Content-Type: application/json

{
  "code": "ABC123XYZ"
}
```

**Request Body:**
```json
{
  "code": "ABC123XYZ"  // From invite link
}
```

**Response:** `200 OK`
```json
{
  "success": true,
  "jid": "1234567890-1640000000@g.us"
}
```

**Error Responses:**
- `400 Bad Request`: Invalid code
- `403 Forbidden`: Not allowed to join

---

### POST /groups/{jid}/leave

Leave a group.

**Request:**
```http
POST /groups/1234567890-1640000000@g.us/leave
Authorization: Bearer your-api-key
```

**Response:** `200 OK`
```json
{
  "success": true,
  "jid": "1234567890-1640000000@g.us"
}
```

**Notes:**
- Permanent action
- Must re-join via invite if needed

---

## Media Handling

### GET /media/{chat_jid}/{msg_id}

Get media information or serve the file.

**Request:**
```http
GET /media/1234567890@s.whatsapp.net/3EB0C6C6F7F75F9C5B8E
Authorization: Bearer your-api-key
```

**Response (If Not Downloaded):** `200 OK`
```json
{
  "chat_jid": "1234567890@s.whatsapp.net",
  "msg_id": "3EB0C6C6F7F75F9C5B8E",
  "media_type": "image",
  "filename": "photo.jpg",
  "mime_type": "image/jpeg",
  "file_length": 524288,
  "downloaded": false,
  "local_path": "",
  "downloaded_at": null
}
```

**Response (If Downloaded):**
Serves the file directly with appropriate `Content-Type` header.

**Notes:**
- If `downloaded: false`, use POST endpoint to download
- If `downloaded: true`, returns the actual file content

---

### POST /media/{chat_jid}/{msg_id}/download

Download and decrypt media file.

**Request:**
```http
POST /media/1234567890@s.whatsapp.net/3EB0C6C6F7F75F9C5B8E/download
Authorization: Bearer your-api-key
```

**Response:** `200 OK`
```json
{
  "success": true,
  "chat_jid": "1234567890@s.whatsapp.net",
  "msg_id": "3EB0C6C6F7F75F9C5B8E",
  "media_type": "image",
  "mime_type": "image/jpeg",
  "local_path": "/data/media/1234567890@s.whatsapp.net/photo.jpg",
  "bytes": 524288,
  "downloaded_at": "2025-12-26T10:30:00Z"
}
```

**Error Responses:**
- `404 Not Found`: Message or media not found
- `400 Bad Request`: No downloadable media metadata
- `500 Internal Server Error`: Download failed

**Download Process:**
1. Fetch metadata from database
2. Download encrypted file from WhatsApp servers
3. Decrypt using stored media key
4. Verify SHA256 hashes
5. Save to local path
6. Update database with download info

**File Location:**
```
{WASVC_DATA_DIR}/media/{chat_jid}/{filename}
```

**Notes:**
- Idempotent: downloading twice returns existing file
- Large files may take time
- HTTP timeout: 5 minutes

---

## History & Sync

### POST /history/backfill

Request older messages for a specific chat.

**Request:**
```http
POST /history/backfill
Authorization: Bearer your-api-key
Content-Type: application/json

{
  "chat_jid": "1234567890@s.whatsapp.net",
  "count": 50,
  "requests": 2,
  "wait_per_request_seconds": 60
}
```

**Request Body:**
```json
{
  "chat_jid": "1234567890@s.whatsapp.net",  // Required
  "count": 50,                                // Optional: msgs per request
  "requests": 1,                              // Optional: number of requests
  "wait_per_request_seconds": 60              // Optional: wait between requests
}
```

**Response:** `200 OK`
```json
{
  "success": true,
  "job_id": "",
  "status": "completed",
  "message": "Added 87 messages (2 requests sent)"
}
```

**How It Works:**
1. Finds oldest message in local database for this chat
2. Sends history sync request to WhatsApp for older messages
3. Waits for history sync events
4. Repeats for `requests` times
5. Returns total messages added

**Notes:**
- Best-effort: WhatsApp may not return all requested messages
- Requires you to have message history on primary device
- Blocking operation (can take several minutes)
- Use sparingly to avoid rate limits

---

### GET /sync/status

Check sync worker status.

**Request:**
```http
GET /sync/status
Authorization: Bearer your-api-key
```

**Response:** `200 OK`
```json
{
  "running": true,
  "state": "connected",
  "messages_synced": 0,
  "started_at": "2025-12-26T10:00:00Z"
}
```

**Notes:**
- `running`: Sync worker is active
- `state`: Current connection state
- `started_at`: When sync worker started (if running)

---

### POST /sync/start

Manually start the sync worker.

**Request:**
```http
POST /sync/start
Authorization: Bearer your-api-key
```

**Response:** `200 OK`
```json
{
  "success": true,
  "message": "sync started"
}
```

**Error Responses:**
- `409 Conflict`: Sync already running

---

### POST /sync/stop

Stop the sync worker.

**Request:**
```http
POST /sync/stop
Authorization: Bearer your-api-key
```

**Response:** `200 OK`
```json
{
  "success": true,
  "message": "sync stopped"
}
```

**Error Responses:**
- `409 Conflict`: Sync not running

**Notes:**
- Disconnects from WhatsApp
- Stops receiving messages
- Does not clear authentication

---

## Diagnostics

### GET /doctor

System diagnostics and health check.

**Request:**
```http
GET /doctor
Authorization: Bearer your-api-key
```

**Response:** `200 OK`
```json
{
  "store_dir": "/data",
  "lock_held": true,
  "authenticated": true,
  "connected": true,
  "fts_enabled": true,
  "message_count": 12847,
  "chat_count": 156,
  "contact_count": 247,
  "group_count": 12
}
```

**Fields:**
- `store_dir`: Data directory location
- `lock_held`: File lock acquired successfully
- `authenticated`: Session authenticated with WhatsApp
- `connected`: Currently connected to WhatsApp
- `fts_enabled`: Full-text search available
- `message_count`: Total messages in database
- `chat_count`: Total chats tracked
- `contact_count`: Contacts in database
- `group_count`: Groups in database

**Usage:**
Quick health check and system overview. Useful for debugging and monitoring.

---

### GET /stats

Quick statistics.

**Request:**
```http
GET /stats
Authorization: Bearer your-api-key
```

**Response:** `200 OK`
```json
{
  "message_count": 12847,
  "state": "connected",
  "has_fts": true
}
```

---

## Error Codes

### Standard Error Codes

| Code | Meaning |
|------|---------|
| `INVALID_REQUEST` | Malformed request body |
| `MISSING_TO` | Recipient not specified |
| `MISSING_MESSAGE` | Message content not specified |
| `MISSING_QUERY` | Search query not specified |
| `MISSING_FILE` | File data/URL not specified |
| `INVALID_FILE_DATA` | Base64 decode failed |
| `DOWNLOAD_FAILED` | File download from URL failed |
| `SEND_FAILED` | Message send failed |
| `SEARCH_FAILED` | Search query failed |
| `NOT_FOUND` | Resource not found |
| `ALREADY_AUTHENTICATED` | Already authenticated |
| `LOGOUT_FAILED` | Logout failed |
| `NOT_INITIALIZED` | Service not initialized |
| `METHOD_NOT_ALLOWED` | HTTP method not allowed |
| `SYNC_START_FAILED` | Sync start failed |
| `SYNC_STOP_FAILED` | Sync stop failed |
| `BACKFILL_FAILED` | History backfill failed |
| `DIAGNOSTICS_FAILED` | Diagnostics query failed |

---

## Webhook Events

### Configuration

Set via environment variables:
```bash
WASVC_WEBHOOK_URL=https://your-app.com/webhook
WASVC_WEBHOOK_SECRET=your-secret-key
WASVC_WEBHOOK_RETRIES=3
WASVC_WEBHOOK_TIMEOUT=10s
```

### Event Format

All webhook events follow this structure:

```json
{
  "type": "message.received",
  "timestamp": "2025-12-26T10:30:00Z",
  "data": { ... }
}
```

### Event Types

#### message.received

Fired when a new message is received.

**Payload:**
```json
{
  "type": "message.received",
  "timestamp": "2025-12-26T10:30:00Z",
  "data": {
    "chat_jid": "1234567890@s.whatsapp.net",
    "chat_name": "John Doe",
    "msg_id": "3EB0C6C6F7F75F9C5B8E",
    "sender_jid": "1234567890@s.whatsapp.net",
    "sender_name": "John Doe",
    "timestamp": "2025-12-26T10:30:00Z",
    "from_me": false,
    "text": "Hello, how are you?",
    "media_type": "",
    "caption": ""
  }
}
```

**Notes:**
- Sent for both incoming and outgoing messages
- `from_me: true` indicates messages you sent
- `media_type`: empty for text, or "image", "video", "audio", "document"

### Webhook Security

**HMAC Signature Verification:**

If `WASVC_WEBHOOK_SECRET` is set, every webhook request includes:

**Header:**
```
X-Webhook-Signature: sha256=<hex_encoded_hmac>
```

**Verification (Node.js):**
```javascript
const crypto = require('crypto');

function verifyWebhook(body, signature, secret) {
  const hmac = crypto.createHmac('sha256', secret);
  hmac.update(body);
  const expected = 'sha256=' + hmac.digest('hex');
  return crypto.timingSafeEqual(
    Buffer.from(signature),
    Buffer.from(expected)
  );
}

// Express.js example
app.post('/webhook', express.raw({type: 'application/json'}), (req, res) => {
  const signature = req.headers['x-webhook-signature'];
  const secret = process.env.WEBHOOK_SECRET;

  if (!verifyWebhook(req.body, signature, secret)) {
    return res.status(401).send('Invalid signature');
  }

  const event = JSON.parse(req.body.toString());
  console.log('Event:', event);
  res.status(200).send('OK');
});
```

**Verification (Python):**
```python
import hmac
import hashlib

def verify_webhook(body: bytes, signature: str, secret: str) -> bool:
    expected = 'sha256=' + hmac.new(
        secret.encode(),
        body,
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(signature, expected)

# Flask example
@app.route('/webhook', methods=['POST'])
def webhook():
    signature = request.headers.get('X-Webhook-Signature')
    secret = os.environ['WEBHOOK_SECRET']

    if not verify_webhook(request.data, signature, secret):
        return 'Invalid signature', 401

    event = request.json
    print('Event:', event)
    return 'OK', 200
```

### Delivery Guarantees

- **At-most-once**: Events may be lost on failure
- **Retry Logic**: 3 retries with exponential backoff (1s, 2s, 4s, 8s, ...)
- **Max Backoff**: 30 seconds
- **Queue Size**: 1000 events (drops oldest if full)
- **Timeout**: 10 seconds per request (configurable)

### Error Handling

**Your webhook endpoint should:**
1. Return `200-299` status code on success
2. Process quickly (< 10 seconds)
3. Handle duplicate events (rare but possible)
4. Log failures for manual retry if needed

---

## Rate Limits

WhatsApp Service itself has no built-in rate limiting. However:

1. **WhatsApp Protocol Limits**:
   - ~40 messages per minute
   - Temporary bans for excessive messaging
   - Varies by account age and history

2. **Recommended Mitigation**:
   - Use external rate limiting (nginx, API gateway)
   - Implement application-level queuing
   - Monitor WhatsApp connection for disconnects

3. **Search Performance**:
   - FTS5 searches: sub-10ms typically
   - Large result sets: use pagination (`limit` parameter)

---

## Best Practices

### 1. Polling

**Don't poll excessively:**
```javascript
// BAD: Polling every 100ms
setInterval(() => fetch('/auth/status'), 100);

// GOOD: Polling every 2-5 seconds
setInterval(() => fetch('/auth/status'), 2000);
```

### 2. Error Handling

**Always check error responses:**
```javascript
const res = await fetch('/messages/text', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer ' + apiKey,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({ to: '1234567890', message: 'Hello' })
});

if (!res.ok) {
  const error = await res.json();
  console.error('Failed:', error.error, error.code);
  return;
}

const data = await res.json();
console.log('Success:', data.message_id);
```

### 3. Large Files

**Use streaming for large files:**
```javascript
// Avoid loading entire file into memory
const stream = fs.createReadStream('large-video.mp4');
const base64 = await streamToBase64(stream);

await fetch('/messages/file', {
  method: 'POST',
  headers: { 'Authorization': 'Bearer ' + apiKey },
  body: JSON.stringify({
    to: '1234567890',
    file_data: base64,
    filename: 'video.mp4'
  })
});
```

### 4. Webhooks

**Use webhooks instead of polling:**
```javascript
// BAD: Polling for new messages
setInterval(async () => {
  const res = await fetch('/search?q=*&limit=1');
  const data = await res.json();
  if (data.messages.length > lastCount) {
    console.log('New message!');
  }
}, 1000);

// GOOD: Use webhooks
app.post('/webhook', (req, res) => {
  if (req.body.type === 'message.received') {
    console.log('New message:', req.body.data);
  }
  res.send('OK');
});
```

---

## Examples

### Complete Authentication Flow

```javascript
const API_KEY = 'your-api-key';
const BASE_URL = 'http://localhost:8080';

async function authenticate() {
  // 1. Initiate auth
  await fetch(`${BASE_URL}/auth/init`, {
    method: 'POST',
    headers: { 'Authorization': `Bearer ${API_KEY}` }
  });

  // 2. Poll for QR code
  let qrCode = null;
  while (!qrCode) {
    const res = await fetch(`${BASE_URL}/auth/qr`, {
      headers: { 'Authorization': `Bearer ${API_KEY}` }
    });
    const data = await res.json();
    if (data.qr_image) {
      qrCode = data.qr_image;
      console.log('Scan this QR code:', qrCode);
    } else {
      await new Promise(r => setTimeout(r, 2000));
    }
  }

  // 3. Poll for authentication
  let authenticated = false;
  while (!authenticated) {
    const res = await fetch(`${BASE_URL}/auth/status`, {
      headers: { 'Authorization': `Bearer ${API_KEY}` }
    });
    const data = await res.json();
    authenticated = data.authenticated;
    if (!authenticated) {
      await new Promise(r => setTimeout(r, 2000));
    }
  }

  console.log('Authenticated!');
}

authenticate();
```

### Send Message with Error Handling

```javascript
async function sendMessage(to, message) {
  try {
    const res = await fetch(`${BASE_URL}/messages/text`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${API_KEY}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ to, message })
    });

    if (!res.ok) {
      const error = await res.json();
      throw new Error(`${error.code}: ${error.error}`);
    }

    const data = await res.json();
    return data.message_id;
  } catch (err) {
    console.error('Send failed:', err.message);
    throw err;
  }
}

sendMessage('1234567890', 'Hello, World!')
  .then(id => console.log('Sent:', id))
  .catch(err => console.error('Error:', err));
```

---

## Troubleshooting

### Authentication Not Working

**Symptom:** QR code never appears or authentication fails

**Solutions:**
1. Check service state: `GET /health`
2. Ensure not already authenticated: `GET /auth/status`
3. Check logs for errors (WA_DEBUG=true)
4. Restart service if stuck

### Messages Not Syncing

**Symptom:** New messages don't appear in database

**Solutions:**
1. Check connection: `GET /auth/status` (should be `authenticated: true`)
2. Check sync worker: `GET /sync/status` (should be `running: true`)
3. Restart sync: `POST /sync/stop` then `POST /sync/start`
4. Check WhatsApp on primary device (must be connected)

### Media Download Fails

**Symptom:** `POST /media/.../download` returns error

**Solutions:**
1. Verify message has media: `GET /media/...` (check `media_type`)
2. Check metadata: `direct_path` and `media_key` must not be empty
3. Ensure connected: `GET /auth/status`
4. Check network/firewall (downloads from WhatsApp CDN)

### Webhooks Not Delivered

**Symptom:** Webhook endpoint not receiving events

**Solutions:**
1. Verify URL is accessible from service
2. Check endpoint returns 200 status
3. Verify HMAC signature (if secret set)
4. Check service logs for webhook errors
5. Test with curl: `curl -X POST -H "Content-Type: application/json" -d '{"test":true}' YOUR_WEBHOOK_URL`

---

This API reference covers all available endpoints and their usage. For implementation details, see the [Architecture Documentation](01-ARCHITECTURE.md).
