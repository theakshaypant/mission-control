import './FilterBar.css'

const SORT_OPTIONS = [
  { value: 'updated_at', label: 'Updated (newest)' },
  { value: 'source', label: 'Source' },
  { value: 'type', label: 'Type' },
]

export function FilterBar({ filters, availableSources, availableTypes, sort, onFilterChange, onSortChange, onRefresh, loading }) {
  function toggleType(type) {
    const current = filters.types ?? []
    const next = current.includes(type)
      ? current.filter((t) => t !== type)
      : [...current, type]
    onFilterChange({ ...filters, types: next })
  }

  return (
    <div className="filter-bar card">
      <div className="filter-bar-inner">
        {/* Source filter */}
        {availableSources.length > 0 && (
          <div className="filter-group">
            <span className="label">Source</span>
            <select
              className="filter-select"
              value={filters.sourceName ?? ''}
              onChange={(e) => onFilterChange({ ...filters, sourceName: e.target.value || '' })}
            >
              <option value="">All</option>
              {availableSources.map((s) => (
                <option key={s} value={s}>{s}</option>
              ))}
            </select>
          </div>
        )}

        {/* Type filter */}
        {availableTypes.length > 0 && (
          <div className="filter-group">
            <span className="label">Type</span>
            {availableTypes.map((t) => (
              <button
                key={t}
                className={`filter-toggle ${(filters.types ?? []).includes(t) ? 'active' : ''}`}
                onClick={() => toggleType(t)}
              >
                {t.toUpperCase()}
              </button>
            ))}
          </div>
        )}

        <div className="filter-divider" />

        {/* Sort */}
        <div className="filter-group">
          <span className="label">Sort</span>
          <select
            className="filter-select"
            value={sort}
            onChange={(e) => onSortChange(e.target.value)}
          >
            {SORT_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>{o.label}</option>
            ))}
          </select>
        </div>

        {/* Refresh */}
        <button
          className="btn btn-ghost filter-refresh"
          onClick={onRefresh}
          disabled={loading}
          aria-label="Refresh items"
        >
          {loading ? <span className="spinner" /> : '↻'} REFRESH
        </button>
      </div>
    </div>
  )
}
