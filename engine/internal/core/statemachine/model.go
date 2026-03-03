package statemachine

import (
	"time"

	"github.com/google/uuid"
)

// TokenStatus represents the status of a token
type TokenStatus string

const (
	// TokenStatusPending - token created but not started
	TokenStatusPending TokenStatus = "pending"

	// TokenStatusActive - token is currently executing
	TokenStatusActive TokenStatus = "active"

	// TokenStatusWaiting - token is waiting for external trigger (user task, event)
	TokenStatusWaiting TokenStatus = "waiting"

	// TokenStatusCompleted - token completed successfully
	TokenStatusCompleted TokenStatus = "completed"

	// TokenStatusFailed - token failed due to error
	TokenStatusFailed TokenStatus = "failed"

	// TokenStatusTerminated - token was terminated
	TokenStatusTerminated TokenStatus = "terminated"

	// TokenStatusSuspended - token is suspended
	TokenStatusSuspended TokenStatus = "suspended"
)

// Token represents a token in the process execution
type Token struct {
	// ID is the unique identifier of the token
	ID string `json:"id"`

	// InstanceID is the process instance this token belongs to
	InstanceID string `json:"instance_id"`

	// NodeID is the current BPMN element ID
	NodeID string `json:"node_id"`

	// Status is the current status of the token
	Status TokenStatus `json:"status"`

	// Variables holds the process variables for this token
	Variables map[string]interface{} `json:"variables"`

	// CreatedAt is when the token was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the token was last updated
	UpdatedAt time.Time `json:"updated_at"`

	// StartedAt is when the token started execution
	StartedAt *time.Time `json:"started_at,omitempty"`

	// CompletedAt is when the token completed
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// Error holds error information if the token failed
	Error *TokenError `json:"error,omitempty"`

	// History holds execution history
	History []TokenHistory `json:"history"`
}

// TokenError holds error information
type TokenError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Stack   string `json:"stack,omitempty"`
}

// TokenHistory holds a history entry for the token
type TokenHistory struct {
	FromStatus TokenStatus `json:"from_status"`
	ToStatus   TokenStatus `json:"to_status"`
	Timestamp  time.Time   `json:"timestamp"`
	Action     string      `json:"action"`
}

// ExecutionResult holds the result of a node execution
type ExecutionResult struct {
	// NextNodeID is the ID of the next node to execute
	NextNodeID string `json:"next_node_id,omitempty"`

	// Variables are the updated process variables
	Variables map[string]interface{} `json:"variables"`

	// Error is the error that occurred during execution
	Error *ExecutionError `json:"error,omitempty"`

	// Await indicates the token should wait for external trigger
	Await bool `json:"await"`

	// AwaitType describes what type of await (user_task, timer, event)
	AwaitType string `json:"await_type,omitempty"`

	// Suspended indicates execution should be suspended
	Suspended bool `json:"suspended"`

	// Terminated indicates the process should be terminated
	Terminated bool `json:"terminated"`
}

// ExecutionError holds execution error information
type ExecutionError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	NodeID  string `json:"node_id"`
}

// ProcessInstanceStatus represents the status of a process instance
type ProcessInstanceStatus string

const (
	// ProcessInstanceStatusPending - instance created but not started
	ProcessInstanceStatusPending ProcessInstanceStatus = "pending"

	// ProcessInstanceStatusRunning - instance is running
	ProcessInstanceStatusRunning ProcessInstanceStatus = "running"

	// ProcessInstanceStatusCompleted - instance completed successfully
	ProcessInstanceStatusCompleted ProcessInstanceStatus = "completed"

	// ProcessInstanceStatusFailed - instance failed
	ProcessInstanceStatusFailed ProcessInstanceStatus = "failed"

	// ProcessInstanceStatusTerminated - instance was terminated
	ProcessInstanceStatusTerminated ProcessInstanceStatus = "terminated"

	// ProcessInstanceStatusSuspended - instance is suspended
	ProcessInstanceStatusSuspended ProcessInstanceStatus = "suspended"
)

// ProcessInstance represents a running process instance
type ProcessInstance struct {
	// ID is the unique identifier
	ID string `json:"id"`

	// ProcessKey is the key of the process definition
	ProcessKey string `json:"process_key"`

	// ProcessVersion is the version of the process
	ProcessVersion string `json:"process_version,omitempty"`

	// Status is the current status
	Status ProcessInstanceStatus `json:"status"`

	// Variables holds the process variables
	Variables map[string]interface{} `json:"variables"`

	// Tokens holds all active tokens
	Tokens map[string]*Token `json:"tokens"`

	// LastActivityAt is the last activity timestamp
	LastActivityAt time.Time `json:"last_activity_at"`

	// CreatedAt is when the instance was created
	CreatedAt time.Time `json:"created_at"`

	// StartedAt is when execution started
	StartedAt *time.Time `json:"started_at,omitempty"`

	// CompletedAt is when execution completed
	CompletedAt *time.Time `json:"completed_at,omitempty"`

	// CompletedBy is who completed the instance
	CompletedBy string `json:"completed_by,omitempty"`

	// Error holds error information if failed
	Error *TokenError `json:"error,omitempty"`
}

// NewToken creates a new token
func NewToken(instanceID, nodeID string, variables map[string]interface{}) *Token {
	now := time.Now()
	return &Token{
		ID:         uuid.New().String(),
		InstanceID: instanceID,
		NodeID:     nodeID,
		Status:     TokenStatusPending,
		Variables:  variables,
		CreatedAt:  now,
		UpdatedAt:  now,
		History:    []TokenHistory{},
	}
}

// NewProcessInstance creates a new process instance
func NewProcessInstance(processKey string, variables map[string]interface{}) *ProcessInstance {
	now := time.Now()
	return &ProcessInstance{
		ID:         uuid.New().String(),
		ProcessKey: processKey,
		Status:     ProcessInstanceStatusPending,
		Variables:  variables,
		Tokens:     make(map[string]*Token),
		CreatedAt:  now,
	}
}
