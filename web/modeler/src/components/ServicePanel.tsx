import React, { useState, useEffect } from 'react';

interface ServiceAction {
  name: string;
  description?: string;
  schema?: {
    input?: any;
    output?: any;
  };
}

interface ServiceInfo {
  name: string;
  version: string;
}

interface MappingParameter {
  name: string;
  type: string;
  required: boolean;
  description?: string;
  defaultValue?: any;
  target?: string;
  source?: string;
}

interface ServiceMapping {
  serviceName: string;
  actionName: string;
  inputParameters: MappingParameter[];
  outputParameters: MappingParameter[];
}

interface ServicePanelProps {
  services?: ServiceInfo[];
  onActionSelect?: (action: ServiceAction & { serviceName: string; mapping?: ServiceMapping }) => void;
}

// Mock API for services (since we can't import from shared easily in this context)
const fetchServices = async (): Promise<ServiceInfo[]> => {
  try {
    const response = await fetch('/api/v1/registry/services');
    if (!response.ok) throw new Error('Failed to fetch services');
    return response.json();
  } catch {
    // Return mock data for demo
    return [
      { name: 'EmailService', version: '1.0.0' },
      { name: 'NotificationService', version: '1.0.0' },
      { name: 'ApprovalService', version: '1.0.0' },
    ];
  }
};

const fetchServiceActions = async (serviceName: string): Promise<{ actions: ServiceAction[] }> => {
  try {
    const response = await fetch(`/api/v1/registry/services/${serviceName}/actions`);
    if (!response.ok) throw new Error('Failed to fetch actions');
    return response.json();
  } catch {
    // Return mock actions with schemas
    const actionsMap: Record<string, ServiceAction[]> = {
      EmailService: [
        { 
          name: 'sendEmail', 
          description: 'Send an email',
          schema: {
            input: {
              type: 'object',
              properties: {
                to: { type: 'string', format: 'email', title: 'Recipient Email' },
                subject: { type: 'string', title: 'Subject' },
                body: { type: 'string', title: 'Body' },
                attachments: { type: 'array', title: 'Attachments' },
              },
              required: ['to', 'subject']
            },
            output: {
              type: 'object',
              properties: {
                messageId: { type: 'string', title: 'Message ID' },
                sentAt: { type: 'string', format: 'date-time', title: 'Sent At' },
                status: { type: 'string', title: 'Status' },
              }
            }
          }
        },
        { name: 'sendTemplate', description: 'Send email from template' },
      ],
      NotificationService: [
        { name: 'notify', description: 'Send notification' },
        { name: 'notifyUser', description: 'Notify specific user' },
      ],
      ApprovalService: [
        { name: 'requestApproval', description: 'Request approval' },
        { name: 'checkStatus', description: 'Check approval status' },
      ],
    };
    return { actions: actionsMap[serviceName] || [] };
  }
};

// Generate mapping from schema
const generateMappingFromSchema = (serviceName: string, actionName: string, schema?: ServiceAction['schema']): ServiceMapping => {
  const inputParameters: MappingParameter[] = [];
  const outputParameters: MappingParameter[] = [];

  if (schema?.input?.properties) {
    for (const [key, prop] of Object.entries(schema.input.properties) as [string, any][]) {
      inputParameters.push({
        name: key,
        type: prop.type || 'string',
        required: schema.input.required?.includes(key) || false,
        description: prop.description,
        defaultValue: prop.default,
        target: key, // default target is the same as name
      });
    }
  }

  if (schema?.output?.properties) {
    for (const [key, prop] of Object.entries(schema.output.properties) as [string, any][]) {
      outputParameters.push({
        name: key,
        type: prop.type || 'string',
        required: false,
        description: prop.description,
        source: key, // default source is the same as name
      });
    }
  }

  return {
    serviceName,
    actionName,
    inputParameters,
    outputParameters,
  };
};

const styles = {
  panel: {
    width: '280px',
    backgroundColor: '#fff',
    borderLeft: '1px solid #e0e0e0',
    display: 'flex',
    flexDirection: 'column' as const,
    height: '100%',
  },
  header: {
    padding: '15px 20px',
    borderBottom: '1px solid #e0e0e0',
    fontSize: '14px',
    fontWeight: 600,
    color: '#1a1a2e',
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
  },
  searchInput: {
    padding: '10px 15px',
    borderBottom: '1px solid #e0e0e0',
  },
  input: {
    width: '100%',
    padding: '8px 12px',
    border: '1px solid #ddd',
    borderRadius: '4px',
    fontSize: '13px',
    outline: 'none',
  },
  list: {
    flex: 1,
    overflowY: 'auto' as const,
    padding: '10px',
  },
  serviceItem: {
    marginBottom: '8px',
  },
  serviceHeader: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    padding: '10px 12px',
    backgroundColor: '#f8f9fa',
    borderRadius: '6px',
    cursor: 'pointer',
    border: 'none',
    width: '100%',
    textAlign: 'left' as const,
    transition: 'all 0.2s',
  },
  serviceName: {
    fontSize: '13px',
    fontWeight: 500,
    color: '#1a1a2e',
  },
  serviceVersion: {
    fontSize: '11px',
    color: '#888',
  },
  actionList: {
    paddingLeft: '15px',
    marginTop: '8px',
  },
  actionItem: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    padding: '8px 12px',
    borderRadius: '4px',
    cursor: 'grab',
    fontSize: '12px',
    color: '#444',
    transition: 'all 0.2s',
    marginBottom: '4px',
  },
  emptyState: {
    padding: '20px',
    textAlign: 'center' as const,
    color: '#888',
    fontSize: '13px',
  },
};

export default function ServicePanel({ services: propServices, onActionSelect }: ServicePanelProps) {
  const [services, setServices] = useState<ServiceInfo[]>(propServices || []);
  const [expanded, setExpanded] = useState<Record<string, boolean>>({});
  const [actions, setActions] = useState<Record<string, ServiceAction[]>>({});
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (propServices && propServices.length > 0) {
      setServices(propServices);
    } else {
      setLoading(true);
      fetchServices()
        .then(setServices)
        .finally(() => setLoading(false));
    }
  }, [propServices]);

  const toggleExpand = async (serviceName: string) => {
    setExpanded((prev) => ({ ...prev, [serviceName]: !prev[serviceName] }));
    
    if (!expanded[serviceName] && !actions[serviceName]) {
      try {
        const data = await fetchServiceActions(serviceName);
        setActions((prev) => ({ ...prev, [serviceName]: data.actions }));
      } catch (err) {
        console.error('Failed to load actions:', err);
      }
    }
  };

  const handleDragStart = (e: React.DragEvent, action: ServiceAction, serviceName: string) => {
    const mapping = generateMappingFromSchema(serviceName, action.name, action.schema);
    const data = {
      ...action,
      serviceName,
      mapping,
    };
    e.dataTransfer.setData('application/json', JSON.stringify(data));
    e.dataTransfer.effectAllowed = 'copy';
  };

  const handleActionClick = (action: ServiceAction, serviceName: string) => {
    const mapping = generateMappingFromSchema(serviceName, action.name, action.schema);
    onActionSelect?.({ ...action, serviceName, mapping });
  };

  if (loading) {
    return (
      <div style={styles.panel}>
        <div style={styles.header}>🔌 Services</div>
        <div style={styles.emptyState}>Loading services...</div>
      </div>
    );
  }

  return (
    <div style={styles.panel}>
      <div style={styles.header}>
        🔌 <span>Services</span>
        <span style={{ marginLeft: 'auto', fontSize: '12px', color: '#888' }}>
          {services.length}
        </span>
      </div>
      
      <div style={styles.searchInput}>
        <input
          type="text"
          placeholder="Search services..."
          style={styles.input}
        />
      </div>

      <div style={styles.list}>
        {services.length === 0 ? (
          <div style={styles.emptyState}>
            No services registered.<br />
            Register services in the registry.
          </div>
        ) : (
          services.map((service) => (
            <div key={service.name} style={styles.serviceItem}>
              <button
                style={{
                  ...styles.serviceHeader,
                  backgroundColor: expanded[service.name] ? '#e8f5e9' : '#f8f9fa',
                }}
                onClick={() => toggleExpand(service.name)}
              >
                <div>
                  <div style={styles.serviceName}>{service.name}</div>
                  <div style={styles.serviceVersion}>v{service.version}</div>
                </div>
                <span>{expanded[service.name] ? '▼' : '▶'}</span>
              </button>

              {expanded[service.name] && actions[service.name] && (
                <div style={styles.actionList}>
                  {actions[service.name].map((action) => (
                    <div
                      key={action.name}
                      style={styles.actionItem}
                      draggable
                      onDragStart={(e) => handleDragStart(e, action, service.name)}
                      onClick={() => handleActionClick(action, service.name)}
                      title={action.description || `Drag to add ${action.name}`}
                    >
                      🎯 <span>{action.name}</span>
                      {action.schema && <span style={{ marginLeft: 'auto' }}>📋</span>}
                    </div>
                  ))}
                </div>
              )}
            </div>
          ))
        )}
      </div>
    </div>
  );
}
