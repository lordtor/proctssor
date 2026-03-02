package bpmn

import (
	"fmt"
)

// Validator validates BPMN processes
type Validator struct {
	errors   []ValidationError
	warnings []ValidationError
}

// NewValidator creates a new BPMN validator
func NewValidator() *Validator {
	return &Validator{
		errors:   make([]ValidationError, 0),
		warnings: make([]ValidationError, 0),
	}
}

// Validate validates a BPMN process and returns validation errors
func Validate(process *Process) []ValidationError {
	validator := NewValidator()
	validator.validate(process)

	// Combine errors and warnings
	result := append(validator.errors, validator.warnings...)
	return result
}

// ValidateWithWarnings validates a BPMN process and returns errors and warnings separately
func ValidateWithWarnings(process *Process) (errors, warnings []ValidationError) {
	validator := NewValidator()
	validator.validate(process)
	return validator.errors, validator.warnings
}

func (v *Validator) validate(process *Process) {
	if process == nil {
		v.addError("", "", "Process is nil")
		return
	}

	// Validate process ID
	if process.ID == "" {
		v.addError(process.ID, process.Name, "Process must have an ID")
	}

	// Validate executable flag
	if process.IsExecutable == "" {
		v.addWarning(process.ID, process.Name, "Process should have isExecutable attribute set")
	}

	// Validate flow elements
	if len(process.FlowElement) == 0 {
		v.addError(process.ID, process.Name, "Process must have at least one flow element")
	}

	// Validate start and end events
	v.validateEvents(process)

	// Validate sequence flows
	v.validateSequenceFlows(process)

	// Validate gateways
	v.validateGateways(process)

	// Validate tasks
	v.validateTasks(process)
}

func (v *Validator) validateEvents(process *Process) {
	var startEvents []*StartEvent
	var endEvents []*EndEvent

	for _, elem := range process.FlowElement {
		if elem == nil {
			continue
		}

		switch e := elem.(type) {
		case *StartEvent:
			startEvents = append(startEvents, e)
		case *EndEvent:
			endEvents = append(endEvents, e)
		}
	}

	// Validate start events
	if len(startEvents) == 0 {
		v.addError(process.ID, process.Name, "Process must have at least one start event")
	} else if len(startEvents) > 1 {
		v.addWarning(process.ID, process.Name, fmt.Sprintf("Process has %d start events (recommended: 1)", len(startEvents)))
	}

	// Validate end events
	if len(endEvents) == 0 {
		v.addError(process.ID, process.Name, "Process must have at least one end event")
	}

	// Check start events have outgoing flows
	for _, se := range startEvents {
		if len(se.Outgoing) == 0 {
			v.addError(se.ID, se.Name, "Start event must have at least one outgoing sequence flow")
		}
	}

	// Check end events have incoming flows
	for _, ee := range endEvents {
		if len(ee.Incoming) == 0 {
			v.addError(ee.ID, ee.Name, "End event must have at least one incoming sequence flow")
		}
	}
}

func (v *Validator) validateSequenceFlows(process *Process) {
	elementIDs := make(map[string]bool)

	// Collect all element IDs
	for _, elem := range process.FlowElement {
		if elem != nil && elem.GetID() != "" {
			elementIDs[elem.GetID()] = true
		}
	}

	// Validate each sequence flow
	for _, flow := range process.SequenceFlow {
		// Check ID
		if flow.ID == "" {
			v.addError(flow.ID, flow.Name, "Sequence flow must have an ID")
			continue
		}

		// Check source and target references
		if flow.SourceRef == "" {
			v.addError(flow.ID, flow.Name, "Sequence flow must have a sourceRef")
		} else if !elementIDs[flow.SourceRef] {
			v.addError(flow.ID, flow.Name, fmt.Sprintf("Sequence flow sourceRef '%s' does not exist", flow.SourceRef))
		}

		if flow.TargetRef == "" {
			v.addError(flow.ID, flow.Name, "Sequence flow must have a targetRef")
		} else if !elementIDs[flow.TargetRef] {
			v.addError(flow.ID, flow.Name, fmt.Sprintf("Sequence flow targetRef '%s' does not exist", flow.TargetRef))
		}

		// Check for source and target being the same
		if flow.SourceRef == flow.TargetRef && flow.SourceRef != "" {
			v.addError(flow.ID, flow.Name, "Sequence flow cannot connect an element to itself")
		}
	}

	// Check for orphan elements (elements with no connections)
	v.checkOrphanElements(process, elementIDs)
}

func (v *Validator) checkOrphanElements(process *Process, elementIDs map[string]bool) {
	// Build connection map
	connected := make(map[string]bool)

	for _, flow := range process.SequenceFlow {
		connected[flow.SourceRef] = true
		connected[flow.TargetRef] = true
	}

	// Check each element
	for _, elem := range process.FlowElement {
		if elem == nil || elem.GetID() == "" {
			continue
		}

		if !connected[elem.GetID()] {
			v.addWarning(elem.GetID(), elem.GetName(), "Element is not connected to any sequence flow")
		}
	}
}

func (v *Validator) validateGateways(process *Process) {
	for _, elem := range process.FlowElement {
		if elem == nil {
			continue
		}

		switch gw := elem.(type) {
		case *ExclusiveGateway:
			v.validateExclusiveGateway(gw)
		case *InclusiveGateway:
			v.validateInclusiveGateway(gw)
		case *ParallelGateway:
			v.validateParallelGateway(gw)
		}
	}
}

func (v *Validator) validateExclusiveGateway(gw *ExclusiveGateway) {
	outgoing := len(gw.Outgoing)
	incoming := len(gw.Incoming)

	// Exclusive gateway should have exactly one incoming or one outgoing
	// (unless it's mixed type)
	if incoming == 0 && outgoing == 0 {
		v.addError(gw.ID, gw.Name, "Exclusive gateway must have at least one incoming or outgoing sequence flow")
	}

	// Check default flow exists if there are multiple outgoing flows
	if outgoing > 1 && gw.Default == "" {
		v.addWarning(gw.ID, gw.Name, "Exclusive gateway with multiple outgoing flows should have a default flow")
	}
}

func (v *Validator) validateInclusiveGateway(gw *InclusiveGateway) {
	outgoing := len(gw.Outgoing)
	incoming := len(gw.Incoming)

	if incoming == 0 && outgoing == 0 {
		v.addError(gw.ID, gw.Name, "Inclusive gateway must have at least one incoming or outgoing sequence flow")
	}

	// Inclusive gateway should have conditions on outgoing flows
	if outgoing > 0 && gw.Default == "" {
		v.addWarning(gw.ID, gw.Name, "Inclusive gateway should have a default flow")
	}
}

func (v *Validator) validateParallelGateway(gw *ParallelGateway) {
	outgoing := len(gw.Outgoing)
	incoming := len(gw.Incoming)

	if incoming == 0 && outgoing == 0 {
		v.addError(gw.ID, gw.Name, "Parallel gateway must have at least one incoming or outgoing sequence flow")
	}

	// Parallel gateway should have equal incoming and outgoing for proper synchronization
	if incoming != outgoing && incoming > 0 && outgoing > 0 {
		v.addWarning(gw.ID, gw.Name, "Parallel gateway has different number of incoming and outgoing flows (may cause synchronization issues)")
	}
}

func (v *Validator) validateTasks(process *Process) {
	for _, elem := range process.FlowElement {
		if elem == nil {
			continue
		}

		switch task := elem.(type) {
		case *UserTask:
			v.validateUserTask(task)
		case *ServiceTask:
			v.validateServiceTask(task)
		case *ScriptTask:
			v.validateScriptTask(task)
		}
	}
}

func (v *Validator) validateUserTask(task *UserTask) {
	if len(task.Outgoing) == 0 {
		v.addError(task.ID, task.Name, "User task must have at least one outgoing sequence flow")
	}
	if len(task.Incoming) == 0 {
		v.addError(task.ID, task.Name, "User task must have at least one incoming sequence flow")
	}
}

func (v *Validator) validateServiceTask(task *ServiceTask) {
	if len(task.Outgoing) == 0 {
		v.addError(task.ID, task.Name, "Service task must have at least one outgoing sequence flow")
	}
	if len(task.Incoming) == 0 {
		v.addError(task.ID, task.Name, "Service task must have at least one incoming sequence flow")
	}

	// Check service task has implementation
	hasImplementation := task.Implementation != "" ||
		task.Class != "" ||
		task.Expression != "" ||
		task.DelegateExpression != "" ||
		task.Topic != "" ||
		task.Type != ""

	if !hasImplementation {
		v.addWarning(task.ID, task.Name, "Service task should have at least one implementation method (class, expression, delegateExpression, topic, or type)")
	}
}

func (v *Validator) validateScriptTask(task *ScriptTask) {
	if len(task.Outgoing) == 0 {
		v.addError(task.ID, task.Name, "Script task must have at least one outgoing sequence flow")
	}
	if len(task.Incoming) == 0 {
		v.addError(task.ID, task.Name, "Script task must have at least one incoming sequence flow")
	}

	// Check script task has script
	if task.Script == "" && task.Resource == "" {
		v.addWarning(task.ID, task.Name, "Script task should have a script or resource defined")
	}
}

func (v *Validator) addError(id, name, message string) {
	v.errors = append(v.errors, ValidationError{
		ElementID:   id,
		ElementName: name,
		Message:     message,
		Severity:    "error",
	})
}

func (v *Validator) addWarning(id, name, message string) {
	v.warnings = append(v.warnings, ValidationError{
		ElementID:   id,
		ElementName: name,
		Message:     message,
		Severity:    "warning",
	})
}
