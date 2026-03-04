import { useEffect, useRef, useState, useCallback } from 'react';
import BpmnModeler from 'bpmn-js/lib/Modeler';
import 'bpmn-js/dist/assets/diagram-js.css';
import 'bpmn-js/dist/assets/bpmn-js.css';
import 'bpmn-js/dist/assets/bpmn-font/css/bpmn-embedded.css';
import axios from 'axios';
import ServicePanel from './components/ServicePanel';

// Types for service mapping
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

interface ServiceActionData {
  name: string;
  description?: string;
  schema?: any;
  serviceName: string;
  mapping?: ServiceMapping;
}

const styles = {
  container: {
    display: 'flex',
    flexDirection: 'column' as const,
    height: '100vh',
    backgroundColor: '#f5f5f5',
  },
  toolbar: {
    display: 'flex',
    alignItems: 'center',
    gap: '10px',
    padding: '10px 20px',
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
    transition: 'all 0.2s',
  },
  primaryButton: {
    backgroundColor: '#4ecca3',
    color: '#1a1a2e',
  },
  secondaryButton: {
    backgroundColor: '#1a1a2e',
    color: '#fff',
  },
  canvasWrapper: {
    flex: 1,
    display: 'flex',
    overflow: 'hidden',
  },
  canvas: {
    flex: 1,
    backgroundColor: '#fff',
  },
  servicePanelWrapper: {
    width: '280px',
    backgroundColor: '#fff',
    borderRight: '1px solid #e0e0e0',
    display: 'flex',
    flexDirection: 'column' as const,
  },
  panel: {
    width: '300px',
    backgroundColor: '#fff',
    borderLeft: '1px solid #e0e0e0',
    padding: '20px',
    overflowY: 'auto' as const,
  },
  panelTitle: {
    fontSize: '16px',
    fontWeight: 600,
    marginBottom: '15px',
    color: '#1a1a2e',
  },
  input: {
    width: '100%',
    padding: '8px 12px',
    border: '1px solid #ddd',
    borderRadius: '4px',
    fontSize: '14px',
    marginBottom: '10px',
  },
  label: {
    display: 'block',
    fontSize: '12px',
    color: '#666',
    marginBottom: '5px',
  },
  status: {
    padding: '8px 16px',
    borderRadius: '4px',
    fontSize: '14px',
  },
  success: {
    backgroundColor: '#d4edda',
    color: '#155724',
  },
  error: {
    backgroundColor: '#f8d7da',
    color: '#721c24',
  },
};

const defaultDiagram = `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
  xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL"
  xmlns:bpmndi="http://www.omg.org/spec/BPMN/20100524/DI"
  xmlns:dc="http://www.omg.org/spec/DD/20100524/DC"
  xmlns:di="http://www.omg.org/spec/DD/20100524/DI"
  id="Definitions_1" targetNamespace="http://bpmn.io/schema/bpmn">
  <bpmn:process id="Process_1" isExecutable="true">
    <bpmn:startEvent id="StartEvent_1">
      <bpmn:outgoing>Flow_1</bpmn:outgoing>
    </bpmn:startEvent>
    <bpmn:task id="Task_1" name="Example Task">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:task>
    <bpmn:endEvent id="EndEvent_1">
      <bpmn:incoming>Flow_2</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="Task_1" />
    <bpmn:sequenceFlow id="Flow_2" sourceRef="Task_1" targetRef="EndEvent_1" />
  </bpmn:process>
  <bpmndi:BPMNDiagram id="BPMNDiagram_1">
    <bpmndi:BPMNPlane id="BPMNPlane_1" bpmnElement="Process_1">
      <bpmndi:BPMNShape id="StartEvent_1_di" bpmnElement="StartEvent_1">
        <dc:Bounds x="180" y="160" width="36" height="36" />
        <bpmndi:BPMNLabel>
          <dc:Bounds x="187" y="203" width="22" height="14" />
        </bpmndi:BPMNLabel>
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Task_1_di" bpmnElement="Task_1">
        <dc:Bounds x="270" y="138" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="EndEvent_1_di" bpmnElement="EndEvent_1">
        <dc:Bounds x="432" y="160" width="36" height="36" />
        <bpmndi:BPMNLabel>
          <dc:Bounds x="439" y="203" width="22" height="14" />
        </bpmndi:BPMNLabel>
      </bpmndi:BPMNShape>
      <bpmndi:BPMNEdge id="Flow_1_di" bpmnElement="Flow_1">
        <di:waypoint x="216" y="178" />
        <di:waypoint x="270" y="178" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_2_di" bpmnElement="Flow_2">
        <di:waypoint x="370" y="178" />
        <di:waypoint x="432" y="178" />
      </bpmndi:BPMNEdge>
    </bpmndi:BPMNPlane>
  </bpmndi:BPMNDiagram>
</bpmn:definitions>`;

export default function Modeler() {
  const canvasRef = useRef<HTMLDivElement>(null);
  const modelerRef = useRef<any>(null);
  const [processName, setProcessName] = useState('My Process');
  const [processVersion, setProcessVersion] = useState('1.0.0');
  const [status, setStatus] = useState<{ type: 'success' | 'error' | null; message: string }>({ type: null, message: '' });
  const [selectedElement, setSelectedElement] = useState<any>(null);
  const [serviceMapping, setServiceMapping] = useState<ServiceMapping | null>(null);

  // Handle service action selection from ServicePanel
  const handleServiceActionSelect = useCallback((actionData: ServiceActionData) => {
    if (actionData.mapping) {
      setServiceMapping(actionData.mapping);
      
      // If an element is selected, apply the mapping to it
      if (selectedElement && modelerRef.current) {
        const modeling = modelerRef.current.get('modeling');
        
        // Update element with service mapping
        modeling.updateProperties(selectedElement, {
          name: actionData.serviceName + '.' + actionData.name,
          serviceName: actionData.serviceName,
          actionName: actionData.name,
          inputParameters: JSON.stringify(actionData.mapping.inputParameters),
          outputParameters: JSON.stringify(actionData.mapping.outputParameters),
        });
        
        setStatus({ type: 'success', message: `Service ${actionData.serviceName}.${actionData.name} applied` });
      }
    }
  }, [selectedElement]);

  // Handle drop on canvas
  const handleCanvasDrop = useCallback((event: React.DragEvent) => {
    event.preventDefault();
    
    try {
      const data = event.dataTransfer.getData('application/json');
      if (!data) return;
      
      const actionData: ServiceActionData = JSON.parse(data);
      
      if (actionData.mapping && modelerRef.current) {
        // Get the position from the drop event
        const container = canvasRef.current?.getBoundingClientRect();
        
        if (container) {
          // Create a new service task
          const elementFactory = modelerRef.current.get('elementFactory');
          const create = modelerRef.current.get('create');
          const shape = elementFactory.createShape({
            type: 'bpmn:ServiceTask',
            name: actionData.serviceName + '.' + actionData.name,
          });
          
          // Add service metadata to the shape
          shape.businessObject.serviceName = actionData.serviceName;
          shape.businessObject.actionName = actionData.name;
          shape.businessObject.inputParameters = JSON.stringify(actionData.mapping?.inputParameters || []);
          shape.businessObject.outputParameters = JSON.stringify(actionData.mapping?.outputParameters || []);
          
          create.start(event, shape);
          
          setServiceMapping(actionData.mapping || null);
          setStatus({ type: 'success', message: `Added ${actionData.serviceName}.${actionData.name}` });
        }
      }
    } catch (err) {
      console.error('Drop failed:', err);
    }
  }, []);

  const handleDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'copy';
  }, []);

  // Load service mapping from selected element
  useEffect(() => {
    if (selectedElement?.businessObject) {
      const bo = selectedElement.businessObject;
      if (bo.serviceName && bo.actionName) {
        try {
          const inputParams = bo.inputParameters ? JSON.parse(bo.inputParameters) : [];
          const outputParams = bo.outputParameters ? JSON.parse(bo.outputParameters) : [];
          setServiceMapping({
            serviceName: bo.serviceName,
            actionName: bo.actionName,
            inputParameters: inputParams,
            outputParameters: outputParams,
          });
        } catch {
          setServiceMapping(null);
        }
      } else {
        setServiceMapping(null);
      }
    }
  }, [selectedElement]);

  useEffect(() => {
    if (!canvasRef.current) return;

    const modeler = new BpmnModeler({
      container: canvasRef.current,
      keyboard: { bindTo: document },
    });

    modeler.importXML(defaultDiagram).then(() => {
      setStatus({ type: 'success', message: 'Diagram loaded successfully' });
    }).catch((err: Error) => {
      setStatus({ type: 'error', message: err.message });
    });

    modeler.on('selection.changed', (e: any) => {
      setSelectedElement(e.newSelection[0] || null);
    });

    modelerRef.current = modeler;

    return () => {
      modeler.destroy();
    };
  }, []);

  const handleDeploy = useCallback(async () => {
    if (!modelerRef.current) return;

    try {
      const { xml } = await modelerRef.current.saveXML({ format: true });
      
      await axios.post('/api/v1/processes/deploy', {
        process_key: processName.replace(/\s+/g, '_'),
        name: processName,
        xml: xml,
        version: parseInt(processVersion) || 1,
      });

      setStatus({ type: 'success', message: `Process deployed successfully` });
    } catch (err: any) {
      setStatus({ type: 'error', message: err.message || 'Failed to deploy process' });
    }
  }, [processName, processVersion]);

  const handleSave = useCallback(async () => {
    if (!modelerRef.current) return;

    try {
      const { xml } = await modelerRef.current.saveXML({ format: true });
      const blob = new Blob([xml], { type: 'application/xml' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${processName}.bpmn`;
      a.click();
      URL.revokeObjectURL(url);
      setStatus({ type: 'success', message: 'Diagram saved successfully' });
    } catch (err: any) {
      setStatus({ type: 'error', message: err.message || 'Failed to save diagram' });
    }
  }, [processName]);

  const handleNew = useCallback(() => {
    if (!modelerRef.current) return;
    
    modelerRef.current.importXML(defaultDiagram).then(() => {
      setStatus({ type: 'success', message: 'New diagram created' });
      setSelectedElement(null);
    });
  }, []);

  return (
    <div style={styles.container}>
      <div style={styles.toolbar}>
        <button 
          style={{ ...styles.button, ...styles.secondaryButton }} 
          onClick={handleNew}
        >
          New
        </button>
        <button 
          style={{ ...styles.button, ...styles.secondaryButton }} 
          onClick={handleSave}
        >
          Save
        </button>
        <button 
          style={{ ...styles.button, ...styles.primaryButton }} 
          onClick={handleDeploy}
        >
          Deploy
        </button>
        
        <div style={{ flex: 1 }} />
        
        <input
          type="text"
          value={processName}
          onChange={(e) => setProcessName(e.target.value)}
          placeholder="Process Name"
          style={{ ...styles.input, width: '200px', marginBottom: 0 }}
        />
        <input
          type="text"
          value={processVersion}
          onChange={(e) => setProcessVersion(e.target.value)}
          placeholder="Version"
          style={{ ...styles.input, width: '100px', marginBottom: 0 }}
        />
      </div>

      {status.type && (
        <div style={{ ...styles.status, ...(status.type === 'success' ? styles.success : styles.error) }}>
          {status.message}
        </div>
      )}

      <div style={styles.canvasWrapper}>
        <div style={styles.servicePanelWrapper}>
          <ServicePanel onActionSelect={handleServiceActionSelect} />
        </div>
        
        <div 
          ref={canvasRef} 
          style={styles.canvas}
          onDrop={handleCanvasDrop}
          onDragOver={handleDragOver}
        />
        
        <div style={styles.panel}>
          <div style={styles.panelTitle}>Properties</div>
          
          {selectedElement ? (
            <>
              <label style={styles.label}>Element Type</label>
              <input
                type="text"
                value={selectedElement.type || ''}
                readOnly
                style={styles.input}
              />
              
              <label style={styles.label}>Element ID</label>
              <input
                type="text"
                value={selectedElement.id || ''}
                readOnly
                style={styles.input}
              />
              
              <label style={styles.label}>Element Name</label>
              <input
                type="text"
                value={selectedElement.businessObject?.name || ''}
                readOnly
                style={styles.input}
              />
              
              {serviceMapping && (
                <>
                  <hr style={{ margin: '15px 0', border: 'none', borderTop: '1px solid #e0e0e0' }} />
                  
                  <div style={styles.panelTitle}>Service Mapping</div>
                  
                  <label style={styles.label}>Service</label>
                  <input
                    type="text"
                    value={serviceMapping.serviceName}
                    readOnly
                    style={styles.input}
                  />
                  
                  <label style={styles.label}>Action</label>
                  <input
                    type="text"
                    value={serviceMapping.actionName}
                    readOnly
                    style={styles.input}
                  />
                  
                  {serviceMapping.inputParameters.length > 0 && (
                    <>
                      <label style={{ ...styles.label, marginTop: '10px' }}>Input Parameters</label>
                      {serviceMapping.inputParameters.map((param, idx) => (
                        <div key={`input-${idx}`} style={{ marginBottom: '8px', padding: '8px', background: '#f5f5f5', borderRadius: '4px' }}>
                          <div style={{ fontSize: '12px', fontWeight: 500 }}>{param.name}</div>
                          <div style={{ fontSize: '11px', color: '#666' }}>Type: {param.type} {param.required && '(required)'}</div>
                          {param.defaultValue !== undefined && (
                            <div style={{ fontSize: '11px', color: '#888' }}>Default: {String(param.defaultValue)}</div>
                          )}
                        </div>
                      ))}
                    </>
                  )}
                  
                  {serviceMapping.outputParameters.length > 0 && (
                    <>
                      <label style={{ ...styles.label, marginTop: '10px' }}>Output Parameters</label>
                      {serviceMapping.outputParameters.map((param, idx) => (
                        <div key={`output-${idx}`} style={{ marginBottom: '8px', padding: '8px', background: '#f5f5f5', borderRadius: '4px' }}>
                          <div style={{ fontSize: '12px', fontWeight: 500 }}>{param.name}</div>
                          <div style={{ fontSize: '11px', color: '#666' }}>Type: {param.type}</div>
                        </div>
                      ))}
                    </>
                  )}
                </>
              )}
            </>
          ) : (
            <p style={{ color: '#666', fontSize: '14px' }}>
              Select an element to view its properties
            </p>
          )}
        </div>
      </div>
    </div>
  );
}
