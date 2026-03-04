package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

// Consumer handles receiving and processing commands from NATS
type Consumer struct {
	conn       *nats.Conn
	js         nats.JetStreamContext
	subject    string
	queueGroup string
	handler    CommandHandler
	responder  *Responder
	readyChan  chan bool
	errChan    chan error
}

// CommandHandler handles incoming workflow commands
type CommandHandler interface {
	HandleCommand(ctx context.Context, cmd *WorkflowCommand) (*ServiceResponse, error)
}

// Responder sends responses back to NATS
type Responder struct {
	js nats.JetStreamContext
}

// NewConsumer creates a new NATS consumer
func NewConsumer(url, subject, queueGroup string, handler CommandHandler) (*Consumer, error) {
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to get JetStream context: %w", err)
	}

	return &Consumer{
		conn:       conn,
		js:         js,
		subject:    subject,
		queueGroup: queueGroup,
		handler:    handler,
		responder:  &Responder{js: js},
		readyChan:  make(chan bool, 1),
		errChan:    make(chan error, 1),
	}, nil
}

// Start starts consuming messages
func (c *Consumer) Start(ctx context.Context) error {
	// Subscribe to the subject
	sub, err := c.js.QueueSubscribe(c.subject, c.queueGroup, c.handleMessage)
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	log.Printf("Subscribed to %s with queue group %s", c.subject, c.queueGroup)
	c.readyChan <- true

	// Wait for context cancellation
	<-ctx.Done()

	// Unsubscribe and close
	if err := sub.Unsubscribe(); err != nil {
		log.Printf("Warning: failed to unsubscribe: %v", err)
	}

	return nil
}

func (c *Consumer) handleMessage(msg *nats.Msg) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var cmd WorkflowCommand
	if err := json.Unmarshal(msg.Data, &cmd); err != nil {
		log.Printf("Failed to unmarshal command: %v", err)
		// Send error response
		c.sendErrorResponse(&cmd, "invalid_command", err.Error())
		msg.Ack()
		return
	}

	log.Printf("Received command: %s (type: %s, operation: %s)", cmd.CommandID, cmd.CommandType, cmd.Operation)

	// Handle the command
	response, err := c.handler.HandleCommand(ctx, &cmd)
	if err != nil {
		log.Printf("Failed to handle command: %v", err)
		c.sendErrorResponse(&cmd, "handler_error", err.Error())
		msg.Ack()
		return
	}

	// Send response back to engine
	if err := c.responder.SendResponse(ctx, response); err != nil {
		log.Printf("Failed to send response: %v", err)
	}

	msg.Ack()
}

func (c *Consumer) sendErrorResponse(cmd *WorkflowCommand, errorCode, errorMessage string) {
	response := &ServiceResponse{
		CommandID:    cmd.CommandID,
		InstanceID:   cmd.InstanceID,
		TokenID:      cmd.TokenID,
		NodeID:       cmd.NodeID,
		Status:       ResponseStatusError,
		ErrorCode:    errorCode,
		ErrorMessage: errorMessage,
		ProcessedAt:  time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := c.responder.SendResponse(ctx, response); err != nil {
		log.Printf("Failed to send error response: %v", err)
	}
}

// SendResponse sends a service response back to the engine
func (r *Responder) SendResponse(ctx context.Context, resp *ServiceResponse) error {
	// Subject for response: wf.resp.{instance_id}
	subject := fmt.Sprintf("wf.resp.%s", resp.InstanceID)

	data, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	_, err = r.js.Publish(subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish response: %w", err)
	}

	log.Printf("Sent response for command %s: status=%s", resp.CommandID, resp.Status)
	return nil
}

// Close closes the consumer connection
func (c *Consumer) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// IsConnected returns whether the consumer is connected
func (c *Consumer) IsConnected() bool {
	return c.conn != nil && c.conn.IsConnected()
}
