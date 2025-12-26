# WhatsApp Service - Technical Documentation

Complete technical documentation for the WhatsApp Service integration platform.

## Overview

WhatsApp Service (wa-svc) is a production-ready, high-performance integration platform that bridges WhatsApp Web Multi-Device protocol to REST APIs and command-line interfaces. Built in Go 1.24, it provides enterprise-grade message archiving, automated notifications, full-text search, and comprehensive management capabilities.

### Key Features

- **Dual Interface**: RESTful HTTP API (`wasvc`) + CLI tool (`wacli`)
- **Continuous Sync**: Automatic message history capture and real-time synchronization
- **Custom Device Identity**: Appears as "WhatsApp-SVC" in linked devices
- **Rich Media**: Full support for images, videos, audio, and documents with automated metadata capture
- **Powerful Search**: SQLite FTS5 with BM25 ranking for ultra-fast offline search
- **Real-time Webhooks**: Event-driven architecture with HMAC signing and retry logic
- **Production Ready**: File locking, graceful shutdown, auto-reconnection, health checks

### Technology Stack

- **Language**: Go 1.24+
- **Protocol**: whatsmeow (WhatsApp Multi-Device)
- **Database**: SQLite3 with FTS5 Full-Text Search
- **Deployment**: Docker & Docker Compose
- **Architecture**: Service-oriented with clean separation of concerns

---

## Documentation Structure

### Core Documentation

1. **[00-INDEX.md](00-INDEX.md)** - Documentation index and navigation
2. **[01-ARCHITECTURE.md](01-ARCHITECTURE.md)** - System architecture and design decisions
   - Executive summary
   - High-level architecture diagrams
   - Component interaction flows
   - Core components deep dive
   - Data flow & integration points
   - Design decisions & rationale
   - Performance characteristics
   - Security considerations

3. **[02-API-REFERENCE.md](02-API-REFERENCE.md)** - Complete REST API documentation
   - All endpoints with request/response examples
   - Authentication methods
   - Error codes and handling
   - Webhook configuration
   - Search query syntax
   - Best practices and examples

4. **[03-DATABASE-SCHEMA.md](03-DATABASE-SCHEMA.md)** - SQLite schema and data models
   - Complete table definitions
   - Full-Text Search (FTS5) implementation
   - Indexes and performance optimization
   - Triggers and constraints
   - Query patterns
   - Migration strategy

5. **[05-CONFIGURATION.md](05-CONFIGURATION.md)** - Environment variables and settings
   - All configuration options explained
   - Server, webhook, and sync settings
   - Security best practices
   - Environment templates (dev/staging/prod)
   - Troubleshooting common issues

6. **[06-DEPLOYMENT.md](06-DEPLOYMENT.md)** - Production deployment guide
   - Docker and Docker Compose setup
   - Reverse proxy configuration (nginx)
   - Health checks and monitoring
   - Backup and recovery procedures
   - Scaling considerations
   - Production checklist

---

## Quick Start

### 1. Installation

```bash
# Clone repository
git clone https://github.com/steipete/wacli.git
cd wacli

# Configure environment
cp .env.example .env
# Edit .env and set WASVC_API_KEY
```

### 2. Deploy with Docker

```bash
# Build and start
docker compose up -d

# View logs
docker compose logs -f
```

### 3. Authenticate

```bash
# Open web UI
open http://localhost:8080

# Or use API
curl -X POST http://localhost:8080/auth/init \
  -H "Authorization: Bearer your-api-key"

# Get QR code
curl http://localhost:8080/auth/qr \
  -H "Authorization: Bearer your-api-key"
```

Scan the QR code with your WhatsApp mobile app.

### 4. Use the API

```bash
# Check status
curl http://localhost:8080/auth/status \
  -H "Authorization: Bearer your-api-key"

# Send a message
curl -X POST http://localhost:8080/messages/text \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "1234567890",
    "message": "Hello from WhatsApp Service!"
  }'

# Search messages
curl "http://localhost:8080/search?q=hello&limit=10" \
  -H "Authorization: Bearer your-api-key"
```

---

## Architecture Highlights

### System Components

```
┌─────────────────────────────────────────────────────────────┐
│                    Client Applications                       │
└──────────────┬─────────────────────────┬────────────────────┘
               │ REST API                │ CLI
   ┌───────────▼──────────┐   ┌──────────▼─────────┐
   │  wasvc (HTTP Server) │   │  wacli (CLI Tool)  │
   └───────────┬──────────┘   └──────────┬─────────┘
               │                         │
               └────────┬────────────────┘
                        │
            ┌───────────▼────────────┐
            │   Service Manager      │
            │  - State Machine       │
            │  - Event Handling      │
            │  - Sync Worker         │
            └───────────┬────────────┘
                        │
       ┌────────────────┼────────────────┐
       │                │                │
┌──────▼─────┐  ┌───────▼──────┐  ┌─────▼────┐
│  App Layer │  │  WA Client   │  │ Webhooks │
└──────┬─────┘  └───────┬──────┘  └──────────┘
       │                │
       │                │
  ┌────▼────────────────▼─────┐
  │    Storage Layer          │
  │  - SQLite (wacli.db)      │
  │  - Session (session.db)   │
  │  - FTS5 Search Index      │
  │  - Media Files            │
  └───────────────────────────┘
```

### Key Design Decisions

1. **Dual Database Strategy**: Separate application data (wacli.db) from WhatsApp protocol data (session.db)
2. **WAL Mode**: Write-Ahead Logging for better concurrency and performance
3. **State Machine**: Centralized state management for reliable connection handling
4. **Upsert Pattern**: Idempotent message storage prevents duplicates during history sync
5. **FTS5 Search**: Fast full-text search with BM25 ranking
6. **Webhook Queue**: Asynchronous event delivery with retry logic
7. **File Locking**: Prevents concurrent access to prevent session conflicts

---

## API Highlights

### Core Endpoints

| Category | Endpoint | Method | Description |
|----------|----------|--------|-------------|
| **Auth** | `/auth/init` | POST | Initiate QR authentication |
| | `/auth/qr` | GET | Get QR code |
| | `/auth/status` | GET | Check auth status |
| | `/auth/logout` | POST | Disconnect session |
| **Messages** | `/messages/text` | POST | Send text message |
| | `/messages/file` | POST | Send file/media |
| | `/search` | GET | Full-text search |
| | `/chats/{jid}/messages` | GET | List messages in chat |
| **Contacts** | `/contacts` | GET | Search contacts |
| | `/contacts/refresh` | POST | Import from WhatsApp |
| | `/contacts/{jid}/alias` | PUT | Set local alias |
| **Groups** | `/groups` | GET | List groups |
| | `/groups/{jid}` | GET | Get group info |
| | `/groups/{jid}/participants` | POST | Manage members |
| | `/groups/{jid}/invite` | GET | Get invite link |
| **Media** | `/media/{chat}/{msg}` | GET | Get media info |
| | `/media/{chat}/{msg}/download` | POST | Download media |
| **Sync** | `/sync/status` | GET | Check sync status |
| | `/history/backfill` | POST | Request older messages |
| **Health** | `/health` | GET | Service health check |
| | `/doctor` | GET | Detailed diagnostics |

### Authentication

Two methods supported:

```http
# Method 1: Bearer token
Authorization: Bearer your-api-key

# Method 2: Custom header
X-API-Key: your-api-key
```

---

## Database Schema

### Core Tables

**messages**: Message storage with full metadata
```sql
CREATE TABLE messages (
    rowid INTEGER PRIMARY KEY AUTOINCREMENT,
    chat_jid TEXT NOT NULL,
    msg_id TEXT NOT NULL,
    sender_jid TEXT,
    ts INTEGER NOT NULL,
    from_me INTEGER NOT NULL,
    text TEXT,
    media_type TEXT,
    -- Media metadata for downloads
    direct_path TEXT,
    media_key BLOB,
    file_sha256 BLOB,
    file_enc_sha256 BLOB,
    local_path TEXT,
    UNIQUE(chat_jid, msg_id)
);
```

**messages_fts**: Full-text search index (FTS5)
```sql
CREATE VIRTUAL TABLE messages_fts USING fts5(
    text, media_caption, filename, chat_name, sender_name
);
```

**chats**: Chat metadata
**contacts**: Contact information
**groups**: Group information
**group_participants**: Group membership
**contact_aliases**: User-defined aliases
**contact_tags**: Contact tagging system

### Search Performance

- FTS5 queries: < 10ms typical
- BM25 ranking for relevance
- Snippet generation with highlighting
- Fallback to LIKE if FTS unavailable

---

## Configuration

### Essential Environment Variables

```bash
# Server
WASVC_HOST=0.0.0.0
WASVC_PORT=8080
WASVC_DATA_DIR=/data

# Security
WASVC_API_KEY=your-strong-random-key

# Webhooks (optional)
WASVC_WEBHOOK_URL=https://your-app.com/webhook
WASVC_WEBHOOK_SECRET=your-webhook-secret
WASVC_WEBHOOK_RETRIES=3
WASVC_WEBHOOK_TIMEOUT=10s

# Sync
WASVC_DOWNLOAD_MEDIA=true
WASVC_REFRESH_CONTACTS=true
WASVC_REFRESH_GROUPS=true

# Debug
WA_DEBUG=false
```

### Docker Compose Example

```yaml
version: '3.8'

services:
  wasvc:
    image: wasvc:latest
    container_name: wasvc
    restart: unless-stopped

    ports:
      - "8080:8080"

    volumes:
      - ./data:/data

    environment:
      WASVC_API_KEY: "${WASVC_API_KEY}"
      WASVC_WEBHOOK_URL: "${WASVC_WEBHOOK_URL}"

    healthcheck:
      test: ["CMD", "wget", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

---

## Deployment

### Production Checklist

- [ ] Set strong `WASVC_API_KEY` (32+ random characters)
- [ ] Configure HTTPS with reverse proxy (nginx/Caddy)
- [ ] Set up webhook authentication (`WASVC_WEBHOOK_SECRET`)
- [ ] Configure automated backups (daily recommended)
- [ ] Set up health check monitoring
- [ ] Configure log rotation
- [ ] Test backup/restore procedures
- [ ] Disable debug logging (`WA_DEBUG=false`)
- [ ] Secure data directory permissions (chmod 700)
- [ ] Set up alerting for service failures
- [ ] Document recovery procedures
- [ ] Plan for disk space (media files can grow large)

### Monitoring

**Health Check Endpoint**:
```bash
curl http://localhost:8080/health
```

**Response**:
```json
{
  "status": "ok",
  "state": "connected",
  "ready": true,
  "version": "wasvc/1.0"
}
```

**Docker Health Check**:
```bash
docker ps  # Shows (healthy) status
docker inspect wasvc | grep Health
```

---

## Security

### Best Practices

1. **API Key**: Use strong random keys, rotate periodically
2. **HTTPS**: Always use HTTPS in production (reverse proxy)
3. **Webhook Signing**: Verify HMAC signatures for webhooks
4. **File Permissions**:
   - Data directory: `700` (owner only)
   - Database files: `600` (owner read/write)
5. **Network**: Bind to localhost if behind reverse proxy
6. **Secrets**: Never commit `.env` to version control
7. **Backups**: Encrypt backups containing sensitive data

### Data Protection

- **session.db**: Contains encryption keys (very sensitive)
- **wacli.db**: Contains message content (sensitive)
- **media/**: Downloaded files (potentially sensitive)

Recommendations:
- Use filesystem encryption for production
- Secure backup storage
- Restrict network access
- Regular security audits

---

## Performance

### Typical Metrics

**Message Processing**:
- Storage rate: ~10,000 messages/second
- FTS5 indexing: ~5,000 messages/second
- Search latency: < 10ms for most queries

**Memory**:
- Idle: ~50 MB
- Active sync: ~100-200 MB
- Peak (large file): ~500 MB

**Database**:
- 100K messages: ~50 MB + ~15 MB FTS index
- 1M messages: ~500 MB + ~150 MB FTS index

### Optimization

- WAL mode for concurrent reads/writes
- Strategic indexing (minimal, rely on FTS)
- Prepared statement reuse
- Transaction batching for bulk operations

---

## Troubleshooting

### Common Issues

**Authentication Not Working**:
1. Check service state: `GET /health`
2. Verify not already authenticated: `GET /auth/status`
3. Enable debug: `WA_DEBUG=true`
4. Review logs for errors

**Messages Not Syncing**:
1. Verify connected: `GET /auth/status`
2. Check sync worker: `GET /sync/status`
3. Ensure WhatsApp on primary device is connected
4. Restart sync if needed

**Media Download Fails**:
1. Verify message has media metadata
2. Check connection status
3. Ensure network access to WhatsApp CDN
4. Check disk space

**High Disk Usage**:
1. Set `WASVC_DOWNLOAD_MEDIA=false`
2. Clean old media files
3. Implement retention policy

---

## Development

### Building from Source

```bash
# Clone repository
git clone https://github.com/steipete/wacli.git
cd wacli

# Build CLI
go build -tags sqlite_fts5 -o wacli ./cmd/wacli

# Build HTTP service
go build -tags sqlite_fts5 -o wasvc ./cmd/wasvc

# Run tests
go test -tags sqlite_fts5 ./...
```

### Project Structure

```
wa-svc/
├── cmd/
│   ├── wacli/        # CLI application
│   └── wasvc/        # HTTP service
├── internal/
│   ├── api/          # HTTP handlers & server
│   ├── app/          # Business logic layer
│   ├── config/       # Configuration
│   ├── lock/         # File locking
│   ├── service/      # Service manager
│   ├── store/        # Database layer
│   ├── wa/           # WhatsApp client wrapper
│   └── webhook/      # Webhook emitter
├── data/             # Runtime data (gitignored)
├── docs/             # Documentation
├── Dockerfile
├── docker-compose.yml
└── go.mod
```

---

## Support & Resources

### Documentation Files

- **[00-INDEX.md](00-INDEX.md)** - Documentation index
- **[01-ARCHITECTURE.md](01-ARCHITECTURE.md)** - Architecture deep dive
- **[02-API-REFERENCE.md](02-API-REFERENCE.md)** - Complete API reference
- **[03-DATABASE-SCHEMA.md](03-DATABASE-SCHEMA.md)** - Database documentation
- **[05-CONFIGURATION.md](05-CONFIGURATION.md)** - Configuration guide
- **[06-DEPLOYMENT.md](06-DEPLOYMENT.md)** - Deployment instructions

### External Resources

- **GitHub**: https://github.com/steipete/wacli
- **whatsmeow Library**: https://github.com/tulir/whatsmeow
- **SQLite FTS5**: https://www.sqlite.org/fts5.html
- **Docker**: https://docs.docker.com

### Getting Help

1. Check relevant documentation section
2. Search existing GitHub issues
3. Enable debug logging (`WA_DEBUG=true`)
4. Review service logs
5. Create GitHub issue with details

---

## License

See [LICENSE](../LICENSE) file for details.

---

## Version Information

- **Documentation Version**: 1.0
- **Service Version**: wasvc/1.0
- **Last Updated**: December 2025
- **Go Version**: 1.24+
- **whatsmeow**: Latest (December 2024)

---

This documentation provides complete technical reference for WhatsApp Service. For specific topics, refer to the individual documentation files listed above.
