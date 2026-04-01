import { useState, useEffect, useMemo, useCallback } from 'react'
import { dismissItem } from './api'
import { useItems } from './hooks/useItems'
import { useSync } from './hooks/useSync'
import { uniqueSources, uniqueTypes } from './utils'

import { SplashScreen } from './components/SplashScreen'
import { Header } from './components/Header'
import { FilterBar } from './components/FilterBar'
import { ItemList } from './components/ItemList'
import { SnoozeModal } from './components/SnoozeModal'
import { ConfigEditor } from './components/ConfigEditor'
import { KpiTiles } from './widgets/KpiTiles'
import { SignalChart } from './widgets/SignalChart'
import { SourceDonut } from './widgets/SourceDonut'
import { NamespaceChart } from './widgets/NamespaceChart'
import { SyncStatus } from './widgets/SyncStatus'

import './styles/global.css'
import './components/SnoozeModal.css'
import './widgets/KpiTiles.css'
import './App.css'

const DEFAULT_FILTERS = {
  sourceName: '',
  types: [],
  signal: '',
  namespace: '',
}

function sortItems(items, sort) {
  const copy = [...items]
  if (sort === 'source') return copy.sort((a, b) => a.source.localeCompare(b.source))
  if (sort === 'type') return copy.sort((a, b) => a.type.localeCompare(b.type))
  // default: updated_at desc (server already returns this order, but client sort after mutations)
  return copy.sort((a, b) => new Date(b.updated_at) - new Date(a.updated_at))
}

export default function App() {
  const [showSplash, setShowSplash] = useState(true)
  const [filters, setFilters] = useState(DEFAULT_FILTERS)
  const [sort, setSort] = useState('updated_at')
  const [snoozeTarget, setSnoozeTarget] = useState(null)
  const [configOpen, setConfigOpen] = useState(false)
  const [lastRefreshed, setLastRefreshed] = useState(null)

  // Fetch items whenever filters change.
  // needsAttention is always forced true so snoozed/dismissed items never appear
  // in the list regardless of which filter toggles are active.
  const { items, loading, error, refresh } = useItems(
    useMemo(() => ({ ...filters, needsAttention: true }), [filters])
  )

  // Fetch all needs-attention items to power widgets and KPI tiles
  const { items: allItems, refresh: refreshAll } = useItems(useMemo(() => ({ needsAttention: true }), []))

  // Fetch snoozed items for the KPI tile
  const { items: snoozedItems, refresh: refreshSnoozed } = useItems(useMemo(() => ({ snoozed: true }), []))

  // Single callback that keeps both lists in sync after any mutation
  const refreshBoth = useCallback(() => { refresh(); refreshAll(); refreshSnoozed() }, [refresh, refreshAll, refreshSnoozed])

  // Update lastRefreshed on each successful load
  useEffect(() => {
    if (!loading && !error) setLastRefreshed(new Date().toISOString())
  }, [loading, error])

  // Sync mutations + status
  const { syncing, syncError, syncStatus, loadStatus, triggerSyncAll, triggerSyncSource } =
    useSync(refreshBoth)

  // Load sync status on mount
  useEffect(() => { loadStatus() }, [loadStatus])

  // Use configured source names from syncStatus as the primary list.
  // Fall back to source_name values found in items for any not yet in syncStatus.
  const availableSources = useMemo(() => {
    const fromConfig = syncStatus.map((s) => s.name)
    const fromItems = [...new Set(allItems.map((i) => i.source_name).filter(Boolean))]
    return [...new Set([...fromConfig, ...fromItems])]
  }, [allItems, syncStatus])
  const availableTypes = useMemo(() => uniqueTypes(allItems), [allItems])
  const sortedItems = useMemo(() => {
    let result = sortItems(items, sort)
    if (filters.signal) result = result.filter((i) => (i.active_signals ?? []).includes(filters.signal))
    if (filters.namespace) result = result.filter((i) => i.namespace === filters.namespace)
    return result
  }, [items, sort, filters.signal, filters.namespace])

  const handleWidgetFilter = useCallback((key, value) => {
    setFilters((f) => ({ ...f, [key]: f[key] === value ? '' : value }))
  }, [])

  const handleDismiss = useCallback(async (id) => {
    await dismissItem(id)
    refreshBoth()
  }, [refreshBoth])

  const handleSnoozeSuccess = useCallback(() => {
    refreshBoth()
  }, [refreshBoth])

  return (
    <div className="app">
      {showSplash && <SplashScreen onDone={() => setShowSplash(false)} />}
      <Header
        syncing={syncing}
        syncError={syncError}
        syncStatus={syncStatus}
        sources={syncStatus.map((s) => s.name)}
        onSyncAll={triggerSyncAll}
        onSyncSource={triggerSyncSource}
        lastRefreshed={lastRefreshed}
        onConfigOpen={() => setConfigOpen(true)}
      />

      <div className="app-body">
        {/* KPI tiles */}
        <div style={{ marginTop: 20 }}>
          <KpiTiles items={allItems} snoozedCount={snoozedItems.length} syncStatus={syncStatus} />
        </div>

        {/* Chart widget row */}
        <div className="widget-row">
          <SignalChart
            items={allItems}
            activeSignal={filters.signal}
            onSignalClick={(sig) => handleWidgetFilter('signal', sig)}
          />
          <SourceDonut
            items={allItems}
            activeSource={filters.sourceName}
            onSourceClick={(src) => handleWidgetFilter('sourceName', src)}
          />
          <NamespaceChart
            items={allItems}
            activeNamespace={filters.namespace}
            onNamespaceClick={(ns) => handleWidgetFilter('namespace', ns)}
          />
        </div>

        {/* Sync status bar */}
        <SyncStatus
          syncStatus={syncStatus}
          syncing={syncing}
          onSyncSource={triggerSyncSource}
        />

        {/* Filter bar + item list */}
        <FilterBar
          filters={filters}
          availableSources={availableSources}
          availableTypes={availableTypes}
          sort={sort}
          onFilterChange={setFilters}
          onSortChange={setSort}
          onRefresh={refresh}
          loading={loading}
        />

        <ItemList
          items={sortedItems}
          loading={loading}
          error={error}
          onDismiss={handleDismiss}
          onSnooze={setSnoozeTarget}
        />
      </div>

      {snoozeTarget && (
        <SnoozeModal
          item={snoozeTarget}
          onClose={() => setSnoozeTarget(null)}
          onSuccess={handleSnoozeSuccess}
        />
      )}

      {configOpen && (
        <ConfigEditor onClose={() => setConfigOpen(false)} />
      )}
    </div>
  )
}
