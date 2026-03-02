package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/workflow-engine/v2/internal/integration/registry"
)

// RegistryHandler handles registry-related HTTP requests
type RegistryHandler struct {
	repo *registry.PostgresRegistryRepository
}

// NewRegistryHandler creates a new registry handler
func NewRegistryHandler(repo *registry.PostgresRegistryRepository) *RegistryHandler {
	return &RegistryHandler{repo: repo}
}

// RegisterRequest represents a service registration request
type RegisterRequest struct {
	Name     string            `json:"name" binding:"required"`
	Type     string            `json:"type" binding:"required"` // worker, handler, external
	Endpoint string            `json:"endpoint" binding:"required"`
	Metadata map[string]string `json:"metadata"`
}

// RegisterResponse represents a registration response
type RegisterResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Endpoint     string `json:"endpoint"`
	Status       string `json:"status"`
	RegisteredAt string `json:"registered_at"`
}

// Register godoc
// @Summary Register a service
// @Description Register a new service in the registry
// @Tags registry
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration request"
// @Success 201 {object} RegisterResponse
// @Router /api/v1/registry [post]
func (h *RegistryHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	service := &registry.Service{
		ID:           uuid.New().String(),
		Name:         req.Name,
		Type:         req.Type,
		Endpoint:     req.Endpoint,
		Metadata:     req.Metadata,
		Status:       "active",
		RegisteredAt: time.Now(),
		HeartbeatAt:  time.Now(),
	}

	if err := h.repo.Register(c.Request.Context(), service); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, RegisterResponse{
		ID:           service.ID,
		Name:         service.Name,
		Type:         service.Type,
		Endpoint:     service.Endpoint,
		Status:       service.Status,
		RegisteredAt: service.RegisteredAt.Format(time.RFC3339),
	})
}

// HeartbeatRequest represents a heartbeat request
type HeartbeatRequest struct {
	ServiceID string `json:"service_id" binding:"required"`
}

// Heartbeat godoc
// @Summary Send heartbeat
// @Description Send a heartbeat to keep a service active
// @Tags registry
// @Accept json
// @Produce json
// @Param request body HeartbeatRequest true "Heartbeat request"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/registry/heartbeat [post]
func (h *RegistryHandler) Heartbeat(c *gin.Context) {
	var req HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.Heartbeat(c.Request.Context(), req.ServiceID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"service_id": req.ServiceID,
		"status":     "active",
	})
}

// ListServices godoc
// @Summary List services
// @Description Get list of registered services
// @Tags registry
// @Produce json
// @Param type query string false "Filter by service type"
// @Success 200 {array} registry.Service
// @Router /api/v1/registry/services [get]
func (h *RegistryHandler) ListServices(c *gin.Context) {
	serviceType := c.Query("type")

	var services []*registry.Service
	var err error

	if serviceType != "" {
		services, err = h.repo.Discover(c.Request.Context(), serviceType)
	} else {
		services, err = h.repo.ListAll(c.Request.Context())
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if services == nil {
		services = []*registry.Service{}
	}

	c.JSON(http.StatusOK, services)
}

// GetService godoc
// @Summary Get service by name
// @Description Get a specific service by name
// @Tags registry
// @Produce json
// @Param name path string true "Service Name"
// @Success 200 {object} registry.Service
// @Router /api/v1/registry/services/{name} [get]
func (h *RegistryHandler) GetService(c *gin.Context) {
	name := c.Param("name")

	service, err := h.repo.DiscoverByName(c.Request.Context(), name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, service)
}

// Unregister godoc
// @Summary Unregister a service
// @Description Remove a service from the registry
// @Tags registry
// @Param name path string true "Service Name"
// @Success 204
// @Router /api/v1/registry/services/{name} [delete]
func (h *RegistryHandler) Unregister(c *gin.Context) {
	name := c.Param("name")

	service, err := h.repo.DiscoverByName(c.Request.Context(), name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.Unregister(c.Request.Context(), service.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// DiscoverServices godoc
// @Summary Discover services by type
// @Description Discover active services of a specific type
// @Tags registry
// @Produce json
// @Param type path string true "Service Type"
// @Success 200 {array} registry.Service
// @Router /api/v1/registry/discover/{type} [get]
func (h *RegistryHandler) DiscoverServices(c *gin.Context) {
	serviceType := c.Param("type")

	services, err := h.repo.Discover(c.Request.Context(), serviceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if services == nil {
		services = []*registry.Service{}
	}

	c.JSON(http.StatusOK, services)
}

// GetActions godoc
// @Summary Get service actions
// @Description Get available actions for a service
// @Tags registry
// @Produce json
// @Param name path string true "Service Name"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/registry/services/{name}/actions [get]
func (h *RegistryHandler) GetActions(c *gin.Context) {
	name := c.Param("name")

	service, err := h.repo.DiscoverByName(c.Request.Context(), name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Return metadata as available actions
	c.JSON(http.StatusOK, gin.H{
		"service_name": service.Name,
		"service_type": service.Type,
		"endpoint":     service.Endpoint,
		"actions":      service.Metadata,
	})
}
