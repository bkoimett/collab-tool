import { useRef, useCallback } from 'react'
import Suggestions from './Suggestions'

const DEBOUNCE_MS = 300

export default function Editor({ content, onChange, suggestions, onAcceptSuggestion, connected }) {
  const debounceRef = useRef(null)
  const textareaRef = useRef(null)

  const handleChange = useCallback((e) => {
    const val = e.target.value
    onChange(val)

    // Debounce suggestion fetch
    clearTimeout(debounceRef.current)
    debounceRef.current = setTimeout(() => {
      // Grab last ~50 chars as prefix for suggestions
      const words = val.trimEnd().split(/\s+/)
      const prefix = words.slice(-3).join(' ')
      if (prefix.length > 2) {
        fetch(`/api/suggest?prefix=${encodeURIComponent(prefix)}`)
          .then(r => r.json())
          .then(data => {
            if (data.suggestions?.length) {
              onAcceptSuggestion(null, data.suggestions)
            }
          })
          .catch(() => {})
      }
    }, DEBOUNCE_MS)
  }, [onChange, onAcceptSuggestion])

  const insertSuggestion = useCallback((word) => {
    const ta = textareaRef.current
    if (!ta) return
    const pos = ta.selectionStart
    const before = content.slice(0, pos)
    const after = content.slice(pos)
    // Add a space before the word if last char isn't whitespace
    const sep = before.endsWith(' ') || before.endsWith('\n') || before === '' ? '' : ' '
    const newContent = before + sep + word + ' ' + after
    onChange(newContent, true) // true = immediate send
    onAcceptSuggestion(word, [])
    // Restore cursor after the inserted word
    requestAnimationFrame(() => {
      const newPos = (before + sep + word + ' ').length
      ta.setSelectionRange(newPos, newPos)
      ta.focus()
    })
  }, [content, onChange, onAcceptSuggestion])

  // Toolbar helpers
  const wrap = useCallback((before, after = before) => {
    const ta = textareaRef.current
    if (!ta) return
    const start = ta.selectionStart
    const end = ta.selectionEnd
    const selected = content.slice(start, end)
    const newContent =
      content.slice(0, start) + before + selected + after + content.slice(end)
    onChange(newContent, true)
    requestAnimationFrame(() => {
      ta.setSelectionRange(start + before.length, end + before.length)
      ta.focus()
    })
  }, [content, onChange])

  const insertLine = useCallback((prefix) => {
    const ta = textareaRef.current
    if (!ta) return
    const pos = ta.selectionStart
    const lineStart = content.lastIndexOf('\n', pos - 1) + 1
    const newContent = content.slice(0, lineStart) + prefix + content.slice(lineStart)
    onChange(newContent, true)
    requestAnimationFrame(() => {
      const newPos = pos + prefix.length
      ta.setSelectionRange(newPos, newPos)
      ta.focus()
    })
  }, [content, onChange])

  const tools = [
    { label: 'B', title: 'Bold', action: () => wrap('**') },
    { label: 'I', title: 'Italic', action: () => wrap('*') },
    { label: 'S', title: 'Strikethrough', action: () => wrap('~~') },
    { label: '`', title: 'Inline code', action: () => wrap('`') },
    { label: 'H1', title: 'Heading 1', action: () => insertLine('# ') },
    { label: 'H2', title: 'Heading 2', action: () => insertLine('## ') },
    { label: 'H3', title: 'Heading 3', action: () => insertLine('### ') },
    { label: '—', title: 'Horizontal rule', action: () => onChange(content + '\n\n---\n\n', true) },
    { label: '•', title: 'Bullet list', action: () => insertLine('- ') },
    { label: '1.', title: 'Numbered list', action: () => insertLine('1. ') },
    { label: '❝', title: 'Blockquote', action: () => insertLine('> ') },
    {
      label: '```', title: 'Code block',
      action: () => wrap('```\n', '\n```')
    },
  ]

  return (
    <div className="editor-pane">
      <div className="pane-header">
        <span className="pane-title">Markdown</span>
        <span className={`conn-badge ${connected ? 'connected' : 'disconnected'}`}>
          {connected ? '● Live' : '○ Reconnecting…'}
        </span>
      </div>
      <div className="toolbar">
        {tools.map(t => (
          <button
            key={t.label}
            className="tool-btn"
            title={t.title}
            onMouseDown={e => { e.preventDefault(); t.action() }}
          >
            {t.label}
          </button>
        ))}
      </div>
      <textarea
        ref={textareaRef}
        className="editor-textarea"
        value={content}
        onChange={handleChange}
        placeholder={'# Start writing…\n\nType markdown here. Share this URL to collaborate in real time.'}
        spellCheck
      />
      <Suggestions
        suggestions={suggestions}
        onAccept={insertSuggestion}
      />
    </div>
  )
}
