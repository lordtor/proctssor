package saga

import (
	"context"
	"fmt"

	"github.com/workflow-engine/v2/internal/core/bpmn"
	"github.com/workflow-engine/v2/internal/core/statemachine"
	"go.uber.org/zap"
)

// ErrorHandler handles errors and triggers compensation
type ErrorHandler struct {
	orchestrator *SagaOrchestrator
	logger       *zap.Logger
}

// NewErrorHandler creates a new saga error handler
func NewErrorHandler(orchestrator *SagaOrchestrator, logger *zap.Logger) *ErrorHandler {
	return &ErrorHandler{
		orchestrator: orchestrator,
		logger:       logger,
	}
}

// HandleError handles an error during process execution and triggers compensation if needed
func (h *ErrorHandler) HandleError(ctx context.Context, saga *Saga, graph *bpmn.Graph, err error, nodeID string) error {
	h.logger.Error("Handling error in saga",
		zap.Error(err),
		zap.String("saga_id", saga.ID),
		zap.String("node_id", nodeID))

	// Determine if compensation should be triggered
	if h.shouldCompensate(saga, nodeID) {
		h.logger.Info("Triggering compensation for saga",
			zap.String("saga_id", saga.ID),
			zap.String("failed_node_id", nodeID))

		compErr := h.orchestrator.Compensate(ctx, saga)
		if compErr != nil {
			h.logger.Error("Compensation failed",
				zap.Error(compErr),
				zap.String("saga_id", saga.ID))
			return fmt.Errorf("error during execution: %w, compensation failed: %v", err, compErr)
		}

		return fmt.Errorf("error during execution: %w, compensation completed", err)
	}

	return fmt.Errorf("error during execution: %w", err)
}

// shouldCompensate determines if compensation should be triggered
func (h *ErrorHandler) shouldCompensate(saga *Saga, nodeID string) bool {
	// Only compensate if the saga has compensation capability
	if !saga.HasCompensation() {
		h.logger.Debug("Saga has no compensation capability",
			zap.String("saga_id", saga.ID))
		return false
	}

	// Only compensate if there are completed steps
	completedSteps := saga.GetCompletedSteps()
	if len(completedSteps) == 0 {
		h.logger.Debug("No completed steps to compensate",
			zap.String("saga_id", saga.ID))
		return false
	}

	return true
}

// HandleErrorWithResult handles error and returns execution result for state machine
func (h *ErrorHandler) HandleErrorWithResult(ctx context.Context, saga *Saga, graph *bpmn.Graph, err error, nodeID string) *statemachine.ExecutionResult {
	handleErr := h.HandleError(ctx, saga, graph, err, nodeID)

	return &statemachine.ExecutionResult{
		Error: &statemachine.ExecutionError{
			Code:    "SAGA_ERROR",
			Message: handleErr.Error(),
			NodeID:  nodeID,
		},
		Terminated: saga.Status == SagaStatusFailed,
	}
}

// CompensationExecutor implements the CompensationHandler interface
type CompensationExecutor struct {
	exec interface {
		ExecuteNode(ctx context.Context, graph *bpmn.Graph, currentNode bpmn.FlowElement, variables map[string]interface{}) (*statemachine.ExecutionResult, error)
	}
	logger *zap.Logger
}

// NewCompensationExecutor creates a new compensation executor
func NewCompensationExecutor(
	exec interface {
		ExecuteNode(ctx context.Context, graph *bpmn.Graph, currentNode bpmn.FlowElement, variables map[string]interface{}) (*statemachine.ExecutionResult, error)
	},
	logger *zap.Logger,
) *CompensationExecutor {
	return &CompensationExecutor{
		exec:   exec,
		logger: logger,
	}
}

// ExecuteCompensation executes a compensation action
func (e *CompensationExecutor) ExecuteCompensation(ctx context.Context, node bpmn.FlowElement, variables map[string]interface{}) (*statemachine.ExecutionResult, error) {
	e.logger.Info("Executing compensation",
		zap.String("node_id", node.GetID()),
		zap.String("node_name", node.GetName()))

	// For compensation, we execute the node with the compensation variables
	// In a real implementation, this would look up the compensation handler
	// from the registry based on the node configuration
	if e.exec == nil {
		return &statemachine.ExecutionResult{
			Variables: variables,
		}, nil
	}

	// Execute as a service task (the compensation node is typically a service task)
	result, err := e.exec.ExecuteNode(ctx, nil, node, variables)
	if err != nil {
		e.logger.Error("Compensation execution failed",
			zap.Error(err),
			zap.String("node_id", node.GetID()))
		return nil, err
	}

	e.logger.Info("Compensation executed successfully",
		zap.String("node_id", node.GetID()))

	return result, nil
}

// SagaErrorHandler is a wrapper that integrates saga compensation with executor
type SagaErrorHandler struct {
	orchestrator *SagaOrchestrator
	executor     interface {
		ExecuteNode(ctx context.Context, graph *bpmn.Graph, currentNode bpmn.FlowElement, variables map[string]interface{}) (*statemachine.ExecutionResult, error)
	}
	logger *zap.Logger
}

// NewSagaErrorHandler creates a new saga error handler
func NewSagaErrorHandler(
	orchestrator *SagaOrchestrator,
	executor interface {
		ExecuteNode(ctx context.Context, graph *bpmn.Graph, currentNode bpmn.FlowElement, variables map[string]interface{}) (*statemachine.ExecutionResult, error)
	},
	logger *zap.Logger,
) *SagaErrorHandler {
	return &SagaErrorHandler{
		orchestrator: orchestrator,
		executor:     executor,
		logger:       logger,
	}
}

// HandleServiceTaskError handles error in a service task and triggers compensation
func (h *SagaErrorHandler) HandleServiceTaskError(ctx context.Context, saga *Saga, graph *bpmn.Graph, node *bpmn.ServiceTask, err error) *statemachine.ExecutionResult {
	h.logger.Error("Service task error",
		zap.Error(err),
		zap.String("saga_id", saga.ID),
		zap.String("node_id", node.GetID()),
		zap.String("node_name", node.GetName()))

	// Check if this node is compensatable
	if node.IsCompensatable() {
		// Add compensation step
		compErr := h.orchestrator.AddStep(ctx, saga, node.GetID(), node.GetName(), node.GetCompensateNodeID(), "", map[string]interface{}{})
		if compErr != nil {
			h.logger.Error("Failed to add compensation step", zap.Error(compErr))
		}
	}

	// Trigger compensation for completed steps
	if saga.HasCompensation() {
		compErr := h.orchestrator.Compensate(ctx, saga)
		if compErr != nil {
			h.logger.Error("Compensation failed", zap.Error(compErr))
			return &statemachine.ExecutionResult{
				Error: &statemachine.ExecutionError{
					Code:    "COMPENSATION_FAILED",
					Message: fmt.Sprintf("execution failed: %v, compensation failed: %v", err, compErr),
					NodeID:  node.GetID(),
				},
			}
		}

		return &statemachine.ExecutionResult{
			Error: &statemachine.ExecutionError{
				Code:    "COMPENSATED",
				Message: fmt.Sprintf("execution failed: %v, compensation completed", err),
				NodeID:  node.GetID(),
			},
		}
	}

	return &statemachine.ExecutionResult{
		Error: &statemachine.ExecutionError{
			Code:    "EXECUTION_FAILED",
			Message: err.Error(),
			NodeID:  node.GetID(),
		},
	}
}

// RegisterSagaHooks registers saga hooks with the executor for automatic error handling
func (h *SagaErrorHandler) RegisterSagaHooks() error {
	// This would integrate with the executor to automatically handle errors
	// For now, it's a placeholder for the integration pattern
	return nil
}
