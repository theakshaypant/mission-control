import { useState, useCallback } from 'react'
import { syncAll, syncSource, fetchSyncStatus } from '../api'

/**
 * Exposes sync mutations and sync status fetching.
 * onSuccess is called after any successful sync to trigger item refresh.
 *
 * @param {() => void} onSuccess
 */
export function useSync(onSuccess) {
  const [syncing, setSyncing] = useState(false)
  const [syncError, setSyncError] = useState(null)
  const [syncStatus, setSyncStatus] = useState([])

  const loadStatus = useCallback(async () => {
    try {
      const data = await fetchSyncStatus()
      setSyncStatus(data ?? [])
    } catch (_) {
      // Non-fatal: sync status widget just shows empty
    }
  }, [])

  const triggerSyncAll = useCallback(async () => {
    setSyncing(true)
    setSyncError(null)
    try {
      await syncAll()
      await loadStatus()
      onSuccess?.()
    } catch (err) {
      setSyncError(err.message ?? 'Sync failed')
    } finally {
      setSyncing(false)
    }
  }, [onSuccess, loadStatus])

  const triggerSyncSource = useCallback(async (source) => {
    setSyncing(true)
    setSyncError(null)
    try {
      await syncSource(source)
      await loadStatus()
      onSuccess?.()
    } catch (err) {
      setSyncError(err.message ?? 'Sync failed')
    } finally {
      setSyncing(false)
    }
  }, [onSuccess, loadStatus])

  return { syncing, syncError, syncStatus, loadStatus, triggerSyncAll, triggerSyncSource }
}
