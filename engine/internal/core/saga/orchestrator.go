package saga

import (
	"context"
	"fmt"
	"time"

	"github.com/workflow-engine/v2/internal/core/bpmn"
	"github.com/workflow-engine/v2/internal/core/statemachine"
	"go.uber.org/zap"
)

// ExecutorInterface defines the interface for executing BPMN nodes
type ExecutorInterface interface {
	ExecuteNode(ctx context.Context, graph *bpmn.Graph, currentNode bpmn.FlowElement, variables map[string]interface{}) (*statemachine.ExecutionResult, error)
}

// CompensationHandler handles compensation actions
type CompensationHandler interface {
	// ExecuteCompensation executes a compensation action
	ExecuteCompensation(ctx context.Context, node bpmn.FlowElement, variables map[string]interface{}) (*statemachine.ExecutionResult, error)
}

// SagaOrchestrator manages saga execution and compensation
type SagaOrchestrator struct {
	executor            ExecutorInterface
	compensationHandler CompensationHandler
	logger              *zap.Logger
	sagaRepository      SagaRepository
}

// SagaRepository defines the interface for saga persistence
type SagaRepository interface {
	// Save saves a saga
	Save(ctx context.Context, saga *Saga) error

	// Get gets a saga by ID
	Get(ctx context.Context, id string) (*Saga, error)

	// GetByInstanceID gets a saga by instance ID
	GetByInstanceID(ctx context.Context, instanceID string) (*Saga, error)

	// Update updates a saga
	Update(ctx context.Context, saga *Saga) error

	// Delete deletes a saga
	Delete(ctx context.Context, id string) error
}

// NewSagaOrchestrator creates a new saga orchestrator
func NewSagaOrchestrator(
	exec ExecutorInterface,
	compensationHandler CompensationHandler,
	logger *zap.Logger,
	sagaRepository SagaRepository,
) *SagaOrchestrator {
	return &SagaOrchestrator{
		executor:            exec,
		compensationHandler: compensationHandler,
		logger:              logger,
		sagaRepository:      sagaRepository,
	}
}

// StartSaga starts a new saga for a process instance
func (o *SagaOrchestrator) StartSaga(ctx context.Context, instanceID, processKey string, variables map[string]interface{}) (*Saga, error) {
	saga := NewSaga(instanceID, processKey, variables)
	saga.Status = SagaStatusRunning

	if o.sagaRepository != nil {
		if err := o.sagaRepository.Save(ctx, saga); err != nil {
			o.logger.Error("Failed to save saga", zap.Error(err), zap.String("saga_id", saga.ID))
			return nil, fmt.Errorf("failed to save saga: %w", err)
		}
	}

	o.logger.Info("Saga started", zap.String("saga_id", saga.ID), zap.String("instance_id", instanceID))
	return saga, nil
}

// AddStep adds a step to the saga
func (o *SagaOrchestrator) AddStep(ctx context.Context, saga *Saga, nodeID, nodeName, compensateNodeID, compensateNodeName string, inputVariables map[string]interface{}) error {
	saga.AddStep(nodeID, nodeName, compensateNodeID, compensateNodeName, inputVariables)
	saga.UpdatedAt = time.Now()

	if o.sagaRepository != nil {
		if err := o.sagaRepository.Update(ctx, saga); err != nil {
			o.logger.Error("Failed to update saga with step", zap.Error(err), zap.String("saga_id", saga.ID))
			return fmt.Errorf("failed to update saga: %w", err)
		}
	}

	o.logger.Debug("Step added to saga",
		zap.String("saga_id", saga.ID),
		zap.String("node_id", nodeID),
		zap.String("compensate_node_id", compensateNodeID))
	return nil
}

// ExecuteStep executes a step in the saga
func (o *SagaOrchestrator) ExecuteStep(ctx context.Context, saga *Saga, graph *bpmn.Graph, stepIndex int) (*SagaStep, error) {
	if stepIndex < 0 || stepIndex >= len(saga.Steps) {
		return nil, fmt.Errorf("invalid step index: %d", stepIndex)
	}

	step := &saga.Steps[stepIndex]
	saga.CurrentStepIndex = stepIndex

	// Get the node to execute
	node, exists := graph.GetElementByID(step.NodeID)
	if !exists {
		err := fmt.Errorf("node not found: %s", step.NodeID)
		step.Status = SagaStepStatusFailed
		step.Error = &SagaError{
			Code:    "NODE_NOT_FOUND",
			Message: err.Error(),
			NodeID:  step.NodeID,
		}
		return step, err
	}

	o.logger.Info("Executing saga step",
		zap.String("saga_id", saga.ID),
		zap.String("node_id", step.NodeID),
		zap.String("node_name", step.NodeName))

	// Execute the node
	result, err := o.executor.ExecuteNode(ctx, graph, node, step.InputVariables)
	if err != nil {
		step.Status = SagaStepStatusFailed
		step.Error = &SagaError{
			Code:    "EXECUTION_FAILED",
			Message: err.Error(),
			NodeID:  step.NodeID,
		}
		o.logger.Error("Step execution failed",
			zap.Error(err),
			zap.String("saga_id", saga.ID),
			zap.String("node_id", step.NodeID))

		// Trigger compensation
		if err := o.Compensate(ctx, saga); err != nil {
			o.logger.Error("Compensation failed", zap.Error(err), zap.String("saga_id", saga.ID))
			saga.Status = SagaStatusFailed
		}

		return step, err
	}

	// Update step with output
	step.Status = SagaStepStatusCompleted
	step.OutputVariables = result.Variables
	now := time.Now()
	step.ExecutedAt = &now

	// Store compensation input (output of current step becomes input for compensation)
	if step.CompensateInputVariables == nil {
		step.CompensateInputVariables = make(map[string]interface{})
	}
	for k, v := range result.Variables {
		step.CompensateInputVariables[k] = v
	}

	// Merge output variables into saga variables
	if saga.Variables == nil {
		saga.Variables = make(map[string]interface{})
	}
	for k, v := range result.Variables {
		saga.Variables[k] = v
	}

	saga.UpdatedAt = time.Now()

	if o.sagaRepository != nil {
		if err := o.sagaRepository.Update(ctx, saga); err != nil {
			o.logger.Error("Failed to update saga after step execution", zap.Error(err))
			return step, fmt.Errorf("failed to update saga: %w", err)
		}
	}

	o.logger.Info("Step executed successfully",
		zap.String("saga_id", saga.ID),
		zap.String("node_id", step.NodeID))

	return step, nil
}

// Compensate performs compensation for all completed steps
func (o *SagaOrchestrator) Compensate(ctx context.Context, saga *Saga) error {
	o.logger.Info("Starting compensation for saga",
		zap.String("saga_id", saga.ID),
		zap.String("instance_id", saga.InstanceID))

	saga.Status = SagaStatusCompensating

	completedSteps := saga.GetCompletedSteps()

	if len(completedSteps) == 0 {
		o.logger.Info("No completed steps to compensate", zap.String("saga_id", saga.ID))
		saga.Status = SagaStatusCompensated
		return nil
	}

	o.logger.Info("Compensating saga steps",
		zap.String("saga_id", saga.ID),
		zap.Int("steps_to_compensate", len(completedSteps)))

	// Process compensation in reverse order
	for _, step := range completedSteps {
		if step.CompensateNodeID == "" {
			o.logger.Warn("Skipping compensation for step without compensation node",
				zap.String("saga_id", saga.ID),
				zap.String("node_id", step.NodeID))
			continue
		}

		if err := o.compensateStep(ctx, saga, &step); err != nil {
			o.logger.Error("Compensation failed for step",
				zap.Error(err),
				zap.String("saga_id", saga.ID),
				zap.String("node_id", step.NodeID))
			// Continue compensating other steps
		}
	}

	saga.UpdatedAt = time.Now()

	// Check if all steps were successfully compensated
	allCompensated := true
	for i := range saga.Steps {
		if saga.Steps[i].Status == SagaStepStatusCompleted {
			saga.Steps[i].Status = SagaStepStatusCompensated
			now := time.Now()
			saga.Steps[i].CompensatedAt = &now
		} else if saga.Steps[i].Status == SagaStepStatusFailed {
			allCompensated = false
		}
	}

	if allCompensated {
		saga.Status = SagaStatusCompensated
		now := time.Now()
		saga.CompletedAt = &now
		o.logger.Info("Saga compensation completed successfully", zap.String("saga_id", saga.ID))
	} else {
		saga.Status = SagaStatusFailed
		o.logger.Error("Saga compensation completed with errors", zap.String("saga_id", saga.ID))
	}

	if o.sagaRepository != nil {
		if err := o.sagaRepository.Update(ctx, saga); err != nil {
			o.logger.Error("Failed to update saga after compensation", zap.Error(err))
			return fmt.Errorf("failed to update saga: %w", err)
		}
	}

	return nil
}

// compensateStep compensates a single step
func (o *SagaOrchestrator) compensateStep(ctx context.Context, saga *Saga, step *SagaStep) error {
	o.logger.Info("Compensating step",
		zap.String("saga_id", saga.ID),
		zap.String("node_id", step.NodeID),
		zap.String("compensate_node_id", step.CompensateNodeID))

	// For compensation, we need to get the compensation node from the graph
	// In a real implementation, this would involve looking up the compensation event/subprocess
	// For now, we'll use the compensation handler if available
	if o.compensationHandler != nil {
		// Create a dummy node for compensation execution
		compensateNode := &bpmn.ServiceTask{
			Task: bpmn.Task{
				BaseElement: bpmn.BaseElement{
					ID:   step.CompensateNodeID,
					Name: step.CompensateNodeName,
				},
			},
		}

		result, err := o.compensationHandler.ExecuteCompensation(ctx, compensateNode, step.CompensateInputVariables)
		if err != nil {
			o.logger.Error("Compensation handler failed",
				zap.Error(err),
				zap.String("saga_id", saga.ID),
				zap.String("node_id", step.NodeID))
			return err
		}

		// Update step with compensation result
		step.CompensateInputVariables = result.Variables
	}

	now := time.Now()
	step.CompensatedAt = &now

	o.logger.Info("Step compensated successfully",
		zap.String("saga_id", saga.ID),
		zap.String("node_id", step.NodeID))

	return nil
}

// GetSaga gets a saga by ID
func (o *SagaOrchestrator) GetSaga(ctx context.Context, id string) (*Saga, error) {
	if o.sagaRepository == nil {
		return nil, fmt.Errorf("saga repository not configured")
	}
	return o.sagaRepository.Get(ctx, id)
}

// GetSagaByInstance gets a saga by instance ID
func (o *SagaOrchestrator) GetSagaByInstance(ctx context.Context, instanceID string) (*Saga, error) {
	if o.sagaRepository == nil {
		return nil, fmt.Errorf("saga repository not configured")
	}
	return o.sagaRepository.GetByInstanceID(ctx, instanceID)
}

// CompleteSaga marks a saga as completed
func (o *SagaOrchestrator) CompleteSaga(ctx context.Context, saga *Saga) error {
	saga.Status = SagaStatusCompleted
	now := time.Now()
	saga.CompletedAt = &now
	saga.UpdatedAt = now

	if o.sagaRepository != nil {
		if err := o.sagaRepository.Update(ctx, saga); err != nil {
			o.logger.Error("Failed to complete saga", zap.Error(err))
			return fmt.Errorf("failed to complete saga: %w", err)
		}
	}

	o.logger.Info("Saga completed", zap.String("saga_id", saga.ID))
	return nil
}

// FailSaga marks a saga as failed
func (o *SagaOrchestrator) FailSaga(ctx context.Context, saga *Saga, err error) {
	saga.Status = SagaStatusFailed
	saga.Error = &SagaError{
		Code:    "SAGA_FAILED",
		Message: err.Error(),
	}
	saga.UpdatedAt = time.Now()

	if o.sagaRepository != nil {
		if updateErr := o.sagaRepository.Update(ctx, saga); updateErr != nil {
			o.logger.Error("Failed to update failed saga", zap.Error(updateErr))
		}
	}

	o.logger.Error("Saga failed", zap.Error(err), zap.String("saga_id", saga.ID))
}

// DeleteSaga deletes a saga
func (o *SagaOrchestrator) DeleteSaga(ctx context.Context, id string) error {
	if o.sagaRepository == nil {
		return fmt.Errorf("saga repository not configured")
	}
	return o.sagaRepository.Delete(ctx, id)
}
