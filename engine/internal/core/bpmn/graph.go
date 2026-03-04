package bpmn

import (
	"fmt"
)

// Graph represents a BPMN process graph
type Graph struct {
	// Nodes is a map of element ID to FlowElement
	Nodes map[string]FlowElement `json:"nodes"`

	// Edges is a map of source element ID to list of target FlowElements
	Edges map[string][]FlowElement `json:"edges"`

	// SequenceFlows holds all sequence flows in the graph
	SequenceFlows []SequenceFlow `json:"sequence_flows"`

	// StartEvents holds all start events
	StartEvents []*StartEvent `json:"start_events"`

	// EndEvents holds all end events
	EndEvents []*EndEvent `json:"end_events"`

	// Gateways holds all gateways by type
	Gateways map[FlowElementType][]FlowElement `json:"gateways"`

	// Tasks holds all tasks (user, service, script, manual)
	Tasks map[FlowElementType][]FlowElement `json:"tasks"`

	// BoundaryEvents holds all boundary events
	BoundaryEvents []*BoundaryEvent `json:"boundary_events"`

	// BoundaryEventsByActivity maps activity ID to its boundary events
	BoundaryEventsByActivity map[string][]*BoundaryEvent `json:"boundary_events_by_activity"`
}

// NewGraph creates a new Graph
func NewGraph() *Graph {
	return &Graph{
		Nodes:                    make(map[string]FlowElement),
		Edges:                    make(map[string][]FlowElement),
		Gateways:                 make(map[FlowElementType][]FlowElement),
		Tasks:                    make(map[FlowElementType][]FlowElement),
		BoundaryEvents:           make([]*BoundaryEvent, 0),
		BoundaryEventsByActivity: make(map[string][]*BoundaryEvent),
	}
}

// BuildGraph builds a graph from a Process
func BuildGraph(process *Process) (*Graph, error) {
	graph := NewGraph()

	// Add all flow elements as nodes
	for _, elem := range process.FlowElement {
		if elem == nil {
			continue
		}
		id := elem.GetID()
		if id == "" {
			continue
		}
		graph.Nodes[id] = elem

		// Categorize elements
		switch elem.GetElementType() {
		case FlowElementTypeStartEvent:
			graph.StartEvents = append(graph.StartEvents, elem.(*StartEvent))
		case FlowElementTypeEndEvent:
			graph.EndEvents = append(graph.EndEvents, elem.(*EndEvent))
		case FlowElementTypeExclusiveGateway,
			FlowElementTypeInclusiveGateway,
			FlowElementTypeParallelGateway,
			FlowElementTypeComplexGateway,
			FlowElementTypeEventBasedGateway:
			graph.Gateways[elem.GetElementType()] = append(graph.Gateways[elem.GetElementType()], elem)
		case FlowElementTypeUserTask,
			FlowElementTypeServiceTask,
			FlowElementTypeScriptTask,
			FlowElementTypeManualTask,
			FlowElementTypeReceiveTask,
			FlowElementTypeSendTask:
			graph.Tasks[elem.GetElementType()] = append(graph.Tasks[elem.GetElementType()], elem)
		case FlowElementTypeBoundaryEvent,
			FlowElementTypeTimerBoundaryEvent,
			FlowElementTypeErrorBoundaryEvent:
			be := elem.(*BoundaryEvent)
			graph.BoundaryEvents = append(graph.BoundaryEvents, be)
			if be.AttachedToRef != "" {
				graph.BoundaryEventsByActivity[be.AttachedToRef] = append(graph.BoundaryEventsByActivity[be.AttachedToRef], be)
			}
		}
	}

	// Build edges from sequence flows
	for _, flow := range process.SequenceFlow {
		graph.SequenceFlows = append(graph.SequenceFlows, flow)

		sourceID := flow.SourceRef
		targetID := flow.TargetRef

		// Check if target exists
		if _, exists := graph.Nodes[targetID]; !exists {
			continue
		}

		// Add edge from source to target
		graph.Edges[sourceID] = append(graph.Edges[sourceID], graph.Nodes[targetID])
	}

	return graph, nil
}

// FindNextNodes finds the next nodes from the current element based on variables
func (g *Graph) FindNextNodes(currentID string, variables map[string]interface{}) ([]FlowElement, error) {
	// Get outgoing flows from current element
	currentElem, exists := g.Nodes[currentID]
	if !exists {
		return nil, fmt.Errorf("element with id %s not found", currentID)
	}

	// Get edges from current element
	nextElems := g.Edges[currentID]

	// Handle gateway logic
	switch currentElem.GetElementType() {
	case FlowElementTypeExclusiveGateway:
		// Exclusive gateway: evaluate conditions, pick first true
		return g.evaluateExclusiveGateway(currentID, variables), nil

	case FlowElementTypeInclusiveGateway:
		// Inclusive gateway: evaluate conditions, pick all true
		return g.evaluateInclusiveGateway(currentID, variables), nil

	case FlowElementTypeParallelGateway:
		// Parallel gateway: all outgoing flows
		return nextElems, nil

	case FlowElementTypeEventBasedGateway:
		// Event-based gateway: wait for first event
		return nextElems, nil
	}

	// For regular elements, return all outgoing flows
	return nextElems, nil
}

// evaluateExclusiveGateway evaluates conditions for exclusive gateway
func (g *Graph) evaluateExclusiveGateway(gatewayID string, variables map[string]interface{}) []FlowElement {
	// Find the outgoing sequence flows for this gateway
	var candidateFlows []struct {
		flow    SequenceFlow
		element FlowElement
	}

	for _, flow := range g.SequenceFlows {
		if flow.SourceRef == gatewayID {
			// Find target element
			if targetElem, exists := g.Nodes[flow.TargetRef]; exists {
				candidateFlows = append(candidateFlows, struct {
					flow    SequenceFlow
					element FlowElement
				}{flow, targetElem})
			}
		}
	}

	// For now, return the default flow if no conditions match
	// In a full implementation, evaluate each condition expression
	for _, candidate := range candidateFlows {
		if candidate.flow.ConditionExpression == nil {
			// No condition = default flow
			return []FlowElement{candidate.element}
		}
	}

	// If no default, return first candidate
	if len(candidateFlows) > 0 {
		return []FlowElement{candidateFlows[0].element}
	}

	return nil
}

// evaluateInclusiveGateway evaluates conditions for inclusive gateway
func (g *Graph) evaluateInclusiveGateway(gatewayID string, variables map[string]interface{}) []FlowElement {
	var result []FlowElement

	// Find outgoing sequence flows
	for _, flow := range g.SequenceFlows {
		if flow.SourceRef == gatewayID {
			// In a full implementation, evaluate the condition
			// For now, include all flows
			if targetElem, exists := g.Nodes[flow.TargetRef]; exists {
				result = append(result, targetElem)
			}
		}
	}

	return result
}

// GetStartNode returns the start event of the process
func (g *Graph) GetStartNode() (*StartEvent, error) {
	if len(g.StartEvents) == 0 {
		return nil, fmt.Errorf("no start event found")
	}
	// Return first start event (processes typically have one)
	return g.StartEvents[0], nil
}

// GetEndNodes returns all end events of the process
func (g *Graph) GetEndNodes() []*EndEvent {
	return g.EndEvents
}

// GetElementByID returns a flow element by its ID
func (g *Graph) GetElementByID(id string) (FlowElement, bool) {
	elem, exists := g.Nodes[id]
	return elem, exists
}

// GetOutgoingEdges returns all outgoing edges from an element
func (g *Graph) GetOutgoingEdges(elementID string) []FlowElement {
	return g.Edges[elementID]
}

// GetIncomingElements returns all elements that have edges to the given element
func (g *Graph) GetIncomingElements(elementID string) []FlowElement {
	var result []FlowElement

	for sourceID, targets := range g.Edges {
		for _, target := range targets {
			if target.GetID() == elementID {
				if sourceElem, exists := g.Nodes[sourceID]; exists {
					result = append(result, sourceElem)
				}
			}
		}
	}

	return result
}

// IsEndNode checks if the given element is an end node
func (g *Graph) IsEndNode(elementID string) bool {
	for _, endEvent := range g.EndEvents {
		if endEvent.ID == elementID {
			return true
		}
	}
	return false
}

// IsStartNode checks if the given element is a start node
func (g *Graph) IsStartNode(elementID string) bool {
	for _, startEvent := range g.StartEvents {
		if startEvent.ID == elementID {
			return true
		}
	}
	return false
}

// GetGateways returns all gateways in the graph
func (g *Graph) GetGateways() []FlowElement {
	var result []FlowElement
	for _, gateways := range g.Gateways {
		result = append(result, gateways...)
	}
	return result
}

// GetTasks returns all tasks in the graph
func (g *Graph) GetTasks() []FlowElement {
	var result []FlowElement
	for _, tasks := range g.Tasks {
		result = append(result, tasks...)
	}
	return result
}

// GetBoundaryEvents returns all boundary events in the graph
func (g *Graph) GetBoundaryEvents() []*BoundaryEvent {
	return g.BoundaryEvents
}

// GetBoundaryEventsForActivity returns all boundary events attached to an activity
func (g *Graph) GetBoundaryEventsForActivity(activityID string) []*BoundaryEvent {
	return g.BoundaryEventsByActivity[activityID]
}

// GetTimerBoundaryEventsForActivity returns all timer boundary events attached to an activity
func (g *Graph) GetTimerBoundaryEventsForActivity(activityID string) []*BoundaryEvent {
	var result []*BoundaryEvent
	for _, be := range g.BoundaryEventsByActivity[activityID] {
		if be.TimerEventDefinition != nil {
			result = append(result, be)
		}
	}
	return result
}

// GetErrorBoundaryEventsForActivity returns all error boundary events attached to an activity
func (g *Graph) GetErrorBoundaryEventsForActivity(activityID string) []*BoundaryEvent {
	var result []*BoundaryEvent
	for _, be := range g.BoundaryEventsByActivity[activityID] {
		if be.ErrorEventDefinition != nil {
			result = append(result, be)
		}
	}
	return result
}

// HasInterruptingBoundaryEvent checks if an activity has an interrupting boundary event
func (g *Graph) HasInterruptingBoundaryEvent(activityID string) bool {
	for _, be := range g.BoundaryEventsByActivity[activityID] {
		if be.IsInterrupting() {
			return true
		}
	}
	return false
}
