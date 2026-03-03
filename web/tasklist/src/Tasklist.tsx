import { useEffect, useState } from 'react';
import { useTaskStore, Task } from './store';
import DynamicForm from './components/DynamicForm';

const styles = {
  container: {
    display: 'flex',
    height: '100vh',
    backgroundColor: '#f5f5f5',
  },
  sidebar: {
    width: '350px',
    backgroundColor: '#fff',
    borderRight: '1px solid #e0e0e0',
    display: 'flex',
    flexDirection: 'column' as const,
  },
  header: {
    padding: '20px',
    borderBottom: '1px solid #e0e0e0',
  },
  title: {
    fontSize: '18px',
    fontWeight: 600,
    color: '#1a1a2e',
    marginBottom: '15px',
  },
  filterButtons: {
    display: 'flex',
    gap: '8px',
  },
  filterButton: {
    padding: '6px 12px',
    border: 'none',
    borderRadius: '4px',
    fontSize: '12px',
    cursor: 'pointer',
    transition: 'all 0.2s',
  },
  activeFilter: {
    backgroundColor: '#4ecca3',
    color: '#1a1a2e',
  },
  inactiveFilter: {
    backgroundColor: '#f0f0f0',
    color: '#666',
  },
  taskList: {
    flex: 1,
    overflowY: 'auto' as const,
    padding: '10px',
  },
  taskCard: {
    padding: '15px',
    marginBottom: '10px',
    borderRadius: '8px',
    cursor: 'pointer',
    transition: 'all 0.2s',
    border: '1px solid #e0e0e0',
    backgroundColor: '#fff',
  },
  selectedTask: {
    borderColor: '#4ecca3',
    backgroundColor: '#f0fff4',
  },
  taskName: {
    fontSize: '14px',
    fontWeight: 500,
    color: '#1a1a2e',
    marginBottom: '5px',
  },
  taskMeta: {
    fontSize: '12px',
    color: '#666',
  },
  taskAssignee: {
    fontSize: '12px',
    color: '#999',
    marginTop: '5px',
  },
  main: {
    flex: 1,
    display: 'flex',
    flexDirection: 'column' as const,
    backgroundColor: '#fff',
  },
  emptyState: {
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    height: '100%',
    color: '#999',
    fontSize: '16px',
  },
  taskHeader: {
    padding: '20px',
    borderBottom: '1px solid #e0e0e0',
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  taskTitle: {
    fontSize: '18px',
    fontWeight: 600,
    color: '#1a1a2e',
  },
  actionButton: {
    padding: '8px 16px',
    border: 'none',
    borderRadius: '4px',
    cursor: 'pointer',
    fontSize: '14px',
    fontWeight: 500,
  },
  claimButton: {
    backgroundColor: '#4ecca3',
    color: '#1a1a2e',
  },
  completeButton: {
    backgroundColor: '#1a1a2e',
    color: '#fff',
  },
  formContainer: {
    flex: 1,
    padding: '20px',
    overflowY: 'auto' as const,
  },
};

const mockTasks: Task[] = [
  {
    id: '1',
    name: 'Review Application',
    processInstanceId: 'inst-001',
    processDefinitionId: 'process-001',
    assignee: null,
    createdAt: new Date().toISOString(),
    dueDate: null,
    variables: {},
    formSchema: {
      fields: [
        { name: 'comments', type: 'textarea', label: 'Comments', required: false },
        { name: 'approved', type: 'boolean', label: 'Approve', required: true },
      ],
    },
  },
  {
    id: '2',
    name: 'Process Payment',
    processInstanceId: 'inst-002',
    processDefinitionId: 'process-002',
    assignee: 'current-user',
    createdAt: new Date(Date.now() - 3600000).toISOString(),
    dueDate: new Date(Date.now() + 86400000).toISOString(),
    variables: { amount: 1000 },
    formSchema: {
      fields: [
        { name: 'paymentMethod', type: 'select', label: 'Payment Method', options: ['Card', 'Bank'], required: true },
      ],
    },
  },
];

export default function Tasklist() {
  const {
    tasks,
    selectedTask,
    loading,
    filter,
    fetchTasks,
    selectTask,
    claimTask,
    unclaimTask,
    completeTask,
    setFilter,
  } = useTaskStore();

  const [formData, setFormData] = useState<Record<string, any>>({});

  useEffect(() => {
    fetchTasks();
  }, []);

  const handleComplete = async () => {
    if (selectedTask) {
      await completeTask(selectedTask.id, formData);
      setFormData({});
    }
  };

  const allTasks = Array.isArray(tasks) && tasks.length > 0 ? tasks : mockTasks;
  const filteredTasks: Task[] = Array.isArray(allTasks) ? allTasks.filter(task => {
    if (filter === 'assigned') return task.assignee !== null;
    if (filter === 'unassigned') return task.assignee === null;
    return true;
  }) : [];

  return (
    <div style={styles.container}>
      <div style={styles.sidebar}>
        <div style={styles.header}>
          <div style={styles.title}>Inbox</div>
          <div style={styles.filterButtons}>
            <button
              style={{
                ...styles.filterButton,
                ...(filter === 'all' ? styles.activeFilter : styles.inactiveFilter),
              }}
              onClick={() => setFilter('all')}
            >
              All
            </button>
            <button
              style={{
                ...styles.filterButton,
                ...(filter === 'assigned' ? styles.activeFilter : styles.inactiveFilter),
              }}
              onClick={() => setFilter('assigned')}
            >
              Assigned
            </button>
            <button
              style={{
                ...styles.filterButton,
                ...(filter === 'unassigned' ? styles.activeFilter : styles.inactiveFilter),
              }}
              onClick={() => setFilter('unassigned')}
            >
              Unassigned
            </button>
          </div>
        </div>

        <div style={styles.taskList}>
          {loading && <div style={{ padding: '20px', textAlign: 'center' }}>Loading...</div>}
          
          {filteredTasks.map((task) => (
            <div
              key={task.id}
              style={{
                ...styles.taskCard,
                ...(selectedTask?.id === task.id ? styles.selectedTask : {}),
              }}
              onClick={() => selectTask(task)}
            >
              <div style={styles.taskName}>{task.name}</div>
              <div style={styles.taskMeta}>
                Created: {new Date(task.createdAt).toLocaleDateString()}
              </div>
              {task.dueDate && (
                <div style={{ ...styles.taskMeta, color: '#e74c3c' }}>
                  Due: {new Date(task.dueDate).toLocaleDateString()}
                </div>
              )}
              <div style={styles.taskAssignee}>
                {task.assignee ? `Assigned to: ${task.assignee}` : 'Unassigned'}
              </div>
            </div>
          ))}
          
          {!loading && filteredTasks.length === 0 && (
            <div style={{ padding: '20px', textAlign: 'center', color: '#999' }}>
              No tasks found
            </div>
          )}
        </div>
      </div>

      <div style={styles.main}>
        {selectedTask ? (
          <>
            <div style={styles.taskHeader}>
              <div style={styles.taskTitle}>{selectedTask.name}</div>
              <div style={{ display: 'flex', gap: '10px' }}>
                {selectedTask.assignee ? (
                  <button
                    style={{ ...styles.actionButton, ...styles.claimButton }}
                    onClick={() => unclaimTask(selectedTask.id)}
                  >
                    Unclaim
                  </button>
                ) : (
                  <button
                    style={{ ...styles.actionButton, ...styles.claimButton }}
                    onClick={() => claimTask(selectedTask.id)}
                  >
                    Claim
                  </button>
                )}
                {selectedTask.assignee && (
                  <button
                    style={{ ...styles.actionButton, ...styles.completeButton }}
                    onClick={handleComplete}
                  >
                    Complete
                  </button>
                )}
              </div>
            </div>

            <div style={styles.formContainer}>
              <DynamicForm
                schema={selectedTask.formSchema}
                values={formData}
                onChange={setFormData}
              />
            </div>
          </>
        ) : (
          <div style={styles.emptyState}>
            Select a task to view details
          </div>
        )}
      </div>
    </div>
  );
}
