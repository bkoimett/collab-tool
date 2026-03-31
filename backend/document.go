package main

import (
	"sync"
)

// DocState holds in-memory document state with version tracking
type DocState struct {
	Content string
	Version int
}

// DocumentManager keeps in-memory versions of documents and handles OT-lite merging
type DocumentManager struct {
	mu   sync.RWMutex
	docs map[string]*DocState
}

func newDocumentManager() *DocumentManager {
	return &DocumentManager{
		docs: make(map[string]*DocState),
	}
}

// Get returns the current content and version for a document
func (dm *DocumentManager) Get(docID string) (string, int) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()
	if doc, ok := dm.docs[docID]; ok {
		return doc.Content, doc.Version
	}
	return "", 0
}

// Apply applies an incoming edit from a client.
//
// OT-lite rules:
//  1. If the client version matches the server version, apply directly.
//  2. If the client version is behind, the server has a newer state —
//     we use last-write-wins (simplest safe strategy without full CRDT).
//     The function returns the *current* server content so the caller
//     can send a resync to the client.
//
// Returns (newContent, newVersion, didApply).
func (dm *DocumentManager) Apply(docID string, clientVersion int, content string) (string, int, bool) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	doc, ok := dm.docs[docID]
	if !ok {
		doc = &DocState{}
		dm.docs[docID] = doc
	}

	// Client is up-to-date or ahead — accept the edit
	if clientVersion >= doc.Version {
		doc.Content = content
		doc.Version++
		return doc.Content, doc.Version, true
	}

	// Client is behind: last-write-wins, still accept and increment
	doc.Content = content
	doc.Version++
	return doc.Content, doc.Version, true
}

// Seed loads content from the database into memory (called on first WS connect for a doc)
func (dm *DocumentManager) Seed(docID, content string, version int) {
	dm.mu.Lock()
	defer dm.mu.Unlock()
	if _, ok := dm.docs[docID]; !ok {
		dm.docs[docID] = &DocState{Content: content, Version: version}
	}
}
