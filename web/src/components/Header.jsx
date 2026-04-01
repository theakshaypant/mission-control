import { useState } from 'react'
import { formatRelative } from '../utils'
import './Header.css'

export function Header({ syncing, syncError, syncStatus, sources, onSyncAll, onSyncSource, lastRefreshed }) {
  const [dropdownOpen, setDropdownOpen] = useState(false)
  const [theme, setTheme] = useState(
    () => document.documentElement.classList.contains('light') ? 'light' : 'dark'
  )

  function toggleTheme() {
    const next = theme === 'dark' ? 'light' : 'dark'
    document.documentElement.classList.remove('dark', 'light')
    document.documentElement.classList.add(next)
    localStorage.setItem('mc-theme', next)
    setTheme(next)
  }

  function handleSyncAll() {
    setDropdownOpen(false)
    onSyncAll()
  }

  function handleSyncSource(src) {
    setDropdownOpen(false)
    onSyncSource(src)
  }

  return (
    <header className="header">
      <div className="header-scanline" aria-hidden />
      <div className="header-inner">
        <div className="header-brand">
          <span className="header-icon">▌▌</span>
          <span className="header-title">MISSION CONTROL</span>
          <span className="header-sub">EYES ONLY</span>
        </div>

        <div className="header-actions">
          {lastRefreshed && (
            <span className="header-refreshed mono">
              BRIEFING {formatRelative(lastRefreshed)}
            </span>
          )}

          {syncError && (
            <span className="header-sync-error">{syncError}</span>
          )}

          <div className="sync-dropdown">
            <button
              className="btn"
              onClick={() => setDropdownOpen((o) => !o)}
              disabled={syncing}
              aria-label="Sync options"
            >
              {syncing ? <span className="spinner" /> : '⟳'}
              SYNC
              <span className="dropdown-arrow">▾</span>
            </button>

            {dropdownOpen && (
              <>
                <div className="dropdown-backdrop" onClick={() => setDropdownOpen(false)} />
                <div className="dropdown-menu">
                  <button className="dropdown-item" onClick={handleSyncAll}>
                    All sources
                  </button>
                  {sources.length > 0 && <div className="dropdown-divider" />}
                  {sources.map((src) => (
                    <button key={src} className="dropdown-item mono" onClick={() => handleSyncSource(src)}>
                      {src}
                    </button>
                  ))}
                </div>
              </>
            )}
          </div>

          <button
            className="btn btn-ghost theme-toggle"
            onClick={toggleTheme}
            aria-label={`Switch to ${theme === 'dark' ? 'light' : 'dark'} mode`}
          >
            {theme === 'dark' ? '○' : '●'}
          </button>
        </div>
      </div>
    </header>
  )
}
