package main

// InboundMessage is sent from a browser client to the server
type InboundMessage struct {
	Type     string `json:"type"`     // "edit" | "join" | "cursor"
	DocID    string `json:"docId"`
	Content  string `json:"content"`
	ClientID string `json:"clientId"`
	Username string `json:"username"`
	Version  int    `json:"version"`
	// Cursor position (optional, for future use)
	CursorLine int `json:"cursorLine,omitempty"`
	CursorCh   int `json:"cursorCh,omitempty"`
}

// OutboundMessage is sent from the server to browser clients
type OutboundMessage struct {
	Type    string     `json:"type"`    // "init" | "update" | "presence" | "ack"
	DocID   string     `json:"docId"`
	Content string     `json:"content,omitempty"`
	Users   []UserInfo `json:"users,omitempty"`
	Version int        `json:"version,omitempty"`
	// The clientId that caused this update (so the sender can ignore echo)
	SenderID string `json:"senderId,omitempty"`
}

// UserInfo describes a connected collaborator
type UserInfo struct {
	ClientID string `json:"clientId"`
	Username string `json:"username"`
}
