import { useEffect, useRef, useState, useCallback } from 'react';
import BpmnViewer from 'bpmn-js/lib/Viewer';
import 'bpmn-js/dist/assets/diagram-js.css';
import 'bpmn-js/dist/assets/bpmn-js.css';
import 'bpmn-js/dist/assets/bpmn-font/css/bpmn-embedded.css';
import axios from 'axios';
import { Token } from '../store';

interface DiagramViewerProps {
  processDefinitionId: string;
  tokens: Token[];
  onSubProcessClick?: (processDefinitionId: string, elementId: string) => void;
}

const styles = {
  container: {
    height: '100%',
    position: 'relative' as const,
  },
  canvas: {
    height: '100%',
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
    <bpmn:task id="Task_1" name="Review">
      <bpmn:incoming>Flow_1</bpmn:incoming>
      <bpmn:outgoing>Flow_2</bpmn:outgoing>
    </bpmn:task>
    <bpmn:task id="Task_2" name="Approve">
      <bpmn:incoming>Flow_2</bpmn:incoming>
      <bpmn:outgoing>Flow_3</bpmn:outgoing>
    </bpmn:task>
    <bpmn:endEvent id="EndEvent_1">
      <bpmn:incoming>Flow_3</bpmn:incoming>
    </bpmn:endEvent>
    <bpmn:sequenceFlow id="Flow_1" sourceRef="StartEvent_1" targetRef="Task_1" />
    <bpmn:sequenceFlow id="Flow_2" sourceRef="Task_1" targetRef="Task_2" />
    <bpmn:sequenceFlow id="Flow_3" sourceRef="Task_2" targetRef="EndEvent_1" />
  </bpmn:process>
  <bpmndi:BPMNDiagram id="BPMNDiagram_1">
    <bpmndi:BPMNPlane id="BPMNPlane_1" bpmnElement="Process_1">
      <bpmndi:BPMNShape id="StartEvent_1_di" bpmnElement="StartEvent_1">
        <dc:Bounds x="180" y="160" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Task_1_di" bpmnElement="Task_1">
        <dc:Bounds x="270" y="138" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="Task_2_di" bpmnElement="Task_2">
        <dc:Bounds x="420" y="138" width="100" height="80" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNShape id="EndEvent_1_di" bpmnElement="EndEvent_1">
        <dc:Bounds x="572" y="160" width="36" height="36" />
      </bpmndi:BPMNShape>
      <bpmndi:BPMNEdge id="Flow_1_di" bpmnElement="Flow_1">
        <di:waypoint x="216" y="178" />
        <di:waypoint x="270" y="178" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_2_di" bpmnElement="Flow_2">
        <di:waypoint x="370" y="178" />
        <di:waypoint x="420" y="178" />
      </bpmndi:BPMNEdge>
      <bpmndi:BPMNEdge id="Flow_3_di" bpmnElement="Flow_3">
        <di:waypoint x="520" y="178" />
        <di:waypoint x="572" y="178" />
      </bpmndi:BPMNEdge>
    </bpmndi:BPMNPlane>
  </bpmndi:BPMNDiagram>
</bpmn:definitions>`;

export default function DiagramViewer({ processDefinitionId, tokens, onSubProcessClick }: DiagramViewerProps) {
  const canvasRef = useRef<HTMLDivElement>(null);
  const viewerRef = useRef<any>(null);

  const [bpmnXml, setBpmnXml] = useState<string | null>(null);

  // Load BPMN XML from backend
  useEffect(() => {
    if (!processDefinitionId) return;

    axios.get(`/api/v1/processes/${processDefinitionId}/xml`)
      .then(res => {
        setBpmnXml(res.data.bpmn_xml);
      })
      .catch(err => {
        console.warn('Failed to load diagram, using default:', err);
        setBpmnXml(defaultDiagram);
      });
  }, [processDefinitionId]);

  useEffect(() => {
    if (!canvasRef.current || !bpmnXml) return;

    const viewer = new BpmnViewer({
      container: canvasRef.current,
    });

    viewer.importXML(bpmnXml).catch(console.error);
    viewerRef.current = viewer;

    // Add click handler for sub-processes
    if (onSubProcessClick) {
      const eventBus = viewer.get('eventBus');
      
      eventBus.on('element.click', (element: any) => {
        if (!element) return;
        
        // Check if element is a sub-process or call activity
        const type = element.type;
        const businessObject = element.businessObject;
        
        // BPMN element types for sub-processes
        const subProcessTypes = [
          'bpmn:SubProcess',
          'bpmn:CallActivity',
          'bpmn:AdHocSubProcess',
          'bpmn:Transaction',
        ];
        
        // Check direct type or via businessObject
        const isSubProcess = subProcessTypes.includes(type) || 
          subProcessTypes.includes(businessObject?.$type);
        
        // Get calledElement for CallActivity
        const calledElement = businessObject?.calledElement;
        
        if (isSubProcess && calledElement) {
          onSubProcessClick(calledElement, element.id);
        } else if (isSubProcess && !calledElement) {
          // Embedded sub-process - try to get the process reference
          // For now, we'll log it - in real implementation would parse the di
          console.log('Embedded sub-process clicked:', element.id);
        }
      });
    }

    return () => {
      viewer.destroy();
    };
  }, [bpmnXml, processDefinitionId]);

  useEffect(() => {
    if (!viewerRef.current || tokens.length === 0) return;

    const canvas = viewerRef.current.get('canvas');
    const elementRegistry = viewerRef.current.get('elementRegistry');
    
    // Clear previous markers first
    canvas.removeMarker('highlight');
    canvas.removeMarker('active');
    
    // Apply new markers
    tokens.forEach((token) => {
      const element = elementRegistry.get(token.elementId);
      if (element) {
        canvas.addMarker(element.id, token.status === 'active' ? 'active' : 'highlight');
      }
    });
  }, [tokens]);

  return (
    <div ref={canvasRef} style={styles.canvas} />
  );
}
