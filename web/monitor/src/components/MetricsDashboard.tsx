import { useEffect } from 'react';
import { useMonitorStore, Metrics } from '../store';

interface MetricsDashboardProps {
  compact?: boolean;
}

const styles = {
  container: {
    padding: '20px',
    backgroundColor: '#fff',
    borderRadius: '8px',
    boxShadow: '0 1px 3px rgba(0,0,0,0.1)',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '20px',
  },
  title: {
    fontSize: '18px',
    fontWeight: 600,
    color: '#1a1a2e',
  },
  refreshButton: {
    padding: '6px 12px',
    backgroundColor: '#1a1a2e',
    color: '#fff',
    border: 'none',
    borderRadius: '4px',
    cursor: 'pointer',
    fontSize: '12px',
  },
  statsGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(3, 1fr)',
    gap: '16px',
    marginBottom: '24px',
  },
  statCard: {
    padding: '16px',
    borderRadius: '8px',
    backgroundColor: '#f8f9fa',
  },
  statLabel: {
    fontSize: '12px',
    color: '#666',
    marginBottom: '4px',
  },
  statValue: {
    fontSize: '24px',
    fontWeight: 600,
    color: '#1a1a2e',
  },
  statSubtext: {
    fontSize: '11px',
    color: '#999',
    marginTop: '4px',
  },
  chartsGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(2, 1fr)',
    gap: '16px',
  },
  chartCard: {
    padding: '16px',
    borderRadius: '8px',
    backgroundColor: '#f8f9fa',
  },
  chartTitle: {
    fontSize: '14px',
    fontWeight: 500,
    color: '#1a1a2e',
    marginBottom: '16px',
  },
  chartContainer: {
    height: '150px',
    display: 'flex',
    alignItems: 'flex-end',
    gap: '8px',
    paddingTop: '10px',
  },
  bar: {
    flex: 1,
    borderRadius: '4px 4px 0 0',
    transition: 'height 0.3s ease',
    position: 'relative' as const,
  },
  barLabel: {
    position: 'absolute' as const,
    bottom: '-20px',
    left: '50%',
    transform: 'translateX(-50%)',
    fontSize: '10px',
    color: '#999',
    whiteSpace: 'nowrap' as const,
  },
  errorRate: {
    color: '#e74c3c',
  },
  successRate: {
    color: '#4ecca3',
  },
  compactContainer: {
    padding: '10px',
  },
  compactStatsGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(3, 1fr)',
    gap: '8px',
  },
  compactStatCard: {
    padding: '8px',
    borderRadius: '4px',
    backgroundColor: '#f8f9fa',
    textAlign: 'center' as const,
  },
  compactStatValue: {
    fontSize: '16px',
    fontWeight: 600,
    color: '#1a1a2e',
  },
  compactStatLabel: {
    fontSize: '10px',
    color: '#666',
  },
};

function MiniChart({ 
  data, 
  color = '#4ecca3',
  maxValue,
  showLabels = true 
}: { 
  data: { date: string; value: number }[];
  color?: string;
  maxValue?: number;
  showLabels?: boolean;
}) {
  const max = maxValue || Math.max(...data.map(d => d.value));
  
  return (
    <div style={styles.chartContainer}>
      {data.map((item, index) => (
        <div
          key={index}
          style={{
            ...styles.bar,
            height: `${(item.value / max) * 100}%`,
            backgroundColor: color,
          }}
        >
          {showLabels && (
            <span style={styles.barLabel}>
              {item.date.slice(5)}
            </span>
          )}
        </div>
      ))}
    </div>
  );
}

export default function MetricsDashboard({ compact = false }: MetricsDashboardProps) {
  const { metrics, fetchMetrics, loading } = useMonitorStore();

  useEffect(() => {
    fetchMetrics();
  }, []);

  const handleRefresh = () => {
    fetchMetrics();
  };

  if (!metrics) {
    return (
      <div style={compact ? { ...styles.container, ...styles.compactContainer } : styles.container}>
        <div style={{ textAlign: 'center', padding: '20px', color: '#999' }}>
          {loading ? 'Loading metrics...' : 'No metrics available'}
        </div>
      </div>
    );
  }

  if (compact) {
    return (
      <div style={styles.compactStatsGrid}>
        <div style={styles.compactStatCard}>
          <div style={styles.compactStatValue}>{metrics.processCount}</div>
          <div style={styles.compactStatLabel}>Total</div>
        </div>
        <div style={styles.compactStatCard}>
          <div style={styles.compactStatValue}>
            {Math.round(metrics.averageDuration / 1000)}s
          </div>
          <div style={styles.compactStatLabel}>Avg Duration</div>
        </div>
        <div style={styles.compactStatCard}>
          <div style={{ 
            ...styles.compactStatValue,
            color: metrics.errorRate > 5 ? '#e74c3c' : '#4ecca3'
          }}>
            {metrics.errorRate}%
          </div>
          <div style={styles.compactStatLabel}>Errors</div>
        </div>
      </div>
    );
  }

  return (
    <div style={styles.container}>
      <div style={styles.header}>
        <div style={styles.title}>Metrics Dashboard</div>
        <button 
          style={styles.refreshButton} 
          onClick={handleRefresh}
          disabled={loading}
        >
          {loading ? 'Refreshing...' : 'Refresh'}
        </button>
      </div>

      <div style={styles.statsGrid}>
        <div style={styles.statCard}>
          <div style={styles.statLabel}>Total Processes</div>
          <div style={styles.statValue}>{metrics.processCount}</div>
          <div style={styles.statSubtext}>
            {metrics.activeCount} active, {metrics.completedCount} completed
          </div>
        </div>
        
        <div style={styles.statCard}>
          <div style={styles.statLabel}>Average Duration</div>
          <div style={styles.statValue}>
            {Math.round(metrics.averageDuration / 1000)}s
          </div>
          <div style={styles.statSubtext}>per process execution</div>
        </div>
        
        <div style={styles.statCard}>
          <div style={styles.statLabel}>Error Rate</div>
          <div style={{ 
            ...styles.statValue,
            color: metrics.errorRate > 5 ? '#e74c3c' : '#4ecca3'
          }}>
            {metrics.errorRate}%
          </div>
          <div style={styles.statSubtext}>
            {metrics.terminatedCount} terminated
          </div>
        </div>
      </div>

      <div style={styles.chartsGrid}>
        <div style={styles.chartCard}>
          <div style={styles.chartTitle}>Duration Trend (last 7 days)</div>
          <MiniChart 
            data={metrics.durationHistory} 
            color="#4ecca3"
          />
        </div>
        
        <div style={styles.chartCard}>
          <div style={styles.chartTitle}>Error Rate Trend (last 7 days)</div>
          <MiniChart 
            data={metrics.errorHistory} 
            color="#e74c3c"
            maxValue={10}
          />
        </div>
      </div>
    </div>
  );
}
