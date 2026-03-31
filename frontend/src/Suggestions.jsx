export default function Suggestions({ suggestions, onAccept }) {
  if (!suggestions || suggestions.length === 0) return null

  return (
    <div className="suggestions">
      <span className="suggestions-label">💡</span>
      {suggestions.map((s, i) => (
        <button key={i} className="suggestion-chip" onClick={() => onAccept(s)}>
          {s}
        </button>
      ))}
    </div>
  )
}
