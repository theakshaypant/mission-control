// api.js — thin fetch wrapper for the mission-control REST API.
//
// Item IDs like "github:org/repo#42" contain characters that need encoding.
// encodeURIComponent handles this correctly:
//   "github:org/repo#42" → "github%3Aorg%2Frepo%2342"
// Go's net/http mux correctly decodes %2F in path wildcards (confirmed by tests).

const BASE = import.meta.env.VITE_API_BASE ?? ''

class ApiError extends Error {
  constructor(status, message) {
    super(message)
    this.status = status
  }
}

async function apiFetch(path, options = {}) {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json', ...options.headers },
    ...options,
  })
  if (!res.ok) {
    let msg = `HTTP ${res.status}`
    try {
      const body = await res.json()
      if (body.error) msg = body.error
    } catch (_) {}
    throw new ApiError(res.status, msg)
  }
  // 204 No Content — no body to parse
  if (res.status === 204) return undefined
  return res.json()
}

// ── Items ──────────────────────────────────────────────────

/**
 * Fetch items with optional filters.
 * @param {{ needsAttention?: boolean, waitsOnMe?: boolean, source?: string, types?: string[] }} filters
 */
export async function fetchItems(filters = {}) {
  const params = new URLSearchParams()
  if (filters.needsAttention) params.set('needs_attention', 'true')
  if (filters.waitsOnMe) params.set('waits_on_me', 'true')
  if (filters.snoozed) params.set('snoozed', 'true')
  if (filters.source) params.set('source', filters.source)
  if (filters.sourceName) params.set('source_name', filters.sourceName)
  for (const t of filters.types ?? []) params.append('type', t)
  const qs = params.toString()
  return apiFetch(`/items${qs ? `?${qs}` : ''}`)
}

/** Fetch items that need attention (shorthand endpoint). */
export async function fetchSummary() {
  return apiFetch('/summary')
}

/** Permanently dismiss an item. */
export async function dismissItem(id) {
  return apiFetch(`/items/${encodeURIComponent(id)}/dismiss`, { method: 'POST' })
}

/**
 * Snooze an item.
 * @param {string} id
 * @param {'for' | 'until'} mode
 * @param {string} value - duration string (e.g. "24h") or date string
 */
export async function snoozeItem(id, mode, value) {
  return apiFetch(`/items/${encodeURIComponent(id)}/snooze`, {
    method: 'POST',
    body: JSON.stringify({ [mode]: value }),
  })
}

// ── Sync ───────────────────────────────────────────────────

/** Trigger a full sync of all sources. */
export async function syncAll() {
  return apiFetch('/sync', { method: 'POST' })
}

/** Trigger a sync for a single named source. */
export async function syncSource(source) {
  return apiFetch(`/sync/${encodeURIComponent(source)}`, { method: 'POST' })
}

/** Get last sync time per configured source. */
export async function fetchSyncStatus() {
  return apiFetch('/sync/status')
}
