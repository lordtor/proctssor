package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/workflow-engine/v2/internal/core/statemachine"
)

// ResponseStatus represents the status of a service response
type ResponseStatus string

const (
	// ResponseStatusSuccess - successful response
	ResponseStatusSuccess ResponseStatus = "success"
	// ResponseStatusError - error response
	ResponseStatusError ResponseStatus = "error"
	// ResponseStatusTimeout - response timeout
	ResponseStatusTimeout ResponseStatus = "timeout"
)

// ServiceResponse represents a response from a service
type ServiceResponse struct {
	// CommandID is the ID of the original command
	CommandID string `json:"command_id"`

	// InstanceID is the process instance ID
	InstanceID string `json:"instance_id"`

	// TokenID is the token ID
	TokenID string `json:"token_id"`

	// NodeID is the BPMN element ID
	NodeID string `json:"node_id"`

	// Status is the response status
	Status ResponseStatus `json:"status"`

	// OutputVariables are the variables returned by the service
	OutputVariables map[string]interface{} `json:"output_variables,omitempty"`

	// ErrorMessage is the error message if failed
	ErrorMessage string `json:"error_message,omitempty"`

	// ErrorCode is the error code if failed
	ErrorCode string `json:"error_code,omitempty"`

	// ProcessedAt is when the response was processed
	ProcessedAt time.Time `json:"processed_at"`
}

// ResponseHandler handles incoming service responses
type ResponseHandler struct {
	conn           *nats.Conn
	js             nats.JetStreamContext
	mu             sync.RWMutex
	subscriptions  map[string]*nats.Subscription
	responseChan   chan *ServiceResponse
	errorHandler   func(*ServiceResponse) error
	successHandler func(*ServiceResponse) error
	dlqPublisher   *Publisher
	timeout        time.Duration
}

// ResponseHandlerConfig holds configuration for the response handler
type ResponseHandlerConfig struct {
	URL          string
	Timeout      time.Duration
	DLQPublisher *Publisher
}

// NewResponseHandler creates a new response handler
func NewResponseHandler(cfg ResponseHandlerConfig) (*ResponseHandler, error) {
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

	handler := &ResponseHandler{
		conn:          conn,
		js:            js,
		subscriptions: make(map[string]*nats.Subscription),
		responseChan:  make(chan *ServiceResponse, 100),
		dlqPublisher:  cfg.DLQPublisher,
		timeout:       cfg.Timeout,
	}

	return handler, nil
}

// SetHandlers sets the success and error handlers
func (h *ResponseHandler) SetHandlers(successHandler, errorHandler func(*ServiceResponse) error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.successHandler = successHandler
	h.errorHandler = errorHandler
}

// Start subscribing to response subjects
func (h *ResponseHandler) Start(ctx context.Context) error {
	// Subscribe to all responses
	subject := "wf.resp.>"

	sub, err := h.js.Subscribe(subject, h.handleMessage, nats.ManualAck(), nats.AckWait(h.timeout))
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", subject, err)
	}

	h.mu.Lock()
	h.subscriptions[subject] = sub
	h.mu.Unlock()

	// Start response processor
	go h.processResponses(ctx)

	return nil
}

// handleMessage handles incoming NATS messages
func (h *ResponseHandler) handleMessage(msg *nats.Msg) {
	var response ServiceResponse
	if err := json.Unmarshal(msg.Data, &response); err != nil {
		fmt.Printf("Failed to unmarshal response: %v\n", err)
		// Send to DLQ if available
		h.sendToDLQ(msg)
		_ = msg.Nak() // Send nak to retry
		return
	}

	response.ProcessedAt = time.Now()

	select {
	case h.responseChan <- &response:
		_ = msg.Ack()
	default:
		fmt.Printf("Response channel full, sending to DLQ\n")
		h.sendToDLQ(msg)
		_ = msg.Ack()
	}
}

// processResponses processes responses from the channel
func (h *ResponseHandler) processResponses(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case response := <-h.responseChan:
			h.processResponse(response)
		}
	}
}

// processResponse handles a single response
func (h *ResponseHandler) processResponse(response *ServiceResponse) {
	h.mu.RLock()
	successHandler := h.successHandler
	errorHandler := h.errorHandler
	h.mu.RUnlock()

	var err error

	switch response.Status {
	case ResponseStatusSuccess:
		if successHandler != nil {
			err = successHandler(response)
		}
	case ResponseStatusError, ResponseStatusTimeout:
		if errorHandler != nil {
			err = errorHandler(response)
		}
	default:
		fmt.Printf("Unknown response status: %s\n", response.Status)
	}

	if err != nil {
		fmt.Printf("Error processing response: %v\n", err)
	}
}

// sendToDLQ sends a message to the dead letter queue
func (h *ResponseHandler) sendToDLQ(msg *nats.Msg) {
	if h.dlqPublisher == nil {
		return
	}

	dlqSubject := fmt.Sprintf("wf.dlq.%s", msg.Subject)
	_, err := h.js.Publish(dlqSubject, msg.Data)
	if err != nil {
		fmt.Printf("Failed to publish to DLQ: %v\n", err)
	}
}

// SubscribeToInstance subscribes to responses for a specific instance
func (h *ResponseHandler) SubscribeToInstance(instanceID string) error {
	subject := fmt.Sprintf("wf.resp.%s.>", instanceID)

	h.mu.RLock()
	if _, exists := h.subscriptions[subject]; exists {
		h.mu.RUnlock()
		return nil
	}
	h.mu.RUnlock()

	sub, err := h.js.Subscribe(subject, h.handleMessage, nats.ManualAck(), nats.AckWait(h.timeout))
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", subject, err)
	}

	h.mu.Lock()
	h.subscriptions[subject] = sub
	h.mu.Unlock()

	return nil
}

// UnsubscribeFromInstance unsubscribes from responses for a specific instance
func (h *ResponseHandler) UnsubscribeFromInstance(instanceID string) error {
	subject := fmt.Sprintf("wf.resp.%s.>", instanceID)

	h.mu.Lock()
	sub, exists := h.subscriptions[subject]
	if !exists {
		h.mu.Unlock()
		return nil
	}
	delete(h.subscriptions, subject)
	h.mu.Unlock()

	return sub.Unsubscribe()
}

// Stop stops the response handler
func (h *ResponseHandler) Stop() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	for subject, sub := range h.subscriptions {
		if err := sub.Unsubscribe(); err != nil {
			fmt.Printf("Failed to unsubscribe from %s: %v\n", subject, err)
		}
	}
	h.subscriptions = make(map[string]*nats.Subscription)

	if h.conn != nil {
		h.conn.Close()
	}

	return nil
}

// CreateWorkflowResponse creates a service response from execution result
func CreateWorkflowResponse(token *statemachine.Token, result *statemachine.ExecutionResult) *ServiceResponse {
	response := &ServiceResponse{
		InstanceID:      token.InstanceID,
		TokenID:         token.ID,
		NodeID:          token.NodeID,
		OutputVariables: result.Variables,
		ProcessedAt:     time.Now(),
	}

	if result.Error != nil {
		response.Status = ResponseStatusError
		response.ErrorCode = result.Error.Code
		response.ErrorMessage = result.Error.Message
	} else {
		response.Status = ResponseStatusSuccess
	}

	return response
}
