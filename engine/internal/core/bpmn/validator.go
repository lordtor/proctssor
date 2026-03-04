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

	// Validate boundary events
	v.validateBoundaryEvents(process)
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
		case *ReceiveTask:
			v.validateReceiveTask(task)
		case *SendTask:
			v.validateSendTask(task)
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

func (v *Validator) validateBoundaryEvents(process *Process) {
	// Collect all valid attachedToRef targets (activities that can have boundary events)
	validTargets := make(map[string]bool)
	for _, elem := range process.FlowElement {
		if elem == nil {
			continue
		}
		switch elem.GetElementType() {
		case FlowElementTypeUserTask,
			FlowElementTypeServiceTask,
			FlowElementTypeScriptTask,
			FlowElementTypeManualTask,
			FlowElementTypeReceiveTask,
			FlowElementTypeSubProcess:
			validTargets[elem.GetID()] = true
		}
	}

	// Validate each boundary event
	for _, elem := range process.FlowElement {
		if elem == nil {
			continue
		}

		be, ok := elem.(*BoundaryEvent)
		if !ok {
			continue
		}

		// Boundary event must have attachedToRef
		if be.AttachedToRef == "" {
			v.addError(be.ID, be.Name, "Boundary event must have attachedToRef")
		} else if !validTargets[be.AttachedToRef] {
			v.addError(be.ID, be.Name, fmt.Sprintf("Boundary event attached to invalid element '%s' (must be a task or subprocess)", be.AttachedToRef))
		}

		// Boundary event must have timer or error definition
		hasTimer := be.TimerEventDefinition != nil
		hasError := be.ErrorEventDefinition != nil

		if !hasTimer && !hasError {
			v.addError(be.ID, be.Name, "Boundary event must have a timer or error event definition")
		}

		// Timer boundary event must have time duration/date/cycle
		if hasTimer {
			if be.TimerEventDefinition.TimeDuration == "" &&
				be.TimerEventDefinition.TimeDate == "" &&
				be.TimerEventDefinition.TimeCycle == "" {
				v.addError(be.ID, be.Name, "Timer boundary event must have timeDuration, timeDate, or timeCycle")
			}
		}

		// Error boundary event should have errorRef
		if hasError && be.ErrorEventDefinition.ErrorRef == "" {
			v.addWarning(be.ID, be.Name, "Error boundary event should have an errorRef")
		}

		// Boundary event must have outgoing flow (to handle the event)
		if len(be.Outgoing) == 0 {
			v.addError(be.ID, be.Name, "Boundary event must have at least one outgoing sequence flow")
		}

		// Non-interrupting boundary events should not have incoming flows
		if len(be.Incoming) > 0 && !be.IsInterrupting() {
			v.addWarning(be.ID, be.Name, "Non-interrupting boundary event should typically not have incoming flows")
		}
	}

	// Check for multiple interrupting boundary events of the same type
	interruptingTimers := make(map[string]int)
	interruptingErrors := make(map[string]int)

	for _, elem := range process.FlowElement {
		be, ok := elem.(*BoundaryEvent)
		if !ok || !be.IsInterrupting() {
			continue
		}

		if be.TimerEventDefinition != nil {
			interruptingTimers[be.AttachedToRef]++
		}
		if be.ErrorEventDefinition != nil {
			interruptingErrors[be.AttachedToRef]++
		}
	}

	for targetID, count := range interruptingTimers {
		if count > 1 {
			v.addWarning("", "", fmt.Sprintf("Activity '%s' has %d interrupting timer boundary events (only one will execute)", targetID, count))
		}
	}

	for targetID, count := range interruptingErrors {
		if count > 1 {
			v.addWarning("", "", fmt.Sprintf("Activity '%s' has %d interrupting error boundary events", targetID, count))
		}
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

func (v *Validator) validateReceiveTask(task *ReceiveTask) {
	if len(task.Outgoing) == 0 {
		v.addError(task.ID, task.Name, "Receive task must have at least one outgoing sequence flow")
	}
	if len(task.Incoming) == 0 {
		v.addError(task.ID, task.Name, "Receive task must have at least one incoming sequence flow")
	}
	// ReceiveTask should have a messageRef for message correlation
	if task.MessageRef == "" {
		v.addWarning(task.ID, task.Name, "Receive task should have a messageRef for message correlation")
	}
}

func (v *Validator) validateSendTask(task *SendTask) {
	if len(task.Outgoing) == 0 {
		v.addError(task.ID, task.Name, "Send task must have at least one outgoing sequence flow")
	}
	if len(task.Incoming) == 0 {
		v.addError(task.ID, task.Name, "Send task must have at least one incoming sequence flow")
	}
	// SendTask should have a messageRef to identify the message
	if task.MessageRef == "" {
		v.addWarning(task.ID, task.Name, "Send task should have a messageRef to identify the message")
	}
}

// ValidateMessageFlow validates the message flows in a collaboration
func (v *Validator) ValidateMessageFlow(collaboration *Collaboration, processes []Process) {
	if collaboration == nil {
		return
	}

	// Build a map of all participants and their process elements
	participantElements := make(map[string]map[string]bool)
	for _, p := range collaboration.Participant {
		participantElements[p.ID] = make(map[string]bool)
		// Find the process for this participant
		for _, proc := range processes {
			if proc.ID == p.ProcessRef {
				for _, elem := range proc.FlowElement {
					if elem != nil {
						participantElements[p.ID][elem.GetID()] = true
					}
				}
			}
		}
	}

	// Validate each message flow
	for _, mf := range collaboration.MessageFlow {
		// Check source and target exist
		if mf.SourceRef == "" {
			v.addError(mf.ID, mf.Name, "Message flow must have a sourceRef")
		}
		if mf.TargetRef == "" {
			v.addError(mf.ID, mf.Name, "Message flow must have a targetRef")
		}

		// Check source is a SendTask or ThrowEvent
		// Check target is a ReceiveTask or CatchEvent
		// This is a simplified check - in a full implementation, you'd verify the element types

		// Check source and target are different
		if mf.SourceRef == mf.TargetRef && mf.SourceRef != "" {
			v.addError(mf.ID, mf.Name, "Message flow cannot connect an element to itself")
		}

		// Check that source and target belong to different participants
		sourceParticipant := v.findParticipantForElement(collaboration, mf.SourceRef, processes)
		targetParticipant := v.findParticipantForElement(collaboration, mf.TargetRef, processes)

		if sourceParticipant != "" && targetParticipant != "" && sourceParticipant == targetParticipant {
			v.addWarning(mf.ID, mf.Name, fmt.Sprintf("Message flow connects elements within the same participant '%s' (should connect different pools)", sourceParticipant))
		}
	}
}

// findParticipantForElement finds which participant contains a given element
func (v *Validator) findParticipantForElement(collaboration *Collaboration, elementID string, processes []Process) string {
	if collaboration == nil {
		return ""
	}

	for _, p := range collaboration.Participant {
		for _, proc := range processes {
			if proc.ID == p.ProcessRef {
				for _, elem := range proc.FlowElement {
					if elem != nil && elem.GetID() == elementID {
						return p.ID
					}
				}
			}
		}
	}
	return ""
}
