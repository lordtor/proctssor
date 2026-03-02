package bpmn

import (
	"encoding/xml"
	"fmt"
	"io"
)

// ProcessParser handles BPMN XML parsing
type ProcessParser struct {
	definitions *Definitions
}

// NewParser creates a new BPMN parser
func NewParser() *ProcessParser {
	return &ProcessParser{}
}

// Parse parses BPMN XML and returns a Process
func Parse(xmlData []byte) (*Process, error) {
	parser := NewParser()
	return parser.ParseBytes(xmlData)
}

// ParseBytes parses BPMN XML bytes and returns a Process
func (p *ProcessParser) ParseBytes(xmlData []byte) (*Process, error) {
	var raw struct {
		Process []struct {
			ID           string `xml:"id,attr"`
			Name         string `xml:"name,attr,omitempty"`
			IsExecutable string `xml:"isExecutable,attr,omitempty"`
			FlowElement  []struct {
				StartEvent *struct {
					ID       string   `xml:"id,attr"`
					Name     string   `xml:"name,attr,omitempty"`
					Outgoing []string `xml:"outgoing"`
					Incoming []string `xml:"incoming"`
				} `xml:"startEvent"`
				EndEvent *struct {
					ID       string   `xml:"id,attr"`
					Name     string   `xml:"name,attr,omitempty"`
					Incoming []string `xml:"incoming"`
					Outgoing []string `xml:"outgoing"`
				} `xml:"endEvent"`
				UserTask *struct {
					ID             string   `xml:"id,attr"`
					Name           string   `xml:"name,attr,omitempty"`
					Implementation string   `xml:"implementation,attr,omitempty"`
					Outgoing       []string `xml:"outgoing"`
					Incoming       []string `xml:"incoming"`
				} `xml:"userTask"`
				ServiceTask *struct {
					ID             string   `xml:"id,attr"`
					Name           string   `xml:"name,attr,omitempty"`
					Implementation string   `xml:"implementation,attr,omitempty"`
					Class          string   `xml:"class,attr,omitempty"`
					Expression     string   `xml:"expression,attr,omitempty"`
					Topic          string   `xml:"topic,attr,omitempty"`
					Type           string   `xml:"type,attr,omitempty"`
					Outgoing       []string `xml:"outgoing"`
					Incoming       []string `xml:"incoming"`
				} `xml:"serviceTask"`
				ScriptTask *struct {
					ID           string   `xml:"id,attr"`
					Name         string   `xml:"name,attr,omitempty"`
					ScriptFormat string   `xml:"scriptFormat,attr,omitempty"`
					Script       string   `xml:"script"`
					Outgoing     []string `xml:"outgoing"`
					Incoming     []string `xml:"incoming"`
				} `xml:"scriptTask"`
				ManualTask *struct {
					ID       string   `xml:"id,attr"`
					Name     string   `xml:"name,attr,omitempty"`
					Outgoing []string `xml:"outgoing"`
					Incoming []string `xml:"incoming"`
				} `xml:"manualTask"`
				ExclusiveGateway *struct {
					ID       string   `xml:"id,attr"`
					Name     string   `xml:"name,attr,omitempty"`
					Default  string   `xml:"default,attr,omitempty"`
					Outgoing []string `xml:"outgoing"`
					Incoming []string `xml:"incoming"`
				} `xml:"exclusiveGateway"`
				InclusiveGateway *struct {
					ID       string   `xml:"id,attr"`
					Name     string   `xml:"name,attr,omitempty"`
					Default  string   `xml:"default,attr,omitempty"`
					Outgoing []string `xml:"outgoing"`
					Incoming []string `xml:"incoming"`
				} `xml:"inclusiveGateway"`
				ParallelGateway *struct {
					ID       string   `xml:"id,attr"`
					Name     string   `xml:"name,attr,omitempty"`
					Outgoing []string `xml:"outgoing"`
					Incoming []string `xml:"incoming"`
				} `xml:"parallelGateway"`
				IntermediateCatchEvent *struct {
					ID       string   `xml:"id,attr"`
					Name     string   `xml:"name,attr,omitempty"`
					Outgoing []string `xml:"outgoing"`
					Incoming []string `xml:"incoming"`
				} `xml:"intermediateCatchEvent"`
				IntermediateThrowEvent *struct {
					ID       string   `xml:"id,attr"`
					Name     string   `xml:"name,attr,omitempty"`
					Outgoing []string `xml:"outgoing"`
					Incoming []string `xml:"incoming"`
				} `xml:"intermediateThrowEvent"`
			} `xml:"flowElement"`
			SequenceFlow []struct {
				ID                  string `xml:"id,attr"`
				Name                string `xml:"name,attr,omitempty"`
				SourceRef           string `xml:"sourceRef,attr"`
				TargetRef           string `xml:"targetRef,attr"`
				ConditionExpression string `xml:"conditionExpression"`
			} `xml:"sequenceFlow"`
		} `xml:"process"`
	}

	if err := xml.Unmarshal(xmlData, &raw); err != nil {
		return nil, fmt.Errorf("failed to unmarshal BPMN XML: %w", err)
	}

	if len(raw.Process) == 0 {
		return nil, fmt.Errorf("no process found in BPMN definitions")
	}

	rp := raw.Process[0]
	process := &Process{
		ID:           rp.ID,
		Name:         rp.Name,
		IsExecutable: rp.IsExecutable,
	}

	// Convert flow elements
	for _, rawElem := range rp.FlowElement {
		if rawElem.StartEvent != nil {
			process.FlowElement = append(process.FlowElement, &StartEvent{
				BaseElement: BaseElement{
					ID:   rawElem.StartEvent.ID,
					Name: rawElem.StartEvent.Name,
				},
				Outgoing: rawElem.StartEvent.Outgoing,
			})
		} else if rawElem.EndEvent != nil {
			process.FlowElement = append(process.FlowElement, &EndEvent{
				BaseElement: BaseElement{
					ID:   rawElem.EndEvent.ID,
					Name: rawElem.EndEvent.Name,
				},
				Incoming: rawElem.EndEvent.Incoming,
			})
		} else if rawElem.UserTask != nil {
			process.FlowElement = append(process.FlowElement, &UserTask{
				Task: Task{
					BaseElement: BaseElement{
						ID:   rawElem.UserTask.ID,
						Name: rawElem.UserTask.Name,
					},
					Outgoing: rawElem.UserTask.Outgoing,
					Incoming: rawElem.UserTask.Incoming,
				},
				Implementation: rawElem.UserTask.Implementation,
			})
		} else if rawElem.ServiceTask != nil {
			process.FlowElement = append(process.FlowElement, &ServiceTask{
				Task: Task{
					BaseElement: BaseElement{
						ID:   rawElem.ServiceTask.ID,
						Name: rawElem.ServiceTask.Name,
					},
					Outgoing: rawElem.ServiceTask.Outgoing,
					Incoming: rawElem.ServiceTask.Incoming,
				},
				Implementation: rawElem.ServiceTask.Implementation,
				Class:          rawElem.ServiceTask.Class,
				Expression:     rawElem.ServiceTask.Expression,
				Topic:          rawElem.ServiceTask.Topic,
				Type:           rawElem.ServiceTask.Type,
			})
		} else if rawElem.ScriptTask != nil {
			process.FlowElement = append(process.FlowElement, &ScriptTask{
				Task: Task{
					BaseElement: BaseElement{
						ID:   rawElem.ScriptTask.ID,
						Name: rawElem.ScriptTask.Name,
					},
					Outgoing: rawElem.ScriptTask.Outgoing,
					Incoming: rawElem.ScriptTask.Incoming,
				},
				ScriptFormat: rawElem.ScriptTask.ScriptFormat,
				Script:       rawElem.ScriptTask.Script,
			})
		} else if rawElem.ManualTask != nil {
			process.FlowElement = append(process.FlowElement, &ManualTask{
				Task: Task{
					BaseElement: BaseElement{
						ID:   rawElem.ManualTask.ID,
						Name: rawElem.ManualTask.Name,
					},
					Outgoing: rawElem.ManualTask.Outgoing,
					Incoming: rawElem.ManualTask.Incoming,
				},
			})
		} else if rawElem.ExclusiveGateway != nil {
			process.FlowElement = append(process.FlowElement, &ExclusiveGateway{
				BaseElement: BaseElement{
					ID:   rawElem.ExclusiveGateway.ID,
					Name: rawElem.ExclusiveGateway.Name,
				},
				Default:  rawElem.ExclusiveGateway.Default,
				Outgoing: rawElem.ExclusiveGateway.Outgoing,
				Incoming: rawElem.ExclusiveGateway.Incoming,
			})
		} else if rawElem.InclusiveGateway != nil {
			process.FlowElement = append(process.FlowElement, &InclusiveGateway{
				BaseElement: BaseElement{
					ID:   rawElem.InclusiveGateway.ID,
					Name: rawElem.InclusiveGateway.Name,
				},
				Default:  rawElem.InclusiveGateway.Default,
				Outgoing: rawElem.InclusiveGateway.Outgoing,
				Incoming: rawElem.InclusiveGateway.Incoming,
			})
		} else if rawElem.ParallelGateway != nil {
			process.FlowElement = append(process.FlowElement, &ParallelGateway{
				BaseElement: BaseElement{
					ID:   rawElem.ParallelGateway.ID,
					Name: rawElem.ParallelGateway.Name,
				},
				Outgoing: rawElem.ParallelGateway.Outgoing,
				Incoming: rawElem.ParallelGateway.Incoming,
			})
		} else if rawElem.IntermediateCatchEvent != nil {
			process.FlowElement = append(process.FlowElement, &IntermediateCatchEvent{
				BaseElement: BaseElement{
					ID:   rawElem.IntermediateCatchEvent.ID,
					Name: rawElem.IntermediateCatchEvent.Name,
				},
				Outgoing: rawElem.IntermediateCatchEvent.Outgoing,
				Incoming: rawElem.IntermediateCatchEvent.Incoming,
			})
		} else if rawElem.IntermediateThrowEvent != nil {
			process.FlowElement = append(process.FlowElement, &IntermediateThrowEvent{
				BaseElement: BaseElement{
					ID:   rawElem.IntermediateThrowEvent.ID,
					Name: rawElem.IntermediateThrowEvent.Name,
				},
				Outgoing: rawElem.IntermediateThrowEvent.Outgoing,
				Incoming: rawElem.IntermediateThrowEvent.Incoming,
			})
		}
	}

	// Convert sequence flows
	for _, rawFlow := range rp.SequenceFlow {
		flow := SequenceFlow{
			BaseElement: BaseElement{
				ID:   rawFlow.ID,
				Name: rawFlow.Name,
			},
			SourceRef: rawFlow.SourceRef,
			TargetRef: rawFlow.TargetRef,
		}
		if rawFlow.ConditionExpression != "" {
			flow.ConditionExpression = &ConditionExpression{
				Text: rawFlow.ConditionExpression,
			}
		}
		process.SequenceFlow = append(process.SequenceFlow, flow)
	}

	return process, nil
}

// ParseFromReader parses BPMN XML from an io.Reader
func ParseFromReader(r io.Reader) (*Process, error) {
	parser := NewParser()
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read XML data: %w", err)
	}
	return parser.ParseBytes(data)
}

// ParseServiceTask parses a ServiceTask from XML
func ParseServiceTask(task *ServiceTask) error {
	if task.Implementation == "" && task.Class == "" && task.Expression == "" && task.Topic == "" && task.Type == "" {
		return fmt.Errorf("service task must have at least one of: implementation, class, expression, topic, type")
	}
	return nil
}

// GetProcessByID finds a process by its ID
func GetProcessByID(definitions *Definitions, id string) (*Process, error) {
	for i := range definitions.Process {
		if definitions.Process[i].ID == id {
			return &definitions.Process[i], nil
		}
	}
	return nil, fmt.Errorf("process with id %s not found", id)
}

// GetFlowElementByID finds a flow element by its ID
func GetFlowElementByID(process *Process, id string) (FlowElement, error) {
	for _, elem := range process.FlowElement {
		if elem.GetID() == id {
			return elem, nil
		}
	}
	return nil, fmt.Errorf("flow element with id %s not found", id)
}

// GetSequenceFlowByID finds a sequence flow by its ID
func GetSequenceFlowByID(process *Process, id string) (*SequenceFlow, error) {
	for i := range process.SequenceFlow {
		if process.SequenceFlow[i].ID == id {
			return &process.SequenceFlow[i], nil
		}
	}
	return nil, fmt.Errorf("sequence flow with id %s not found", id)
}

// GetOutgoingFlows gets all outgoing sequence flows from an element
func GetOutgoingFlows(process *Process, elementID string) []SequenceFlow {
	var flows []SequenceFlow
	for _, flow := range process.SequenceFlow {
		if flow.SourceRef == elementID {
			flows = append(flows, flow)
		}
	}
	return flows
}

// GetIncomingFlows gets all incoming sequence flows to an element
func GetIncomingFlows(process *Process, elementID string) []SequenceFlow {
	var flows []SequenceFlow
	for _, flow := range process.SequenceFlow {
		if flow.TargetRef == elementID {
			flows = append(flows, flow)
		}
	}
	return flows
}
