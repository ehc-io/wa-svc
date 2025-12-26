# WhatsApp Service (wa-svc)

A high-performance WhatsApp integration platform that bridges the WhatsApp Web Multi-Device protocol to REST APIs and CLI tools. Built in Go, it provides message archiving, full-text search, media handling, and webhook notifications.

## Features

| Feature | Description |
|---------|-------------|
| **REST API** | Full HTTP API for messaging, contacts, groups, and media |
| **CLI Tool** | Command-line interface for all operations |
| **Message Sync** | Automatic real-time message capture and storage |
| **Full-Text Search** | SQLite FTS5 with BM25 ranking for instant search |
| **Media Support** | Send/receive images, videos, audio, and documents |
| **Webhooks** | Real-time event notifications with HMAC signing |
| **Contact Management** | Import contacts, set aliases, add tags |
| **Group Management** | Create, manage, and monitor group chats |
| **History Backfill** | Request older messages on demand |
| **Auto-Reconnect** | Automatic reconnection with exponential backoff |

## Quick Start

### Prerequisites

- Go 1.24+
- SQLite3 with FTS5 support (included in most distributions)

### Installation

```bash
# Clone the repository
git clone https://github.com/ehc-io/wa-svc
cd wa-svc

# Build the binaries
go build -o wasvc ./cmd/wasvc
go build -o wacli ./cmd/wacli
```

### Running the HTTP Server

```bash
# Set configuration
export WASVC_API_KEY="your-secret-api-key"
export WASVC_DATA_DIR="./data"

# Start the server
./wasvc
```

The server starts at `http://localhost:8080` by default.

### Authentication (First Time Setup)

1. **Start authentication flow:**
   ```bash
   curl -X POST http://localhost:8080/auth/init \
     -H "Authorization: Bearer your-secret-api-key"
   ```

2. **Get QR code:**
   ```bash
   curl http://localhost:8080/auth/qr \
     -H "Authorization: Bearer your-secret-api-key"
   ```

3. **Scan the QR code** with your WhatsApp mobile app (Settings → Linked Devices → Link a Device)

4. **Check status:**
   ```bash
   curl http://localhost:8080/auth/status \
     -H "Authorization: Bearer your-secret-api-key"
   ```

## Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `WASVC_HOST` | `0.0.0.0` | HTTP server host |
| `WASVC_PORT` | `8080` | HTTP server port |
| `WASVC_DATA_DIR` | `./data` | Data storage directory |
| `WASVC_API_KEY` | *(none)* | API key for authentication |
| `WASVC_WEBHOOK_URL` | *(none)* | Webhook endpoint URL |
| `WASVC_WEBHOOK_SECRET` | *(none)* | HMAC secret for webhook signing |
| `WA_DEBUG` | `false` | Enable verbose logging |

## API Overview

### Authentication
| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/auth/init` | Start QR authentication |
| `GET` | `/auth/qr` | Get QR code (base64 PNG) |
| `GET` | `/auth/status` | Check connection status |
| `POST` | `/auth/logout` | Disconnect and clear session |

### Messaging
| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/messages/text` | Send text message |
| `POST` | `/messages/file` | Send file/media message |
| `GET` | `/search` | Full-text search messages |

### Chats
| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/chats` | List recent chats |
| `GET` | `/chats/{jid}/messages` | Get chat messages |

### Contacts
| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/contacts` | Search contacts |
| `GET` | `/contacts/{jid}` | Get single contact |
| `POST` | `/contacts/refresh` | Import from WhatsApp |
| `PUT` | `/contacts/{jid}/alias` | Set contact alias |
| `POST` | `/contacts/{jid}/tags` | Add tag to contact |

### Groups
| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/groups` | List groups |
| `GET` | `/groups/{jid}` | Get group details |
| `POST` | `/groups/refresh` | Import from WhatsApp |
| `PUT` | `/groups/{jid}/name` | Rename group |
| `POST` | `/groups/{jid}/participants` | Add/remove members |
| `GET` | `/groups/{jid}/invite` | Get invite link |

### Media
| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/media/{chat}/{msg}` | Get media info or file |
| `POST` | `/media/{chat}/{msg}/download` | Download media |

### System
| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health check |
| `GET` | `/doctor` | Diagnostics |
| `GET` | `/stats` | Quick statistics |
| `POST` | `/history/backfill` | Request older messages |

## CLI Usage

```bash
# Authenticate
./wacli auth login

# Send a message
./wacli send text 1234567890 "Hello, World!"

# Send a file
./wacli send file 1234567890 photo.jpg --caption "Check this out!"

# Search messages
./wacli search "meeting tomorrow"

# List chats
./wacli chats list

# List contacts
./wacli contacts search john

# System diagnostics
./wacli doctor
```

## Practical Examples

### Sending Text Messages

**Via API:**
```bash
# Send a simple text message (use phone number without + or spaces)
curl -X POST http://localhost:8080/messages/text \
  -H "Authorization: Bearer your-secret-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "1234567890",
    "message": "Hello from WhatsApp Service!"
  }'

# Response
{
  "success": true,
  "msg_id": "3EB0C6C6F7F75F9C5B8E",
  "timestamp": "2025-12-26T10:30:00Z"
}
```

**Via CLI:**
```bash
# Send text message
./wacli send text 1234567890 "Hello, World!"

# Send to a group (use group JID)
./wacli send text 123456789-1234567890@g.us "Hello group!"
```

### Sending Images and Files

**Via API (multipart form):**
```bash
# Send an image with caption
curl -X POST http://localhost:8080/messages/file \
  -H "Authorization: Bearer your-secret-api-key" \
  -F "to=1234567890" \
  -F "file=@/path/to/photo.jpg" \
  -F "caption=Check out this photo!"

# Send a document
curl -X POST http://localhost:8080/messages/file \
  -H "Authorization: Bearer your-secret-api-key" \
  -F "to=1234567890" \
  -F "file=@/path/to/document.pdf" \
  -F "caption=Here is the report"

# Response
{
  "success": true,
  "msg_id": "3EB0C6C6F7F75F9C5B8F",
  "media_type": "image"
}
```

**Via CLI:**
```bash
# Send an image
./wacli send file 1234567890 photo.jpg --caption "Check this out!"

# Send a video
./wacli send file 1234567890 video.mp4 --caption "Watch this"

# Send a document
./wacli send file 1234567890 report.pdf
```

### Downloading Media

**Step 1: Get media info from a message:**
```bash
# Get media metadata for a specific message
curl http://localhost:8080/media/{chat_jid}/{msg_id} \
  -H "Authorization: Bearer your-secret-api-key"

# Response
{
  "msg_id": "3EB0C6C6F7F75F9C5B8E",
  "media_type": "image",
  "mimetype": "image/jpeg",
  "file_size": 245678,
  "filename": "photo.jpg",
  "downloaded": false
}
```

**Step 2: Download the media file:**
```bash
# Trigger download (saves to data/media/)
curl -X POST http://localhost:8080/media/{chat_jid}/{msg_id}/download \
  -H "Authorization: Bearer your-secret-api-key"

# Response
{
  "success": true,
  "local_path": "media/1234567890/3EB0C6C6F7F75F9C5B8E.jpg"
}

# Get the actual file (after download)
curl http://localhost:8080/media/{chat_jid}/{msg_id}?download=true \
  -H "Authorization: Bearer your-secret-api-key" \
  -o downloaded_photo.jpg
```

### Looking Up Contacts and Sending Messages

**Step 1: Search for a contact by name:**
```bash
# Search contacts by name
curl "http://localhost:8080/contacts?q=john" \
  -H "Authorization: Bearer your-secret-api-key"

# Response
{
  "contacts": [
    {
      "jid": "1234567890@s.whatsapp.net",
      "name": "John Smith",
      "phone": "1234567890",
      "alias": "Johnny"
    },
    {
      "jid": "0987654321@s.whatsapp.net",
      "name": "John Doe",
      "phone": "0987654321"
    }
  ]
}
```

**Step 2: Send a message to the found contact:**
```bash
# Use the phone number from the JID (without @s.whatsapp.net)
curl -X POST http://localhost:8080/messages/text \
  -H "Authorization: Bearer your-secret-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "1234567890",
    "message": "Hey John, how are you?"
  }'
```

**Complete workflow via CLI:**
```bash
# Search for contact
./wacli contacts search john

# Output:
# JID                          Name         Phone
# 1234567890@s.whatsapp.net    John Smith   1234567890
# 0987654321@s.whatsapp.net    John Doe     0987654321

# Send message using the phone number
./wacli send text 1234567890 "Hey John!"
```

**Set a custom alias for easier lookup:**
```bash
# Set an alias for a contact
curl -X PUT http://localhost:8080/contacts/1234567890@s.whatsapp.net/alias \
  -H "Authorization: Bearer your-secret-api-key" \
  -H "Content-Type: application/json" \
  -d '{"alias": "Johnny"}'

# Now search by alias
curl "http://localhost:8080/contacts?q=johnny" \
  -H "Authorization: Bearer your-secret-api-key"
```

### Searching Message History

```bash
# Full-text search across all messages
curl "http://localhost:8080/search?q=meeting+tomorrow&limit=20" \
  -H "Authorization: Bearer your-secret-api-key"

# Response
{
  "results": [
    {
      "chat_jid": "1234567890@s.whatsapp.net",
      "msg_id": "3EB0C6C6F7F75F9C5B8E",
      "sender_name": "John Smith",
      "text": "Let's have a meeting tomorrow at 10am",
      "timestamp": "2025-12-25T15:30:00Z",
      "snippet": "Let's have a <b>meeting</b> <b>tomorrow</b> at 10am"
    }
  ],
  "total": 1
}

# Get all messages from a specific chat
curl "http://localhost:8080/chats/1234567890@s.whatsapp.net/messages?limit=50" \
  -H "Authorization: Bearer your-secret-api-key"
```

## Webhooks

Configure webhooks to receive real-time notifications:

```bash
export WASVC_WEBHOOK_URL="https://your-app.com/webhook"
export WASVC_WEBHOOK_SECRET="your-hmac-secret"
```

### Event Format

```json
{
  "type": "message.received",
  "timestamp": "2025-12-26T10:30:00Z",
  "data": {
    "chat_jid": "1234567890@s.whatsapp.net",
    "msg_id": "3EB0C6C6F7F75F9C5B8E",
    "sender_jid": "1234567890@s.whatsapp.net",
    "text": "Hello!",
    "from_me": false
  }
}
```

### HMAC Verification (Node.js)

```javascript
const crypto = require('crypto');

function verifyWebhook(body, signature, secret) {
  const hmac = crypto.createHmac('sha256', secret);
  hmac.update(body);
  const expected = 'sha256=' + hmac.digest('hex');
  return crypto.timingSafeEqual(Buffer.from(signature), Buffer.from(expected));
}
```

## Docker

```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o wasvc ./cmd/wasvc

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/wasvc /usr/local/bin/
EXPOSE 8080
CMD ["wasvc"]
```

```yaml
# docker-compose.yml
version: '3.8'
services:
  wasvc:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./data:/data
    environment:
      - WASVC_DATA_DIR=/data
      - WASVC_API_KEY=${WASVC_API_KEY}
      - WASVC_WEBHOOK_URL=${WASVC_WEBHOOK_URL}
```

## Architecture

```
wa-svc/
├── cmd/
│   ├── wasvc/          # HTTP server
│   └── wacli/          # CLI tool
├── internal/
│   ├── api/            # HTTP handlers & middleware
│   ├── app/            # Business logic
│   ├── config/         # Configuration
│   ├── service/        # Service manager & state machine
│   ├── store/          # SQLite storage & FTS5
│   ├── wa/             # WhatsApp client wrapper
│   └── webhook/        # Webhook emitter
└── data/               # Runtime data (session, database, media)
```

## Documentation

See the `docs/` folder for detailed documentation:

- [Architecture](docs/01-ARCHITECTURE.md) - System design and components
- [API Reference](docs/02-API-REFERENCE.md) - Complete API documentation
- [Database Schema](docs/03-DATABASE-SCHEMA.md) - Database structure
- [Configuration](docs/05-CONFIGURATION.md) - All configuration options
- [Deployment](docs/06-DEPLOYMENT.md) - Production deployment guide
- [API Examples](docs/API-EXAMPLES.md) - Practical usage examples

## Security Notes

- Store `WASVC_API_KEY` securely (use secrets management in production)
- The `data/` directory contains session keys - never commit to version control
- Use HTTPS in production (reverse proxy with TLS)
- Enable webhook HMAC signing for secure event delivery

## License

MIT License - see [LICENSE](LICENSE) for details.
