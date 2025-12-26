# WhatsApp Service - Database Schema Documentation

Complete documentation of the SQLite database architecture, schema design, and Full-Text Search implementation.

## Table of Contents

- [Overview](#overview)
- [Database Files](#database-files)
- [Schema Design](#schema-design)
- [Table Definitions](#table-definitions)
- [Full-Text Search (FTS5)](#full-text-search-fts5)
- [Indexes & Performance](#indexes--performance)
- [Data Models](#data-models)
- [Triggers](#triggers)
- [Queries & Operations](#queries--operations)
- [Migration Strategy](#migration-strategy)

---

## Overview

WhatsApp Service uses a dual-database architecture:

1. **wacli.db**: Application data (messages, contacts, groups, search index)
2. **session.db**: WhatsApp protocol data (managed by whatsmeow library)

### Design Principles

- **Normalization**: Balanced 3NF with denormalization for performance
- **Idempotency**: Upsert patterns for safe message replay
- **Performance**: Strategic indexing and FTS5 for search
- **Integrity**: Foreign keys with cascade deletes
- **Concurrency**: WAL mode for read/write parallelism

---

## Database Files

### wacli.db (Application Database)

**Location**: `{WASVC_DATA_DIR}/wacli.db`

**Purpose**: Stores all application-level data including messages, contacts, groups, and search indices.

**Configuration**:
```sql
PRAGMA journal_mode=WAL;         -- Write-Ahead Logging
PRAGMA synchronous=NORMAL;       -- Balanced durability
PRAGMA temp_store=MEMORY;        -- Temp tables in RAM
PRAGMA foreign_keys=ON;          -- Enforce FK constraints
```

**Size Estimates**:
- 100K messages: ~50 MB
- FTS5 index: +30% overhead (~15 MB)
- 1M messages: ~500 MB + ~150 MB FTS

### session.db (WhatsApp Session)

**Location**: `{WASVC_DATA_DIR}/session.db`

**Purpose**: WhatsApp encryption keys, device identity, and protocol state.

**Managed By**: whatsmeow library (do not modify directly)

**Security**: Contains sensitive cryptographic material
- Permissions: `0600` (owner read/write only)
- Never commit to version control
- Backup separately from application data

---

## Schema Design

### Entity-Relationship Diagram

```
┌─────────────┐
│    chats    │──┐
└─────────────┘  │
                 │ 1:N
                 ↓
           ┌──────────┐
           │ messages │
           └──────────┘
                 ↑
                 │ 1:1
                 │
           ┌────────────┐
           │messages_fts│ (FTS5 Virtual Table)
           └────────────┘

┌──────────────┐
│   contacts   │──┐
└──────────────┘  │ 1:1
                  ↓
         ┌─────────────────┐
         │ contact_aliases │
         └─────────────────┘
                  ↓ 1:N
         ┌─────────────────┐
         │  contact_tags   │
         └─────────────────┘

┌─────────┐
│ groups  │──┐
└─────────┘  │ 1:N
             ↓
    ┌────────────────────┐
    │ group_participants │
    └────────────────────┘
```

---

## Table Definitions

### chats

Stores chat metadata for direct messages, groups, and broadcast lists.

**Schema**:
```sql
CREATE TABLE chats (
    jid TEXT PRIMARY KEY,           -- WhatsApp JID
    kind TEXT NOT NULL,             -- dm|group|broadcast|unknown
    name TEXT,                      -- Display name
    last_message_ts INTEGER         -- Unix timestamp
);
```

**Fields**:
- `jid`: WhatsApp Jabber ID (unique identifier)
  - User: `1234567890@s.whatsapp.net`
  - Group: `1234567890-1640000000@g.us`
  - Broadcast: `broadcast@s.whatsapp.net`
- `kind`: Chat type classification
- `name`: Resolved display name (from contacts or group info)
- `last_message_ts`: Unix timestamp of last message (for sorting)

**Constraints**:
- Primary key on `jid`
- `kind` should be one of: `dm`, `group`, `broadcast`, `unknown`

**Usage**:
```sql
-- Get recent chats
SELECT jid, kind, name, last_message_ts
FROM chats
ORDER BY last_message_ts DESC
LIMIT 50;

-- Find chats by name
SELECT * FROM chats
WHERE LOWER(name) LIKE LOWER('%project%')
ORDER BY last_message_ts DESC;
```

---

### contacts

Stores contact information imported from WhatsApp.

**Schema**:
```sql
CREATE TABLE contacts (
    jid TEXT PRIMARY KEY,           -- WhatsApp JID
    phone TEXT,                     -- Phone number
    push_name TEXT,                 -- Name from WhatsApp push
    full_name TEXT,                 -- Full contact name
    first_name TEXT,                -- First name only
    business_name TEXT,             -- Business account name
    updated_at INTEGER NOT NULL     -- Unix timestamp
);
```

**Fields**:
- `jid`: WhatsApp JID (unique identifier)
- `phone`: Phone number without country code prefix
- `push_name`: Name set in WhatsApp (shown in notifications)
- `full_name`: Complete name from contact card
- `first_name`: First name only
- `business_name`: Name for business accounts
- `updated_at`: Last update timestamp

**Name Priority** (for display):
1. Alias (from `contact_aliases` table)
2. `full_name`
3. `push_name`
4. `business_name`
5. `first_name`
6. `jid`

**Usage**:
```sql
-- Get best display name
SELECT jid,
       COALESCE(
           (SELECT alias FROM contact_aliases WHERE contact_aliases.jid = contacts.jid),
           NULLIF(full_name, ''),
           NULLIF(push_name, ''),
           NULLIF(business_name, ''),
           NULLIF(first_name, ''),
           jid
       ) AS display_name
FROM contacts
WHERE jid = '1234567890@s.whatsapp.net';
```

---

### groups

Stores group metadata.

**Schema**:
```sql
CREATE TABLE groups (
    jid TEXT PRIMARY KEY,           -- Group JID
    name TEXT,                      -- Group name
    owner_jid TEXT,                 -- Creator's JID
    created_ts INTEGER,             -- Creation timestamp
    updated_at INTEGER NOT NULL     -- Last update
);
```

**Fields**:
- `jid`: Group JID (format: `{id}-{timestamp}@g.us`)
- `name`: Group display name
- `owner_jid`: JID of group creator
- `created_ts`: Unix timestamp when group was created
- `updated_at`: Last refresh timestamp

**Relationships**:
- 1:N with `group_participants`

---

### group_participants

Stores group membership and roles.

**Schema**:
```sql
CREATE TABLE group_participants (
    group_jid TEXT NOT NULL,        -- Group JID
    user_jid TEXT NOT NULL,         -- Participant JID
    role TEXT,                      -- admin|superadmin|<empty>
    updated_at INTEGER NOT NULL,    -- Last update
    PRIMARY KEY (group_jid, user_jid),
    FOREIGN KEY (group_jid) REFERENCES groups(jid) ON DELETE CASCADE
);
```

**Fields**:
- `group_jid`: Reference to groups table
- `user_jid`: Participant's WhatsApp JID
- `role`: Participant role
  - `""` (empty): Regular member
  - `"admin"`: Group admin
  - `"superadmin"`: Group owner
- `updated_at`: Last update timestamp

**Cascade Delete**: When a group is deleted, all participants are automatically removed.

**Usage**:
```sql
-- Get all admins in a group
SELECT user_jid, role
FROM group_participants
WHERE group_jid = '1234567890-1640000000@g.us'
  AND role IN ('admin', 'superadmin');

-- Count members
SELECT COUNT(*) as member_count
FROM group_participants
WHERE group_jid = '1234567890-1640000000@g.us';
```

---

### contact_aliases

Stores user-defined contact aliases (local only).

**Schema**:
```sql
CREATE TABLE contact_aliases (
    jid TEXT PRIMARY KEY,           -- Contact JID
    alias TEXT NOT NULL,            -- User-defined alias
    notes TEXT,                     -- Optional notes
    updated_at INTEGER NOT NULL     -- Last update
);
```

**Fields**:
- `jid`: Contact JID
- `alias`: User-defined display name (overrides contact name)
- `notes`: Free-form notes field (reserved for future use)
- `updated_at`: Last modification timestamp

**Notes**:
- Local only (not synced to WhatsApp)
- Takes precedence over contact names in display
- One alias per contact

---

### contact_tags

Stores user-defined tags for contacts.

**Schema**:
```sql
CREATE TABLE contact_tags (
    jid TEXT NOT NULL,              -- Contact JID
    tag TEXT NOT NULL,              -- Tag name
    updated_at INTEGER NOT NULL,    -- Last update
    PRIMARY KEY (jid, tag)
);
```

**Fields**:
- `jid`: Contact JID
- `tag`: Tag identifier (e.g., "vip", "client", "family")
- `updated_at`: Last modification timestamp

**Constraints**:
- Composite primary key (one tag per contact)
- Multiple tags per contact supported

**Usage**:
```sql
-- Get all tags for a contact
SELECT tag FROM contact_tags
WHERE jid = '1234567890@s.whatsapp.net'
ORDER BY tag;

-- Find contacts with specific tag
SELECT DISTINCT jid FROM contact_tags
WHERE tag = 'vip';
```

---

### messages

Core table storing all message data and metadata.

**Schema**:
```sql
CREATE TABLE messages (
    rowid INTEGER PRIMARY KEY AUTOINCREMENT,
    chat_jid TEXT NOT NULL,         -- Chat identifier
    chat_name TEXT,                 -- Denormalized chat name
    msg_id TEXT NOT NULL,           -- WhatsApp message ID
    sender_jid TEXT,                -- Sender JID (empty if from_me)
    sender_name TEXT,               -- Denormalized sender name
    ts INTEGER NOT NULL,            -- Unix timestamp
    from_me INTEGER NOT NULL,       -- 1 if sent by us, 0 if received
    text TEXT,                      -- Message text content
    media_type TEXT,                -- image|video|audio|document
    media_caption TEXT,             -- Media caption
    filename TEXT,                  -- Media filename
    mime_type TEXT,                 -- MIME type
    direct_path TEXT,               -- WhatsApp CDN path
    media_key BLOB,                 -- Decryption key
    file_sha256 BLOB,               -- Plaintext SHA256
    file_enc_sha256 BLOB,           -- Encrypted SHA256
    file_length INTEGER,            -- File size in bytes
    local_path TEXT,                -- Local file path (if downloaded)
    downloaded_at INTEGER,          -- Download timestamp
    UNIQUE(chat_jid, msg_id),
    FOREIGN KEY (chat_jid) REFERENCES chats(jid) ON DELETE CASCADE
);
```

**Fields**:

**Core Fields**:
- `rowid`: Auto-increment primary key (used by FTS5)
- `chat_jid`: Chat this message belongs to
- `chat_name`: Denormalized chat name (for search)
- `msg_id`: WhatsApp message identifier (unique within chat)
- `sender_jid`: Who sent the message (empty for own messages)
- `sender_name`: Denormalized sender name (for search)
- `ts`: Message timestamp (Unix seconds)
- `from_me`: Boolean (1 = sent by us, 0 = received)

**Content Fields**:
- `text`: Text message content (or caption for media)
- `media_type`: Type of media attachment (if any)
  - `""`: No media (text only)
  - `"image"`: Image/photo
  - `"video"`: Video file
  - `"audio"`: Audio/voice message
  - `"document"`: Document/file
- `media_caption`: Caption for media messages

**Media Metadata** (for download):
- `filename`: Original or suggested filename
- `mime_type`: File MIME type (e.g., `image/jpeg`)
- `direct_path`: WhatsApp CDN path
- `media_key`: Encryption key (binary)
- `file_sha256`: SHA256 of decrypted file
- `file_enc_sha256`: SHA256 of encrypted file
- `file_length`: File size in bytes

**Download Tracking**:
- `local_path`: Absolute path to downloaded file
- `downloaded_at`: Unix timestamp when downloaded

**Constraints**:
- Unique constraint on `(chat_jid, msg_id)` - prevents duplicates
- Foreign key to `chats` with cascade delete

**Indexes**:
```sql
CREATE INDEX idx_messages_chat_ts ON messages(chat_jid, ts);
CREATE INDEX idx_messages_ts ON messages(ts);
```

**Upsert Pattern**:
```sql
INSERT INTO messages(chat_jid, msg_id, text, ts, from_me, ...)
VALUES (?, ?, ?, ?, ?, ...)
ON CONFLICT(chat_jid, msg_id) DO UPDATE SET
    text = excluded.text,
    media_key = CASE
        WHEN excluded.media_key IS NOT NULL AND length(excluded.media_key) > 0
        THEN excluded.media_key
        ELSE messages.media_key
    END,
    -- ... preserves existing non-NULL values
```

**Benefits of Upsert**:
- Idempotent: can insert same message multiple times safely
- History sync replays don't create duplicates
- Can update media metadata separately from text

---

## Full-Text Search (FTS5)

### messages_fts Virtual Table

**Schema**:
```sql
CREATE VIRTUAL TABLE messages_fts USING fts5(
    text,                           -- Message text
    media_caption,                  -- Media captions
    filename,                       -- File names
    chat_name,                      -- Denormalized chat name
    sender_name                     -- Denormalized sender name
);
```

**Design Decisions**:

1. **External Content Table**:
   - FTS5 table has same `rowid` as `messages` table
   - Triggers keep tables in sync
   - FTS table only stores indexed content, not source data

2. **Indexed Fields**:
   - `text`: Primary message content
   - `media_caption`: Captions on photos/videos/docs
   - `filename`: Document and file names
   - `chat_name`: Enables searching by chat/contact name
   - `sender_name`: Enables searching by sender

3. **Not Indexed**:
   - Timestamps (use WHERE clause filters)
   - JIDs (use exact match filters)
   - Media metadata (binary data)

### FTS5 Features

**Ranking**: BM25 algorithm (Okapi BM25)
```sql
SELECT * FROM messages_fts
WHERE messages_fts MATCH 'hello'
ORDER BY bm25(messages_fts)
LIMIT 50;
```

**Snippet Generation**:
```sql
SELECT snippet(messages_fts, 0, '[', ']', '…', 12) as snippet
FROM messages_fts
WHERE messages_fts MATCH 'hello';
-- Result: "... say [hello] to everyone ..."
```

**Query Operators**:
```
hello                   -- Single term
"hello world"           -- Phrase search
hello AND world         -- Both terms required
hello OR world          -- Either term
hello NOT spam          -- Exclude term
hel*                    -- Prefix search
```

### FTS Performance

**Index Size**: ~30% of message data size

**Query Performance**:
- Simple term: < 10ms
- Complex boolean: < 50ms
- Large result sets: Limited by LIMIT clause

**Index Update**: Automatic via triggers (real-time)

### Fallback to LIKE

If FTS5 is unavailable (SQLite compiled without FTS support):

```sql
-- Fallback query
SELECT * FROM messages
WHERE LOWER(text) LIKE LOWER('%hello%')
   OR LOWER(media_caption) LIKE LOWER('%hello%')
   OR LOWER(filename) LIKE LOWER('%hello%')
ORDER BY ts DESC;
```

**Performance**: 100-1000x slower for large datasets

**Detection**:
```go
// Store initialization
if ftsErr != nil {
    d.ftsEnabled = false  // Falls back to LIKE
}
```

---

## Indexes & Performance

### Existing Indexes

**messages Table**:
```sql
-- Composite index for chat message listings
CREATE INDEX idx_messages_chat_ts ON messages(chat_jid, ts);

-- Global timestamp index for recent messages
CREATE INDEX idx_messages_ts ON messages(ts);
```

**Index Usage**:

1. **idx_messages_chat_ts**:
   - Query: List messages in a chat
   - Sort: By timestamp (chronological or reverse)
   - Example: `SELECT * FROM messages WHERE chat_jid = ? ORDER BY ts DESC`

2. **idx_messages_ts**:
   - Query: Global recent messages
   - Sort: By timestamp
   - Example: `SELECT * FROM messages ORDER BY ts DESC LIMIT 50`

### Index Strategy

**Philosophy**: Minimal indexing, rely on FTS5 for search

**Why Minimal Indexes?**:
- Indexes increase write overhead
- WAL mode reduces read/write conflicts
- Most queries are FTS-based or simple PK lookups
- Messages are mostly append-only

**Not Indexed**:
- `sender_jid`: Low selectivity, covered by FTS
- `from_me`: Binary flag, not selective enough
- `media_type`: Low cardinality, filter after FTS

### Query Performance

**Fast Queries** (indexed):
```sql
-- Single chat messages (uses idx_messages_chat_ts)
SELECT * FROM messages
WHERE chat_jid = '1234567890@s.whatsapp.net'
ORDER BY ts DESC;

-- Recent messages globally (uses idx_messages_ts)
SELECT * FROM messages
ORDER BY ts DESC
LIMIT 100;

-- Full-text search (uses messages_fts)
SELECT * FROM messages_fts
WHERE messages_fts MATCH 'hello'
ORDER BY bm25(messages_fts);
```

**Slower Queries** (table scan):
```sql
-- Filter by sender (not indexed)
SELECT * FROM messages
WHERE sender_jid = '1234567890@s.whatsapp.net';

-- Media-only messages (not indexed)
SELECT * FROM messages
WHERE media_type = 'image';
```

**Optimization**: Add index if needed:
```sql
-- If frequently filtering by sender
CREATE INDEX idx_messages_sender ON messages(sender_jid);
```

---

## Data Models

### Go Structs

**Chat**:
```go
type Chat struct {
    JID           string
    Kind          string
    Name          string
    LastMessageTS time.Time
}
```

**Message**:
```go
type Message struct {
    ChatJID   string
    ChatName  string
    MsgID     string
    SenderJID string
    Timestamp time.Time
    FromMe    bool
    Text      string
    MediaType string
    Snippet   string  // Only populated for FTS search results
}
```

**Contact**:
```go
type Contact struct {
    JID       string
    Phone     string
    Name      string    // Best available name
    Alias     string    // User-defined alias
    Tags      []string
    UpdatedAt time.Time
}
```

**Group**:
```go
type Group struct {
    JID       string
    Name      string
    OwnerJID  string
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

**MediaDownloadInfo**:
```go
type MediaDownloadInfo struct {
    ChatJID       string
    ChatName      string
    MsgID         string
    MediaType     string
    Filename      string
    MimeType      string
    DirectPath    string
    MediaKey      []byte
    FileSHA256    []byte
    FileEncSHA256 []byte
    FileLength    uint64
    LocalPath     string
    DownloadedAt  time.Time
}
```

---

## Triggers

### FTS5 Sync Triggers

**Insert Trigger**:
```sql
CREATE TRIGGER messages_ai AFTER INSERT ON messages BEGIN
    INSERT INTO messages_fts(rowid, text, media_caption, filename, chat_name, sender_name)
    VALUES (
        new.rowid,
        COALESCE(new.text, ''),
        COALESCE(new.media_caption, ''),
        COALESCE(new.filename, ''),
        COALESCE(new.chat_name, ''),
        COALESCE(new.sender_name, '')
    );
END;
```

**Delete Trigger**:
```sql
CREATE TRIGGER messages_ad AFTER DELETE ON messages BEGIN
    DELETE FROM messages_fts WHERE rowid = old.rowid;
END;
```

**Update Trigger**:
```sql
CREATE TRIGGER messages_au AFTER UPDATE ON messages BEGIN
    DELETE FROM messages_fts WHERE rowid = old.rowid;
    INSERT INTO messages_fts(rowid, text, media_caption, filename, chat_name, sender_name)
    VALUES (
        new.rowid,
        COALESCE(new.text, ''),
        COALESCE(new.media_caption, ''),
        COALESCE(new.filename, ''),
        COALESCE(new.chat_name, ''),
        COALESCE(new.sender_name, '')
    );
END;
```

**Why Delete+Insert for Update?**:
- FTS5 doesn't support direct updates
- Delete old entry, insert new entry
- Ensures index stays in sync

**Performance Impact**:
- Triggers execute within same transaction
- Minimal overhead (FTS5 is optimized)
- Index updates are batched in WAL mode

---

## Queries & Operations

### Common Query Patterns

**List Recent Chats**:
```sql
SELECT jid, kind, name, last_message_ts
FROM chats
ORDER BY last_message_ts DESC
LIMIT 50;
```

**Search Messages**:
```sql
-- FTS5
SELECT m.chat_jid, m.msg_id, m.text, m.ts,
       snippet(f, 0, '[', ']', '…', 12) as snippet
FROM messages_fts f
JOIN messages m ON f.rowid = m.rowid
WHERE f MATCH 'hello world'
ORDER BY bm25(f)
LIMIT 50;

-- With filters
SELECT m.* FROM messages_fts f
JOIN messages m ON f.rowid = m.rowid
WHERE f MATCH 'hello'
  AND m.chat_jid = '1234567890@s.whatsapp.net'
  AND m.ts > 1640000000
ORDER BY bm25(f);
```

**Get Message Context**:
```sql
-- Get 5 messages before and after a specific message
WITH target AS (
    SELECT ts FROM messages
    WHERE chat_jid = ? AND msg_id = ?
)
SELECT * FROM (
    SELECT * FROM messages
    WHERE chat_jid = ? AND ts < (SELECT ts FROM target)
    ORDER BY ts DESC LIMIT 5
) UNION ALL
SELECT * FROM messages
WHERE chat_jid = ? AND msg_id = ?
UNION ALL
SELECT * FROM messages
WHERE chat_jid = ? AND ts > (SELECT ts FROM target)
ORDER BY ts ASC LIMIT 5;
```

**Contact Search**:
```sql
SELECT c.jid, c.phone,
       COALESCE(a.alias, c.full_name, c.push_name, c.jid) as display_name
FROM contacts c
LEFT JOIN contact_aliases a ON a.jid = c.jid
WHERE LOWER(display_name) LIKE LOWER('%john%')
   OR LOWER(c.phone) LIKE LOWER('%john%')
ORDER BY display_name;
```

---

## Migration Strategy

### Current Version: 1.0

Schema is initialized on first run. No migrations needed yet.

### Future Migrations

**Approach**: SQL migration files

**Example Migration** (v1 → v2):
```sql
-- migrations/002_add_message_status.sql
ALTER TABLE messages ADD COLUMN status TEXT DEFAULT 'sent';
CREATE INDEX idx_messages_status ON messages(status);
```

**Execution**:
1. Track schema version in metadata table
2. Apply migrations sequentially
3. Use transactions for atomicity
4. Backup before migration

**Schema Version Table**:
```sql
CREATE TABLE schema_version (
    version INTEGER PRIMARY KEY,
    applied_at INTEGER NOT NULL
);
```

### Backup Strategy

**Before Migration**:
```bash
# Create backup
cp wacli.db wacli.db.backup

# Verify backup
sqlite3 wacli.db.backup "PRAGMA integrity_check;"
```

**Rollback**: Restore from backup

---

## Performance Tuning

### PRAGMA Settings

**Current Configuration**:
```sql
PRAGMA journal_mode=WAL;         -- Write-Ahead Logging
PRAGMA synchronous=NORMAL;       -- Balanced safety/speed
PRAGMA temp_store=MEMORY;        -- Temp tables in RAM
PRAGMA foreign_keys=ON;          -- FK enforcement
```

**Tuning Options**:

**For Write-Heavy Workload**:
```sql
PRAGMA wal_autocheckpoint=1000;  -- Checkpoint every 1000 pages
PRAGMA cache_size=-64000;        -- 64MB page cache
```

**For Read-Heavy Workload**:
```sql
PRAGMA cache_size=-128000;       -- 128MB page cache
PRAGMA mmap_size=268435456;      -- 256MB memory-mapped I/O
```

**For Maximum Safety** (slower):
```sql
PRAGMA synchronous=FULL;         -- Full fsync on commit
```

### Vacuum & Maintenance

**Periodic Vacuum**:
```sql
-- Reclaim space and defragment
VACUUM;

-- Rebuild indexes
REINDEX;

-- Update statistics
ANALYZE;
```

**Frequency**: Monthly for active instances

**Optimize FTS**:
```sql
-- Merge FTS segments
INSERT INTO messages_fts(messages_fts) VALUES('optimize');
```

---

## Security Considerations

### File Permissions

**Database Files**:
```bash
chmod 600 wacli.db session.db     # Owner read/write only
chmod 700 /data                     # Owner access only
```

**Why**:
- Contains message content (privacy)
- session.db contains encryption keys (security)

### SQL Injection

**Safe Practices**:
```go
// GOOD: Use parameterized queries
db.Query("SELECT * FROM messages WHERE chat_jid = ?", chatJID)

// BAD: String concatenation
db.Query("SELECT * FROM messages WHERE chat_jid = '" + chatJID + "'")
```

All database operations use parameterized queries (protection built-in).

### Encryption at Rest

**Current**: SQLite databases are not encrypted

**Options**:
1. **Filesystem Encryption**: Use encrypted filesystem (LUKS, FileVault, BitLocker)
2. **SQLite Encryption Extension** (SEE): Commercial extension
3. **SQLCipher**: Open-source alternative

**Recommendation**: Filesystem-level encryption for production deployments containing sensitive data.

---

## Troubleshooting

### Database Locked Errors

**Symptom**: `database is locked` error

**Causes**:
- Another process holds exclusive lock
- WAL checkpoint in progress
- Deadlock (rare)

**Solutions**:
```sql
-- Check WAL file size
PRAGMA wal_checkpoint(TRUNCATE);

-- Increase busy timeout
PRAGMA busy_timeout=5000;  -- 5 seconds
```

### FTS5 Not Available

**Symptom**: Search falls back to LIKE queries

**Check**:
```bash
sqlite3 wacli.db "PRAGMA compile_options;" | grep FTS
```

**Solution**: Rebuild SQLite with FTS5 support or use pre-compiled binary

### Corrupted Database

**Detection**:
```sql
PRAGMA integrity_check;
```

**Recovery**:
```bash
# Dump to SQL
sqlite3 wacli.db .dump > backup.sql

# Create new database
rm wacli.db
sqlite3 wacli.db < backup.sql
```

---

This schema documentation covers the complete database design. For API usage of these tables, see the [API Reference](02-API-REFERENCE.md).
