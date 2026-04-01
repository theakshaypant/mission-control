import { formatRelative } from '../utils'
import './Widget.css'

export function SyncStatus({ syncStatus, syncing, onSyncSource }) {
  if (syncStatus.length === 0) return null

  return (
    <div className="sync-status-bar card">
      <span className="label sync-status-title">Sync Status</span>
      <div className="sync-status-sources">
        {syncStatus.map((src) => {
          const ok = !!src.last_synced_at
          return (
            <button
              key={src.name}
              className="sync-source-chip"
              onClick={() => onSyncSource(src.name)}
              disabled={syncing}
              title={`Click to sync ${src.name}`}
            >
              <span className={`sync-dot ${ok ? 'online' : 'offline'}`} />
              <span className="mono sync-source-name">{src.name}</span>
              <span className="sync-source-time">
                {ok ? formatRelative(src.last_synced_at) : 'never'}
              </span>
            </button>
          )
        })}
      </div>
    </div>
  )
}
