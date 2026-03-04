import { useEffect, useState } from 'react';
import { useMonitorStore, ProcessInstance } from './store';
import DiagramViewer from './components/DiagramViewer';
import TokenTracker from './components/TokenTracker';
import VariablesPanel from './components/VariablesPanel';
import MetricsDashboard from './components/MetricsDashboard';

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
  },
  statusBar: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    marginTop: '10px',
    fontSize: '12px',
  },
  statusDot: {
    width: '8px',
    height: '8px',
    borderRadius: '50%',
  },
  connected: {
    backgroundColor: '#4ecca3',
  },
  disconnected: {
    backgroundColor: '#e74c3c',
  },
  filterButtons: {
    display: 'flex',
    gap: '8px',
    marginTop: '15px',
  },
  filterButton: {
    padding: '6px 12px',
    border: 'none',
    borderRadius: '4px',
    fontSize: '12px',
    cursor: 'pointer',
  },
  activeTab: {
    backgroundColor: '#1a1a2e',
    color: '#fff',
  },
  inactiveTab: {
    backgroundColor: '#f0f0f0',
    color: '#666',
  },
  instanceList: {
    flex: 1,
    overflowY: 'auto' as const,
    padding: '10px',
  },
  instanceCard: {
    padding: '15px',
    marginBottom: '10px',
    borderRadius: '8px',
    cursor: 'pointer',
    border: '1px solid #e0e0e0',
    backgroundColor: '#fff',
    transition: 'all 0.2s',
  },
  selectedInstance: {
    borderColor: '#4ecca3',
    backgroundColor: '#f0fff4',
  },
  instanceName: {
    fontSize: '14px',
    fontWeight: 500,
    color: '#1a1a2e',
  },
  instanceMeta: {
    fontSize: '12px',
    color: '#666',
    marginTop: '5px',
  },
  statusBadge: {
    display: 'inline-block',
    padding: '2px 8px',
    borderRadius: '12px',
    fontSize: '10px',
    fontWeight: 500,
    textTransform: 'uppercase' as const,
  },
  activeBadge: {
    backgroundColor: '#d4edda',
    color: '#155724',
  },
  completedBadge: {
    backgroundColor: '#cce5ff',
    color: '#004085',
  },
  terminatedBadge: {
    backgroundColor: '#f8d7da',
    color: '#721c24',
  },
  main: {
    flex: 1,
    display: 'flex',
    flexDirection: 'column' as const,
  },
  toolbar: {
    display: 'flex',
    alignItems: 'center',
    gap: '10px',
    padding: '15px 20px',
    backgroundColor: '#fff',
    borderBottom: '1px solid #e0e0e0',
  },
  button: {
    padding: '8px 16px',
    border: 'none',
    borderRadius: '4px',
    cursor: 'pointer',
    fontSize: '14px',
    fontWeight: 500,
  },
  dangerButton: {
    backgroundColor: '#e74c3c',
    color: '#fff',
  },
  primaryButton: {
    backgroundColor: '#1a1a2e',
    color: '#fff',
  },
  content: {
    flex: 1,
    display: 'flex',
    overflow: 'hidden',
  },
  diagramContainer: {
    flex: 1,
    backgroundColor: '#fff',
    overflow: 'auto',
  },
  rightPanel: {
    width: '350px',
    backgroundColor: '#fff',
    borderLeft: '1px solid #e0e0e0',
    display: 'flex',
    flexDirection: 'column' as const,
  },
  panelHeader: {
    padding: '15px 20px',
    borderBottom: '1px solid #e0e0e0',
    fontSize: '14px',
    fontWeight: 600,
    color: '#1a1a2e',
  },
  panelContent: {
    flex: 1,
    overflowY: 'auto' as const,
    padding: '15px',
  },
  breadcrumb: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    fontSize: '14px',
    color: '#666',
  },
  breadcrumbItem: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
  },
  breadcrumbLink: {
    color: '#1a1a2e',
    cursor: 'pointer',
    textDecoration: 'underline',
  },
  breadcrumbSeparator: {
    color: '#999',
  },
  breadcrumbCurrent: {
    color: '#1a1a2e',
    fontWeight: 500,
  },
  backButton: {
    padding: '6px 12px',
    border: '1px solid #e0e0e0',
    borderRadius: '4px',
    backgroundColor: '#fff',
    cursor: 'pointer',
    fontSize: '12px',
    display: 'flex',
    alignItems: 'center',
    gap: '4px',
  },
};

const mockInstances: ProcessInstance[] = [
  {
    id: 'inst-001',
    processDefinitionId: 'process-001',
    processDefinitionKey: 'approval-process',
    status: 'active',
    startTime: new Date(Date.now() - 3600000).toISOString(),
    endTime: null,
    variables: { requestId: 'REQ-001', amount: 5000 },
  },
  {
    id: 'inst-002',
    processDefinitionId: 'process-002',
    processDefinitionKey: 'payment-process',
    status: 'completed',
    startTime: new Date(Date.now() - 7200000).toISOString(),
    endTime: new Date(Date.now() - 1800000).toISOString(),
    variables: { orderId: 'ORD-001', amount: 1000 },
  },
  {
    id: 'inst-003',
    processDefinitionId: 'process-001',
    processDefinitionKey: 'approval-process',
    status: 'suspended',
    startTime: new Date(Date.now() - 86400000).toISOString(),
    endTime: null,
    variables: { requestId: 'REQ-002', amount: 10000 },
  },
];

export default function Monitor() {
  const [activeTab, setActiveTab] = useState<'instances' | 'metrics'>('instances');
  
  const {
    instances,
    selectedInstance,
    tokens,
    events,
    loading,
    wsConnected,
    processStack,
    currentProcessDefinitionId,
    fetchInstances,
    selectInstance,
    terminateInstance,
    suspendInstance,
    resumeInstance,
    connectWebSocket,
    disconnectWebSocket,
    drillDown,
    drillUp,
    resetNavigation,
  } = useMonitorStore();

  useEffect(() => {
    fetchInstances();
    connectWebSocket();
    
    return () => {
      disconnectWebSocket();
    };
  }, []);

  const allInstances = Array.isArray(instances) && instances.length > 0 ? instances : mockInstances;
  const displayInstances: ProcessInstance[] = Array.isArray(allInstances) ? allInstances : [];

  const getStatusBadgeStyle = (status: string) => {
    switch (status) {
      case 'active':
        return { ...styles.statusBadge, ...styles.activeBadge };
      case 'completed':
        return { ...styles.statusBadge, ...styles.completedBadge };
      case 'terminated':
      case 'suspended':
        return { ...styles.statusBadge, ...styles.terminatedBadge };
      default:
        return styles.statusBadge;
    }
  };

  return (
    <div style={styles.container}>
      <div style={styles.sidebar}>
        <div style={styles.header}>
          <div style={styles.title}>Process Monitor</div>
          <div style={styles.statusBar}>
            <div style={{ 
              ...styles.statusDot, 
              ...(wsConnected ? styles.connected : styles.disconnected) 
            }} />
            <span>{wsConnected ? 'Connected' : 'Disconnected'}</span>
          </div>
          <div style={styles.filterButtons}>
            <button 
              style={{ ...styles.filterButton, ...(activeTab === 'instances' ? styles.activeTab : styles.inactiveTab) }}
              onClick={() => setActiveTab('instances')}
            >
              Instances
            </button>
            <button 
              style={{ ...styles.filterButton, ...(activeTab === 'metrics' ? styles.activeTab : styles.inactiveTab) }}
              onClick={() => setActiveTab('metrics')}
            >
              Metrics
            </button>
          </div>
        </div>

        {activeTab === 'instances' ? (
          <div style={styles.instanceList}>
            {loading && <div style={{ padding: '20px', textAlign: 'center' }}>Loading...</div>}
            
            {displayInstances.map((instance) => (
              <div
                key={instance.id}
                style={{
                  ...styles.instanceCard,
                  ...(selectedInstance?.id === instance.id ? styles.selectedInstance : {}),
                }}
                onClick={() => selectInstance(instance)}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <div style={styles.instanceName}>{instance.processDefinitionKey}</div>
                  <span style={getStatusBadgeStyle(instance.status)}>{instance.status}</span>
                </div>
                <div style={styles.instanceMeta}>
                  ID: {instance.id}
                </div>
                <div style={styles.instanceMeta}>
                  Started: {new Date(instance.startTime).toLocaleString()}
                </div>
              </div>
            ))}
            
            {!loading && displayInstances.length === 0 && (
              <div style={{ padding: '20px', textAlign: 'center', color: '#999' }}>
                No instances found
              </div>
            )}
          </div>
        ) : (
          <div style={styles.instanceList}>
            <MetricsDashboard compact />
          </div>
        )}
      </div>

      <div style={styles.main}>
        {activeTab === 'metrics' ? (
          <div style={{ padding: '20px', overflow: 'auto' }}>
            <MetricsDashboard />
          </div>
        ) : selectedInstance ? (
          <>
            <div style={styles.toolbar}>
              <div style={{ fontSize: '16px', fontWeight: 500 }}>
                Instance: {selectedInstance.id}
              </div>
              <div style={{ flex: 1 }} />
              
              {selectedInstance.status === 'active' && (
                <>
                  <button
                    style={{ ...styles.button, ...styles.dangerButton }}
                    onClick={() => terminateInstance(selectedInstance.id)}
                  >
                    Terminate
                  </button>
                  <button
                    style={{ ...styles.button, backgroundColor: '#ffc107', color: '#000' }}
                    onClick={() => suspendInstance(selectedInstance.id)}
                  >
                    Suspend
                  </button>
                </>
              )}
              
              {selectedInstance.status === 'suspended' && (
                <button
                  style={{ ...styles.button, ...styles.primaryButton }}
                  onClick={() => resumeInstance(selectedInstance.id)}
                >
                  Resume
                </button>
              )}
            </div>

            <div style={styles.content}>
              <div style={styles.diagramContainer}>
                {/* Breadcrumb navigation */}
                {(processStack.length > 0 || currentProcessDefinitionId) && (
                  <div style={{ 
                    padding: '12px 20px', 
                    borderBottom: '1px solid #e0e0e0',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '12px',
                  }}>
                    {(processStack.length > 0 || currentProcessDefinitionId) && (
                      <button
                        style={styles.backButton}
                        onClick={() => drillUp()}
                      >
                        ← Back
                      </button>
                    )}
                    <div style={styles.breadcrumb}>
                      <span 
                        style={styles.breadcrumbLink}
                        onClick={() => resetNavigation()}
                      >
                        {selectedInstance.processDefinitionKey}
                      </span>
                      {processStack.map((item, index) => (
                        <span key={index} style={styles.breadcrumbItem}>
                          <span style={styles.breadcrumbSeparator}>›</span>
                          <span style={styles.breadcrumbCurrent}>
                            {item.elementName || 'Sub Process'}
                          </span>
                        </span>
                      ))}
                    </div>
                  </div>
                )}
                <DiagramViewer 
                  processDefinitionId={currentProcessDefinitionId || selectedInstance.processDefinitionId} 
                  tokens={tokens}
                  onSubProcessClick={(subProcessId, elementId) => {
                    drillDown(subProcessId, elementId);
                  }}
                />
              </div>
              
              <div style={styles.rightPanel}>
                <div style={styles.panelHeader}>
                  Active Tokens ({tokens.length})
                </div>
                <div style={styles.panelContent}>
                  <TokenTracker tokens={tokens.length > 0 ? tokens : [
                    { id: '1', processInstanceId: selectedInstance.id, elementId: 'Task_1', elementName: 'Review', status: 'active', arrivedAt: new Date().toISOString() },
                    { id: '2', processInstanceId: selectedInstance.id, elementId: 'Task_2', elementName: 'Approve', status: 'waiting', arrivedAt: new Date().toISOString() },
                  ]} />
                </div>

                <div style={styles.panelHeader}>
                  Variables
                </div>
                <div style={styles.panelContent}>
                  <VariablesPanel variables={selectedInstance.variables} />
                </div>

                <div style={styles.panelHeader}>
                  Events ({events.length})
                </div>
                <div style={styles.panelContent}>
                  {(events.length > 0 ? events : [
                    { id: '1', processInstanceId: selectedInstance.id, eventType: 'started', elementId: 'StartEvent', timestamp: selectedInstance.startTime, variables: {} },
                    { id: '2', processInstanceId: selectedInstance.id, eventType: 'task.completed', elementId: 'Task_1', timestamp: new Date().toISOString(), variables: {} },
                  ]).map((event) => (
                    <div key={event.id} style={{ padding: '10px', marginBottom: '8px', backgroundColor: '#f5f5f5', borderRadius: '4px', fontSize: '12px' }}>
                      <div style={{ fontWeight: 500 }}>{event.eventType}</div>
                      <div style={{ color: '#666' }}>{event.elementId}</div>
                      <div style={{ color: '#999' }}>{new Date(event.timestamp).toLocaleString()}</div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </>
        ) : activeTab === 'metrics' ? (
          <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%', color: '#999' }}>
            Select 'Metrics' tab to view dashboard
          </div>
        ) : (
          <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%', color: '#999' }}>
            Select an instance to view details
          </div>
        )}
      </div>
    </div>
  );
}
