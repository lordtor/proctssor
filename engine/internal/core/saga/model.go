package saga

import (
	"time"

	"github.com/google/uuid"
)

// SagaStatus represents the status of a saga
type SagaStatus string

const (
	// SagaStatusPending - saga created but not started
	SagaStatusPending SagaStatus = "pending"

	// SagaStatusRunning - saga is currently executing
	SagaStatusRunning SagaStatus = "running"

	// SagaStatusCompensating - saga is compensating (rolling back)
	SagaStatusCompensating SagaStatus = "compensating"

	// SagaStatusCompleted - saga completed successfully
	SagaStatusCompleted SagaStatus = "completed"

	// SagaStatusFailed - saga failed and could not be compensated
	SagaStatusFailed SagaStatus = "failed"

	// SagaStatusCompensated - saga failed but was successfully compensated
	SagaStatusCompensated SagaStatus = "compensated"
)

// SagaStepStatus represents the status of a saga step
type SagaStepStatus string

const (
	// SagaStepStatusPending - step not yet executed
	SagaStepStatusPending SagaStepStatus = "pending"

	// SagaStepStatusCompleted - step completed successfully
	SagaStepStatusCompleted SagaStepStatus = "completed"

	// SagaStepStatusCompensated - step was compensated
	SagaStepStatusCompensated SagaStepStatus = "compensated"

	// SagaStepStatusFailed - step failed
	SagaStepStatusFailed SagaStepStatus = "failed"
)

// SagaStep represents a single step in a saga
type SagaStep struct {
	// ID is the unique identifier of the step
	ID string `json:"id"`

	// NodeID is the BPMN node ID this step corresponds to
	NodeID string `json:"node_id"`

	// NodeName is the human-readable name of the node
	NodeName string `json:"node_name"`

	// Status is the current status of the step
	Status SagaStepStatus `json:"status"`

	// CompensateNodeID is the node to execute for compensation
	CompensateNodeID string `json:"compensate_node_id,omitempty"`

	// CompensateNodeName is the human-readable name of the compensation node
	CompensateNodeName string `json:"compensate_node_name,omitempty"`

	// InputVariables are the variables passed to the step
	InputVariables map[string]interface{} `json:"input_variables"`

	// OutputVariables are the variables returned by the step
	OutputVariables map[string]interface{} `json:"output_variables"`

	// CompensateInputVariables are the variables to use for compensation
	CompensateInputVariables map[string]interface{} `json:"compensate_input_variables"`

	// Error holds error information if the step failed
	Error *SagaError `json:"error,omitempty"`

	// ExecutedAt is when the step was executed
	ExecutedAt *time.Time `json:"executed_at,omitempty"`

	// CompensatedAt is when the step was compensated
	CompensatedAt *time.Time `json:"compensated_at,omitempty"`
}

// SagaError holds error information for saga steps
type SagaError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	NodeID  string `json:"node_id"`
}

// Saga represents a saga orchestrator for compensating transactions
type Saga struct {
	// ID is the unique identifier of the saga
	ID string `json:"id"`

	// InstanceID is the process instance this saga belongs to
	InstanceID string `json:"instance_id"`

	// ProcessKey is the key of the process definition
	ProcessKey string `json:"process_key"`

	// Status is the current status of the saga
	Status SagaStatus `json:"status"`

	// Steps holds all steps in the saga
	Steps []SagaStep `json:"steps"`

	// CurrentStepIndex is the index of the currently executing step
	CurrentStepIndex int `json:"current_step_index"`

	// Variables holds the saga-level variables
	Variables map[string]interface{} `json:"variables"`

	// Error holds error information if the saga failed
	Error *SagaError `json:"error,omitempty"`

	// CreatedAt is when the saga was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the saga was last updated
	UpdatedAt time.Time `json:"updated_at"`

	// CompletedAt is when the saga completed
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// NewSaga creates a new saga
func NewSaga(instanceID, processKey string, variables map[string]interface{}) *Saga {
	now := time.Now()
	return &Saga{
		ID:               uuid.New().String(),
		InstanceID:       instanceID,
		ProcessKey:       processKey,
		Status:           SagaStatusPending,
		Steps:            []SagaStep{},
		CurrentStepIndex: -1,
		Variables:        variables,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

// AddStep adds a step to the saga
func (s *Saga) AddStep(nodeID, nodeName, compensateNodeID, compensateNodeName string, inputVariables map[string]interface{}) {
	step := SagaStep{
		ID:                       uuid.New().String(),
		NodeID:                   nodeID,
		NodeName:                 nodeName,
		Status:                   SagaStepStatusPending,
		CompensateNodeID:         compensateNodeID,
		CompensateNodeName:       compensateNodeName,
		InputVariables:           inputVariables,
		OutputVariables:          make(map[string]interface{}),
		CompensateInputVariables: make(map[string]interface{}),
	}
	s.Steps = append(s.Steps, step)
}

// GetCompletedSteps returns all completed steps in reverse order (for compensation)
func (s *Saga) GetCompletedSteps() []SagaStep {
	var completed []SagaStep
	for i := len(s.Steps) - 1; i >= 0; i-- {
		if s.Steps[i].Status == SagaStepStatusCompleted {
			completed = append(completed, s.Steps[i])
		}
	}
	return completed
}

// GetPendingSteps returns all pending steps
func (s *Saga) GetPendingSteps() []SagaStep {
	var pending []SagaStep
	for i := s.CurrentStepIndex + 1; i < len(s.Steps); i++ {
		if s.Steps[i].Status == SagaStepStatusPending {
			pending = append(pending, s.Steps[i])
		}
	}
	return pending
}

// HasCompensation returns true if the saga can be compensated
func (s *Saga) HasCompensation() bool {
	for _, step := range s.Steps {
		if step.CompensateNodeID != "" && step.Status == SagaStepStatusCompleted {
			return true
		}
	}
	return false
}

// SagaHistoryEntry represents a history entry for the saga
type SagaHistoryEntry struct {
	// StepID is the ID of the step this entry relates to
	StepID string `json:"step_id"`

	// NodeID is the BPMN node ID
	NodeID string `json:"node_id"`

	// Action is the action performed (executed, compensated, failed)
	Action string `json:"action"`

	// Timestamp is when the action was performed
	Timestamp time.Time `json:"timestamp"`

	// Variables are the variables at the time of the action
	Variables map[string]interface{} `json:"variables,omitempty"`

	// Error is the error if the action failed
	Error *SagaError `json:"error,omitempty"`
}
