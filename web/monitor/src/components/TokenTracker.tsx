import { Token } from '../store';

interface TokenTrackerProps {
  tokens: Token[];
}

const styles = {
  token: {
    padding: '10px',
    marginBottom: '8px',
    borderRadius: '4px',
    backgroundColor: '#f5f5f5',
  },
  tokenActive: {
    borderLeft: '3px solid #4ecca3',
  },
  tokenWaiting: {
    borderLeft: '3px solid #ffc107',
  },
  tokenCompleted: {
    borderLeft: '3px solid #6c757d',
    opacity: 0.7,
  },
  tokenName: {
    fontSize: '14px',
    fontWeight: 500,
    color: '#1a1a2e',
  },
  tokenMeta: {
    fontSize: '12px',
    color: '#666',
    marginTop: '4px',
  },
};

export default function TokenTracker({ tokens }: TokenTrackerProps) {
  if (tokens.length === 0) {
    return (
      <div style={{ color: '#999', fontSize: '14px' }}>
        No active tokens
      </div>
    );
  }

  const getTokenStyle = (status: string) => {
    switch (status) {
      case 'active':
        return { ...styles.token, ...styles.tokenActive };
      case 'waiting':
        return { ...styles.token, ...styles.tokenWaiting };
      case 'completed':
        return { ...styles.token, ...styles.tokenCompleted };
      default:
        return styles.token;
    }
  };

  return (
    <div>
      {tokens.map((token) => (
        <div key={token.id} style={getTokenStyle(token.status)}>
          <div style={styles.tokenName}>
            {token.elementName || token.elementId}
          </div>
          <div style={styles.tokenMeta}>
            Status: {token.status}
          </div>
          <div style={{ ...styles.tokenMeta, color: '#999' }}>
            Arrived: {new Date(token.arrivedAt).toLocaleTimeString()}
          </div>
        </div>
      ))}
    </div>
  );
}
