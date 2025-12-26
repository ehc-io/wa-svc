# WhatsApp Service - Practical API Examples

Real-world examples for common use cases using curl, JavaScript, and Python.

## Setup

Set your API configuration:

```bash
# Environment variables for all examples
export API_URL="http://localhost:8080"
export API_KEY="your-secret-api-key"
```

---

## Authentication

### Complete Authentication Flow

**curl:**
```bash
# 1. Start authentication
curl -X POST "$API_URL/auth/init" \
  -H "Authorization: Bearer $API_KEY"

# 2. Get QR code (poll until available)
curl "$API_URL/auth/qr" \
  -H "Authorization: Bearer $API_KEY" | jq '.qr_image'

# 3. Check if authenticated (poll until true)
curl "$API_URL/auth/status" \
  -H "Authorization: Bearer $API_KEY" | jq '.authenticated'
```

**JavaScript:**
```javascript
const API_URL = 'http://localhost:8080';
const API_KEY = 'your-secret-api-key';

async function authenticate() {
  // Start auth
  await fetch(`${API_URL}/auth/init`, {
    method: 'POST',
    headers: { 'Authorization': `Bearer ${API_KEY}` }
  });

  // Poll for QR code
  let qrImage = null;
  while (!qrImage) {
    const res = await fetch(`${API_URL}/auth/qr`, {
      headers: { 'Authorization': `Bearer ${API_KEY}` }
    });
    const data = await res.json();
    if (data.qr_image) {
      qrImage = data.qr_image;
      console.log('Scan this QR code with WhatsApp');
      // Display in browser: <img src="${qrImage}" />
    } else {
      await new Promise(r => setTimeout(r, 2000));
    }
  }

  // Poll for authentication
  while (true) {
    const res = await fetch(`${API_URL}/auth/status`, {
      headers: { 'Authorization': `Bearer ${API_KEY}` }
    });
    const data = await res.json();
    if (data.authenticated) {
      console.log('Connected to WhatsApp!');
      return true;
    }
    await new Promise(r => setTimeout(r, 2000));
  }
}
```

**Python:**
```python
import requests
import time

API_URL = 'http://localhost:8080'
API_KEY = 'your-secret-api-key'
HEADERS = {'Authorization': f'Bearer {API_KEY}'}

def authenticate():
    # Start auth
    requests.post(f'{API_URL}/auth/init', headers=HEADERS)

    # Poll for QR code
    qr_image = None
    while not qr_image:
        res = requests.get(f'{API_URL}/auth/qr', headers=HEADERS)
        data = res.json()
        if 'qr_image' in data:
            qr_image = data['qr_image']
            print('QR Code available - scan with WhatsApp')
            # Save or display the base64 PNG image
        else:
            time.sleep(2)

    # Poll for authentication
    while True:
        res = requests.get(f'{API_URL}/auth/status', headers=HEADERS)
        data = res.json()
        if data.get('authenticated'):
            print('Connected to WhatsApp!')
            return True
        time.sleep(2)
```

---

## Sending Messages

### Send Text Message

**curl:**
```bash
curl -X POST "$API_URL/messages/text" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "1234567890",
    "message": "Hello from WhatsApp Service!"
  }'
```

**JavaScript:**
```javascript
async function sendText(to, message) {
  const res = await fetch(`${API_URL}/messages/text`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${API_KEY}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ to, message })
  });

  if (!res.ok) {
    const error = await res.json();
    throw new Error(error.error);
  }

  const data = await res.json();
  return data.message_id;
}

// Usage
const msgId = await sendText('1234567890', 'Hello from WhatsApp Service!');
console.log('Sent:', msgId);
```

**Python:**
```python
def send_text(to: str, message: str) -> str:
    res = requests.post(
        f'{API_URL}/messages/text',
        headers={**HEADERS, 'Content-Type': 'application/json'},
        json={'to': to, 'message': message}
    )
    res.raise_for_status()
    return res.json()['message_id']

# Usage
msg_id = send_text('1234567890', 'Hello from WhatsApp Service!')
print(f'Sent: {msg_id}')
```

### Send Image with Caption

**curl:**
```bash
# From file (base64 encoded)
curl -X POST "$API_URL/messages/file" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d "{
    \"to\": \"1234567890\",
    \"file_data\": \"$(base64 -i photo.jpg)\",
    \"filename\": \"photo.jpg\",
    \"caption\": \"Check out this image!\",
    \"mime_type\": \"image/jpeg\"
  }"

# From URL
curl -X POST "$API_URL/messages/file" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "to": "1234567890",
    "file_url": "https://example.com/image.jpg",
    "caption": "Image from URL"
  }'
```

**JavaScript:**
```javascript
async function sendImage(to, imagePath, caption = '') {
  // Read file and convert to base64
  const fs = require('fs');
  const fileBuffer = fs.readFileSync(imagePath);
  const base64Data = fileBuffer.toString('base64');
  const filename = imagePath.split('/').pop();

  const res = await fetch(`${API_URL}/messages/file`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${API_KEY}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      to,
      file_data: base64Data,
      filename,
      caption,
      mime_type: 'image/jpeg'
    })
  });

  return (await res.json()).message_id;
}

// Browser: Send from file input
async function sendImageFromInput(to, fileInput, caption = '') {
  const file = fileInput.files[0];
  const base64 = await new Promise(resolve => {
    const reader = new FileReader();
    reader.onload = () => resolve(reader.result.split(',')[1]);
    reader.readAsDataURL(file);
  });

  const res = await fetch(`${API_URL}/messages/file`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${API_KEY}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      to,
      file_data: base64,
      filename: file.name,
      caption,
      mime_type: file.type
    })
  });

  return (await res.json()).message_id;
}
```

**Python:**
```python
import base64

def send_image(to: str, image_path: str, caption: str = '') -> str:
    with open(image_path, 'rb') as f:
        file_data = base64.b64encode(f.read()).decode()

    res = requests.post(
        f'{API_URL}/messages/file',
        headers={**HEADERS, 'Content-Type': 'application/json'},
        json={
            'to': to,
            'file_data': file_data,
            'filename': image_path.split('/')[-1],
            'caption': caption,
            'mime_type': 'image/jpeg'
        }
    )
    res.raise_for_status()
    return res.json()['message_id']

# Usage
msg_id = send_image('1234567890', 'photo.jpg', 'Check this out!')
```

### Send Document (PDF, etc.)

**curl:**
```bash
curl -X POST "$API_URL/messages/file" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d "{
    \"to\": \"1234567890\",
    \"file_data\": \"$(base64 -i document.pdf)\",
    \"filename\": \"report.pdf\",
    \"caption\": \"Monthly report\",
    \"mime_type\": \"application/pdf\"
  }"
```

**Python:**
```python
def send_document(to: str, file_path: str, caption: str = '') -> str:
    import mimetypes

    mime_type, _ = mimetypes.guess_type(file_path)
    with open(file_path, 'rb') as f:
        file_data = base64.b64encode(f.read()).decode()

    res = requests.post(
        f'{API_URL}/messages/file',
        headers={**HEADERS, 'Content-Type': 'application/json'},
        json={
            'to': to,
            'file_data': file_data,
            'filename': file_path.split('/')[-1],
            'caption': caption,
            'mime_type': mime_type or 'application/octet-stream'
        }
    )
    res.raise_for_status()
    return res.json()['message_id']
```

---

## Searching Messages

### Basic Search

**curl:**
```bash
# Search for a word
curl "$API_URL/search?q=meeting&limit=20" \
  -H "Authorization: Bearer $API_KEY"

# Search exact phrase
curl "$API_URL/search?q=\"project%20deadline\"&limit=20" \
  -H "Authorization: Bearer $API_KEY"
```

**JavaScript:**
```javascript
async function searchMessages(query, limit = 50) {
  const res = await fetch(
    `${API_URL}/search?q=${encodeURIComponent(query)}&limit=${limit}`,
    { headers: { 'Authorization': `Bearer ${API_KEY}` } }
  );
  const data = await res.json();
  return data.messages;
}

// Usage
const results = await searchMessages('meeting tomorrow');
results.forEach(msg => {
  console.log(`[${msg.chat_name}] ${msg.text}`);
});
```

**Python:**
```python
def search_messages(query: str, limit: int = 50) -> list:
    res = requests.get(
        f'{API_URL}/search',
        headers=HEADERS,
        params={'q': query, 'limit': limit}
    )
    res.raise_for_status()
    return res.json()['messages']

# Usage
results = search_messages('meeting tomorrow')
for msg in results:
    print(f"[{msg['chat_name']}] {msg['text']}")
```

### Advanced Search Examples

```bash
# Search with prefix matching
curl "$API_URL/search?q=meet*" -H "Authorization: Bearer $API_KEY"

# Boolean OR search
curl "$API_URL/search?q=meeting%20OR%20conference" -H "Authorization: Bearer $API_KEY"

# Find messages with attachments (by filename)
curl "$API_URL/search?q=.pdf" -H "Authorization: Bearer $API_KEY"
```

---

## Chat Management

### List Recent Chats

**curl:**
```bash
curl "$API_URL/chats?limit=20" \
  -H "Authorization: Bearer $API_KEY"
```

**JavaScript:**
```javascript
async function getChats(limit = 50) {
  const res = await fetch(`${API_URL}/chats?limit=${limit}`, {
    headers: { 'Authorization': `Bearer ${API_KEY}` }
  });
  return (await res.json()).chats;
}

// Usage
const chats = await getChats();
chats.forEach(chat => {
  console.log(`${chat.name} (${chat.kind}): ${chat.last_message_ts}`);
});
```

### Get Messages from a Chat

**curl:**
```bash
# Replace JID with actual chat JID
curl "$API_URL/chats/1234567890@s.whatsapp.net/messages?limit=50" \
  -H "Authorization: Bearer $API_KEY"
```

**JavaScript:**
```javascript
async function getChatMessages(chatJid, limit = 50) {
  const res = await fetch(
    `${API_URL}/chats/${encodeURIComponent(chatJid)}/messages?limit=${limit}`,
    { headers: { 'Authorization': `Bearer ${API_KEY}` } }
  );
  return (await res.json()).messages;
}

// Usage
const messages = await getChatMessages('1234567890@s.whatsapp.net');
messages.forEach(msg => {
  const direction = msg.from_me ? 'You' : msg.sender_jid;
  console.log(`${direction}: ${msg.text}`);
});
```

---

## Contact Management

### Search Contacts

**curl:**
```bash
curl "$API_URL/contacts?q=john" \
  -H "Authorization: Bearer $API_KEY"
```

### Import Contacts from WhatsApp

**curl:**
```bash
curl -X POST "$API_URL/contacts/refresh" \
  -H "Authorization: Bearer $API_KEY"
```

### Set Contact Alias

**curl:**
```bash
curl -X PUT "$API_URL/contacts/1234567890@s.whatsapp.net/alias" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"alias": "Johnny"}'
```

### Add Tag to Contact

**curl:**
```bash
curl -X POST "$API_URL/contacts/1234567890@s.whatsapp.net/tags" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"tag": "vip"}'
```

**Python:**
```python
def set_contact_alias(jid: str, alias: str):
    res = requests.put(
        f'{API_URL}/contacts/{jid}/alias',
        headers={**HEADERS, 'Content-Type': 'application/json'},
        json={'alias': alias}
    )
    res.raise_for_status()
    return res.json()

def add_contact_tag(jid: str, tag: str):
    res = requests.post(
        f'{API_URL}/contacts/{jid}/tags',
        headers={**HEADERS, 'Content-Type': 'application/json'},
        json={'tag': tag}
    )
    res.raise_for_status()
    return res.json()

# Usage
set_contact_alias('1234567890@s.whatsapp.net', 'Johnny')
add_contact_tag('1234567890@s.whatsapp.net', 'client')
```

---

## Group Management

### List Groups

**curl:**
```bash
curl "$API_URL/groups" \
  -H "Authorization: Bearer $API_KEY"
```

### Get Group Details

**curl:**
```bash
curl "$API_URL/groups/1234567890-1640000000@g.us" \
  -H "Authorization: Bearer $API_KEY"
```

### Add Members to Group

**curl:**
```bash
curl -X POST "$API_URL/groups/1234567890-1640000000@g.us/participants" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "action": "add",
    "users": ["1111111111", "2222222222"]
  }'
```

### Promote to Admin

**curl:**
```bash
curl -X POST "$API_URL/groups/1234567890-1640000000@g.us/participants" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "action": "promote",
    "users": ["1111111111"]
  }'
```

### Get Group Invite Link

**curl:**
```bash
curl "$API_URL/groups/1234567890-1640000000@g.us/invite" \
  -H "Authorization: Bearer $API_KEY"
```

**JavaScript:**
```javascript
async function manageGroupMembers(groupJid, action, users) {
  const res = await fetch(`${API_URL}/groups/${encodeURIComponent(groupJid)}/participants`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${API_KEY}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ action, users })
  });
  return res.json();
}

// Add members
await manageGroupMembers('1234567890-1640000000@g.us', 'add', ['1111111111', '2222222222']);

// Remove member
await manageGroupMembers('1234567890-1640000000@g.us', 'remove', ['3333333333']);

// Promote to admin
await manageGroupMembers('1234567890-1640000000@g.us', 'promote', ['1111111111']);

// Demote from admin
await manageGroupMembers('1234567890-1640000000@g.us', 'demote', ['1111111111']);
```

---

## Media Handling

### Download Media from a Message

**curl:**
```bash
# Check if media is available
curl "$API_URL/media/1234567890@s.whatsapp.net/3EB0C6C6F7F75F9C5B8E" \
  -H "Authorization: Bearer $API_KEY"

# Download the media
curl -X POST "$API_URL/media/1234567890@s.whatsapp.net/3EB0C6C6F7F75F9C5B8E/download" \
  -H "Authorization: Bearer $API_KEY"

# Get the file (after download)
curl "$API_URL/media/1234567890@s.whatsapp.net/3EB0C6C6F7F75F9C5B8E" \
  -H "Authorization: Bearer $API_KEY" \
  --output downloaded_file.jpg
```

**JavaScript:**
```javascript
async function downloadMedia(chatJid, msgId) {
  // Check media info
  const infoRes = await fetch(
    `${API_URL}/media/${encodeURIComponent(chatJid)}/${msgId}`,
    { headers: { 'Authorization': `Bearer ${API_KEY}` } }
  );
  const info = await infoRes.json();

  if (!info.downloaded) {
    // Download the media
    await fetch(
      `${API_URL}/media/${encodeURIComponent(chatJid)}/${msgId}/download`,
      {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${API_KEY}` }
      }
    );
  }

  // Get the file
  const fileRes = await fetch(
    `${API_URL}/media/${encodeURIComponent(chatJid)}/${msgId}`,
    { headers: { 'Authorization': `Bearer ${API_KEY}` } }
  );

  return fileRes.blob();
}
```

**Python:**
```python
def download_media(chat_jid: str, msg_id: str, output_path: str):
    # Check media info
    info_res = requests.get(
        f'{API_URL}/media/{chat_jid}/{msg_id}',
        headers=HEADERS
    )
    info = info_res.json()

    if not info.get('downloaded'):
        # Download the media
        requests.post(
            f'{API_URL}/media/{chat_jid}/{msg_id}/download',
            headers=HEADERS
        )

    # Get the file
    file_res = requests.get(
        f'{API_URL}/media/{chat_jid}/{msg_id}',
        headers=HEADERS
    )

    with open(output_path, 'wb') as f:
        f.write(file_res.content)

    return output_path

# Usage
download_media('1234567890@s.whatsapp.net', '3EB0C6C6F7F75F9C5B8E', 'photo.jpg')
```

---

## History & Sync

### Request Older Messages (Backfill)

**curl:**
```bash
curl -X POST "$API_URL/history/backfill" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "chat_jid": "1234567890@s.whatsapp.net",
    "count": 100,
    "requests": 3
  }'
```

**Python:**
```python
def backfill_chat(chat_jid: str, count: int = 50, requests: int = 1):
    res = requests.post(
        f'{API_URL}/history/backfill',
        headers={**HEADERS, 'Content-Type': 'application/json'},
        json={
            'chat_jid': chat_jid,
            'count': count,
            'requests': requests
        },
        timeout=300  # 5 minutes - backfill can be slow
    )
    res.raise_for_status()
    return res.json()

# Usage - get older messages for a chat
result = backfill_chat('1234567890@s.whatsapp.net', count=100, requests=2)
print(f"Added {result['message']}")
```

### Check Sync Status

**curl:**
```bash
curl "$API_URL/sync/status" \
  -H "Authorization: Bearer $API_KEY"
```

---

## Webhook Handler Examples

### Express.js (Node.js)

```javascript
const express = require('express');
const crypto = require('crypto');

const app = express();
const WEBHOOK_SECRET = process.env.WEBHOOK_SECRET;

// Parse raw body for HMAC verification
app.use('/webhook', express.raw({ type: 'application/json' }));

function verifySignature(body, signature) {
  const hmac = crypto.createHmac('sha256', WEBHOOK_SECRET);
  hmac.update(body);
  const expected = 'sha256=' + hmac.digest('hex');
  return crypto.timingSafeEqual(Buffer.from(signature), Buffer.from(expected));
}

app.post('/webhook', (req, res) => {
  const signature = req.headers['x-webhook-signature'];

  if (WEBHOOK_SECRET && !verifySignature(req.body, signature)) {
    return res.status(401).send('Invalid signature');
  }

  const event = JSON.parse(req.body.toString());

  switch (event.type) {
    case 'message.received':
      const msg = event.data;
      console.log(`[${msg.chat_name}] ${msg.from_me ? 'You' : msg.sender_name}: ${msg.text}`);

      // Auto-reply example
      if (!msg.from_me && msg.text.toLowerCase() === 'hello') {
        sendText(msg.chat_jid, 'Hi there! How can I help you?');
      }
      break;

    default:
      console.log('Unknown event:', event.type);
  }

  res.status(200).send('OK');
});

app.listen(3000, () => console.log('Webhook server running on port 3000'));
```

### Flask (Python)

```python
from flask import Flask, request
import hmac
import hashlib
import os
import json

app = Flask(__name__)
WEBHOOK_SECRET = os.environ.get('WEBHOOK_SECRET', '')

def verify_signature(body: bytes, signature: str) -> bool:
    if not WEBHOOK_SECRET:
        return True
    expected = 'sha256=' + hmac.new(
        WEBHOOK_SECRET.encode(),
        body,
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(signature, expected)

@app.route('/webhook', methods=['POST'])
def webhook():
    signature = request.headers.get('X-Webhook-Signature', '')

    if not verify_signature(request.data, signature):
        return 'Invalid signature', 401

    event = request.json

    if event['type'] == 'message.received':
        msg = event['data']
        direction = 'You' if msg['from_me'] else msg.get('sender_name', 'Unknown')
        print(f"[{msg['chat_name']}] {direction}: {msg['text']}")

        # Auto-reply example
        if not msg['from_me'] and msg['text'].lower() == 'hello':
            send_text(msg['chat_jid'], 'Hi there! How can I help you?')

    return 'OK', 200

if __name__ == '__main__':
    app.run(port=3000)
```

---

## Health Monitoring

### Check Service Health

**curl:**
```bash
# Simple health check
curl "$API_URL/health"

# Detailed diagnostics
curl "$API_URL/doctor" \
  -H "Authorization: Bearer $API_KEY"

# Quick stats
curl "$API_URL/stats" \
  -H "Authorization: Bearer $API_KEY"
```

**JavaScript:**
```javascript
async function checkHealth() {
  const healthRes = await fetch(`${API_URL}/health`);
  const health = await healthRes.json();

  if (health.status !== 'ok' || !health.ready) {
    console.error('Service unhealthy:', health);
    return false;
  }

  return true;
}

async function getDiagnostics() {
  const res = await fetch(`${API_URL}/doctor`, {
    headers: { 'Authorization': `Bearer ${API_KEY}` }
  });
  return res.json();
}

// Usage
const diagnostics = await getDiagnostics();
console.log(`Messages: ${diagnostics.message_count}`);
console.log(`Chats: ${diagnostics.chat_count}`);
console.log(`FTS enabled: ${diagnostics.fts_enabled}`);
```

---

## Complete Integration Example

### Auto-Responder Bot

```javascript
const express = require('express');
const crypto = require('crypto');
const fetch = require('node-fetch');

const app = express();
const API_URL = process.env.API_URL || 'http://localhost:8080';
const API_KEY = process.env.API_KEY;
const WEBHOOK_SECRET = process.env.WEBHOOK_SECRET;

app.use('/webhook', express.raw({ type: 'application/json' }));

// Send message helper
async function sendMessage(to, text) {
  const res = await fetch(`${API_URL}/messages/text`, {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${API_KEY}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ to, message: text })
  });
  return res.json();
}

// Search messages helper
async function searchMessages(query) {
  const res = await fetch(`${API_URL}/search?q=${encodeURIComponent(query)}&limit=5`, {
    headers: { 'Authorization': `Bearer ${API_KEY}` }
  });
  return (await res.json()).messages;
}

// Webhook handler
app.post('/webhook', async (req, res) => {
  // Verify signature
  if (WEBHOOK_SECRET) {
    const sig = req.headers['x-webhook-signature'];
    const hmac = crypto.createHmac('sha256', WEBHOOK_SECRET);
    hmac.update(req.body);
    const expected = 'sha256=' + hmac.digest('hex');
    if (!crypto.timingSafeEqual(Buffer.from(sig), Buffer.from(expected))) {
      return res.status(401).send('Invalid signature');
    }
  }

  const event = JSON.parse(req.body.toString());

  if (event.type === 'message.received') {
    const msg = event.data;

    // Skip messages from self
    if (msg.from_me) {
      return res.send('OK');
    }

    const text = msg.text.toLowerCase();

    // Command handler
    if (text.startsWith('!')) {
      const [cmd, ...args] = text.slice(1).split(' ');

      switch (cmd) {
        case 'help':
          await sendMessage(msg.chat_jid,
            'Available commands:\n' +
            '!help - Show this message\n' +
            '!search <query> - Search messages\n' +
            '!ping - Check if bot is alive'
          );
          break;

        case 'search':
          const query = args.join(' ');
          if (query) {
            const results = await searchMessages(query);
            if (results.length > 0) {
              const response = results.map(r =>
                `[${r.chat_name}] ${r.text.slice(0, 50)}...`
              ).join('\n');
              await sendMessage(msg.chat_jid, `Found ${results.length} results:\n${response}`);
            } else {
              await sendMessage(msg.chat_jid, 'No messages found');
            }
          }
          break;

        case 'ping':
          await sendMessage(msg.chat_jid, 'Pong! Bot is alive.');
          break;
      }
    }
  }

  res.send('OK');
});

app.listen(3000, () => {
  console.log('Bot running on port 3000');
});
```

---

## Error Handling

### Common Error Codes

| Code | Meaning | Solution |
|------|---------|----------|
| `MISSING_TO` | Recipient not specified | Add `to` field |
| `MISSING_MESSAGE` | Message content empty | Add `message` field |
| `SEND_FAILED` | WhatsApp send failed | Check connection status |
| `NOT_FOUND` | Resource not found | Verify JID/ID exists |
| `ALREADY_AUTHENTICATED` | Already logged in | Use existing session |

### Error Handling Pattern

```javascript
async function safeApiCall(fn) {
  try {
    return await fn();
  } catch (error) {
    if (error.response) {
      const data = await error.response.json();
      console.error(`API Error: ${data.code} - ${data.error}`);

      // Handle specific errors
      switch (data.code) {
        case 'SEND_FAILED':
          // Check connection and retry
          const health = await checkHealth();
          if (!health) {
            console.log('Service disconnected, waiting for reconnect...');
          }
          break;
        case 'NOT_FOUND':
          // Resource doesn't exist
          break;
      }
    }
    throw error;
  }
}

// Usage
await safeApiCall(() => sendText('1234567890', 'Hello'));
```

```python
def safe_api_call(fn):
    try:
        return fn()
    except requests.HTTPError as e:
        data = e.response.json()
        print(f"API Error: {data.get('code')} - {data.get('error')}")

        if data.get('code') == 'SEND_FAILED':
            # Check connection status
            health = requests.get(f'{API_URL}/health').json()
            if health.get('status') != 'ok':
                print('Service unhealthy, waiting...')

        raise

# Usage
safe_api_call(lambda: send_text('1234567890', 'Hello'))
```
