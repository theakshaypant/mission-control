import { useState, useEffect, useCallback, useRef } from 'react'
import { fetchItems } from '../api'

/**
 * Fetch items whenever filters change. Exposes refresh() for manual re-fetches
 * after mutations (dismiss, snooze, sync).
 *
 * @param {object} filters - { needsAttention, waitsOnMe, source, types }
 * @returns {{ items: object[], loading: boolean, error: string|null, refresh: () => void }}
 */
export function useItems(filters) {
  const [items, setItems] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)
  const [tick, setTick] = useState(0)

  // Stable refresh callback — increments tick to re-trigger the effect
  const refresh = useCallback(() => setTick((t) => t + 1), [])

  // Serialize filters so useEffect has a stable primitive dependency
  const filtersKey = JSON.stringify(filters)

  useEffect(() => {
    const controller = new AbortController()
    let cancelled = false

    setLoading(true)
    setError(null)

    fetchItems(filters)
      .then((data) => {
        if (!cancelled) {
          setItems(data ?? [])
          setLoading(false)
        }
      })
      .catch((err) => {
        if (!cancelled) {
          setError(err.message ?? 'Failed to load items')
          setLoading(false)
        }
      })

    return () => {
      cancelled = true
      controller.abort()
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filtersKey, tick])

  return { items, loading, error, refresh }
}
