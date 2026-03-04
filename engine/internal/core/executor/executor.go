package executor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/workflow-engine/v2/internal/core/bpmn"
	"github.com/workflow-engine/v2/internal/core/statemachine"
	"github.com/workflow-engine/v2/internal/integration/nats"
	"go.uber.org/zap"
)

// Context keys for execution context
type contextKey string

const (
	ContextKeyInstanceID contextKey = "instance_id"
	ContextKeyNodeID     contextKey = "node_id"
	ContextKeyNodeName   contextKey = "node_name"
)

// GetInstanceID retrieves instance ID from context
func GetInstanceID(ctx context.Context) string {
	if id, ok := ctx.Value(ContextKeyInstanceID).(string); ok {
		return id
	}
	return ""
}

// GetNodeID retrieves node ID from context
func GetNodeID(ctx context.Context) string {
	if id, ok := ctx.Value(ContextKeyNodeID).(string); ok {
		return id
	}
	return ""
}

// Executor executes BPMN nodes
type Executor struct {
	// TaskHandlers holds handlers for different task types
	TaskHandlers map[string]TaskHandler

	// VariableManagers holds variable managers
	VariableManagers map[string]VariableManager
}

// TaskHandler handles execution of a specific task type
type TaskHandler interface {
	Execute(ctx context.Context, node bpmn.FlowElement, variables map[string]interface{}) (*statemachine.ExecutionResult, error)
}

// VariableManager manages process variables
type VariableManager interface {
	Get(ctx context.Context, instanceID, name string) (interface{}, error)
	Set(ctx context.Context, instanceID, name string, value interface{}) error
	Delete(ctx context.Context, instanceID, name string) error
}

// DefaultExecutor is the default implementation of Executor
type DefaultExecutor struct {
	taskHandlers  map[string]TaskHandler
	registryCache interface {
		Get(ctx context.Context, name string) (interface{}, bool)
	}
	natsPublisher *nats.Publisher
	logger        *zap.Logger
}

// NewExecutor creates a new executor
func NewExecutor(
	registryCache interface {
		Get(ctx context.Context, name string) (interface{}, bool)
	},
	natsPublisher *nats.Publisher,
	logger *zap.Logger,
) *DefaultExecutor {
	return &DefaultExecutor{
		taskHandlers:  make(map[string]TaskHandler),
		registryCache: registryCache,
		natsPublisher: natsPublisher,
		logger:        logger,
	}
}

// ExecuteNode executes a BPMN node
func (e *DefaultExecutor) ExecuteNode(ctx context.Context, graph *bpmn.Graph, currentNode bpmn.FlowElement, variables map[string]interface{}) (*statemachine.ExecutionResult, error) {
	if currentNode == nil {
		return nil, fmt.Errorf("current node is nil")
	}

	nodeID := currentNode.GetID()

	// Execute based on node type
	switch elem := currentNode.(type) {
	case *bpmn.StartEvent:
		return e.executeStartEvent(ctx, elem, variables)
	case *bpmn.EndEvent:
		return e.executeEndEvent(ctx, elem, variables)
	case *bpmn.UserTask:
		return e.executeUserTask(ctx, elem, variables)
	case *bpmn.ServiceTask:
		return e.executeServiceTask(ctx, elem, variables)
	case *bpmn.ScriptTask:
		return e.executeScriptTask(ctx, elem, variables)
	case *bpmn.ManualTask:
		return e.executeManualTask(ctx, elem, variables)
	case *bpmn.ReceiveTask:
		return e.executeReceiveTask(ctx, elem, variables)
	case *bpmn.SendTask:
		return e.executeSendTask(ctx, elem, variables)
	case *bpmn.ExclusiveGateway:
		return e.executeExclusiveGateway(ctx, graph, elem, variables)
	case *bpmn.InclusiveGateway:
		return e.executeInclusiveGateway(ctx, graph, elem, variables)
	case *bpmn.ParallelGateway:
		return e.executeParallelGateway(ctx, graph, elem, variables)
	case *bpmn.IntermediateCatchEvent:
		return e.executeIntermediateCatchEvent(ctx, elem, variables)
	case *bpmn.IntermediateThrowEvent:
		return e.executeIntermediateThrowEvent(ctx, elem, variables)
	case *bpmn.BoundaryEvent:
		return e.executeBoundaryEvent(ctx, graph, elem, variables)
	default:
		return nil, fmt.Errorf("unsupported node type: %T for node %s", currentNode, nodeID)
	}
}

// executeStartEvent executes a start event
func (e *DefaultExecutor) executeStartEvent(ctx context.Context, node *bpmn.StartEvent, variables map[string]interface{}) (*statemachine.ExecutionResult, error) {
	// Start events just pass through to next node
	// Get outgoing flows
	if len(node.Outgoing) == 0 {
		return nil, fmt.Errorf("start event %s has no outgoing flows", node.ID)
	}

	// Return first outgoing flow target
	return &statemachine.ExecutionResult{
		NextNodeID: node.Outgoing[0],
		Variables:  variables,
	}, nil
}

// executeEndEvent executes an end event
func (e *DefaultExecutor) executeEndEvent(ctx context.Context, node *bpmn.EndEvent, variables map[string]interface{}) (*statemachine.ExecutionResult, error) {
	// End event terminates the token flow
	return &statemachine.ExecutionResult{
		NextNodeID: "",
		Variables:  variables,
		Terminated: true,
	}, nil
}

// executeUserTask executes a user task
func (e *DefaultExecutor) executeUserTask(ctx context.Context, node *bpmn.UserTask, variables map[string]interface{}) (*statemachine.ExecutionResult, error) {
	// User tasks require external completion
	return &statemachine.ExecutionResult{
		Variables: variables,
		Await:     true,
		AwaitType: "user_task",
	}, nil
}

// executeServiceTask executes a service task
func (e *DefaultExecutor) executeServiceTask(ctx context.Context, node *bpmn.ServiceTask, variables map[string]interface{}) (*statemachine.ExecutionResult, error) {
	// Check for implementation
	if node.Class != "" {
		return e.executeDelegateClass(ctx, node.Class, variables)
	}
	if node.Expression != "" {
		return e.executeExpression(ctx, node.Expression, variables)
	}
	if node.DelegateExpression != "" {
		return e.executeDelegateExpression(ctx, node.DelegateExpression, variables, GetInstanceID(ctx), GetNodeID(ctx), node.GetName())
	}
	if node.Topic != "" {
		return e.executeExternalTask(ctx, node.Topic, variables)
	}

	// No implementation - skip to next node
	return &statemachine.ExecutionResult{
		Variables: variables,
	}, nil
}

// executeDelegateClass executes a delegate class
func (e *DefaultExecutor) executeDelegateClass(ctx context.Context, class string, variables map[string]interface{}) (*statemachine.ExecutionResult, error) {
	// TODO: Implement delegate class execution
	return &statemachine.ExecutionResult{
		Variables: variables,
	}, nil
}

// executeExpression executes an expression
func (e *DefaultExecutor) executeExpression(ctx context.Context, expression string, variables map[string]interface{}) (*statemachine.ExecutionResult, error) {
	evaluator := NewExpressionEvaluator(variables)
	_, err := evaluator.Evaluate(expression)
	if err != nil {
		e.logger.Error("Failed to evaluate expression", zap.String("expression", expression), zap.Error(err))
		return nil, fmt.Errorf("expression evaluation failed: %w", err)
	}

	// Store the evaluated result in variables for potential use
	if variables == nil {
		variables = make(map[string]interface{})
	}
	variables["_lastExpressionResult"] = true

	return &statemachine.ExecutionResult{
		Variables: variables,
	}, nil
}

// executeExternalTask executes an external task by publishing to NATS
func (e *DefaultExecutor) executeExternalTask(ctx context.Context, topic string, variables map[string]interface{}) (*statemachine.ExecutionResult, error) {
	instanceID := GetInstanceID(ctx)
	nodeID := GetNodeID(ctx)

	if e.natsPublisher == nil {
		return nil, fmt.Errorf("NATS publisher not configured for external task execution")
	}

	// Create a token for this execution
	token := &statemachine.Token{
		ID:         uuid.New().String(),
		InstanceID: instanceID,
		NodeID:     nodeID,
		Status:     statemachine.TokenStatusWaiting,
		Variables:  variables,
	}

	// Publish external task command to NATS
	cmd := &nats.WorkflowCommand{
		CommandType:    nats.CommandTypeServiceTask,
		InstanceID:     instanceID,
		TokenID:        token.ID,
		NodeID:         nodeID,
		ServiceName:    topic,
		Operation:      "execute",
		InputVariables: variables,
		MaxRetries:     3,
	}

	if err := e.natsPublisher.PublishCommand(ctx, cmd); err != nil {
		e.logger.Error("Failed to publish external task",
			zap.String("topic", topic),
			zap.String("instanceID", instanceID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to publish external task: %w", err)
	}

	e.logger.Info("External task published",
		zap.String("topic", topic),
		zap.String("instanceID", instanceID),
		zap.String("tokenID", token.ID))

	// Return await result - external task waits for response
	return &statemachine.ExecutionResult{
		Variables: variables,
		Await:     true,
		AwaitType: "external_task",
	}, nil
}

// executeScriptTask executes a script task
func (e *DefaultExecutor) executeScriptTask(ctx context.Context, node *bpmn.ScriptTask, variables map[string]interface{}) (*statemachine.ExecutionResult, error) {
	// TODO: Implement script execution based on ScriptFormat
	return &statemachine.ExecutionResult{
		Variables: variables,
	}, nil
}

// executeManualTask executes a manual task (passes through)
func (e *DefaultExecutor) executeManualTask(ctx context.Context, node *bpmn.ManualTask, variables map[string]interface{}) (*statemachine.ExecutionResult, error) {
	return &statemachine.ExecutionResult{
		Variables: variables,
	}, nil
}

// executeReceiveTask executes a receive task (waits for message)
func (e *DefaultExecutor) executeReceiveTask(ctx context.Context, node *bpmn.ReceiveTask, variables map[string]interface{}) (*statemachine.ExecutionResult, error) {
	// ReceiveTask waits for a message to arrive
	// It should have a messageRef that identifies the message to wait for
	return &statemachine.ExecutionResult{
		Variables:      variables,
		Await:          true,
		AwaitType:      "message",
		MessageRef:     node.MessageRef,
		CorrelationKey: "businessKey",
	}, nil
}

// executeSendTask executes a send task (sends message)
func (e *DefaultExecutor) executeSendTask(ctx context.Context, node *bpmn.SendTask, variables map[string]interface{}) (*statemachine.ExecutionResult, error) {
	// SendTask sends a message to another participant
	// The message is sent via messageRef and can be correlated by business key

	instanceID := GetInstanceID(ctx)
	businessKey := ""
	if bk, ok := variables["businessKey"]; ok {
		if bkStr, ok := bk.(string); ok {
			businessKey = bkStr
		}
	}

	// Publish message command to NATS for cross-process communication
	if e.natsPublisher != nil && node.MessageRef != "" {
		cmd := &nats.WorkflowCommand{
			CommandType:    nats.CommandTypeMessage,
			InstanceID:     instanceID,
			NodeID:         node.MessageRef,
			Operation:      businessKey,
			InputVariables: variables,
			MaxRetries:     3,
		}

		if err := e.natsPublisher.PublishCommand(ctx, cmd); err != nil {
			e.logger.Error("Failed to send message", zap.Error(err), zap.String("messageRef", node.MessageRef))
			return nil, fmt.Errorf("failed to send message: %w", err)
		}
		e.logger.Info("Message sent", zap.String("messageRef", node.MessageRef), zap.String("instanceID", instanceID))
	}

	// Move to next node
	nextNodeID := ""
	if len(node.Outgoing) > 0 {
		nextNodeID = node.Outgoing[0]
	}

	return &statemachine.ExecutionResult{
		Variables:  variables,
		NextNodeID: nextNodeID,
	}, nil
}

// executeExclusiveGateway executes an exclusive gateway
func (e *DefaultExecutor) executeExclusiveGateway(ctx context.Context, graph *bpmn.Graph, node *bpmn.ExclusiveGateway, variables map[string]interface{}) (*statemachine.ExecutionResult, error) {
	// Find outgoing flows and evaluate conditions
	outgoingFlows := bpmn.GetOutgoingFlows(&bpmn.Process{
		SequenceFlow: graph.SequenceFlows,
	}, node.ID)

	for _, flow := range outgoingFlows {
		if flow.ConditionExpression != nil && flow.ConditionExpression.Text != "" {
			// Evaluate condition - for now use default flow or first
			// TODO: Implement condition evaluation
			continue
		}
		// Default flow (no condition)
		return &statemachine.ExecutionResult{
			NextNodeID: flow.TargetRef,
			Variables:  variables,
		}, nil
	}

	// No default, return first
	if len(outgoingFlows) > 0 {
		return &statemachine.ExecutionResult{
			NextNodeID: outgoingFlows[0].TargetRef,
			Variables:  variables,
		}, nil
	}

	return nil, fmt.Errorf("exclusive gateway %s has no outgoing flows", node.ID)
}

// executeInclusiveGateway executes an inclusive gateway
func (e *DefaultExecutor) executeInclusiveGateway(ctx context.Context, graph *bpmn.Graph, node *bpmn.InclusiveGateway, variables map[string]interface{}) (*statemachine.ExecutionResult, error) {
	// Inclusive gateway evaluates all conditions and takes all true paths
	// This may result in multiple tokens
	outgoingFlows := bpmn.GetOutgoingFlows(&bpmn.Process{
		SequenceFlow: graph.SequenceFlows,
	}, node.ID)

	// For now, just take first valid flow
	for _, flow := range outgoingFlows {
		return &statemachine.ExecutionResult{
			NextNodeID: flow.TargetRef,
			Variables:  variables,
		}, nil
	}

	return nil, fmt.Errorf("inclusive gateway %s has no outgoing flows", node.ID)
}

// executeParallelGateway executes a parallel gateway
func (e *DefaultExecutor) executeParallelGateway(ctx context.Context, graph *bpmn.Graph, node *bpmn.ParallelGateway, variables map[string]interface{}) (*statemachine.ExecutionResult, error) {
	// Parallel gateway - diverging: activates all outgoing flows
	// Converging: waits for all incoming tokens before proceeding
	outgoingFlows := bpmn.GetOutgoingFlows(&bpmn.Process{
		SequenceFlow: graph.SequenceFlows,
	}, node.ID)

	if len(outgoingFlows) > 0 {
		return &statemachine.ExecutionResult{
			NextNodeID: outgoingFlows[0].TargetRef,
			Variables:  variables,
		}, nil
	}

	return nil, fmt.Errorf("parallel gateway %s has no outgoing flows", node.ID)
}

// executeIntermediateCatchEvent executes an intermediate catch event
func (e *DefaultExecutor) executeIntermediateCatchEvent(ctx context.Context, node *bpmn.IntermediateCatchEvent, variables map[string]interface{}) (*statemachine.ExecutionResult, error) {
	// Intermediate catch events wait for external events
	return &statemachine.ExecutionResult{
		Variables: variables,
		Await:     true,
		AwaitType: "event",
	}, nil
}

// executeIntermediateThrowEvent executes an intermediate throw event
func (e *DefaultExecutor) executeIntermediateThrowEvent(ctx context.Context, node *bpmn.IntermediateThrowEvent, variables map[string]interface{}) (*statemachine.ExecutionResult, error) {
	// Intermediate throw events just pass through
	return &statemachine.ExecutionResult{
		Variables: variables,
	}, nil
}

// executeBoundaryEvent executes a boundary event
func (e *DefaultExecutor) executeBoundaryEvent(ctx context.Context, graph *bpmn.Graph, node *bpmn.BoundaryEvent, variables map[string]interface{}) (*statemachine.ExecutionResult, error) {
	// Boundary events are triggered by timer or error, and transition to their outgoing flow
	if len(node.Outgoing) == 0 {
		return nil, fmt.Errorf("boundary event %s has no outgoing flows", node.ID)
	}

	return &statemachine.ExecutionResult{
		NextNodeID: node.Outgoing[0],
		Variables:  variables,
	}, nil
}

// GetBoundaryEventsForActivity returns boundary events attached to an activity
func (e *DefaultExecutor) GetBoundaryEventsForActivity(graph *bpmn.Graph, activityID string) []*bpmn.BoundaryEvent {
	return graph.GetBoundaryEventsForActivity(activityID)
}

// GetTimerBoundaryEventsForActivity returns timer boundary events attached to an activity
func (e *DefaultExecutor) GetTimerBoundaryEventsForActivity(graph *bpmn.Graph, activityID string) []*bpmn.BoundaryEvent {
	return graph.GetTimerBoundaryEventsForActivity(activityID)
}

// GetErrorBoundaryEventsForActivity returns error boundary events attached to an activity
func (e *DefaultExecutor) GetErrorBoundaryEventsForActivity(graph *bpmn.Graph, activityID string) []*bpmn.BoundaryEvent {
	return graph.GetErrorBoundaryEventsForActivity(activityID)
}

// HasInterruptingBoundaryEvent checks if an activity has an interrupting boundary event
func (e *DefaultExecutor) HasInterruptingBoundaryEvent(graph *bpmn.Graph, activityID string) bool {
	return graph.HasInterruptingBoundaryEvent(activityID)
}

// ParseTimerDuration parses ISO 8601 duration string and returns duration
func ParseTimerDuration(durationStr string) (time.Duration, error) {
	// Handle ISO 8601 duration format (e.g., PT5M, PT1H30M, PT10S)
	return time.ParseDuration(durationStr)
}

// GetTimerDurationForBoundaryEvent calculates the duration for a timer boundary event
func GetTimerDurationForBoundaryEvent(be *bpmn.BoundaryEvent) (time.Duration, error) {
	if be.TimerEventDefinition == nil {
		return 0, fmt.Errorf("boundary event has no timer definition")
	}

	if be.TimerEventDefinition.TimeDuration != "" {
		return ParseTimerDuration(be.TimerEventDefinition.TimeDuration)
	}

	if be.TimerEventDefinition.TimeCycle != "" {
		// For cycles, return the first occurrence duration
		// Format: R3/PT10M means repeat 3 times every 10 minutes
		// We'll parse just the duration part
		durationStr := be.TimerEventDefinition.TimeCycle
		if idx := strings.Index(durationStr, "/"); idx != -1 {
			durationStr = durationStr[idx+1:]
		}
		return ParseTimerDuration(durationStr)
	}

	if be.TimerEventDefinition.TimeDate != "" {
		// For specific date, calculate time until that date
		timeDate, err := time.Parse(time.RFC3339, be.TimerEventDefinition.TimeDate)
		if err != nil {
			return 0, fmt.Errorf("invalid timeDate format: %w", err)
		}
		return timeDate.Sub(time.Now()), nil
	}

	return 0, fmt.Errorf("timer boundary event has no valid time definition")
}

// RegisterTaskHandler registers a custom task handler
func (e *DefaultExecutor) RegisterTaskHandler(taskType string, handler TaskHandler) {
	e.taskHandlers[taskType] = handler
}

// GetTaskHandler gets a registered task handler
func (e *DefaultExecutor) GetTaskHandler(taskType string) (TaskHandler, bool) {
	handler, exists := e.taskHandlers[taskType]
	return handler, exists
}
