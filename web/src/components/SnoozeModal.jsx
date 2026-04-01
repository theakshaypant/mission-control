import { useState, useEffect, useRef } from 'react'
import { snoozeItem } from '../api'

const QUICK_DURATIONS = ['1h', '4h', '24h', '7d']

export function SnoozeModal({ item, onClose, onSuccess }) {
  const [mode, setMode] = useState('for')
  const [forValue, setForValue] = useState('24h')
  const [untilValue, setUntilValue] = useState(() => {
    const d = new Date()
    d.setDate(d.getDate() + 1)
    const y = d.getFullYear()
    const m = String(d.getMonth() + 1).padStart(2, '0')
    const day = String(d.getDate()).padStart(2, '0')
    return `${y}-${m}-${day}`
  })
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState(null)
  const modalRef = useRef(null)

  useEffect(() => {
    function onKeyDown(e) {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', onKeyDown)
    modalRef.current?.focus()
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [onClose])

  async function handleSubmit(e) {
    e.preventDefault()
    if (!forValue && !untilValue) return
    // For "until date" mode, attach the local timezone offset so the server
    // receives a full RFC3339 timestamp (e.g. "2026-04-02T00:00:00+05:30")
    // rather than a bare date that would be parsed as midnight UTC.
    let value
    if (mode === 'for') {
      value = forValue
    } else {
      const offset = -new Date().getTimezoneOffset() // minutes, positive = UTC+
      const sign = offset >= 0 ? '+' : '-'
      const h = String(Math.floor(Math.abs(offset) / 60)).padStart(2, '0')
      const m = String(Math.abs(offset) % 60).padStart(2, '0')
      value = `${untilValue}T00:00:00${sign}${h}:${m}`
    }

    setSubmitting(true)
    setError(null)
    try {
      await snoozeItem(item.id, mode, value)
      onSuccess()
      onClose()
    } catch (err) {
      setError(err.message ?? 'Snooze failed')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div
        className="modal"
        onClick={(e) => e.stopPropagation()}
        ref={modalRef}
        tabIndex={-1}
        role="dialog"
        aria-modal="true"
        aria-label="Snooze item"
      >
        <div className="modal-header">
          <span className="modal-label">◷ SNOOZE ASSET</span>
          <button className="btn btn-ghost modal-close" onClick={onClose} aria-label="Close">✕</button>
        </div>

        <p className="modal-item-title">{item.title}</p>
        <p className="modal-item-id mono">{item.id}</p>

        <div className="modal-divider" />

        <form onSubmit={handleSubmit}>
          <div className="modal-mode-tabs">
            <button
              type="button"
              className={`modal-tab ${mode === 'for' ? 'active' : ''}`}
              onClick={() => setMode('for')}
            >
              For duration
            </button>
            <button
              type="button"
              className={`modal-tab ${mode === 'until' ? 'active' : ''}`}
              onClick={() => setMode('until')}
            >
              Until date
            </button>
          </div>

          {mode === 'for' ? (
            <div className="modal-field">
              <div className="quick-picks">
                {QUICK_DURATIONS.map((d) => (
                  <button
                    key={d}
                    type="button"
                    className={`filter-toggle ${forValue === d ? 'active' : ''}`}
                    onClick={() => setForValue(d)}
                  >
                    {d}
                  </button>
                ))}
              </div>
              <input
                className="modal-input mono"
                type="text"
                value={forValue}
                onChange={(e) => setForValue(e.target.value)}
                placeholder="e.g. 2h30m, 7d"
                aria-label="Duration"
              />
            </div>
          ) : (
            <div className="modal-field">
              <input
                className="modal-input"
                type="date"
                value={untilValue}
                onChange={(e) => setUntilValue(e.target.value)}
                aria-label="Until date"
              />
            </div>
          )}

          {error && <p className="modal-error">{error}</p>}

          <div className="modal-actions">
            <button type="button" className="btn" onClick={onClose}>Cancel</button>
            <button type="submit" className="btn btn-primary" disabled={submitting}>
              {submitting ? <span className="spinner" /> : '◷'} Snooze
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
