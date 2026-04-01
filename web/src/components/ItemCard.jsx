import { useState } from 'react'
import { formatRelative, truncate } from '../utils'
import './ItemCard.css'

export function ItemCard({ item, onDismiss, onSnooze }) {
  const [expanded, setExpanded] = useState(false)
  const [dismissing, setDismissing] = useState(false)

  async function handleDismiss(e) {
    e.stopPropagation()
    setDismissing(true)
    try {
      await onDismiss(item.id)
    } finally {
      setDismissing(false)
    }
  }

  function handleSnooze(e) {
    e.stopPropagation()
    onSnooze(item)
  }

  const hasAttention = (item.active_signals ?? []).length > 0
  const isAssigned = item.is_assigned

  return (
    <div
      className={`item-card card ${expanded ? 'expanded' : ''} ${hasAttention ? 'has-attention' : ''}`}
      onClick={() => setExpanded((e) => !e)}
      role="button"
      aria-expanded={expanded}
      tabIndex={0}
      onKeyDown={(e) => e.key === 'Enter' && setExpanded((v) => !v)}
    >
      <div className="item-card-main">
        <div className="item-card-indicator" />

        <div className="item-card-meta">
          <span className="badge badge-source">{item.source}</span>
          <span className="badge badge-type">{item.type}</span>
          <span className="item-namespace mono">{item.namespace}</span>
        </div>

        <div className="item-card-title">
          {expanded ? item.title : truncate(item.title, 80)}
        </div>

        <div className="item-card-signals">
          {(item.active_signals ?? []).map((sig) => (
            <span key={sig} className="badge badge-signal">{sig}</span>
          ))}
          {isAssigned && <span className="badge badge-assigned">ASSIGNED</span>}
        </div>

        <div className="item-card-right">
          <span className="item-updated mono">{formatRelative(item.updated_at)}</span>
          <span className="item-chevron">{expanded ? '▲' : '▼'}</span>
        </div>
      </div>

      {expanded && (
        <div className="item-card-expanded" onClick={(e) => e.stopPropagation()}>
          <div className="expanded-divider" />

          <div className="expanded-row">
            <span className="label">ID</span>
            <span className="mono expanded-id">{item.id}</span>
          </div>

          <div className="expanded-row">
            <span className="label">Updated</span>
            <span className="mono">{new Date(item.updated_at).toLocaleString()}</span>
          </div>

          {item.url && (
            <div className="expanded-row">
              <span className="label">Source</span>
              <a
                href={item.url}
                target="_blank"
                rel="noreferrer"
                className="expanded-url"
                onClick={(e) => e.stopPropagation()}
              >
                {item.url} ↗
              </a>
            </div>
          )}

          <div className="expanded-actions">
            <button
              className="btn btn-danger"
              onClick={handleDismiss}
              disabled={dismissing}
              aria-label="Dismiss this item"
            >
              {dismissing ? <span className="spinner" /> : '✕'} DISMISS
            </button>
            <button
              className="btn"
              onClick={handleSnooze}
              aria-label="Snooze this item"
            >
              ◷ SNOOZE
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
