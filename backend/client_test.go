package main

import (
	// "fmt"
	"net/url"
	// "os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWebSocketConnection(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Wait a bit for server to be ready
	time.Sleep(1 * time.Second)

	// Connect to WebSocket
	u := url.URL{
		Scheme:   "ws",
		Host:     "localhost:8080",
		Path:     "/ws",
		RawQuery: "docId=test123",
	}

	t.Logf("Connecting to %s", u.String())

	// Set up dialer with timeout
	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}

	c, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer c.Close()

	t.Log("Connected successfully")

	// Test sending a message
	testMessage := []byte("Hello, WebSocket!")
	err = c.WriteMessage(websocket.TextMessage, testMessage)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	t.Logf("Sent message: %s", testMessage)

	// Test receiving message with timeout
	c.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, message, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	t.Logf("Received message: %s", message)

	if string(message) != string(testMessage) {
		t.Errorf("Expected %s, got %s", testMessage, message)
	} else {
		t.Log("Message echo test passed!")
	}
}

func TestMultipleConnections(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Create two connections
	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws", RawQuery: "docId=test123"}

	// First client
	c1, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Failed to connect first client: %v", err)
	}
	defer c1.Close()

	// Second client
	c2, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Failed to connect second client: %v", err)
	}
	defer c2.Close()

	t.Log("Both clients connected")

	// Send message from first client
	testMessage := []byte("Broadcast test")
	err = c1.WriteMessage(websocket.TextMessage, testMessage)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Second client should receive it
	c2.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, message, err := c2.ReadMessage()
	if err != nil {
		t.Fatalf("Second client didn't receive message: %v", err)
	}

	if string(message) != string(testMessage) {
		t.Errorf("Expected %s, got %s", testMessage, message)
	} else {
		t.Log("Broadcast test passed!")
	}
}
