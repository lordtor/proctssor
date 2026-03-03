package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

// Client represents a WebSocket client
type Client struct {
	// Hub reference
	hub *Hub

	// The WebSocket connection
	conn *websocket.Conn

	// Buffered channel of outbound messages
	send chan []byte

	// Instance ID this client is subscribed to
	InstanceID string
}

// Message represents a WebSocket message
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn, instanceID string) *Client {
	return &Client{
		hub:        hub,
		conn:       conn,
		send:       make(chan []byte, 256),
		InstanceID: instanceID,
	}
}

// ReadPump reads messages from the WebSocket connection
func (c *Client) ReadPump() {
	defer func() {
		c.hub.UnregisterClient(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle incoming message
		c.handleMessage(message)
	}
}

// WritePump writes messages to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current WebSocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage handles incoming WebSocket messages
func (c *Client) handleMessage(data []byte) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("Error parsing message: %v", err)
		c.sendError("Invalid message format")
		return
	}

	switch msg.Type {
	case "ping":
		c.sendJSON(InstanceUpdate{Type: "pong", InstanceID: c.InstanceID, Timestamp: time.Now()})
	case "subscribe":
		// Already subscribed on connection
		c.sendJSON(InstanceUpdate{Type: "subscribed", InstanceID: c.InstanceID, Timestamp: time.Now()})
	case "unsubscribe":
		c.hub.UnregisterClient(c)
	default:
		log.Printf("Unknown message type: %s", msg.Type)
	}
}

// sendJSON sends a JSON message to the client
func (c *Client) sendJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	select {
	case c.send <- data:
		return nil
	default:
		return err
	}
}

// sendError sends an error message to the client
func (c *Client) sendError(message string) {
	c.sendJSON(map[string]string{"type": "error", "message": message})
}

// Upgrader exported for handlers
var Upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Get origin from request
		origin := r.Header.Get("Origin")
		// In development, allow all origins
		if os.Getenv("GIN_MODE") == "debug" || os.Getenv("ENV") == "development" {
			return true
		}
		// In production, validate against allowed origins
		allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
		if allowedOrigins != "" {
			allowed := false
			for _, o := range strings.Split(allowedOrigins, ",") {
				if strings.TrimSpace(o) == origin {
					allowed = true
					break
				}
			}
			if !allowed {
				log.Printf("[WARN] WebSocket origin rejected: %s (allowed: %s)", origin, allowedOrigins)
				return false
			}
			return true
		}
		// No origin header is fine (same-origin request)
		if origin == "" {
			return true
		}
		return false
	},
}
