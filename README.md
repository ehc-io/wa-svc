### **Project README**

# WhatsApp-SVC (wa-svc)

`wa-svc` is a high-performance WhatsApp integration service that provides a bridge between the WhatsApp Web protocol and your applications via a REST API and CLI. It is designed for message archiving, automated notifications, and local message search.

## 🚀 Key Capabilities

*   **Continuous Synchronization**: Automatically captures and stores message history in a local SQLite database.
*   **Custom Device Identity**: Registers as a distinct "Desktop" device named `WhatsApp-SVC` in your linked devices list.
*   **Rich Media Support**: 
    *   Send and receive images, documents, and files.
    *   Automated metadata capture for reliable media downloading and decryption.
*   **Powerful Search**: Built-in Full-Text Search (FTS5) with BM25 ranking for ultra-fast offline searching of your entire message history.
*   **Flexible Interface**: 
    *   **`wasvc`**: A RESTful HTTP API for server-side integrations.
    *   **`wacli`**: A command-line tool for local management and testing.
*   **Webhooks**: Real-time delivery of incoming messages and status updates to your external services.

## 🛠 Tech Stack

*   **Language**: Go 1.24+
*   **Protocol**: [whatsmeow](https://github.com/tulir/whatsmeow) (WhatsApp Multi-Device)
*   **Database**: SQLite3 with FTS5 Full-Text Search
*   **Containerization**: Docker & Docker Compose

## 🚦 Getting Started

### 1. Environment Configuration
Create a `.env` file based on `.env.example`:
```bash
WA_DEBUG=false
WASVC_HTTP_ADDR=0.0.0.0:8080
WASVC_API_KEY=your_secure_key
```

### 2. Deployment
```bash
docker compose build
docker compose up -d
```

### 3. Authentication
1.  Initiate pairing: `POST /auth/init`
2.  Retrieve the QR code: `GET /auth/qr` (or open the service URL in a browser for the Web UI).
3.  Scan with your WhatsApp phone.

## 📡 API Highlights

| Endpoint | Method | Description |
| :--- | :--- | :--- |
| `/messages/text` | `POST` | Send a plain text message |
| `/messages/file` | `POST` | Send images or documents (base64) |
| `/media/{jid}/{msg_id}/download` | `POST` | Trigger background media download |
| `/search?q=...` | `GET` | Full-text search across all stored messages |
| `/auth/status` | `GET` | Check connection and authentication state |

## 🔧 Troubleshooting & Debugging
Enable verbose logging for the WhatsApp protocol by setting the environment variable:
`WA_DEBUG=true`

This will output detailed connection states and device property validation logs to the container output.