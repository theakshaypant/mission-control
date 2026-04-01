import { BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, Cell } from 'recharts'
import { countSignals } from '../utils'
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

function barColor(entry, index, activeSignal) {
  if (activeSignal) return activeSignal === entry.label ? 'var(--red)' : 'var(--bg-surface)'
  return index === 0 ? 'var(--red)' : 'var(--border-bright)'
}

export function SignalChart({ items, activeSignal, onSignalClick }) {
  const data = countSignals(items).slice(0, 8)

  if (data.length === 0) {
    return (
      <div className="widget card">
        <div className="widget-header">
          <span className="label">Signal Breakdown</span>
        </div>
        <div className="widget-empty">No signals</div>
      </div>
    )
  }

  return (
    <div className="widget card">
      <div className="widget-header">
        <span className="label">Signal Breakdown</span>
        {activeSignal
          ? <button className="widget-filter-clear" onClick={() => onSignalClick(activeSignal)}>{activeSignal} ×</button>
          : <span className="widget-count">{data.length} types</span>
        }
      </div>
      <ResponsiveContainer width="100%" height={Math.max(120, data.length * 28)}>
        <BarChart data={data} layout="vertical" margin={{ top: 4, right: 16, bottom: 4, left: 8 }}>
          <XAxis type="number" hide />
          <YAxis
            type="category"
            dataKey="label"
            width={130}
            tick={{ fill: 'var(--text-muted)', fontSize: 10, fontFamily: 'var(--font-mono)' }}
            tickLine={false}
            axisLine={false}
          />
          <Tooltip content={<CustomTooltip />} cursor={{ fill: 'var(--bg-hover)' }} />
          <Bar dataKey="count" radius={[0, 2, 2, 0]} maxBarSize={14} cursor="pointer" onClick={(d) => onSignalClick(d.label)}>
            {data.map((entry, i) => (
              <Cell
                key={entry.label}
                fill={barColor(entry, i, activeSignal)}
                opacity={activeSignal && activeSignal !== entry.label ? 0.35 : 1}
              />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  )
}
