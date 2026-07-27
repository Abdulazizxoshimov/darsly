export function ReactionsOverlay({ reactions }) {
  return (
    <div className="reactions-overlay">
      {reactions.map((r) => (
        <div key={r.id} className="float-reaction" style={{ left: `${r.left}%` }}>
          <span>{r.emoji}</span>
          <span>{r.name}</span>
        </div>
      ))}
    </div>
  )
}
