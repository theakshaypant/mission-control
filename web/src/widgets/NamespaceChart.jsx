import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, Cell } from 'recharts'
import { countBy } from '../utils'
import './Widget.css'

const CustomTooltip = ({ active, payload }) => {
  if (!active || !payload?.length) return null
  return (
    <div className="chart-tooltip">
      <span className="mono">{payload[0].payload.label}</span>
      <span className="chart-tooltip-value">{payload[0].value}</span>
    </div>
  )
}

function barColor(entry, index, activeNamespace) {
  if (activeNamespace) return activeNamespace === entry.label ? 'var(--blue)' : 'var(--bg-surface)'
  return index === 0 ? 'var(--blue)' : 'var(--border-bright)'
}

export function NamespaceChart({ items, activeNamespace, onNamespaceClick }) {
  const data = countBy(items, 'namespace').slice(0, 6)

  if (data.length === 0) {
    return (
      <div className="widget card">
        <div className="widget-header">
          <span className="label">Namespace Intel</span>
        </div>
        <div className="widget-empty">No data</div>
      </div>
    )
  }

  return (
    <div className="widget card">
      <div className="widget-header">
        <span className="label">Namespace Intel</span>
        {activeNamespace
          ? <button className="widget-filter-clear" onClick={() => onNamespaceClick(activeNamespace)}>{activeNamespace} ×</button>
          : <span className="widget-count">Top {data.length}</span>
        }
      </div>
      <ResponsiveContainer width="100%" height={Math.max(120, data.length * 28)}>
        <BarChart data={data} layout="vertical" margin={{ top: 4, right: 16, bottom: 4, left: 8 }}>
          <XAxis type="number" hide />
          <YAxis
            type="category"
            dataKey="label"
            width={120}
            tick={{ fill: 'var(--text-muted)', fontSize: 10, fontFamily: 'var(--font-mono)' }}
            tickLine={false}
            axisLine={false}
          />
          <Tooltip content={<CustomTooltip />} cursor={{ fill: 'var(--bg-hover)' }} />
          <Bar dataKey="count" radius={[0, 2, 2, 0]} maxBarSize={14} cursor="pointer" onClick={(d) => onNamespaceClick(d.label)}>
            {data.map((entry, i) => (
              <Cell
                key={entry.label}
                fill={barColor(entry, i, activeNamespace)}
                opacity={activeNamespace && activeNamespace !== entry.label ? 0.35 : 1}
              />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  )
}
