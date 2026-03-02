package bpmn

import (
	"encoding/xml"
	"time"
)

// BPMN 2.0 XML Namespaces
const (
	XMLNamespaceBPMN   = "http://www.omg.org/spec/BPMN/20100524/MODEL"
	XMLNamespaceBPMNDI = "http://www.omg.org/spec/BPMN/20100524/DI"
	XMLNamespaceDC     = "http://www.omg.org/spec/DD/20100524/DC"
	XMLNamespaceDI     = "http://www.omg.org/spec/DD/20100524/DI"
	XMLNamespaceBioc   = "http://www.omg.org/spec/BPMN/non-normative/di/bpmndi"
)

// Definitions represents the root BPMN 2.0 XML element
type Definitions struct {
	XMLName         xml.Name         `xml:"definitions"`
	ID              string           `xml:"id,attr"`
	Name            string           `xml:"name,attr"`
	TargetNamespace string           `xml:"targetNamespace,attr"`
	Exporter        string           `xml:"exporter,attr,omitempty"`
	ExporterVersion string           `xml:"exporterVersion,attr,omitempty"`
	Import          []Import         `xml:"import"`
	ItemDefinition  []ItemDefinition `xml:"itemDefinition"`
	Process         []Process        `xml:"process"`
	Collaboration   *Collaboration   `xml:"participant"`
	BPMNDiagram     *BPMNDiagram     `xml:"bpmndi:BPMNDiagram"`
}

// Import represents a BPMN import
type Import struct {
	XMLNamespace string `xml:"namespace,attr"`
	Location     string `xml:"location,attr"`
	ImportType   string `xml:"importType,attr"`
}

// ItemDefinition defines the type of data being used
type ItemDefinition struct {
	ID           string `xml:"id,attr"`
	Name         string `xml:"name,attr,omitempty"`
	ItemKind     string `xml:"itemKind,attr,omitempty"` // ItemKind: Information, Physical, Resource
	StructureRef string `xml:"structureRef,attr,omitempty"`
}

// Process represents a BPMN Process
type Process struct {
	XMLName      xml.Name       `xml:"process"`
	ID           string         `xml:"id,attr"`
	Name         string         `xml:"name,attr,omitempty"`
	IsExecutable string         `xml:"isExecutable,attr,omitempty"`
	IsClosed     string         `xml:"isClosed,attr,omitempty"`
	ProcessType  string         `xml:"processType,attr,omitempty"` // Private, Public
	Auditing     *Auditing      `xml:"auditing"`
	Monitoring   *Monitoring    `xml:"monitoring"`
	Property     []Property     `xml:"property"`
	FlowElement  []FlowElement  `xml:"flowElement"`
	SequenceFlow []SequenceFlow `xml:"sequenceFlow"`
	ResourceRole []ResourceRole `xml:"resourceRole"`
}

// Auditing holds auditing information
type Auditing struct {
	Documentation []Documentation `xml:"documentation"`
}

// Monitoring holds monitoring information
type Monitoring struct {
	Documentation []Documentation `xml:"documentation"`
}

// Documentation holds documentation text
type Documentation struct {
	ID         string `xml:"id,attr"`
	TextFormat string `xml:"textFormat,attr,omitempty"`
	Text       string `xml:"text"`
}

// Property defines a property of the process
type Property struct {
	ID             string `xml:"id,attr"`
	Name           string `xml:"name,attr,omitempty"`
	ItemSubjectRef string `xml:"itemSubjectRef,attr,omitempty"`
}

// ResourceRole defines a resource role
type ResourceRole struct {
	XMLName                      xml.Name                      `xml:"resourceRole"`
	ID                           string                        `xml:"id,attr"`
	Name                         string                        `xml:"name,attr,omitempty"`
	ResourceRef                  string                        `xml:"resourceRef,attr,omitempty"`
	ResourceAssignmentExpression *ResourceAssignmentExpression `xml:"resourceAssignmentExpression"`
}

// ResourceAssignmentExpression defines resource assignment
type ResourceAssignmentExpression struct {
	Expression *BaseElement `xml:"expression"`
}

// Collaboration represents a collaboration diagram
type Collaboration struct {
	XMLName     xml.Name      `xml:"participant"`
	ID          string        `xml:"id,attr"`
	Name        string        `xml:"name,attr,omitempty"`
	ProcessRef  string        `xml:"processRef,attr,omitempty"`
	Participant []Participant `xml:"participant"`
	MessageFlow []MessageFlow `xml:"messageFlow"`
}

// Participant represents a participant in a collaboration
type Participant struct {
	ID         string `xml:"id,attr"`
	Name       string `xml:"name,attr,omitempty"`
	ProcessRef string `xml:"processRef,attr,omitempty"`
}

// MessageFlow represents a message flow
type MessageFlow struct {
	ID        string `xml:"id,attr"`
	Name      string `xml:"name,attr,omitempty"`
	SourceRef string `xml:"sourceRef,attr"`
	TargetRef string `xml:"targetRef,attr"`
}

// FlowElement is an interface implemented by all flow elements
type FlowElement interface {
	GetID() string
	GetName() string
	GetElementType() FlowElementType
}

// BaseElement is embedded in all BPMN elements
type BaseElement struct {
	ID                string             `xml:"id,attr"`
	Name              string             `xml:"name,attr,omitempty"`
	Documentation     []Documentation    `xml:"documentation"`
	ExtensionElements *ExtensionElements `xml:"extensionElements"`
}

// ExtensionElements holds extension elements
type ExtensionElements struct {
	Any []interface{} `xml:",any"`
}

// FlowElementType represents the type of flow element
type FlowElementType int

const (
	FlowElementTypeUnknown FlowElementType = iota
	FlowElementTypeStartEvent
	FlowElementTypeEndEvent
	FlowElementTypeIntermediateCatchEvent
	FlowElementTypeIntermediateThrowEvent
	FlowElementTypeUserTask
	FlowElementTypeServiceTask
	FlowElementTypeScriptTask
	FlowElementTypeManualTask
	FlowElementTypeReceiveTask
	FlowElementTypeSendTask
	FlowElementTypeExclusiveGateway
	FlowElementTypeInclusiveGateway
	FlowElementTypeParallelGateway
	FlowElementTypeComplexGateway
	FlowElementTypeEventBasedGateway
	FlowElementTypeSubProcess
	FlowElementTypeAdHocSubProcess
	FlowElementTypeTransaction
)

// StartEvent represents a Start Event
type StartEvent struct {
	BaseElement
	Outgoing []string    `xml:"outgoing"`
	Outputs  []OutputSet `xml:"outputSet"`
}

// GetID returns the element ID
func (e *StartEvent) GetID() string { return e.ID }

// GetName returns the element name
func (e *StartEvent) GetName() string { return e.Name }

// GetElementType returns the element type
func (e *StartEvent) GetElementType() FlowElementType { return FlowElementTypeStartEvent }

// EndEvent represents an End Event
type EndEvent struct {
	BaseElement
	Incoming []string   `xml:"incoming"`
	Inputs   []InputSet `xml:"inputSet"`
}

// GetID returns the element ID
func (e *EndEvent) GetID() string { return e.ID }

// GetName returns the element name
func (e *EndEvent) GetName() string { return e.Name }

// GetElementType returns the element type
func (e *EndEvent) GetElementType() FlowElementType { return FlowElementTypeEndEvent }

// IntermediateCatchEvent represents an intermediate catch event
type IntermediateCatchEvent struct {
	BaseElement
	Name     string   `xml:"name,attr,omitempty"`
	Incoming []string `xml:"incoming"`
	Outgoing []string `xml:"outgoing"`
}

// GetID returns the element ID
func (e *IntermediateCatchEvent) GetID() string { return e.ID }

// GetName returns the element name
func (e *IntermediateCatchEvent) GetName() string { return e.Name }

// GetElementType returns the element type
func (e *IntermediateCatchEvent) GetElementType() FlowElementType {
	return FlowElementTypeIntermediateCatchEvent
}

// IntermediateThrowEvent represents an intermediate throw event
type IntermediateThrowEvent struct {
	BaseElement
	Name     string   `xml:"name,attr,omitempty"`
	Incoming []string `xml:"incoming"`
	Outgoing []string `xml:"outgoing"`
}

// GetID returns the element ID
func (e *IntermediateThrowEvent) GetID() string { return e.ID }

// GetName returns the element name
func (e *IntermediateThrowEvent) GetName() string { return e.Name }

// GetElementType returns the element type
func (e *IntermediateThrowEvent) GetElementType() FlowElementType {
	return FlowElementTypeIntermediateThrowEvent
}

// Task represents a base Task
type Task struct {
	BaseElement
	Incoming []string `xml:"incoming"`
	Outgoing []string `xml:"outgoing"`
}

// UserTask represents a User Task
type UserTask struct {
	Task
	Implementation string         `xml:"implementation,attr,omitempty"`
	Rendering      *Rendering     `xml:"rendering"`
	ResourceRole   []ResourceRole `xml:"resourceRole"`
	Priority       string         `xml:"priority,attr,omitempty"`
	DueDate        string         `xml:"dueDate,attr,omitempty"`
}

// GetID returns the element ID
func (t *UserTask) GetID() string { return t.ID }

// GetName returns the element name
func (t *UserTask) GetName() string { return t.Name }

// GetElementType returns the element type
func (t *UserTask) GetElementType() FlowElementType { return FlowElementTypeUserTask }

// ServiceTask represents a Service Task
type ServiceTask struct {
	Task
	Implementation     string `xml:"implementation,attr,omitempty"` // webService, delegate, etc.
	OperationRef       string `xml:"operationRef,attr,omitempty"`
	Class              string `xml:"class,attr,omitempty"`
	DelegateExpression string `xml:"delegateExpression,attr,omitempty"`
	Expression         string `xml:"expression,attr,omitempty"`
	ResultVariable     string `xml:"resultVariable,attr,omitempty"`
	Topic              string `xml:"topic,attr,omitempty"`
	Type               string `xml:"type,attr,omitempty"`
	Retries            string `xml:"retries,attr,omitempty"`
	RetryCycle         string `xml:"retryCycle,attr,omitempty"`
}

// GetID returns the element ID
func (t *ServiceTask) GetID() string { return t.ID }

// GetName returns the element name
func (t *ServiceTask) GetName() string { return t.Name }

// GetElementType returns the element type
func (t *ServiceTask) GetElementType() FlowElementType { return FlowElementTypeServiceTask }

// ScriptTask represents a Script Task
type ScriptTask struct {
	Task
	ScriptFormat       string `xml:"scriptFormat,attr,omitempty"`
	Script             string `xml:"script"`
	AutoStoreVariables string `xml:"autoStoreVariables,attr,omitempty"`
	Resource           string `xml:"resource,attr,omitempty"`
}

// GetID returns the element ID
func (t *ScriptTask) GetID() string { return t.ID }

// GetName returns the element name
func (t *ScriptTask) GetName() string { return t.Name }

// GetElementType returns the element type
func (t *ScriptTask) GetElementType() FlowElementType { return FlowElementTypeScriptTask }

// ManualTask represents a Manual Task
type ManualTask struct {
	Task
}

// GetID returns the element ID
func (t *ManualTask) GetID() string { return t.ID }

// GetName returns the element name
func (t *ManualTask) GetName() string { return t.Name }

// GetElementType returns the element type
func (t *ManualTask) GetElementType() FlowElementType { return FlowElementTypeManualTask }

// ReceiveTask represents a Receive Task
type ReceiveTask struct {
	Task
	MessageRef   string `xml:"messageRef,attr,omitempty"`
	Instantiate  string `xml:"instantiate,attr,omitempty"`
	OperationRef string `xml:"operationRef,attr,omitempty"`
}

// GetID returns the element ID
func (t *ReceiveTask) GetID() string { return t.ID }

// GetName returns the element name
func (t *ReceiveTask) GetName() string { return t.Name }

// GetElementType returns the element type
func (t *ReceiveTask) GetElementType() FlowElementType { return FlowElementTypeReceiveTask }

// SendTask represents a Send Task
type SendTask struct {
	Task
	Implementation string `xml:"implementation,attr,omitempty"`
	MessageRef     string `xml:"messageRef,attr,omitempty"`
	OperationRef   string `xml:"operationRef,attr,omitempty"`
}

// GetID returns the element ID
func (t *SendTask) GetID() string { return t.ID }

// GetName returns the element name
func (t *SendTask) GetName() string { return t.Name }

// GetElementType returns the element type
func (t *SendTask) GetElementType() FlowElementType { return FlowElementTypeSendTask }

// ExclusiveGateway represents an Exclusive (XOR) Gateway
type ExclusiveGateway struct {
	BaseElement
	GatewayType string   `xml:"gatewayDirection,attr"` // Converging, Diverging, Mixed
	Incoming    []string `xml:"incoming"`
	Outgoing    []string `xml:"outgoing"`
	Default     string   `xml:"default,attr,omitempty"`
}

// GetID returns the element ID
func (g *ExclusiveGateway) GetID() string { return g.ID }

// GetName returns the element name
func (g *ExclusiveGateway) GetName() string { return g.Name }

// GetElementType returns the element type
func (g *ExclusiveGateway) GetElementType() FlowElementType { return FlowElementTypeExclusiveGateway }

// InclusiveGateway represents an Inclusive (OR) Gateway
type InclusiveGateway struct {
	BaseElement
	GatewayType string   `xml:"gatewayDirection,attr"` // Converging, Diverging, Mixed
	Incoming    []string `xml:"incoming"`
	Outgoing    []string `xml:"outgoing"`
	Default     string   `xml:"default,attr,omitempty"`
}

// GetID returns the element ID
func (g *InclusiveGateway) GetID() string { return g.ID }

// GetName returns the element name
func (g *InclusiveGateway) GetName() string { return g.Name }

// GetElementType returns the element type
func (g *InclusiveGateway) GetElementType() FlowElementType { return FlowElementTypeInclusiveGateway }

// ParallelGateway represents a Parallel (AND) Gateway
type ParallelGateway struct {
	BaseElement
	GatewayType string   `xml:"gatewayDirection,attr"` // Converging, Diverging, Mixed
	Incoming    []string `xml:"incoming"`
	Outgoing    []string `xml:"outgoing"`
}

// GetID returns the element ID
func (g *ParallelGateway) GetID() string { return g.ID }

// GetName returns the element name
func (g *ParallelGateway) GetName() string { return g.Name }

// GetElementType returns the element type
func (g *ParallelGateway) GetElementType() FlowElementType { return FlowElementTypeParallelGateway }

// ComplexGateway represents a Complex Gateway
type ComplexGateway struct {
	BaseElement
	GatewayType         string   `xml:"gatewayDirection,attr"` // Converging, Diverging, Mixed
	Incoming            []string `xml:"incoming"`
	Outgoing            []string `xml:"outgoing"`
	Default             string   `xml:"default,attr,omitempty"`
	ActivationCondition string   `xml:"activationCondition,attr,omitempty"`
}

// GetID returns the element ID
func (g *ComplexGateway) GetID() string { return g.ID }

// GetName returns the element name
func (g *ComplexGateway) GetName() string { return g.Name }

// GetElementType returns the element type
func (g *ComplexGateway) GetElementType() FlowElementType { return FlowElementTypeComplexGateway }

// EventBasedGateway represents an Event-Based Gateway
type EventBasedGateway struct {
	BaseElement
	GatewayType string   `xml:"gatewayDirection,attr"` // Converging, Diverging, Mixed
	Instantiate string   `xml:"instantiate,attr,omitempty"`
	Incoming    []string `xml:"incoming"`
	Outgoing    []string `xml:"outgoing"`
}

// GetID returns the element ID
func (g *EventBasedGateway) GetID() string { return g.ID }

// GetName returns the element name
func (g *EventBasedGateway) GetName() string { return g.Name }

// GetElementType returns the element type
func (g *EventBasedGateway) GetElementType() FlowElementType { return FlowElementTypeEventBasedGateway }

// SequenceFlow represents a Sequence Flow
type SequenceFlow struct {
	BaseElement
	SourceRef           string               `xml:"sourceRef,attr"`
	TargetRef           string               `xml:"targetRef,attr"`
	ConditionExpression *ConditionExpression `xml:"conditionExpression"`
}

// ConditionExpression defines a condition
type ConditionExpression struct {
	Type               string `xml:"type,attr,omitempty"`
	EvaluatesToTypeRef string `xml:"evaluatesToTypeRef,attr,omitempty"`
	Text               string `xml:",chardata"`
}

// SubProcess represents a Sub-Process
type SubProcess struct {
	BaseElement
	Name             string         `xml:"name,attr,omitempty"`
	TriggeredByEvent string         `xml:"triggeredByEvent,attr,omitempty"`
	ProcessType      string         `xml:"processType,attr,omitempty"`
	Incoming         []string       `xml:"incoming"`
	Outgoing         []string       `xml:"outgoing"`
	FlowElement      []FlowElement  `xml:"flowElement"`
	SequenceFlow     []SequenceFlow `xml:"sequenceFlow"`
}

// GetID returns the element ID
func (s *SubProcess) GetID() string { return s.ID }

// GetName returns the element name
func (s *SubProcess) GetName() string { return s.Name }

// GetElementType returns the element type
func (s *SubProcess) GetElementType() FlowElementType { return FlowElementTypeSubProcess }

// AdHocSubProcess represents an Ad-Hoc Sub-Process
type AdHocSubProcess struct {
	SubProcess
	CancelRemainingInstances string `xml:"cancelRemainingInstances,attr,omitempty"`
	CompletionCondition      string `xml:"completionCondition,attr,omitempty"`
}

// GetElementType returns the element type
func (s *AdHocSubProcess) GetElementType() FlowElementType { return FlowElementTypeAdHocSubProcess }

// Transaction represents a Transaction Sub-Process
type Transaction struct {
	SubProcess
	Method string `xml:"method,attr,omitempty"`
}

// GetElementType returns the element type
func (t *Transaction) GetElementType() FlowElementType { return FlowElementTypeTransaction }

// Rendering holds rendering information for tasks
type Rendering struct {
	InputOutput *InputOutput `xml:"inputOutput"`
}

// InputOutput defines input and output parameters
type InputOutput struct {
	InputParameter  []InputParameter  `xml:"inputParameter"`
	OutputParameter []OutputParameter `xml:"outputParameter"`
}

// InputParameter defines an input parameter
type InputParameter struct {
	Name string `xml:"name,attr"`
	Text string `xml:"text,attr,omitempty"`
}

// OutputParameter defines an output parameter
type OutputParameter struct {
	Name string `xml:"name,attr"`
	Text string `xml:"text,attr,omitempty"`
}

// InputSet defines input set
type InputSet struct {
	Name          string   `xml:"name,attr,omitempty"`
	ItemSemantics string   `xml:"itemSemantics,attr,omitempty"`
	DataInputRefs []string `xml:"dataInputRefs"`
}

// OutputSet defines output set
type OutputSet struct {
	Name           string   `xml:"name,attr,omitempty"`
	ItemSemantics  string   `xml:"itemSemantics,attr,omitempty"`
	DataOutputRefs []string `xml:"dataOutputRefs"`
}

// BPMNDiagram holds diagram information
type BPMNDiagram struct {
	XMLName   xml.Name   `xml:"bpmndi:BPMNDiagram"`
	ID        string     `xml:"id,attr"`
	Name      string     `xml:"name,attr,omitempty"`
	BPMNPlane *BPMNPlane `xml:"bpmndi:BPMNPlane"`
}

// BPMNPlane holds the plane information
type BPMNPlane struct {
	XMLName     xml.Name      `xml:"bpmndi:BPMNPlane"`
	ID          string        `xml:"id,attr"`
	BPMNElement []BPMNElement `xml:"bpmndi:BPMNElement"`
}

// BPMNElement holds element reference
type BPMNElement struct {
	ID   string `xml:"id,attr"`
	Name string `xml:"name,attr,omitempty"`
}

// ProcessInstance represents a running instance of a process
type ProcessInstance struct {
	ID             string                 `json:"id"`
	ProcessID      string                 `json:"process_id"`
	Status         string                 `json:"status"` // pending, running, completed, terminated, suspended
	Variables      map[string]interface{} `json:"variables"`
	CurrentElement string                 `json:"current_element,omitempty"`
	StartedAt      time.Time              `json:"started_at"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
	CreatedBy      string                 `json:"created_by,omitempty"`
}

// Token represents a token in the process
type Token struct {
	ID          string                 `json:"id"`
	ProcessID   string                 `json:"process_id"`
	ElementID   string                 `json:"element_id"`
	Status      string                 `json:"status"` // active, waiting, completed
	Variables   map[string]interface{} `json:"variables"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	ElementID   string `json:"element_id"`
	ElementName string `json:"element_name,omitempty"`
	Message     string `json:"message"`
	Severity    string `json:"severity"` // error, warning
}
