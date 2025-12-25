This is a system design perspective of building a "Headless WhatsApp Service" on top of `wacli` project. 
The existing codebase already separates the core logic (`internal/app`, `internal/wa`) from the CLI command layer (`cmd/wacli`), which is the hardest part of the refactoring process.

Here is a system design breakdown of how you would transform this CLI tool into an independent service.

### 1. High-Level Architecture

Instead of a CLI that spins up, runs a task, and dies, you will move to a **Daemon/Server Model**. The application becomes a long-running process with two primary interfaces: an **HTTP API** for incoming commands and a **Webhook Emitter** for outgoing events.

**The Layers:**

1.  **API Layer (HTTP/REST):** Replaces the `cobra` commands.
2.  **Service Manager (The "Brain"):** A central singleton process that holds the active `whatsmeow` client in memory and manages the connection lifecycle.
3.  **Sync Worker:** A background goroutine (derived from `wacli sync`) that continuously ingests messages.
4.  **Data Layer:** The existing SQLite database (with FTS5) used for storage and search.

---

### 2. Functional Breakdown

#### A. Authentication (The Web Interface Flow)
In the CLI, the QR code is printed to the terminal. In your service, this needs to be decoupled.

*   **State Machine:** The Service Manager needs states: `UNAUTHENTICATED`, `PAIRING`, `CONNECTED`, `OFFLINE`.
*   **The API Endpoint:** Create a `GET /auth/qr` endpoint.
*   **The Flow:**
    1.  When the service starts (or via a `POST /auth/init` trigger), it attempts to connect.
    2.  If not logged in, the `whatsmeow` client generates a QR event.
    3.  Instead of printing to stdout, your Service Manager captures this QR string and stores it in a temporary in-memory cache (volatile state).
    4.  Your frontend polls (or uses Server-Sent Events) to fetch this string from `GET /auth/qr` and renders it using a JS library (like `qrcode.js`).
    5.  Once scanned, the Service Manager transitions to `CONNECTED` and clears the cached QR code.

#### B. Monitoring for New Messages (Event Ingestion)
You need to convert the internal Go channels into external notifications.

*   **Ingestion:** The existing "Sync" logic receives `events.Message` from WhatsApp.
*   **Dispatch:** Instead of just writing to SQLite, the Sync Worker should pass the message payload to an **Event Dispatcher**.
*   **Delivery (Webhooks):** The simplest design is a configurable Webhook URL.
    *   The service sends a `POST` request to your consuming application with the JSON payload of the message.
    *   *Design Tip:* Implement a simple retry queue for these webhooks. If your consuming app is down, the service shouldn't drop the message notification.

#### C. Sending and Receiving
*   **Receiving:** Covered by the Sync Worker and Webhooks above.
*   **Sending:**
    *   Create `POST /messages/text` and `POST /messages/file`.
    *   These endpoints must check if the Service Manager is in the `CONNECTED` state.
    *   They utilize the existing `client.SendText` logic.
    *   *Concurrency Note:* Since sending and syncing happen simultaneously, relying on `whatsmeow`'s internal thread safety is mostly fine, but you must ensure your SQLite usage allows concurrent reads (search) and writes (sync/send logging).

#### D. Searching the Database
This is the strongest selling point of using `wacli` as the base.

*   **Endpoint:** `GET /search?q=query&limit=50`.
*   **Logic:** This maps directly to the `db.SearchMessages` function in the existing code.
*   **Performance:** Since the database is local to the service, this API response will be sub-millisecond, regardless of your internet connection or WhatsApp's cloud status.

---

### 3. Critical Technical Challenges

To make this production-ready, you must address specific architectural constraints found in the current codebase:

#### 1. The Locking Mechanism
*   **Current Issue:** `wacli` uses a file lock (`LOCK`) to ensure only one process runs.
*   **Service Solution:** Your API service effectively becomes the *only* process. You must ensure you don't accidentally run the CLI commands manually while the API service is running, or they will crash/block.
*   **Architecture:** The API Server owns the Lock.

#### 2. SQLite Concurrency (WAL Mode)
*   **The Problem:** In a service, a user might Search (Read) at the exact same millisecond a new message arrives (Write). Default SQLite might lock the whole DB.
*   **The Fix:** The repo already enables **WAL (Write-Ahead Logging)** mode in `internal/store/store.go`. This is crucial. It allows one writer and multiple readers simultaneously. You must ensure this remains enabled and is robust.

#### 3. Session Persistence
*   **The Problem:** If the service crashes or restarts, it must auto-reconnect without asking for a QR code again.
*   **The Solution:** The `session.db` (whatsmeow store) must be persistent. When your Service Manager starts, it must check `client.IsAuthed()`. If true, it skips the QR generation and goes straight to `Connect()`.

#### 4. Media Handling
*   **The Problem:** Accessing media files (images/videos) via API.
*   **The Solution:**
    *   You need a static file server handler in your API (e.g., `GET /media/{message_id}`).
    *   When the API receives a request for media, it checks if it's downloaded. If not, it triggers the existing `DownloadMedia` logic on demand, saves it to the `media/` folder, and then serves the file.

### 4. Deployment Model

I recommend packaging this as a **Docker Container**.

*   **Volume 1 (`/data`):** Mounts the `~/.wacli` directory. This ensures your database and session keys persist across container restarts.
*   **Network:** Exposes port `8080` for your API.
*   **Environment Variables:** Configure the Webhook URL and Auth credentials for the API itself (so strangers can't read your messages).

### Summary

You are essentially building a **"Sidecar"** pattern.
The `wacli` logic runs as a sidecar service, handling the complex WhatsApp protocol and local caching. Your main application consumes it via a clean REST API, completely shielded from the complexities of encryption, socket management, and history syncing. This is a very robust architecture for a personal messaging service.