import { useState, useEffect, useRef } from 'react'
import { fetchConfig, saveConfig } from '../api'
import './ConfigEditor.css'

export function ConfigEditor({ onClose }) {
  const [yaml, setYaml] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState(null)
  const [saved, setSaved] = useState(false)
  const modalRef = useRef(null)

  useEffect(() => {
    fetchConfig()
      .then((text) => { setYaml(text); setLoading(false) })
      .catch((err) => { setError(err.message ?? 'Failed to load config'); setLoading(false) })
  }, [])

  useEffect(() => {
    function onKeyDown(e) {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', onKeyDown)
    modalRef.current?.focus()
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [onClose])

  async function handleSave(e) {
    e.preventDefault()
    setSaving(true)
    setError(null)
    setSaved(false)
    try {
      await saveConfig(yaml)
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    } catch (err) {
      setError(err.message ?? 'Save failed')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div
        className="modal config-modal"
        onClick={(e) => e.stopPropagation()}
        ref={modalRef}
        tabIndex={-1}
        role="dialog"
        aria-modal="true"
        aria-label="Edit configuration"
      >
        <div className="modal-header">
          <span className="modal-label">⚙ CONFIG — SOURCES</span>
          <button className="btn btn-ghost modal-close" onClick={onClose} aria-label="Close">✕</button>
        </div>

        <p className="config-hint mono">
          Edit your sources below. Removed sources will have their items dismissed automatically.
          The server config (address, etc.) is not shown here.
        </p>

        <div className="modal-divider" />

        <form onSubmit={handleSave}>
          {loading ? (
            <div className="config-loading">Loading…</div>
          ) : (
            <textarea
              className="config-textarea mono"
              value={yaml}
              onChange={(e) => setYaml(e.target.value)}
              spellCheck={false}
              aria-label="Sources YAML"
            />
          )}

          {error && <p className="modal-error">{error}</p>}

          <div className="modal-actions">
            <button type="button" className="btn" onClick={onClose}>Cancel</button>
            <button type="submit" className="btn btn-primary" disabled={saving || loading}>
              {saving ? <span className="spinner" /> : saved ? '✓' : '⚙'}{' '}
              {saved ? 'Applied' : 'Save & Apply'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
