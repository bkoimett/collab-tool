/**
 * WebSocket manager — handles connection, reconnection, and message routing.
 */

export class WSManager {
  constructor({ docId, clientId, username, onMessage, onOpen, onClose }) {
    this.docId = docId
    this.clientId = clientId
    this.username = username
    this.onMessage = onMessage
    this.onOpen = onOpen
    this.onClose = onClose
    this.ws = null
    this.reconnectDelay = 1000
    this.maxReconnectDelay = 16000
    this.shouldReconnect = true
    this._connect()
  }

  _connect() {
    const host = import.meta.env.VITE_API_BASE
      ? import.meta.env.VITE_API_BASE.replace('https://', '').replace('http://', '')
      : window.location.host
    const proto = import.meta.env.VITE_API_BASE?.startsWith('https') ? 'wss' : 'ws'
    const url = `${proto}://${host}/ws?docId=${encodeURIComponent(this.docId)}&clientId=${this.clientId}&username=${encodeURIComponent(this.username)}`

    this.ws = new WebSocket(url)

    this.ws.onopen = () => {
      this.reconnectDelay = 1000
      this.onOpen?.()
    }

    this.ws.onmessage = (e) => {
      try {
        const msg = JSON.parse(e.data)
        this.onMessage?.(msg)
      } catch (err) {
        console.error('[ws] parse error', err)
      }
    }

    this.ws.onclose = () => {
      this.onClose?.()
      if (this.shouldReconnect) {
        setTimeout(() => this._connect(), this.reconnectDelay)
        this.reconnectDelay = Math.min(this.reconnectDelay * 2, this.maxReconnectDelay)
      }
    }

    this.ws.onerror = (err) => {
      console.error('[ws] error', err)
    }
  }

  send(msg) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(msg))
      return true
    }
    return false
  }

  destroy() {
    this.shouldReconnect = false
    this.ws?.close()
  }
}

/** Generate a short random client ID */
export function generateClientId() {
  return Math.random().toString(36).slice(2, 8)
}

/** Generate a random username */
export function generateUsername() {
  const adjectives = ['Swift', 'Bright', 'Bold', 'Calm', 'Cool', 'Dark', 'Fast', 'Kind', 'Lazy', 'Wise']
  const nouns = ['Fox', 'Bear', 'Owl', 'Cat', 'Dog', 'Elk', 'Jay', 'Ant', 'Bee', 'Bat']
  return adjectives[Math.floor(Math.random() * adjectives.length)] +
    nouns[Math.floor(Math.random() * nouns.length)]
}
