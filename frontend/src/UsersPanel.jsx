export default function UsersPanel({ users, currentClientId }) {
  if (!users.length) return null

  return (
    <div className="users-panel">
      <div className="users-header">
        <span className="users-dot" />
        {users.length} online
      </div>
      <div className="users-list">
        {users.map((u) => (
          <div key={u.clientId} className={`user-chip ${u.clientId === currentClientId ? 'me' : ''}`}>
            <span className="user-avatar">{(u.username || 'U')[0].toUpperCase()}</span>
            <span className="user-name">{u.username || 'Anonymous'}</span>
            {u.clientId === currentClientId && <span className="user-you">(you)</span>}
          </div>
        ))}
      </div>
    </div>
  )
}
