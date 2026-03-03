import React, { useState, useEffect } from 'react';

interface ProcessEvent {
  id: string;
  processInstanceId: string;
  eventType: string;
  elementId: string;
  timestamp: string;
  variables?: Record<string, any>;
}

interface ExecutionLogProps {
  instanceId: string;
}

const styles = {
  container: {
    maxHeight: '300px',
    overflowY: 'auto' as const,
  },
  eventItem: {
    padding: '10px',
    marginBottom: '8px',
    backgroundColor: '#f8f9fa',
    borderRadius: '6px',
    borderLeft: '3px solid #4ecca3',
  },
  eventHeader: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: '4px',
  },
  eventType: {
    fontSize: '12px',
    fontWeight: 600,
    color: '#1a1a2e',
    textTransform: 'capitalize' as const,
  },
  timestamp: {
    fontSize: '10px',
    color: '#888',
  },
  elementId: {
    fontSize: '11px',
    color: '#666',
  },
  emptyState: {
    padding: '20px',
    textAlign: 'center' as const,
    color: '#888',
    fontSize: '12px',
  },
  loading: {
    padding: '15px',
    textAlign: 'center' as const,
    color: '#666',
    fontSize: '12px',
  },
};

// Mock events for demo
const mockEvents: ProcessEvent[] = [
  {
    id: '1',
    processInstanceId: '',
    eventType: 'process_started',
    elementId: 'StartEvent',
    timestamp: new Date(Date.now() - 3600000).toISOString(),
  },
  {
    id: '2',
    processInstanceId: '',
    eventType: 'task_completed',
    elementId: 'Task_Review',
    timestamp: new Date(Date.now() - 1800000).toISOString(),
  },
  {
    id: '3',
    processInstanceId: '',
    eventType: 'task_created',
    elementId: 'Task_Approve',
    timestamp: new Date(Date.now() - 900000).toISOString(),
  },
];

export default function ExecutionLog({ instanceId }: ExecutionLogProps) {
  const [events, setEvents] = useState<ProcessEvent[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!instanceId) return;

    setLoading(true);
    // Fetch events from API
    fetch(`/api/v1/instances/${instanceId}/events`)
      .then(res => res.json())
      .then(data => setEvents(data))
      .catch(() => {
        // Use mock data for demo
        setEvents(mockEvents.map(e => ({ ...e, processInstanceId: instanceId })));
      })
      .finally(() => setLoading(false));
  }, [instanceId]);

  const formatTimestamp = (timestamp: string) => {
    const date = new Date(timestamp);
    return date.toLocaleTimeString('en-US', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    });
  };

  const getEventIcon = (eventType: string) => {
    if (eventType.includes('started') || eventType.includes('created')) return '🟢';
    if (eventType.includes('completed') || eventType.includes('ended')) return '✅';
    if (eventType.includes('error') || eventType.includes('failed')) return '❌';
    if (eventType.includes('suspended') || eventType.includes('paused')) return '⏸️';
    return '⚪';
  };

  if (loading) {
    return <div style={styles.loading}>Loading events...</div>;
  }

  if (events.length === 0) {
    return (
      <div style={styles.emptyState}>
        No events recorded yet
      </div>
    );
  }

  return (
    <div style={styles.container}>
      {events.map((event) => (
        <div key={event.id} style={styles.eventItem}>
          <div style={styles.eventHeader}>
            <span style={styles.eventType}>
              {getEventIcon(event.eventType)} {event.eventType.replace(/_/g, ' ')}
            </span>
            <span style={styles.timestamp}>{formatTimestamp(event.timestamp)}</span>
          </div>
          <div style={styles.elementId}>
            📍 {event.elementId}
          </div>
        </div>
      ))}
    </div>
  );
}
