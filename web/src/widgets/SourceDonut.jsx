import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer, Legend } from 'recharts'
import { countBy } from '../utils'
import './Widget.css'

const COLORS = ['var(--red)', 'var(--blue)', 'var(--green)', 'var(--amber)']

const CustomTooltip = ({ active, payload }) => {
  if (!active || !payload?.length) return null
  return (
    <div className="chart-tooltip">
      <span className="mono">{payload[0].name}</span>
      <span className="chart-tooltip-value">{payload[0].value}</span>
    </div>
  )
}

const renderLegend = (props) => {
  const { payload } = props
  return (
    <div className="donut-legend">
      {payload.map((entry) => (
        <div key={entry.value} className="donut-legend-item">
          <span className="donut-legend-dot" style={{ background: entry.color }} />
          <span className="mono donut-legend-label">{entry.value}</span>
          <span className="donut-legend-count">{entry.payload.count}</span>
        </div>
      ))}
    </div>
  )
}

export function SourceDonut({ items, activeSource, onSourceClick }) {
  const data = countBy(items, 'source_name')

  if (data.length === 0) {
    return (
      <div className="widget card">
        <div className="widget-header">
          <span className="label">Assets by Source</span>
        </div>
        <div className="widget-empty">No data</div>
      </div>
    )
  }

  return (
    <div className="widget card">
      <div className="widget-header">
        <span className="label">Assets by Source</span>
        {activeSource
          ? <button className="widget-filter-clear" onClick={() => onSourceClick(activeSource)}>{activeSource} ×</button>
          : <span className="widget-count">{items.length} total</span>
        }
      </div>
      <ResponsiveContainer width="100%" height={180}>
        <PieChart>
          <Pie
            data={data}
            dataKey="count"
            nameKey="label"
            cx="50%"
            cy="50%"
            innerRadius={48}
            outerRadius={72}
            paddingAngle={2}
            cursor="pointer"
            onClick={(d) => onSourceClick(d.label)}
          >
            {data.map((entry, i) => (
              <Cell
                key={entry.label}
                fill={COLORS[i % COLORS.length]}
                opacity={activeSource && activeSource !== entry.label ? 0.25 : 1}
              />
            ))}
          </Pie>
          <Tooltip content={<CustomTooltip />} />
          <Legend content={renderLegend} />
        </PieChart>
      </ResponsiveContainer>
    </div>
  )
}
