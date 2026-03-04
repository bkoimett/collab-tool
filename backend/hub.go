package main

import (
	"log"
    "sync"
    "github.com/gorilla/websocket"
)

type Client struct {
    hub   *Hub
    conn  *websocket.Conn
    send  chan []byte
    docID string
}

type Hub struct {
    clients    map[*Client]bool
    broadcast  chan []byte
    register   chan *Client
    unregister chan *Client
    documents  map[string]string // docID -> content
    mu         sync.RWMutex      // Add mutex for thread safety
}

func newHub() *Hub {
    return &Hub{
        clients:    make(map[*Client]bool),
        broadcast:  make(chan []byte),
        register:   make(chan *Client),
        unregister: make(chan *Client),
        documents:  make(map[string]string),
    }
}

func (h *Hub) run() {
    for {
        select {
        case client := <-h.register:
            h.mu.Lock()
            h.clients[client] = true
            h.mu.Unlock()
            
            // Optional: Send current document state to new client
            h.mu.RLock()
            if content, exists := h.documents[client.docID]; exists {
                client.send <- []byte(content)
            }
            h.mu.RUnlock()
            
        case client := <-h.unregister:
            h.mu.Lock()
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                close(client.send)
            }
            h.mu.Unlock()
            
        case message := <-h.broadcast:
            h.mu.RLock()
            for client := range h.clients {
                select {
                case client.send <- message:
                default:
                    // Don't close immediately, just skip if channel is full
                    log.Printf("Client %v channel full, skipping", client.docID)
                }
            }
            h.mu.RUnlock()
        }
    }
}