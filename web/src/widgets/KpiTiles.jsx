import './KpiTiles.css'

function Tile({ label, value, sub, accent }) {
  return (
    <div className={`kpi-tile card ${accent ? 'kpi-accent' : ''}`}>
      <span className="kpi-label label">{label}</span>
      <span className="kpi-value">{value ?? '—'}</span>
      {sub && <span className="kpi-sub">{sub}</span>}
    </div>
  )
}

export function KpiTiles({ items, snoozedCount, syncStatus }) {
  const priority = items.filter((i) => (i.active_signals ?? []).length > 0).length
  const sources = syncStatus.length || new Set(items.map((i) => i.source_name).filter(Boolean)).size
  const lastSync = syncStatus.length > 0
    ? syncStatus
        .filter((s) => s.last_synced_at)
        .sort((a, b) => new Date(b.last_synced_at) - new Date(a.last_synced_at))[0]
    : null

  function lastSyncLabel() {
    if (!lastSync) return 'Never'
    const diff = Math.round((Date.now() - new Date(lastSync.last_synced_at)) / 60000)
    if (diff < 1) return 'Just now'
    if (diff < 60) return `${diff}m ago`
    return `${Math.round(diff / 60)}h ago`
  }

  return (
    <div className="kpi-row">
      <Tile label="Priority" value={priority} accent={priority > 0} />
      <Tile label="Sources" value={sources} />
      <Tile label="Snoozed" value={snoozedCount} />
      <Tile label="Last Briefing" value={lastSyncLabel()} sub={lastSync?.name} />
    </div>
  )
}
