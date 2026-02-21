package main

import (
    "time"
    "fmt"
    "log"
    "net/http"
    
    "github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true // Allow all origins (for development)
    },
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
}

var hub *Hub // Global hub instance

func handleConnections(w http.ResponseWriter, r *http.Request) {
    // Get document ID from query params
    docID := r.URL.Query().Get("docId")
    if docID == "" {
        docID = "default" // Fallback document
    }
    
    log.Printf("New connection attempt for document: %s", docID)

    // Upgrade to WebSocket
    ws, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println("Upgrade error:", err)
        return
    }
    
    log.Printf("WebSocket connection upgraded for document: %s", docID)

    // Create and register client
    client := &Client{
        hub:   hub,
        conn:  ws,
        send:  make(chan []byte, 256),
        docID: docID,
    }
    client.hub.register <- client
    
    log.Printf("Client registered for document: %s", docID)

    // Start client read/write pumps
    go client.writePump()
    go client.readPump()
}

func (c *Client) readPump() {
    defer func() {
        log.Printf("Client disconnected from document: %s", c.docID)
        c.hub.unregister <- c
        c.conn.Close()
    }()
    
    c.conn.SetReadLimit(512 * 1024) // 512KB max message size
    
    for {
        _, message, err := c.conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                log.Printf("WebSocket read error: %v", err)
            }
            break
        }
        
        log.Printf("Received message for document %s: %s", c.docID, string(message))
        
        // Here you could save to database or process the message
        // For now, broadcast to all clients
        c.hub.broadcast <- message
    }
}

func (c *Client) writePump() {
    defer c.conn.Close()
    
    for {
        select {
        case message, ok := <-c.send:
            if !ok {
                // The hub closed the channel
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }
            
            c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
                log.Printf("Write error: %v", err)
                return
            }
        }
    }
}

func main() {
    // Initialize database
    db := initDB()
    defer db.Close()
    
    // Initialize and start hub
    hub = newHub()
    go hub.run()
    
    // Set up routes
    http.HandleFunc("/ws", handleConnections)
    
    // Add a simple health check endpoint
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("Server is running"))
    })
    
    fmt.Println("Websocket server started on http://localhost:8080")
    fmt.Println("Database connected and ready")
    fmt.Println("Test with: ws://localhost:8080/ws?docId=test123")
    
    err := http.ListenAndServe(":8080", nil) 
    if err != nil {
        fmt.Println("ListenAndServe error:", err)
    }
}