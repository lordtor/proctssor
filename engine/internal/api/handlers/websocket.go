package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	ws "github.com/workflow-engine/v2/internal/api/websocket"
)

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	hub *ws.Hub
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *ws.Hub) *WebSocketHandler {
	return &WebSocketHandler{hub: hub}
}

// HandleWebSocket handles WebSocket connections for instance updates
// @Summary Connect to instance WebSocket
// @Description Upgrade HTTP connection to WebSocket for real-time instance updates
// @Tags websocket
// @Produce json
// @Param id path string true "Instance ID"
// @Success 101 {string} Switching Protocols
// @Router /ws/instances/{id} [get]
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	instanceID := c.Param("id")
	if instanceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Instance ID required"})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := ws.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	// Create client
	client := ws.NewClient(h.hub, conn, instanceID)

	// Register client
	h.hub.RegisterClient(client)

	// Start pumps
	go client.WritePump()
	go client.ReadPump()
}

// Handler functions use the exported Upgrader from websocket package
