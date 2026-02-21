package main

import (
	"github.com/gorilla/websocket"
	_ "github.com/gorilla/websocket"
	// _ "golang.org/x/sys/windows/registry"
	// _ "golang.org/x/text/message"
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
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}
