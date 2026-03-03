import React from 'react';

interface Task {
  id: string;
  name: string;
  processInstanceId: string;
  processDefinitionId: string;
  assignee: string | null;
  createdAt: string;
  dueDate?: string;
  priority?: number;
  variables?: Record<string, any>;
}

interface TaskCardProps {
  task: Task;
  selected?: boolean;
  onSelect?: (task: Task) => void;
}

const styles = {
  card: {
    padding: '14px',
    marginBottom: '10px',
    borderRadius: '8px',
    cursor: 'pointer',
    transition: 'all 0.2s',
    border: '1px solid #e8e8e8',
    backgroundColor: '#fff',
  },
  selected: {
    borderColor: '#4ecca3',
    backgroundColor: '#f0fff4',
  },
  header: {
    display: 'flex',
    alignItems: 'flex-start',
    justifyContent: 'space-between',
    marginBottom: '8px',
  },
  name: {
    fontSize: '14px',
    fontWeight: 500,
    color: '#1a1a2e',
    lineHeight: 1.4,
  },
  priority: {
    padding: '2px 6px',
    borderRadius: '4px',
    fontSize: '10px',
    fontWeight: 600,
    textTransform: 'uppercase' as const,
  },
  highPriority: {
    backgroundColor: '#ffebee',
    color: '#c62828',
  },
  mediumPriority: {
    backgroundColor: '#fff3e0',
    color: '#e65100',
  },
  lowPriority: {
    backgroundColor: '#e8f5e9',
    color: '#2e7d32',
  },
  meta: {
    fontSize: '11px',
    color: '#666',
    marginBottom: '4px',
  },
  assignee: {
    fontSize: '11px',
    color: '#888',
    marginTop: '6px',
    display: 'flex',
    alignItems: 'center',
    gap: '4px',
  },
  dueDate: {
    fontSize: '11px',
    marginTop: '6px',
  },
  overdue: {
    color: '#e53935',
  },
};

export default function TaskCard({ task, selected, onSelect }: TaskCardProps) {
  const getPriorityStyle = (priority?: number) => {
    if (!priority) return {};
    if (priority >= 80) return styles.highPriority;
    if (priority >= 50) return styles.mediumPriority;
    return styles.lowPriority;
  };

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr);
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  const isOverdue = () => {
    if (!task.dueDate) return false;
    return new Date(task.dueDate) < new Date();
  };

  const handleClick = () => {
    onSelect?.(task);
  };

  return (
    <div
      style={{
        ...styles.card,
        ...(selected ? styles.selected : {}),
      }}
      onClick={handleClick}
    >
      <div style={styles.header}>
        <div style={styles.name}>{task.name}</div>
        {task.priority !== undefined && task.priority !== null && (
          <span style={{ ...styles.priority, ...getPriorityStyle(task.priority) }}>
            {task.priority}
          </span>
        )}
      </div>

      <div style={styles.meta}>
        📋 {task.processDefinitionId.split('-')[0] || 'Process'}
      </div>

      <div style={styles.meta}>
        🕐 {formatDate(task.createdAt)}
      </div>

      {task.dueDate && (
        <div style={{ ...styles.dueDate, ...(isOverdue() ? styles.overdue : {}) }}>
          ⏰ Due: {formatDate(task.dueDate)}
        </div>
      )}

      <div style={styles.assignee}>
        👤 {task.assignee ? task.assignee : 'Unassigned'}
      </div>
    </div>
  );
}
