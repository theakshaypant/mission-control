import { ItemCard } from './ItemCard'

export function ItemList({ items, loading, error, onDismiss, onSnooze }) {
  if (loading) {
    return (
      <div className="state-loading">
        <span className="spinner" style={{ width: 20, height: 20 }} />
        <span className="state-label">Retrieving intelligence…</span>
      </div>
    )
  }

  if (error) {
    return (
      <div className="state-error">
        <span style={{ fontSize: 24 }}>✕</span>
        <span className="state-label">Transmission failure</span>
        <span style={{ fontSize: 12 }}>{error}</span>
      </div>
    )
  }

  if (items.length === 0) {
    return (
      <div className="state-empty">
        <span style={{ fontSize: 32, opacity: 0.3 }}>◎</span>
        <span className="state-label">No active targets</span>
        <span style={{ fontSize: 12 }}>All assets are clear.</span>
      </div>
    )
  }

  return (
    <div>
      {items.map((item) => (
        <ItemCard
          key={item.id}
          item={item}
          onDismiss={onDismiss}
          onSnooze={onSnooze}
        />
      ))}
    </div>
  )
}
