package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/workflow-engine/v2/internal/core/statemachine"
)

// CommandType represents the type of command to publish
type CommandType string

const (
	// CommandTypeServiceTask - execute a service task
	CommandTypeServiceTask CommandType = "service_task"
	// CommandTypeUserTask - create a user task
	CommandTypeUserTask CommandType = "user_task"
	// CommandTypeTimer - timer event triggered
	CommandTypeTimer CommandType = "timer"
	// CommandTypeSignal - signal event received
	CommandTypeSignal CommandType = "signal"
	// CommandTypeMessage - message event received
	CommandTypeMessage CommandType = "message"
)

// WorkflowCommand represents a command to be published to NATS
type WorkflowCommand struct {
	// CommandID is the unique identifier for this command
	CommandID string `json:"command_id"`

	// CommandType is the type of command
	CommandType CommandType `json:"command_type"`

	// InstanceID is the process instance ID
	InstanceID string `json:"instance_id"`

	// TokenID is the token ID this command relates to
	TokenID string `json:"token_id"`

	// NodeID is the BPMN element ID
	NodeID string `json:"node_id"`

	// ServiceName is the target service name
	ServiceName string `json:"service_name,omitempty"`

	// Operation is the operation to perform
	Operation string `json:"operation"`

	// InputVariables are the variables to pass to the service
	InputVariables map[string]interface{} `json:"input_variables"`

	// CreatedAt is when the command was created
	CreatedAt time.Time `json:"created_at"`

	// ExpiresAt is when the command should expire
	ExpiresAt time.Time `json:"expires_at"`

	// RetryCount is the current retry attempt
	RetryCount int `json:"retry_count"`

	// MaxRetries is the maximum number of retries
	MaxRetries int `json:"max_retries"`
}

// Publisher handles publishing commands to NATS
type Publisher struct {
	conn    *nats.Conn
	js      nats.JetStreamContext
	timeout time.Duration
}

// PublisherConfig holds configuration for the publisher
type PublisherConfig struct {
	URL     string
	Timeout time.Duration
}

// NewPublisher creates a new NATS publisher
func NewPublisher(cfg PublisherConfig) (*Publisher, error) {
	conn, err := nats.Connect(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to get JetStream context: %w", err)
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &Publisher{
		conn:    conn,
		js:      js,
		timeout: cfg.Timeout,
	}, nil
}

// PublishCommand publishes a command to NATS
func (p *Publisher) PublishCommand(ctx context.Context, cmd *WorkflowCommand) error {
	// Generate command ID if not set
	if cmd.CommandID == "" {
		cmd.CommandID = uuid.New().String()
	}

	// Set timestamps
	if cmd.CreatedAt.IsZero() {
		cmd.CreatedAt = time.Now()
	}
	if cmd.ExpiresAt.IsZero() {
		cmd.ExpiresAt = cmd.CreatedAt.Add(p.timeout)
	}

	// Marshal command to JSON
	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	// Determine subject based on command type
	subject := p.getSubject(cmd)

	// Publish with headers
	headers := nats.Header{
		"Command-ID":   []string{cmd.CommandID},
		"Instance-ID":  []string{cmd.InstanceID},
		"Token-ID":     []string{cmd.TokenID},
		"Command-Type": []string{string(cmd.CommandType)},
		"Node-ID":      []string{cmd.NodeID},
		"Expires-At":   []string{cmd.ExpiresAt.Format(time.RFC3339)},
		"Retry-Count":  []string{strconv.Itoa(cmd.RetryCount)},
	}
	if cmd.ServiceName != "" {
		headers["Service-Name"] = []string{cmd.ServiceName}
	}

	_, err = p.js.PublishMsg(&nats.Msg{
		Subject: subject,
		Data:    data,
		Header:  headers,
	})
	if err != nil {
		return fmt.Errorf("failed to publish command: %w", err)
	}

	return nil
}

// PublishServiceTask publishes a service task command
func (p *Publisher) PublishServiceTask(ctx context.Context, token *statemachine.Token, nodeID, serviceName, operation string) error {
	cmd := &WorkflowCommand{
		CommandType:    CommandTypeServiceTask,
		InstanceID:     token.InstanceID,
		TokenID:        token.ID,
		NodeID:         nodeID,
		ServiceName:    serviceName,
		Operation:      operation,
		InputVariables: token.Variables,
		MaxRetries:     3,
	}

	return p.PublishCommand(ctx, cmd)
}

// PublishUserTask publishes a user task command
func (p *Publisher) PublishUserTask(ctx context.Context, token *statemachine.Token, nodeID string) error {
	cmd := &WorkflowCommand{
		CommandType:    CommandTypeUserTask,
		InstanceID:     token.InstanceID,
		TokenID:        token.ID,
		NodeID:         nodeID,
		InputVariables: token.Variables,
		MaxRetries:     0, // User tasks don't retry
	}

	return p.PublishCommand(ctx, cmd)
}

// PublishTimerCommand publishes a timer trigger command
func (p *Publisher) PublishTimerCommand(ctx context.Context, instanceID, tokenID, nodeID string, variables map[string]interface{}) error {
	cmd := &WorkflowCommand{
		CommandType:    CommandTypeTimer,
		InstanceID:     instanceID,
		TokenID:        tokenID,
		NodeID:         nodeID,
		InputVariables: variables,
		MaxRetries:     3,
	}

	return p.PublishCommand(ctx, cmd)
}

// PublishSignalCommand publishes a signal event command
func (p *Publisher) PublishSignalCommand(ctx context.Context, instanceID, signalName string, variables map[string]interface{}) error {
	cmd := &WorkflowCommand{
		CommandType:    CommandTypeSignal,
		InstanceID:     instanceID,
		NodeID:         signalName,
		Operation:      signalName,
		InputVariables: variables,
		MaxRetries:     3,
	}

	return p.PublishCommand(ctx, cmd)
}

// PublishMessageCommand publishes a message event command
func (p *Publisher) PublishMessageCommand(ctx context.Context, instanceID, messageName string, correlationKey string, variables map[string]interface{}) error {
	cmd := &WorkflowCommand{
		CommandType:    CommandTypeMessage,
		InstanceID:     instanceID,
		NodeID:         messageName,
		Operation:      correlationKey,
		InputVariables: variables,
		MaxRetries:     3,
	}

	return p.PublishCommand(ctx, cmd)
}

// getSubject returns the NATS subject based on command type
func (p *Publisher) getSubject(cmd *WorkflowCommand) string {
	switch cmd.CommandType {
	case CommandTypeServiceTask:
		return fmt.Sprintf("wf.cmd.service.%s.%s", cmd.ServiceName, cmd.InstanceID)
	case CommandTypeUserTask:
		return fmt.Sprintf("wf.cmd.user.%s.%s", cmd.NodeID, cmd.InstanceID)
	case CommandTypeTimer:
		return fmt.Sprintf("wf.cmd.timer.%s", cmd.InstanceID)
	case CommandTypeSignal:
		return fmt.Sprintf("wf.cmd.signal.%s", cmd.InstanceID)
	case CommandTypeMessage:
		return fmt.Sprintf("wf.cmd.message.%s.%s", cmd.NodeID, cmd.InstanceID)
	default:
		return fmt.Sprintf("wf.cmd.%s", cmd.InstanceID)
	}
}

// Close closes the publisher connection
func (p *Publisher) Close() {
	if p.conn != nil {
		p.conn.Close()
	}
}

// IsConnected returns whether the publisher is connected
func (p *Publisher) IsConnected() bool {
	return p.conn != nil && p.conn.IsConnected()
}
