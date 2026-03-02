package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"
)

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	// Registered clients by instance ID
	clients map[string]map[*Client]bool

	// Register requests from the clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Broadcast messages to specific instance
	broadcast chan *BroadcastMessage

	// Lock for clients map
	mu sync.RWMutex
}

// BroadcastMessage represents a message to broadcast
type BroadcastMessage struct {
	InstanceID string
	Message    []byte
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan *BroadcastMessage, 256),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.InstanceID] == nil {
				h.clients[client.InstanceID] = make(map[*Client]bool)
			}
			h.clients[client.InstanceID][client] = true
			h.mu.Unlock()
			log.Printf("Client registered for instance: %s", client.InstanceID)

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.InstanceID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)
					if len(clients) == 0 {
						delete(h.clients, client.InstanceID)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("Client unregistered for instance: %s", client.InstanceID)

		case message := <-h.broadcast:
			h.mu.RLock()
			clients := h.clients[message.InstanceID]
			h.mu.RUnlock()

			for client := range clients {
				select {
				case client.send <- message.Message:
				default:
					h.mu.Lock()
					close(client.send)
					delete(h.clients[message.InstanceID], client)
					h.mu.Unlock()
				}
			}
		}
	}
}

// RegisterClient registers a client to the hub
func (h *Hub) RegisterClient(client *Client) {
	h.register <- client
}

// UnregisterClient unregisters a client from the hub
func (h *Hub) UnregisterClient(client *Client) {
	h.unregister <- client
}

// BroadcastToInstance broadcasts a message to all clients subscribed to an instance
func (h *Hub) BroadcastToInstance(instanceID string, message interface{}) {
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling message: %v", err)
		return
	}

	h.broadcast <- &BroadcastMessage{
		InstanceID: instanceID,
		Message:    data,
	}
}

// GetClientCount returns the number of clients for an instance
func (h *Hub) GetClientCount(instanceID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.clients[instanceID]; ok {
		return len(clients)
	}
	return 0
}

// InstanceUpdate represents an instance update message
type InstanceUpdate struct {
	Type       string                 `json:"type"` // started, completed, suspended, resumed, terminated, task_created, task_completed
	InstanceID string                 `json:"instance_id"`
	Timestamp  time.Time              `json:"timestamp"`
	Data       map[string]interface{} `json:"data,omitempty"`
}
