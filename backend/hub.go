package main

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// Client represents a connected WebSocket user
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	docID    string
	clientID string
	username string
}

// Hub manages all active WebSocket clients, grouped by document
type Hub struct {
	// docID -> set of clients
	rooms      map[string]map[*Client]bool
	broadcast  chan BroadcastMsg
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// BroadcastMsg carries a message to be sent to all clients in a document room
type BroadcastMsg struct {
	docID   string
	payload []byte
	// exclude this client from receiving the broadcast (the sender)
	exclude *Client
}

func newHub() *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]bool),
		broadcast:  make(chan BroadcastMsg, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.rooms[client.docID] == nil {
				h.rooms[client.docID] = make(map[*Client]bool)
			}
			h.rooms[client.docID][client] = true
			h.mu.Unlock()
			log.Printf("[hub] client %s joined doc %s", client.clientID, client.docID)
			h.broadcastPresence(client.docID)

		case client := <-h.unregister:
			h.mu.Lock()
			if room, ok := h.rooms[client.docID]; ok {
				if _, ok := room[client]; ok {
					delete(room, client)
					close(client.send)
					if len(room) == 0 {
						delete(h.rooms, client.docID)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("[hub] client %s left doc %s", client.clientID, client.docID)
			h.broadcastPresence(client.docID)

		case msg := <-h.broadcast:
			h.mu.RLock()
			room := h.rooms[msg.docID]
			for client := range room {
				if client == msg.exclude {
					continue
				}
				select {
				case client.send <- msg.payload:
				default:
					log.Printf("[hub] send buffer full for client %s, dropping message", client.clientID)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// broadcastPresence sends a presence update to all clients in a document room
func (h *Hub) broadcastPresence(docID string) {
	h.mu.RLock()
	room := h.rooms[docID]
	users := make([]UserInfo, 0, len(room))
	for c := range room {
		users = append(users, UserInfo{
			ClientID: c.clientID,
			Username: c.username,
		})
	}
	h.mu.RUnlock()

	msg := OutboundMessage{
		Type:  "presence",
		DocID: docID,
		Users: users,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.mu.RLock()
	for c := range h.rooms[docID] {
		select {
		case c.send <- data:
		default:
		}
	}
	h.mu.RUnlock()
}

// usersInDoc returns the count and list of users editing a document
func (h *Hub) usersInDoc(docID string) []UserInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()
	room := h.rooms[docID]
	users := make([]UserInfo, 0, len(room))
	for c := range room {
		users = append(users, UserInfo{ClientID: c.clientID, Username: c.username})
	}
	return users
}
