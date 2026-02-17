package main

import (
	"fmt"
	"log"
	"net/http"
	
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true // Allow all origins (for development)
    },
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
    // Upgrade initial GET request to a WebSocket
    ws, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        fmt.Println(err)
        return
    }
    defer ws.Close()

    for {
        messageType, p, err := ws.ReadMessage()
        if err != nil {
            log.Println("Read error:", err)
            break
        }
        
        // Echo message back
        if err := ws.WriteMessage(messageType, p); err != nil {
            log.Println("Write error:", err)
            break
        }
    }	
}

func main() {
	http.HandleFunc("/ws", handleConnections)

	fmt.Println("Websocket server started on http://localhost:8080")
	err := http.ListenAndServe(":8080", nil) 
	if err != nil {
		fmt.Println("ListenAndServe error:", err)
	}

}