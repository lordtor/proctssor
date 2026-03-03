import React from 'react';

interface ControlPanelProps {
  status: string;
  onAction: (action: 'suspend' | 'resume' | 'terminate') => void;
  isLoading?: boolean;
}

const styles = {
  container: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
  },
  button: {
    padding: '6px 12px',
    border: 'none',
    borderRadius: '4px',
    cursor: 'pointer',
    fontSize: '12px',
    fontWeight: 500,
    transition: 'all 0.2s',
    display: 'flex',
    alignItems: 'center',
    gap: '4px',
  },
  suspendButton: {
    backgroundColor: '#ffc107',
    color: '#000',
  },
  resumeButton: {
    backgroundColor: '#4ecca3',
    color: '#1a1a2e',
  },
  terminateButton: {
    backgroundColor: '#dc3545',
    color: '#fff',
  },
  disabledButton: {
    opacity: 0.6,
    cursor: 'not-allowed',
  },
  statusBadge: {
    padding: '4px 10px',
    borderRadius: '12px',
    fontSize: '11px',
    fontWeight: 600,
    textTransform: 'uppercase' as const,
  },
  activeStatus: {
    backgroundColor: '#d4edda',
    color: '#155724',
  },
  completedStatus: {
    backgroundColor: '#cce5ff',
    color: '#004085',
  },
  suspendedStatus: {
    backgroundColor: '#fff3cd',
    color: '#856404',
  },
  terminatedStatus: {
    backgroundColor: '#f8d7da',
    color: '#721c24',
  },
};

export default function ControlPanel({ status, onAction, isLoading = false }: ControlPanelProps) {
  const getStatusBadgeStyle = () => {
    switch (status) {
      case 'active':
        return { ...styles.statusBadge, ...styles.activeStatus };
      case 'completed':
        return { ...styles.statusBadge, ...styles.completedStatus };
      case 'suspended':
        return { ...styles.statusBadge, ...styles.suspendedStatus };
      case 'terminated':
        return { ...styles.statusBadge, ...styles.terminatedStatus };
      default:
        return styles.statusBadge;
    }
  };

  const handleAction = (action: 'suspend' | 'resume' | 'terminate') => {
    if (isLoading) return;
    onAction(action);
  };

  return (
    <div style={styles.container}>
      <span style={getStatusBadgeStyle()}>{status}</span>
      
      {status === 'active' && (
        <>
          <button
            style={{
              ...styles.button,
              ...styles.suspendButton,
              ...(isLoading ? styles.disabledButton : {}),
            }}
            onClick={() => handleAction('suspend')}
            disabled={isLoading}
            title="Suspend process instance"
          >
            ⏸️ Suspend
          </button>
          
          <button
            style={{
              ...styles.button,
              ...styles.terminateButton,
              ...(isLoading ? styles.disabledButton : {}),
            }}
            onClick={() => handleAction('terminate')}
            disabled={isLoading}
            title="Terminate process instance"
          >
            🛑 Terminate
          </button>
        </>
      )}
      
      {status === 'suspended' && (
        <button
          style={{
            ...styles.button,
            ...styles.resumeButton,
            ...(isLoading ? styles.disabledButton : {}),
          }}
          onClick={() => handleAction('resume')}
          disabled={isLoading}
          title="Resume suspended process instance"
        >
          ▶️ Resume
        </button>
      )}
    </div>
  );
}
