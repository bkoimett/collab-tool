package main

import (
	"database/sql"
    "fmt"
    "log"
    "net/http"
    "time"
    
    "github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
}

var (
    hub *Hub
    db  *sql.DB
)

func handleConnections(w http.ResponseWriter, r *http.Request) {
    docID := r.URL.Query().Get("docId")
    if docID == "" {
        docID = "default"
    }

    log.Printf("New connection attempt for document: %s", docID)

    ws, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println("Upgrade error:", err)
        return
    }

    // Load document from database
    content, err := getDocument(db, docID)
    if err != nil {
        log.Printf("Error loading document %s: %v", docID, err)
    }

    client := &Client{
        hub:   hub,
        conn:  ws,
        send:  make(chan []byte, 256),
        docID: docID,
    }
    
    // Send current document state to client
    if content != "" {
        client.send <- []byte(content)
    }
    
    client.hub.register <- client

    log.Printf("Client registered for document: %s", docID)

    go client.writePump()
    go client.readPump()
}

func (c *Client) readPump() {
    defer func() {
        log.Printf("Client disconnected from document: %s", c.docID)
        c.hub.unregister <- c
        c.conn.Close()
    }()

    c.conn.SetReadLimit(512 * 1024)
    c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    c.conn.SetPongHandler(func(string) error {
        c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
        return nil
    })

    for {
        _, message, err := c.conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                log.Printf("WebSocket read error: %v", err)
            }
            break
        }

        log.Printf("Received message for document %s: %s", c.docID, string(message))

        // Save to database
        err = saveDocument(db, c.docID, string(message))
        if err != nil {
            log.Printf("Error saving document %s: %v", c.docID, err)
        }

        // Broadcast to all clients
        c.hub.broadcast <- message
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
        case message, ok := <-c.send:
            c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if !ok {
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }

            err := c.conn.WriteMessage(websocket.TextMessage, message)
            if err != nil {
                log.Printf("Write error: %v", err)
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

func main() {
    // Initialize database
    db = initDB()  // Use the global db variable
    defer db.Close()

    // Initialize and start hub
    hub = newHub()
    go hub.run()

    // Set up routes
    http.HandleFunc("/ws", handleConnections)
    
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("Server is running"))
    })

    // Add REST endpoints for document management
    http.HandleFunc("/documents", func(w http.ResponseWriter, r *http.Request) {
        switch r.Method {
        case "GET":
            id := r.URL.Query().Get("id")
            content, err := getDocument(db, id)
            if err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
                return
            }
            w.Header().Set("Content-Type", "application/json")
            fmt.Fprintf(w, `{"id":"%s","content":"%s"}`, id, content)
        }
    })

    fmt.Println("Websocket server started on http://localhost:8080")
    fmt.Println("Database connected and ready")
    fmt.Println("Test with: ws://localhost:8080/ws?docId=test123")

    err := http.ListenAndServe(":8080", nil)
    if err != nil {
        log.Fatal("ListenAndServe error:", err)
    }
}