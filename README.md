# Collab-Tool — Real-time Collaborative Markdown Editor

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org/)
[![React](https://img.shields.io/badge/React-18+-61DAFB?logo=react)](https://reactjs.org/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15+-4169E1?logo=postgresql)](https://www.postgresql.org/)
[![WebSocket](https://img.shields.io/badge/WebSocket-Real--time-010101)](https://developer.mozilla.org/en-US/docs/Web/API/WebSocket)

A real-time collaborative markdown editor. Multiple users can edit the same document simultaneously with live preview, user presence, and AI text suggestions.

---

## ✨ Features

| Feature | Status | Notes |
|---------|--------|-------|
| Real-time collaboration | ✅ | WebSocket broadcast per document room |
| Live markdown preview | ✅ | GFM, syntax highlighting, tables, task lists |
| User presence panel | ✅ | See who's editing in real time |
| AI text suggestions | ✅ | Markov chain trained on document content |
| Version-based OT | ✅ | Last-write-wins with sequential versioning |
| PostgreSQL persistence | ✅ | Auto-save + 30 s flush |
| Markdown toolbar | ✅ | Bold, italic, headings, code, lists, quotes |
| Word/char counter | ✅ | Live status bar |
| Editable username | ✅ | Persisted in sessionStorage |
| Shareable URL | ✅ | Copy link button |
| Split / editor / preview | ✅ | Layout toggle |
| Docker Compose | ✅ | One-command start |
| Responsive | ✅ | Mobile layout support |

---

## 📁 Project Structure

```
collab-editor/
├── backend/
│   ├── main.go          # HTTP server, WebSocket handler, REST API
│   ├── hub.go           # Per-document WebSocket rooms + presence
│   ├── document.go      # In-memory doc state, OT-lite version tracking
│   ├── database.go      # PostgreSQL layer (initDB, get, save, list)
│   ├── messages.go      # Inbound / outbound message types
│   ├── markov.go        # Bigram/trigram Markov chain for suggestions
│   ├── go.mod
│   ├── go.sum
│   └── Dockerfile
├── frontend/
│   ├── src/
│   │   ├── main.jsx         # React entry point
│   │   ├── App.jsx          # Root component, WS wiring
│   │   ├── Editor.jsx       # Textarea + toolbar + debounce
│   │   ├── Preview.jsx      # react-markdown renderer
│   │   ├── UsersPanel.jsx   # Online collaborators list
│   │   ├── Suggestions.jsx  # AI word chips
│   │   ├── index.css        # Full dark theme
│   │   └── utils/
│   │       └── websocket.js # Reconnecting WS manager
│   ├── index.html
│   ├── package.json
│   ├── vite.config.js
│   └── Dockerfile
├── docker-compose.yml
└── README.md
```

---

## 🚀 Quick Start

### Option A — Docker Compose (recommended)

```bash
git clone <your-repo>
cd collab-editor
docker-compose up --build
```

- Frontend: http://localhost:5173
- Backend API: http://localhost:8080

### Option B — Manual (local dev)

**Prerequisites:** Go 1.21+, Node 18+, PostgreSQL 15+

```bash
# 1. Database
sudo -u postgres psql << 'EOF'
CREATE USER editor_user WITH PASSWORD 'password123';
CREATE DATABASE collab_editor OWNER editor_user;
GRANT ALL PRIVILEGES ON DATABASE collab_editor TO editor_user;
\c collab_editor
GRANT ALL ON SCHEMA public TO editor_user;
EOF

# 2. Backend
cd backend
go mod download
go run .
# Server starts on :8080

# 3. Frontend (new terminal)
cd frontend
npm install
npm run dev
# Dev server starts on :5173
```

Open http://localhost:5173 — a random document ID is created for you. Share the URL to collaborate.

---

## 🌐 WebSocket Protocol

**Connect:**
```
ws://localhost:8080/ws?docId=<id>&clientId=<id>&username=<name>
```

**Client → Server:**
```json
{ "type": "edit",  "docId": "abc", "content": "# Hello", "clientId": "x1", "version": 4 }
{ "type": "join",  "docId": "abc", "clientId": "x1", "username": "Alice" }
```

**Server → Client:**
```json
{ "type": "init",     "docId": "abc", "content": "...", "version": 0,  "users": [...] }
{ "type": "update",   "docId": "abc", "content": "...", "version": 5,  "senderId": "x2" }
{ "type": "ack",      "docId": "abc", "version": 5 }
{ "type": "presence", "docId": "abc", "users": [{"clientId":"x1","username":"Alice"}] }
```

---

## 🔌 REST API

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/docs` | List all documents (with preview) |
| POST | `/api/docs` | Create a document `{"id":"...","content":"..."}` |
| GET | `/api/docs/:id` | Get document + online users |
| PUT | `/api/docs/:id` | Update document (REST fallback) |
| GET | `/api/suggest?prefix=...` | Get AI word suggestions |
| GET | `/health` | Health check |

---

## 🧠 How It Works

### Real-time Sync Flow

1. Client connects via WebSocket → receives `init` with current content + version
2. User types → debounced (300 ms) `edit` message sent to server
3. Server applies **OT-lite**: accepts edit, increments version
4. Server broadcasts `update` to all **other** clients in the same document room
5. Server sends `ack` (with new version) back to the sender
6. Client version ref is updated on `ack` — keeps local version in sync
7. Document is flushed to PostgreSQL every 30 s (and immediately on each edit in the background)

### Conflict Resolution (OT-lite)

- Every edit carries the client's current `version`
- Server maintains a monotonically increasing version per document
- If client version ≥ server version → edit applied directly
- If client version < server version (behind) → last-write-wins, version still incremented
- Full CRDTs are not needed for a synchronous collaborative editor where users can see each other's cursors

### Markov Chain AI

- Trained on a built-in markdown corpus (headers, lists, paragraphs, code patterns)
- Re-trained on each document's content as it's loaded and updated
- Bigram (1-word context) + trigram (2-word context) for better quality
- Suggestions appear as clickable chips below the editor

---

## ⚙️ Environment Variables

```env
# backend/.env  (or export in shell)
DATABASE_URL=postgres://editor_user:password123@localhost/collab_editor?sslmode=disable
PORT=8080
CORS_ORIGINS=http://localhost:5173,https://yourdomain.com
```

---

## 📈 Roadmap

- [ ] JWT authentication + named accounts
- [ ] Document version history & rollback
- [ ] Cursor position sharing (show collaborator carets)
- [ ] Export to PDF / HTML
- [ ] Comment threads
- [ ] Offline editing with sync-on-reconnect
- [ ] Full CRDT (Yjs or Automerge) for true conflict-free merging
