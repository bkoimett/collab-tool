package main

import (
	"fmt"
	"html"
	"net/http"
	

	// "github.com/gorilla/websocket"
)


func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Welcome to my webpage hosted on %q", html.EscapeString(r.URL.Path))
	})

	// http.HandleFunc("/ws", handleConnections)

	fmt.Println("Websocket server started on http://localhost:8080")
	err := http.ListenAndServe(":8080", nil) 
	if err != nil {
		fmt.Println("ListenAndServe error:", err)
	}

}