// Package events provides event types for inter-component communication
package events

import (
	"time"
)

// ProcessEvent represents a BPMN process event (for NATS)
type ProcessEvent struct {
	InstanceID  string                 `json:"instance_id"`
	ProcessID   string                 `json:"process_id"`
	NodeID      string                 `json:"node_id"`
	NodeName    string                 `json:"node_name"`
	EventType   string                 `json:"event_type"`   // "service_task.execute"
	ServiceName string                 `json:"service_name"` // "git-config"
	Action      string                 `json:"action"`       // "createGroup"
	Variables   map[string]interface{} `json:"variables"`
	Timestamp   time.Time              `json:"timestamp"`
	ReplyTo     string                 `json:"reply_to"` // "wf.resp.{instance_id}"
}

// ServiceResponse represents a response from a microservice
type ServiceResponse struct {
	InstanceID string                 `json:"instance_id"`
	NodeID     string                 `json:"node_id"`
	Success    bool                   `json:"success"`
	Result     map[string]interface{} `json:"result,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
}

// RegistryChange represents a registry change event (for NOTIFY)
type RegistryChange struct {
	Name   string `json:"name"`
	Action string `json:"action"` // "created", "updated", "deleted"
}

// InstanceUpdate represents an instance update event (for WebSocket)
type InstanceUpdate struct {
	Type       string                 `json:"type"` // "started", "completed", "waiting", "error"
	InstanceID string                 `json:"instance_id"`
	NodeID     string                 `json:"node_id,omitempty"`
	Status     string                 `json:"status"`
	Variables  map[string]interface{} `json:"variables,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
}

// TaskUpdate represents a task update event (for SSE)
type TaskUpdate struct {
	Type      string                 `json:"type"` // "created", "completed", "assigned"
	TaskID    string                 `json:"task_id"`
	Assignee  string                 `json:"assignee,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}
