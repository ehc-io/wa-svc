# WhatsApp Service - Configuration Guide

Complete configuration reference for WhatsApp Service environment variables and settings.

## Table of Contents

- [Quick Start](#quick-start)
- [Environment Variables](#environment-variables)
- [Server Configuration](#server-configuration)
- [Authentication Settings](#authentication-settings)
- [Webhook Configuration](#webhook-configuration)
- [Sync Settings](#sync-settings)
- [Debug & Logging](#debug--logging)
- [Docker Configuration](#docker-configuration)
- [Security Best Practices](#security-best-practices)

---

## Quick Start

### Minimal Configuration

Create `.env` file:
```bash
# Minimal setup
WASVC_API_KEY=your-secret-api-key-here
```

### Recommended Production Configuration

```bash
# Server
WASVC_HOST=0.0.0.0
WASVC_PORT=8080
WASVC_DATA_DIR=/data

# Security
WASVC_API_KEY=your-strong-random-api-key

# Webhooks
WASVC_WEBHOOK_URL=https://your-app.com/webhook
WASVC_WEBHOOK_SECRET=your-webhook-secret
WASVC_WEBHOOK_RETRIES=3
WASVC_WEBHOOK_TIMEOUT=10s

# Sync
WASVC_DOWNLOAD_MEDIA=true
WASVC_REFRESH_CONTACTS=true
WASVC_REFRESH_GROUPS=true

# Operations
WASVC_SHUTDOWN_TIMEOUT=30s

# Debug (disable in production)
WA_DEBUG=false
```

---

## Environment Variables

### Configuration Loading

Environment variables are loaded from:
1. System environment
2. `.env` file (if present)
3. Docker environment (if containerized)

**Priority**: System environment > `.env` file

---

## Server Configuration

### WASVC_HOST

**Description**: Network interface to bind the HTTP server.

**Default**: `0.0.0.0` (all interfaces)

**Values**:
- `0.0.0.0`: All network interfaces (default)
- `127.0.0.1`: Localhost only (local access)
- `192.168.1.10`: Specific interface IP

**Example**:
```bash
WASVC_HOST=0.0.0.0  # Allow external connections
WASVC_HOST=127.0.0.1  # Localhost only
```

**Use Cases**:
- `0.0.0.0`: Production deployment behind reverse proxy
- `127.0.0.1`: Development or secure local-only access

---

### WASVC_PORT

**Description**: TCP port for the HTTP server.

**Default**: `8080`

**Range**: `1-65535` (privileged ports 1-1024 require root)

**Example**:
```bash
WASVC_PORT=8080   # Default
WASVC_PORT=3000   # Alternative port
```

**Note**: If changing from default, update health check URLs and client configurations.

---

### WASVC_HTTP_ADDR

**Description**: Alternative to setting HOST and PORT separately.

**Format**: `HOST:PORT`

**Default**: `0.0.0.0:8080`

**Example**:
```bash
WASVC_HTTP_ADDR=0.0.0.0:8080
WASVC_HTTP_ADDR=127.0.0.1:3000
```

**Note**: If set, overrides `WASVC_HOST` and `WASVC_PORT`.

---

### WASVC_DATA_DIR

**Description**: Directory for storing databases and media files.

**Default**: `/data` (in Docker), `~/.wacli` (CLI)

**Structure**:
```
{WASVC_DATA_DIR}/
├── session.db        # WhatsApp session (whatsmeow)
├── wacli.db          # Application database
├── wacli.db-shm      # Shared memory (WAL mode)
├── wacli.db-wal      # Write-ahead log
├── LOCK              # File lock
└── media/            # Downloaded media files
    ├── {chat_jid}/
    │   ├── photo1.jpg
    │   └── document.pdf
    └── ...
```

**Requirements**:
- **Writable**: Service must have write permissions
- **Persistent**: Must survive container restarts
- **Sufficient Space**: Plan for message and media storage growth

**Example**:
```bash
# Linux/macOS
WASVC_DATA_DIR=/var/lib/wasvc

# Docker
WASVC_DATA_DIR=/data  # Internal path, map volume to host
```

**Docker Volume Mapping**:
```yaml
volumes:
  - ./data:/data  # Maps ./data on host to /data in container
```

---

## Authentication Settings

### WASVC_API_KEY

**Description**: API key for HTTP endpoint authentication.

**Default**: None (authentication disabled)

**Format**: Any string (recommend 32+ characters, random)

**Generation**:
```bash
# Linux/macOS
openssl rand -base64 32

# Node.js
node -e "console.log(require('crypto').randomBytes(32).toString('base64'))"

# Python
python3 -c "import secrets; print(secrets.token_urlsafe(32))"
```

**Usage**:
```bash
WASVC_API_KEY=Kx7mP9vN4qR2tY6wZ8aB1cD5eF3gH0iJ
```

**Client Headers**:
```http
# Method 1: Bearer token
Authorization: Bearer Kx7mP9vN4qR2tY6wZ8aB1cD5eF3gH0iJ

# Method 2: Custom header
X-API-Key: Kx7mP9vN4qR2tY6wZ8aB1cD5eF3gH0iJ
```

**Security**:
- **Required** for production deployments
- Store securely (environment variables, secrets manager)
- Rotate periodically
- Never commit to version control

**Exempted Endpoints** (no auth required):
- `GET /` (Web UI)
- `GET /health`
- `GET /healthz`

---

## Webhook Configuration

### WASVC_WEBHOOK_URL

**Description**: URL to send webhook events (POST requests).

**Default**: None (webhooks disabled)

**Format**: `http://` or `https://` URL

**Example**:
```bash
WASVC_WEBHOOK_URL=https://your-app.com/api/webhooks/whatsapp
WASVC_WEBHOOK_URL=http://localhost:3000/webhook
```

**Requirements**:
- Must return `200-299` status code for success
- Should respond within timeout (default 10s)
- Handle `application/json` POST requests

**Event Types**:
- `message.received`: New message received

**Payload Example**:
```json
{
  "type": "message.received",
  "timestamp": "2025-12-26T10:30:00Z",
  "data": {
    "chat_jid": "1234567890@s.whatsapp.net",
    "text": "Hello!",
    ...
  }
}
```

---

### WASVC_WEBHOOK_SECRET

**Description**: Secret key for HMAC signature verification.

**Default**: None (signatures disabled)

**Format**: Any string (recommend 32+ characters)

**Generation**: Same as `WASVC_API_KEY`

**Example**:
```bash
WASVC_WEBHOOK_SECRET=wH5kP8nM2vR9tY3xZ6aB4cD7eF1gH0iJ
```

**Usage**:
Service includes `X-Webhook-Signature` header:
```
X-Webhook-Signature: sha256=a1b2c3d4e5f6...
```

**Verification** (Node.js):
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
```

**Security**:
- **Strongly recommended** for production
- Prevents webhook forgery
- Validates request authenticity

---

### WASVC_WEBHOOK_RETRIES

**Description**: Number of retry attempts for failed webhook deliveries.

**Default**: `3`

**Range**: `0-10` (0 = no retries)

**Example**:
```bash
WASVC_WEBHOOK_RETRIES=3   # Default
WASVC_WEBHOOK_RETRIES=5   # More retries
WASVC_WEBHOOK_RETRIES=0   # No retries (not recommended)
```

**Retry Logic**:
- Exponential backoff: 1s, 2s, 4s, 8s, ... (max 30s)
- Retries on HTTP errors or timeouts
- Gives up after max retries, event is dropped

**Recommendation**: Keep at default (3) for most use cases.

---

### WASVC_WEBHOOK_TIMEOUT

**Description**: HTTP request timeout for webhook calls.

**Default**: `10s`

**Format**: Duration string (`1s`, `500ms`, `1m`)

**Example**:
```bash
WASVC_WEBHOOK_TIMEOUT=10s   # Default
WASVC_WEBHOOK_TIMEOUT=30s   # Longer timeout
WASVC_WEBHOOK_TIMEOUT=5s    # Shorter timeout
```

**Considerations**:
- Must be shorter than your endpoint's processing time
- Too short: unnecessary retries
- Too long: delays event processing

**Recommendation**: `10s` for most cases, increase if endpoint is slow.

---

## Sync Settings

### WASVC_DOWNLOAD_MEDIA

**Description**: Automatically download media files in background.

**Default**: `true`

**Values**: `true` | `false`

**Example**:
```bash
WASVC_DOWNLOAD_MEDIA=true   # Auto-download enabled
WASVC_DOWNLOAD_MEDIA=false  # Manual download only
```

**Behavior**:
- `true`: Downloads media automatically when messages are received
- `false`: Only download when explicitly requested via API

**Storage Impact**:
- `true`: Higher disk usage (stores all media)
- `false`: Minimal disk usage (download on demand)

**Recommendation**:
- `true`: If you need immediate media access
- `false`: If disk space is limited or media rarely needed

**Note**: Even with `false`, media can be downloaded via `POST /media/{chat}/{msg}/download`.

---

### WASVC_REFRESH_CONTACTS

**Description**: Refresh contact list from WhatsApp on startup.

**Default**: `true`

**Values**: `true` | `false`

**Example**:
```bash
WASVC_REFRESH_CONTACTS=true   # Refresh on startup
WASVC_REFRESH_CONTACTS=false  # Skip refresh
```

**Behavior**:
- `true`: Imports contacts from WhatsApp after successful connection
- `false`: Uses existing contacts from database

**Startup Impact**:
- `true`: Adds 1-3 seconds to startup time (depends on contact count)
- `false`: Faster startup

**Recommendation**: `true` for keeping contacts in sync.

---

### WASVC_REFRESH_GROUPS

**Description**: Refresh group list from WhatsApp on startup.

**Default**: `true`

**Values**: `true` | `false`

**Example**:
```bash
WASVC_REFRESH_GROUPS=true   # Refresh on startup
WASVC_REFRESH_GROUPS=false  # Skip refresh
```

**Behavior**:
- `true`: Imports group info and participants after connection
- `false`: Uses existing group data from database

**Startup Impact**:
- `true`: Adds 1-5 seconds (depends on group count)
- `false`: Faster startup

**Recommendation**: `true` for keeping group info current.

---

## Debug & Logging

### WA_DEBUG

**Description**: Enable verbose logging for WhatsApp protocol.

**Default**: `false`

**Values**: `true` | `false`

**Example**:
```bash
WA_DEBUG=false  # Production (quiet)
WA_DEBUG=true   # Development (verbose)
```

**Output When Enabled**:
```
[WA-DEBUG] DeviceProps before Connect:
[WA-DEBUG]   Os: WhatsApp-SVC
[WA-DEBUG]   PlatformType: DESKTOP
[WA-DEBUG]   RequireFullSync: false
```

**Use Cases**:
- Debugging authentication issues
- Verifying device properties
- Troubleshooting connection problems

**Production**: Set to `false` (reduces log noise)

**Development**: Set to `true` for troubleshooting

---

### WASVC_SHUTDOWN_TIMEOUT

**Description**: Graceful shutdown timeout duration.

**Default**: `30s`

**Format**: Duration string

**Example**:
```bash
WASVC_SHUTDOWN_TIMEOUT=30s   # Default
WASVC_SHUTDOWN_TIMEOUT=1m    # Longer grace period
WASVC_SHUTDOWN_TIMEOUT=10s   # Faster shutdown
```

**Shutdown Sequence**:
1. Receive SIGTERM/SIGINT
2. Stop accepting new requests
3. Wait for in-flight requests (up to timeout)
4. Close WhatsApp connection
5. Close databases
6. Release file lock
7. Exit

**Considerations**:
- Too short: May terminate active operations
- Too long: Delays restart/redeployment

**Recommendation**: `30s` for most cases.

---

## Docker Configuration

### docker-compose.yml Example

**Production Setup**:
```yaml
version: '3.8'

services:
  wasvc:
    image: your-registry/wasvc:latest
    container_name: wasvc
    restart: unless-stopped

    ports:
      - "8080:8080"

    volumes:
      - ./data:/data

    environment:
      # Server
      WASVC_HOST: "0.0.0.0"
      WASVC_PORT: "8080"
      WASVC_DATA_DIR: "/data"

      # Security
      WASVC_API_KEY: "${WASVC_API_KEY}"

      # Webhooks
      WASVC_WEBHOOK_URL: "${WASVC_WEBHOOK_URL}"
      WASVC_WEBHOOK_SECRET: "${WASVC_WEBHOOK_SECRET}"
      WASVC_WEBHOOK_RETRIES: "3"
      WASVC_WEBHOOK_TIMEOUT: "10s"

      # Sync
      WASVC_DOWNLOAD_MEDIA: "true"
      WASVC_REFRESH_CONTACTS: "true"
      WASVC_REFRESH_GROUPS: "true"

      # Operations
      WASVC_SHUTDOWN_TIMEOUT: "30s"

      # Debug
      WA_DEBUG: "false"

    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 5s
```

**Load from .env file**:
```yaml
env_file:
  - .env
```

---

## Security Best Practices

### 1. API Key Management

**DO**:
- Generate strong random keys (32+ characters)
- Store in environment variables or secrets manager
- Rotate periodically (every 90 days)
- Use different keys per environment (dev/staging/prod)

**DON'T**:
- Hardcode in source code
- Commit to version control
- Share across services
- Use weak/predictable keys

---

### 2. Webhook Security

**DO**:
- Always set `WASVC_WEBHOOK_SECRET`
- Use HTTPS for webhook URLs
- Verify HMAC signatures
- Validate request origin

**DON'T**:
- Use HTTP for sensitive webhooks
- Skip signature verification
- Expose webhook endpoints publicly without auth

---

### 3. Network Security

**Production Checklist**:
- [ ] Use reverse proxy (nginx, Caddy)
- [ ] Enable HTTPS/TLS
- [ ] Configure firewall rules
- [ ] Restrict `WASVC_HOST` if needed
- [ ] Use internal network for webhooks if possible

**Reverse Proxy Example** (nginx):
```nginx
server {
    listen 443 ssl http2;
    server_name whatsapp.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Timeouts for large files
        proxy_read_timeout 300s;
        proxy_send_timeout 300s;
    }
}
```

---

### 4. File Permissions

**Data Directory**:
```bash
# Directory: owner access only
chmod 700 /path/to/data

# Database files: owner read/write
chmod 600 /path/to/data/*.db

# Media files: owner read/write
chmod 600 /path/to/data/media/**/*
```

**User**:
```bash
# Run as non-root user
chown -R wasvc:wasvc /path/to/data
```

**Docker**:
```dockerfile
# Already configured in Dockerfile
USER wasvc  # Non-root user (UID 1000)
```

---

### 5. Secrets Management

**Docker Secrets** (Swarm/Kubernetes):
```yaml
secrets:
  api_key:
    external: true
  webhook_secret:
    external: true

services:
  wasvc:
    secrets:
      - api_key
      - webhook_secret
    environment:
      WASVC_API_KEY_FILE: /run/secrets/api_key
      WASVC_WEBHOOK_SECRET_FILE: /run/secrets/webhook_secret
```

**Environment File** (.env):
```bash
# .env (never commit to git)
WASVC_API_KEY=...
WASVC_WEBHOOK_SECRET=...
```

**Ensure .env is in .gitignore**:
```bash
echo ".env" >> .gitignore
```

---

## Configuration Validation

### Startup Checks

Service validates configuration on startup:

**Required Checks**:
- Data directory exists and is writable
- Port is available
- API key meets minimum length (if set)

**Warnings**:
- API key not set (authentication disabled)
- Webhook URL set but secret not set (unsigned webhooks)

**Errors** (prevents startup):
- Data directory not writable
- Port already in use
- Invalid environment variable format

---

## Environment Templates

### Development

```bash
# .env.development
WASVC_HOST=127.0.0.1
WASVC_PORT=8080
WASVC_DATA_DIR=./dev-data
WASVC_API_KEY=dev-key-not-secure
WASVC_WEBHOOK_URL=http://localhost:3000/webhook
WA_DEBUG=true
WASVC_DOWNLOAD_MEDIA=false
```

### Staging

```bash
# .env.staging
WASVC_HOST=0.0.0.0
WASVC_PORT=8080
WASVC_DATA_DIR=/data
WASVC_API_KEY=${STAGING_API_KEY}
WASVC_WEBHOOK_URL=https://staging-app.example.com/webhook
WASVC_WEBHOOK_SECRET=${STAGING_WEBHOOK_SECRET}
WA_DEBUG=false
```

### Production

```bash
# .env.production
WASVC_HOST=0.0.0.0
WASVC_PORT=8080
WASVC_DATA_DIR=/data
WASVC_API_KEY=${PROD_API_KEY}
WASVC_WEBHOOK_URL=https://app.example.com/webhook
WASVC_WEBHOOK_SECRET=${PROD_WEBHOOK_SECRET}
WASVC_WEBHOOK_RETRIES=5
WASVC_WEBHOOK_TIMEOUT=15s
WASVC_DOWNLOAD_MEDIA=true
WASVC_REFRESH_CONTACTS=true
WASVC_REFRESH_GROUPS=true
WA_DEBUG=false
```

---

## Troubleshooting

### Configuration Not Applied

**Issue**: Changes to `.env` not taking effect

**Solutions**:
1. Restart service: `docker compose restart`
2. Check file location (must be in working directory)
3. Verify syntax (no spaces around `=`)
4. Check for typos in variable names

### Port Already in Use

**Issue**: `bind: address already in use`

**Solutions**:
```bash
# Find process using port
lsof -i :8080
netstat -tulpn | grep 8080

# Change port
WASVC_PORT=8081
```

### Permission Denied on Data Directory

**Issue**: Cannot write to data directory

**Solutions**:
```bash
# Check permissions
ls -la /data

# Fix permissions
sudo chown -R 1000:1000 /data  # Docker user
sudo chmod -R 700 /data
```

---

For deployment instructions using these configurations, see the [Deployment Guide](06-DEPLOYMENT.md).
