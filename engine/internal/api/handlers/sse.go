package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// SSEHandler handles Server-Sent Events
type SSEHandler struct {
	taskNotifier     *TaskNotifier
	registryNotifier *RegistryNotifier
}

// TaskNotifier notifies about task updates
type TaskNotifier struct {
	clients map[string]map[chan []byte]bool
}

// NewTaskNotifier creates a new task notifier
func NewTaskNotifier() *TaskNotifier {
	return &TaskNotifier{
		clients: make(map[string]map[chan []byte]bool),
	}
}

// Register registers a client for task notifications
func (t *TaskNotifier) Register(assignee string, ch chan []byte) {
	if t.clients[assignee] == nil {
		t.clients[assignee] = make(map[chan []byte]bool)
	}
	t.clients[assignee][ch] = true
}

// Unregister removes a client
func (t *TaskNotifier) Unregister(assignee string, ch chan []byte) {
	if t.clients[assignee] != nil {
		delete(t.clients[assignee], ch)
	}
}

// Notify notifies all clients for an assignee
func (t *TaskNotifier) Notify(assignee string, data interface{}) {
	if t.clients[assignee] == nil {
		return
	}

	dataBytes, _ := json.Marshal(data)
	for ch := range t.clients[assignee] {
		select {
		case ch <- dataBytes:
		default:
		}
	}
}

// RegistryNotifier notifies about registry updates
type RegistryNotifier struct {
	clients map[chan []byte]bool
}

// NewRegistryNotifier creates a new registry notifier
func NewRegistryNotifier() *RegistryNotifier {
	return &RegistryNotifier{
		clients: make(map[chan []byte]bool),
	}
}

// Register registers a client
func (r *RegistryNotifier) Register(ch chan []byte) {
	r.clients[ch] = true
}

// Unregister removes a client
func (r *RegistryNotifier) Unregister(ch chan []byte) {
	delete(r.clients, ch)
}

// Notify notifies all clients
func (r *RegistryNotifier) Notify(data interface{}) {
	dataBytes, _ := json.Marshal(data)
	for ch := range r.clients {
		select {
		case ch <- dataBytes:
		default:
		}
	}
}

// NewSSEHandler creates a new SSE handler
func NewSSEHandler() *SSEHandler {
	return &SSEHandler{
		taskNotifier:     NewTaskNotifier(),
		registryNotifier: NewRegistryNotifier(),
	}
}

// TaskUpdate represents a task update event
type TaskUpdate struct {
	Type      string                 `json:"type"` // created, completed, assigned, unassigned
	TaskID    string                 `json:"task_id"`
	Assignee  string                 `json:"assignee,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// HandleTaskSSE handles SSE for task notifications
// @Summary Connect to task notifications
// @Description Get Server-Sent Events stream for task notifications
// @Tags sse
// @Produce text/event-stream
// @Param assignee query string true "User ID to filter tasks"
// @Success 200 {string} text/event-stream
// @Router /sse/tasks [get]
func (h *SSEHandler) HandleTaskSSE(c *gin.Context) {
	assignee := c.Query("assignee")
	if assignee == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "assignee parameter required"})
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Create channel for this client
	clientChan := make(chan []byte, 10)
	defer h.taskNotifier.Unregister(assignee, clientChan)
	h.taskNotifier.Register(assignee, clientChan)

	// Send initial connection message
	c.SSEvent("connected", gin.H{
		"message":  "Connected to task notifications",
		"assignee": assignee,
	})
	c.Writer.Flush()

	// Keep connection open and send messages
	notify := c.Request.Context().Done()
	for {
		select {
		case <-notify:
			return
		case data := <-clientChan:
			c.SSEvent("task_update", data)
			c.Writer.Flush()
		}
	}
}

// HandleRegistrySSE handles SSE for registry notifications
// @Summary Connect to registry notifications
// @Description Get Server-Sent Events stream for service registry updates
// @Tags sse
// @Produce text/event-stream
// @Success 200 {string} text/event-stream
// @Router /sse/registry [get]
func (h *SSEHandler) HandleRegistrySSE(c *gin.Context) {
	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Create channel for this client
	clientChan := make(chan []byte, 10)
	defer h.registryNotifier.Unregister(clientChan)
	h.registryNotifier.Register(clientChan)

	// Send initial connection message
	c.SSEvent("connected", gin.H{
		"message": "Connected to registry notifications",
	})
	c.Writer.Flush()

	// Keep connection open and send messages
	notify := c.Request.Context().Done()
	for {
		select {
		case <-notify:
			return
		case data := <-clientChan:
			c.SSEvent("registry_update", data)
			c.Writer.Flush()
		}
	}
}

// NotifyTaskCreated notifies about a new task
func (h *SSEHandler) NotifyTaskCreated(taskID, assignee string, data map[string]interface{}) {
	h.taskNotifier.Notify(assignee, TaskUpdate{
		Type:      "created",
		TaskID:    taskID,
		Assignee:  assignee,
		Timestamp: time.Now(),
		Data:      data,
	})
}

// NotifyTaskCompleted notifies about a completed task
func (h *SSEHandler) NotifyTaskCompleted(taskID, assignee string) {
	h.taskNotifier.Notify(assignee, TaskUpdate{
		Type:      "completed",
		TaskID:    taskID,
		Assignee:  assignee,
		Timestamp: time.Now(),
	})
}

// NotifyServiceRegistered notifies about a new service
func (h *SSEHandler) NotifyServiceRegistered(serviceName, serviceType string) {
	h.registryNotifier.Notify(gin.H{
		"type":         "registered",
		"name":         serviceName,
		"service_type": serviceType,
		"timestamp":    time.Now(),
	})
}

// NotifyServiceUnregistered notifies about an unregistered service
func (h *SSEHandler) NotifyServiceUnregistered(serviceName string) {
	h.registryNotifier.Notify(gin.H{
		"type":      "unregistered",
		"name":      serviceName,
		"timestamp": time.Now(),
	})
}

// Helper to send SSE
func sendSSEvent(c *gin.Context, event string, data interface{}) error {
	c.SSEvent(event, data)
	c.Writer.Flush()
	return nil
}

// Helper to create error response
func sseError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

// FormatSSE formats data for SSE
func FormatSSE(event string, data interface{}) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal data: %w", err)
	}
	return fmt.Sprintf("event: %s\ndata: %s\n\n", event, jsonData), nil
}
