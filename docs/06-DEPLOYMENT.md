# WhatsApp Service - Deployment Guide

Production-ready deployment guide for WhatsApp Service using Docker and Docker Compose.

## Table of Contents

- [Quick Start](#quick-start)
- [Prerequisites](#prerequisites)
- [Docker Deployment](#docker-deployment)
- [Docker Compose Deployment](#docker-compose-deployment)
- [Production Setup](#production-setup)
- [Health Checks & Monitoring](#health-checks--monitoring)
- [Backup & Recovery](#backup--recovery)
- [Scaling Considerations](#scaling-considerations)
- [Troubleshooting](#troubleshooting)

---

## Quick Start

### 1. Clone and Setup

```bash
git clone https://github.com/steipete/wacli.git
cd wacli
cp .env.example .env
```

### 2. Configure Environment

Edit `.env`:
```bash
WASVC_API_KEY=your-strong-random-key-here
WASVC_WEBHOOK_URL=https://your-app.com/webhook  # Optional
WASVC_WEBHOOK_SECRET=your-webhook-secret         # Optional
```

### 3. Deploy

```bash
docker compose up -d
```

### 4. Authenticate

```bash
# Open in browser
open http://localhost:8080

# Or use API
curl -X POST http://localhost:8080/auth/init \
  -H "Authorization: Bearer your-api-key"
```

Scan the QR code with WhatsApp mobile app.

---

## Prerequisites

### System Requirements

**Minimum**:
- CPU: 1 core
- RAM: 512 MB
- Disk: 10 GB (for databases and media)
- OS: Linux, macOS, Windows with Docker

**Recommended**:
- CPU: 2+ cores
- RAM: 2 GB
- Disk: 50+ GB (SSD preferred)
- OS: Linux (Ubuntu 22.04 LTS or newer)

### Software Requirements

**Required**:
- Docker 24.0+ or Docker Desktop
- Docker Compose 2.0+

**Optional**:
- Reverse proxy (nginx, Caddy, Traefik)
- Process manager (systemd, supervisord)

### Installation

**Docker** (Ubuntu/Debian):
```bash
# Install Docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# Install Docker Compose (if not included)
sudo apt-get install docker-compose-plugin

# Verify
docker --version
docker compose version
```

**Docker Desktop** (macOS/Windows):
Download from: https://www.docker.com/products/docker-desktop

---

## Docker Deployment

### Build Image

```bash
# Build from source
docker build -t wasvc:latest .

# With specific Go version
docker build --build-arg GO_VERSION=1.24 -t wasvc:latest .
```

### Run Container

```bash
docker run -d \
  --name wasvc \
  -p 8080:8080 \
  -v $(pwd)/data:/data \
  -e WASVC_API_KEY=your-api-key \
  --restart unless-stopped \
  wasvc:latest
```

**Explanation**:
- `-d`: Detached mode (background)
- `--name wasvc`: Container name
- `-p 8080:8080`: Port mapping (host:container)
- `-v $(pwd)/data:/data`: Volume mount for persistence
- `-e WASVC_API_KEY`: Environment variable
- `--restart unless-stopped`: Auto-restart policy

### Environment Variables

```bash
docker run -d \
  --name wasvc \
  -p 8080:8080 \
  -v $(pwd)/data:/data \
  -e WASVC_API_KEY=your-api-key \
  -e WASVC_WEBHOOK_URL=https://your-app.com/webhook \
  -e WASVC_WEBHOOK_SECRET=your-secret \
  -e WA_DEBUG=false \
  wasvc:latest
```

Or use env file:
```bash
docker run -d \
  --name wasvc \
  -p 8080:8080 \
  -v $(pwd)/data:/data \
  --env-file .env \
  wasvc:latest
```

---

## Docker Compose Deployment

### Basic Setup

Create `docker-compose.yml`:
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
      WASVC_HOST: "0.0.0.0"
      WASVC_PORT: "8080"
      WASVC_DATA_DIR: "/data"
      WASVC_API_KEY: "${WASVC_API_KEY}"

    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 5s
```

### Start Service

```bash
# Start in background
docker compose up -d

# View logs
docker compose logs -f

# Stop service
docker compose down

# Restart service
docker compose restart
```

---

## Production Setup

### Full Production Configuration

**docker-compose.yml**:
```yaml
version: '3.8'

services:
  wasvc:
    build: .
    container_name: wasvc-prod
    restart: unless-stopped

    ports:
      - "127.0.0.1:8080:8080"  # Bind to localhost only

    volumes:
      - ./data:/data:rw
      - /etc/localtime:/etc/localtime:ro  # Sync timezone

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
      WASVC_WEBHOOK_RETRIES: "5"
      WASVC_WEBHOOK_TIMEOUT: "15s"

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
      start_period: 10s

    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

    networks:
      - wasvc-network

networks:
  wasvc-network:
    driver: bridge
```

**.env** (production):
```bash
# Security
WASVC_API_KEY=<generated-strong-key>
WASVC_WEBHOOK_SECRET=<generated-strong-key>

# Webhooks
WASVC_WEBHOOK_URL=https://your-production-app.com/api/webhooks/whatsapp
```

### Reverse Proxy Setup

**nginx Configuration**:

`/etc/nginx/sites-available/wasvc`:
```nginx
upstream wasvc {
    server 127.0.0.1:8080;
}

server {
    listen 80;
    server_name whatsapp.yourdomain.com;

    # Redirect to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name whatsapp.yourdomain.com;

    # SSL Configuration
    ssl_certificate /etc/letsencrypt/live/whatsapp.yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/whatsapp.yourdomain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    # Security Headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    # Client Max Body Size (for large file uploads)
    client_max_body_size 100M;

    # Timeouts (for large file operations)
    proxy_read_timeout 300s;
    proxy_send_timeout 300s;
    proxy_connect_timeout 60s;

    location / {
        proxy_pass http://wasvc;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket support (if needed in future)
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }

    # Health check endpoint (bypass auth if needed)
    location /health {
        proxy_pass http://wasvc;
        access_log off;
    }

    # Access and Error Logs
    access_log /var/log/nginx/wasvc-access.log;
    error_log /var/log/nginx/wasvc-error.log;
}
```

**Enable and Reload**:
```bash
sudo ln -s /etc/nginx/sites-available/wasvc /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

**Let's Encrypt SSL**:
```bash
sudo apt-get install certbot python3-certbot-nginx
sudo certbot --nginx -d whatsapp.yourdomain.com
```

---

## Health Checks & Monitoring

### Built-in Health Check

**Endpoint**: `GET /health`

**Response**:
```json
{
  "status": "ok",
  "state": "connected",
  "ready": true,
  "version": "wasvc/1.0",
  "timestamp": "2025-12-26T10:30:00Z"
}
```

**Status Codes**:
- `200 OK`: Service healthy
- `503 Service Unavailable`: Service degraded

### Docker Health Check

Configured in docker-compose.yml:
```yaml
healthcheck:
  test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
  interval: 30s      # Check every 30 seconds
  timeout: 10s       # Timeout after 10 seconds
  retries: 3         # Mark unhealthy after 3 failures
  start_period: 10s  # Grace period on startup
```

**Check Status**:
```bash
docker ps  # Shows (healthy) or (unhealthy)
docker inspect wasvc | grep Health -A 20
```

### External Monitoring

**Uptime Monitoring**:
- UptimeRobot
- Pingdom
- StatusCake

**Example Check**:
```bash
curl -f http://localhost:8080/health || exit 1
```

### Logging

**View Logs**:
```bash
# Docker Compose
docker compose logs -f wasvc

# Docker
docker logs -f wasvc

# Last 100 lines
docker logs --tail 100 wasvc
```

**Log Rotation** (configured in docker-compose.yml):
```yaml
logging:
  driver: "json-file"
  options:
    max-size: "10m"    # Rotate at 10MB
    max-file: "3"      # Keep 3 files
```

**Centralized Logging**:

For production, consider:
- ELK Stack (Elasticsearch, Logstash, Kibana)
- Loki + Grafana
- CloudWatch Logs (AWS)
- Stackdriver (GCP)

**Example with Loki**:
```yaml
logging:
  driver: "loki"
  options:
    loki-url: "http://loki:3100/loki/api/v1/push"
    loki-retries: "5"
    loki-batch-size: "400"
```

---

## Backup & Recovery

### What to Backup

**Critical Data**:
1. `data/session.db` - WhatsApp session (encryption keys)
2. `data/wacli.db` - Message database
3. `.env` - Configuration (secure storage)

**Optional**:
4. `data/media/` - Downloaded media files (can be re-downloaded)

### Backup Script

**backup.sh**:
```bash
#!/bin/bash
set -e

BACKUP_DIR="/backups/wasvc"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
DATA_DIR="./data"

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Stop service (optional, for consistency)
# docker compose stop wasvc

# Backup databases
echo "Backing up databases..."
tar -czf "$BACKUP_DIR/wasvc-db-$TIMESTAMP.tar.gz" \
  "$DATA_DIR/session.db" \
  "$DATA_DIR/wacli.db"

# Backup media (optional)
# tar -czf "$BACKUP_DIR/wasvc-media-$TIMESTAMP.tar.gz" \
#   "$DATA_DIR/media"

# Restart service
# docker compose start wasvc

# Clean old backups (keep last 7 days)
find "$BACKUP_DIR" -name "wasvc-db-*.tar.gz" -mtime +7 -delete

echo "Backup completed: $BACKUP_DIR/wasvc-db-$TIMESTAMP.tar.gz"
```

**Cron Job** (daily at 2 AM):
```bash
# Edit crontab
crontab -e

# Add line
0 2 * * * /path/to/backup.sh >> /var/log/wasvc-backup.log 2>&1
```

### Restore from Backup

```bash
#!/bin/bash
BACKUP_FILE="$1"
DATA_DIR="./data"

if [ -z "$BACKUP_FILE" ]; then
  echo "Usage: ./restore.sh <backup-file>"
  exit 1
fi

# Stop service
docker compose stop wasvc

# Backup current data (just in case)
mv "$DATA_DIR" "$DATA_DIR.old"
mkdir -p "$DATA_DIR"

# Extract backup
tar -xzf "$BACKUP_FILE" -C "$DATA_DIR" --strip-components=1

# Restart service
docker compose start wasvc

echo "Restore completed from: $BACKUP_FILE"
```

### Off-site Backup

**Amazon S3**:
```bash
#!/bin/bash
BACKUP_FILE="/backups/wasvc/wasvc-db-$(date +%Y%m%d_%H%M%S).tar.gz"
S3_BUCKET="s3://your-bucket/wasvc-backups/"

# Create backup
tar -czf "$BACKUP_FILE" data/session.db data/wacli.db

# Upload to S3
aws s3 cp "$BACKUP_FILE" "$S3_BUCKET"

# Clean local backups
find /backups/wasvc -name "*.tar.gz" -mtime +1 -delete
```

---

## Scaling Considerations

### Current Limitations

**Single Instance Only**:
- WhatsApp protocol allows ONE active connection per device
- Running multiple instances causes session conflicts
- File locking prevents concurrent access

**Not Horizontally Scalable**:
- Cannot run behind load balancer with multiple instances
- Cannot use container orchestration (Kubernetes) for scaling

### Vertical Scaling

**Increase Resources**:
```yaml
# docker-compose.yml
services:
  wasvc:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 4G
        reservations:
          cpus: '1'
          memory: 2G
```

### High Availability

**Strategies**:

1. **Active-Standby**:
   - Primary instance running
   - Standby instance ready with recent backup
   - Manual/automated failover on primary failure

2. **Backup Automation**:
   - Frequent backups (every 15-30 minutes)
   - Fast restore process (< 5 minutes)
   - Acceptable downtime during restore

3. **Monitoring & Alerts**:
   - Health check monitoring
   - Automated restart on failure
   - Alert on repeated failures

**Not Recommended**:
- Active-Active (causes session conflicts)
- Load balancing (not supported)

---

## Troubleshooting

### Container Won't Start

**Check Logs**:
```bash
docker compose logs wasvc
docker logs wasvc
```

**Common Issues**:

1. **Port Already in Use**:
   ```bash
   # Find process
   sudo lsof -i :8080
   # Change port in docker-compose.yml or stop conflicting service
   ```

2. **Permission Denied on Data Directory**:
   ```bash
   # Fix permissions
   sudo chown -R 1000:1000 ./data
   sudo chmod -R 700 ./data
   ```

3. **Invalid Configuration**:
   ```bash
   # Validate env file
   cat .env
   # Check for typos, missing quotes, etc.
   ```

### Service Unhealthy

**Check Health**:
```bash
curl http://localhost:8080/health
docker inspect wasvc | grep -A 20 Health
```

**Solutions**:
1. Check if WhatsApp connection established
2. Verify network connectivity
3. Review logs for errors
4. Restart service if needed

### Authentication Fails

**Symptoms**:
- QR code never appears
- Scan succeeds but doesn't connect

**Solutions**:
1. Check `WA_DEBUG=true` in logs
2. Verify internet connectivity
3. Ensure not already authenticated
4. Try logout and re-authenticate
5. Check WhatsApp servers status

### Messages Not Syncing

**Symptoms**:
- New messages don't appear in database
- Search returns old results

**Check**:
```bash
# Sync status
curl -H "Authorization: Bearer your-key" \
  http://localhost:8080/sync/status
```

**Solutions**:
1. Verify connection: `GET /auth/status`
2. Restart sync: `POST /sync/stop` then `POST /sync/start`
3. Check WhatsApp on primary device
4. Review logs for errors

### High Disk Usage

**Check Usage**:
```bash
du -sh ./data
du -sh ./data/media
du -sh ./data/*.db
```

**Solutions**:
1. Set `WASVC_DOWNLOAD_MEDIA=false` to stop auto-downloads
2. Clean old media files manually
3. Implement media retention policy
4. Use external storage (S3, etc.) for media

### Memory Leaks

**Monitor**:
```bash
docker stats wasvc
```

**If memory grows unbounded**:
1. Check for excessive webhook queue
2. Review for connection leaks
3. Restart service daily as workaround
4. Report issue with logs

---

## Production Checklist

Before going to production:

- [ ] Set strong `WASVC_API_KEY`
- [ ] Configure `WASVC_WEBHOOK_SECRET` if using webhooks
- [ ] Set `WA_DEBUG=false`
- [ ] Configure reverse proxy with HTTPS
- [ ] Set up automated backups
- [ ] Configure log rotation
- [ ] Set up health check monitoring
- [ ] Test failover/restore procedure
- [ ] Document recovery procedures
- [ ] Set up alerting for service down
- [ ] Review and harden firewall rules
- [ ] Ensure data directory has sufficient space
- [ ] Test with production load
- [ ] Review security best practices
- [ ] Prepare incident response plan

---

For configuration details, see [Configuration Guide](05-CONFIGURATION.md).

For API usage, see [API Reference](02-API-REFERENCE.md).
