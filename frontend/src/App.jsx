import { useState, useEffect, useRef, useCallback } from 'react'
import Editor from './Editor'
import Preview from './Preview'
import UsersPanel from './UsersPanel'
import { WSManager, generateClientId, generateUsername } from './utils/websocket'

// Persist identity across page reloads
function getIdentity() {
  let id = sessionStorage.getItem('clientId')
  let name = sessionStorage.getItem('username')
  if (!id) { id = generateClientId(); sessionStorage.setItem('clientId', id) }
  if (!name) { name = generateUsername(); sessionStorage.setItem('username', name) }
  return { clientId: id, username: name }
}

// Extract or generate a doc ID from the URL path
function getDocId() {
  const path = window.location.pathname.replace(/^\//, '')
  if (path && path !== 'new') return path
  const id = Math.random().toString(36).slice(2, 10)
  window.history.replaceState(null, '', '/' + id)
  return id
}

export default function App() {
  const [content, setContent] = useState('')
  const [version, setVersion] = useState(0)
  const [users, setUsers] = useState([])
  const [connected, setConnected] = useState(false)
  const [suggestions, setSuggestions] = useState([])
  const [layout, setLayout] = useState('split') // 'split' | 'editor' | 'preview'
  const [copied, setCopied] = useState(false)
  const [username, setUsername] = useState('')
  const [editingName, setEditingName] = useState(false)
  const [nameInput, setNameInput] = useState('')

  const wsRef = useRef(null)
  const versionRef = useRef(0)
  const { clientId, username: initialUsername } = getIdentity()
  const docId = getDocId()

  useEffect(() => {
    setUsername(initialUsername)
  }, [initialUsername])

  // Connect WebSocket
  useEffect(() => {
    const ws = new WSManager({
      docId,
      clientId,
      username: sessionStorage.getItem('username') || initialUsername,
      onOpen: () => setConnected(true),
      onClose: () => setConnected(false),
      onMessage: (msg) => {
        switch (msg.type) {
          case 'init':
            setContent(msg.content || '')
            setVersion(msg.version || 0)
            versionRef.current = msg.version || 0
            setUsers(msg.users || [])
            break
          case 'update':
            // Ignore our own echo (server excludes sender, but just in case)
            if (msg.senderId === clientId) break
            setContent(msg.content || '')
            setVersion(msg.version)
            versionRef.current = msg.version
            break
          case 'ack':
            setVersion(msg.version)
            versionRef.current = msg.version
            break
          case 'presence':
            setUsers(msg.users || [])
            break
        }
      },
    })
    wsRef.current = ws
    return () => ws.destroy()
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  // Debounce sending edits over WS
  const debounceRef = useRef(null)
  const handleContentChange = useCallback((newContent, immediate = false) => {
    setContent(newContent)
    clearTimeout(debounceRef.current)
    const send = () => {
      wsRef.current?.send({
        type: 'edit',
        docId,
        content: newContent,
        clientId,
        version: versionRef.current,
      })
    }
    if (immediate) {
      send()
    } else {
      debounceRef.current = setTimeout(send, 300)
    }
  }, [clientId, docId])

  const handleAcceptSuggestion = useCallback((word, newSuggestions) => {
    if (newSuggestions !== null) setSuggestions(newSuggestions)
  }, [])

  const copyLink = () => {
    navigator.clipboard.writeText(window.location.href)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  const saveUsername = () => {
    const trimmed = nameInput.trim()
    if (trimmed) {
      setUsername(trimmed)
      sessionStorage.setItem('username', trimmed)
      wsRef.current?.send({ type: 'join', docId, clientId, username: trimmed })
    }
    setEditingName(false)
  }

  return (
    <div className="app">
      {/* ── Top bar ── */}
      <header className="topbar">
        <div className="topbar-left">
          <div className="logo">
            <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M12 20h9"/><path d="M16.5 3.5a2.121 2.121 0 0 1 3 3L7 19l-4 1 1-4L16.5 3.5z"/>
            </svg>
            <span>Collab<strong>MD</strong></span>
          </div>
          <div className="doc-id">doc: <code>{docId}</code></div>
        </div>

        <div className="topbar-center">
          <div className="layout-toggle">
            {['split', 'editor', 'preview'].map(l => (
              <button
                key={l}
                className={`layout-btn ${layout === l ? 'active' : ''}`}
                onClick={() => setLayout(l)}
                title={l.charAt(0).toUpperCase() + l.slice(1)}
              >
                {l === 'split' ? '⬛⬛' : l === 'editor' ? '📝' : '👁'}
              </button>
            ))}
          </div>
        </div>

        <div className="topbar-right">
          <UsersPanel users={users} currentClientId={clientId} />

          {editingName ? (
            <div className="name-edit">
              <input
                autoFocus
                value={nameInput}
                onChange={e => setNameInput(e.target.value)}
                onKeyDown={e => { if (e.key === 'Enter') saveUsername(); if (e.key === 'Escape') setEditingName(false) }}
                placeholder="Your name…"
                className="name-input"
              />
              <button className="btn-sm" onClick={saveUsername}>✓</button>
            </div>
          ) : (
            <button
              className="btn-ghost"
              title="Change your display name"
              onClick={() => { setNameInput(username); setEditingName(true) }}
            >
              👤 {username}
            </button>
          )}

          <button className="btn-primary" onClick={copyLink}>
            {copied ? '✓ Copied!' : '🔗 Share'}
          </button>
        </div>
      </header>

      {/* ── Main area ── */}
      <main className={`main layout-${layout}`}>
        {layout !== 'preview' && (
          <Editor
            content={content}
            onChange={handleContentChange}
            suggestions={suggestions}
            onAcceptSuggestion={handleAcceptSuggestion}
            connected={connected}
          />
        )}
        {layout !== 'editor' && (
          <Preview content={content} />
        )}
      </main>

      {/* ── Status bar ── */}
      <footer className="statusbar">
        <span>{content.split(/\s+/).filter(Boolean).length} words</span>
        <span>{content.length} chars</span>
        <span>v{version}</span>
        <span className={connected ? 'status-ok' : 'status-err'}>
          {connected ? '● connected' : '○ reconnecting'}
        </span>
      </footer>
    </div>
  )
}
