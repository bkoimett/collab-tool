package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		allowed := os.Getenv("CORS_ORIGINS")
		if allowed == "" || allowed == "*" {
			return true
		}
		origin := r.Header.Get("Origin")
		for _, o := range strings.Split(allowed, ",") {
			if strings.TrimSpace(o) == origin {
				return true
			}
		}
		return false
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var (
	hub    *Hub
	docMgr *DocumentManager
	sqlDB  *sql.DB
)

func main() {
	sqlDB = initDB()
	defer sqlDB.Close()

	docMgr = newDocumentManager()
	hub = newHub()
	go hub.run()

	// Periodic flush of in-memory docs to Postgres every 30 s
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			hub.mu.RLock()
			for docID := range hub.rooms {
				content, version := docMgr.Get(docID)
				if err := dbSaveDocument(sqlDB, docID, content, version); err != nil {
					log.Printf("[persist] %s: %v", docID, err)
				}
			}
			hub.mu.RUnlock()
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", wsHandler)
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/api/docs", docsHandler)
	mux.HandleFunc("/api/docs/", docByIDHandler)
	mux.HandleFunc("/api/suggest", suggestHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("🚀  Server on http://localhost:%s\n", port)
	fmt.Printf("    WS   ws://localhost:%s/ws?docId=<id>&username=<name>\n", port)
	fmt.Printf("    API  http://localhost:%s/api/docs\n", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func docsHandler(w http.ResponseWriter, r *http.Request) {
	setCORS(w, r)
	if r.Method == http.MethodOptions {
		return
	}
	switch r.Method {
	case http.MethodGet:
		docs, err := dbListDocuments(sqlDB)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		type Summary struct {
			ID      string `json:"id"`
			Version int    `json:"version"`
			Preview string `json:"preview"`
		}
		out := []Summary{}
		for _, d := range docs {
			p := d.Content
			if len(p) > 120 {
				p = p[:120] + "…"
			}
			out = append(out, Summary{d.ID, d.Version, p})
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(out)

	case http.MethodPost:
		var body struct {
			ID      string `json:"id"`
			Content string `json:"content"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if body.ID == "" {
			body.ID = randomID(8)
		}
		if err := dbCreateDocument(sqlDB, body.ID, body.Content); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": body.ID})

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func docByIDHandler(w http.ResponseWriter, r *http.Request) {
	setCORS(w, r)
	if r.Method == http.MethodOptions {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/docs/")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		doc, err := dbGetDocument(sqlDB, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if doc == nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": doc.ID, "content": doc.Content,
			"version": doc.Version, "users": hub.usersInDoc(id),
		})
	case http.MethodPut:
		var body struct {
			Content string `json:"content"`
			Version int    `json:"version"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		newContent, newVersion, _ := docMgr.Apply(id, body.Version, body.Content)
		_ = dbSaveDocument(sqlDB, id, newContent, newVersion)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": id, "content": newContent, "version": newVersion,
		})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func suggestHandler(w http.ResponseWriter, r *http.Request) {
	setCORS(w, r)
	prefix := r.URL.Query().Get("prefix")
	suggestions := globalMarkov.Suggest(prefix, 5)
	if suggestions == nil {
		suggestions = []string{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"prefix": prefix, "suggestions": suggestions,
	})
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	docID := q.Get("docId")
	if docID == "" {
		docID = "default"
	}
	clientID := q.Get("clientId")
	if clientID == "" {
		clientID = randomID(6)
	}
	username := q.Get("username")
	if username == "" {
		username = "User-" + clientID[:4]
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ws] upgrade: %v", err)
		return
	}

	// Seed in-memory state from DB on first connection
	if content, version := docMgr.Get(docID); content == "" && version == 0 {
		if persisted, err := dbGetDocument(sqlDB, docID); err == nil && persisted != nil {
			docMgr.Seed(docID, persisted.Content, persisted.Version)
			go globalMarkov.Train(persisted.Content)
		}
	}

	client := &Client{
		hub: hub, conn: conn,
		send:     make(chan []byte, 256),
		docID:    docID,
		clientID: clientID,
		username: username,
	}
	hub.register <- client

	content, version := docMgr.Get(docID)
	if initData, err := json.Marshal(OutboundMessage{
		Type: "init", DocID: docID,
		Content: content, Version: version,
		Users: hub.usersInDoc(docID),
	}); err == nil {
		client.send <- initData
	}

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(512 * 1024)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[ws] read error %s: %v", c.clientID, err)
			}
			break
		}
		var msg InboundMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			log.Printf("[ws] bad message from %s: %v", c.clientID, err)
			continue
		}
		switch msg.Type {
		case "edit":
			newContent, newVersion, _ := docMgr.Apply(c.docID, msg.Version, msg.Content)
			go globalMarkov.Train(newContent)
			go func(id, content string, ver int) {
				if err := dbSaveDocument(sqlDB, id, content, ver); err != nil {
					log.Printf("[ws] persist %s: %v", id, err)
				}
			}(c.docID, newContent, newVersion)

			if outData, err := json.Marshal(OutboundMessage{
				Type: "update", DocID: c.docID,
				Content: newContent, Version: newVersion, SenderID: c.clientID,
			}); err == nil {
				hub.broadcast <- BroadcastMsg{docID: c.docID, payload: outData, exclude: c}
			}
			if ackData, err := json.Marshal(OutboundMessage{
				Type: "ack", DocID: c.docID, Version: newVersion,
			}); err == nil {
				c.send <- ackData
			}
		case "join":
			if msg.Username != "" {
				c.username = msg.Username
			}
			hub.broadcastPresence(c.docID)
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, nil)
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func setCORS(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = "*"
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

func randomID(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
