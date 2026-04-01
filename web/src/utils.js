/**
 * Format a date/ISO string as a relative time (e.g. "3 hours ago").
 */
export function formatRelative(dateStr) {
  if (!dateStr) return '—'
  const diff = (new Date(dateStr).getTime() - Date.now()) / 1000 // seconds, negative = past
  const abs = Math.abs(diff)
  const fmt = new Intl.RelativeTimeFormat('en', { numeric: 'auto' })

  if (abs < 60) return fmt.format(Math.round(diff), 'second')
  if (abs < 3600) return fmt.format(Math.round(diff / 60), 'minute')
  if (abs < 86400) return fmt.format(Math.round(diff / 3600), 'hour')
  if (abs < 604800) return fmt.format(Math.round(diff / 86400), 'day')
  return fmt.format(Math.round(diff / 604800), 'week')
}

/**
 * Count occurrences of each value in an array of strings.
 * @returns {{ label: string, count: number }[]} sorted by count desc
 */
export function countBy(items, key) {
  const map = {}
  for (const item of items) {
    const val = item[key] ?? 'unknown'
    map[val] = (map[val] ?? 0) + 1
  }
  return Object.entries(map)
    .map(([label, count]) => ({ label, count }))
    .sort((a, b) => b.count - a.count)
}

/**
 * Flatten active_signals across all items into a counted list.
 * @returns {{ label: string, count: number }[]} sorted by count desc
 */
export function countSignals(items) {
  const map = {}
  for (const item of items) {
    for (const sig of item.active_signals ?? []) {
      map[sig] = (map[sig] ?? 0) + 1
    }
  }
  return Object.entries(map)
    .map(([label, count]) => ({ label, count }))
    .sort((a, b) => b.count - a.count)
}

/** Derive unique source names from item list. */
export function uniqueSources(items) {
  return [...new Set(items.map((i) => i.source_name))].filter(Boolean)
}

/** Derive unique types from item list. */
export function uniqueTypes(items) {
  return [...new Set(items.map((i) => i.type))].filter(Boolean)
}

/** Truncate a string to n chars, appending ellipsis. */
export function truncate(str, n) {
  if (!str || str.length <= n) return str
  return str.slice(0, n - 1) + '…'
}
