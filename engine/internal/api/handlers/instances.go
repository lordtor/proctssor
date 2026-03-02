package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/workflow-engine/v2/internal/core/statemachine"
	"github.com/workflow-engine/v2/internal/integration/postgres"
)

// InstanceHandler handles instance-related HTTP requests
type InstanceHandler struct {
	instanceRepo *postgres.PostgresInstanceRepository
	processRepo  *postgres.PostgresProcessRepository
}

// NewInstanceHandler creates a new instance handler
func NewInstanceHandler(instanceRepo *postgres.PostgresInstanceRepository, processRepo *postgres.PostgresProcessRepository) *InstanceHandler {
	return &InstanceHandler{
		instanceRepo: instanceRepo,
		processRepo:  processRepo,
	}
}

// StartRequest represents a process start request
type StartRequest struct {
	Variables map[string]interface{} `json:"variables"`
	Initiator string                 `json:"initiator"`
}

// StartResponse represents a process start response
type StartResponse struct {
	InstanceID string                 `json:"instance_id"`
	ProcessID  string                 `json:"process_id"`
	Status     string                 `json:"status"`
	Variables  map[string]interface{} `json:"variables,omitempty"`
	StartedAt  string                 `json:"started_at"`
}

// Start godoc
// @Summary Start a process instance
// @Description Start a new process instance from a deployed process
// @Tags instances
// @Accept json
// @Produce json
// @Param id path string true "Process ID"
// @Param request body StartRequest true "Start request"
// @Success 201 {object} StartResponse
// @Router /api/v1/instances [post]
func (h *InstanceHandler) Start(c *gin.Context) {
	var req StartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Variables = make(map[string]interface{})
	}

	processKey := c.Param("id")
	if processKey == "" {
		processKey = c.GetString("process_key")
	}

	// Get process definition
	process, _, err := h.processRepo.GetProcessByKey(c.Request.Context(), processKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Process not found: " + err.Error()})
		return
	}

	// Create new instance
	instance := statemachine.NewProcessInstance(processKey, req.Variables)
	instance.Status = statemachine.ProcessInstanceStatusRunning
	now := time.Now()
	instance.StartedAt = &now

	// Save to database
	if err := h.instanceRepo.CreateInstance(c.Request.Context(), instance); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, StartResponse{
		InstanceID: instance.ID,
		ProcessID:  process.ID,
		Status:     string(instance.Status),
		Variables:  instance.Variables,
		StartedAt:  instance.StartedAt.Format(time.RFC3339),
	})
}

// List godoc
// @Summary List process instances
// @Description Get list of process instances with optional filters
// @Tags instances
// @Produce json
// @Param process_key query string false "Filter by process key"
// @Param status query string false "Filter by status"
// @Param limit query int false "Limit results"
// @Success 200 {array} statemachine.ProcessInstance
// @Router /api/v1/instances [get]
func (h *InstanceHandler) List(c *gin.Context) {
	filter := postgres.InstanceFilter{
		ProcessKey: c.Query("process_key"),
		Status:     c.Query("status"),
	}

	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filter.Limit = l
		}
	}

	instances, err := h.instanceRepo.ListInstances(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if instances == nil {
		instances = []*statemachine.ProcessInstance{}
	}

	c.JSON(http.StatusOK, instances)
}

// GetByID godoc
// @Summary Get instance by ID
// @Description Get a specific process instance by ID
// @Tags instances
// @Produce json
// @Param id path string true "Instance ID"
// @Success 200 {object} statemachine.ProcessInstance
// @Router /api/v1/instances/{id} [get]
func (h *InstanceHandler) GetByID(c *gin.Context) {
	id := c.Param("id")

	instance, err := h.instanceRepo.GetInstanceByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, instance)
}

// Suspend godoc
// @Summary Suspend an instance
// @Description Suspend a running process instance
// @Tags instances
// @Param id path string true "Instance ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/instances/{id}/suspend [post]
func (h *InstanceHandler) Suspend(c *gin.Context) {
	id := c.Param("id")

	instance, err := h.instanceRepo.GetInstanceByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if instance.Status != statemachine.ProcessInstanceStatusRunning {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Instance is not running"})
		return
	}

	instance.Status = statemachine.ProcessInstanceStatusSuspended

	if err := h.instanceRepo.UpdateInstance(c.Request.Context(), instance); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"instance_id": instance.ID,
		"status":      instance.Status,
	})
}

// Resume godoc
// @Summary Resume an instance
// @Description Resume a suspended process instance
// @Tags instances
// @Param id path string true "Instance ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/instances/{id}/resume [post]
func (h *InstanceHandler) Resume(c *gin.Context) {
	id := c.Param("id")

	instance, err := h.instanceRepo.GetInstanceByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if instance.Status != statemachine.ProcessInstanceStatusSuspended {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Instance is not suspended"})
		return
	}

	instance.Status = statemachine.ProcessInstanceStatusRunning

	if err := h.instanceRepo.UpdateInstance(c.Request.Context(), instance); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"instance_id": instance.ID,
		"status":      instance.Status,
	})
}

// Terminate godoc
// @Summary Terminate an instance
// @Description Terminate a process instance
// @Tags instances
// @Param id path string true "Instance ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/instances/{id}/terminate [post]
func (h *InstanceHandler) Terminate(c *gin.Context) {
	id := c.Param("id")

	instance, err := h.instanceRepo.GetInstanceByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	if instance.Status == statemachine.ProcessInstanceStatusCompleted ||
		instance.Status == statemachine.ProcessInstanceStatusTerminated {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Instance already terminated or completed"})
		return
	}

	now := time.Now()
	instance.Status = statemachine.ProcessInstanceStatusTerminated
	instance.CompletedAt = &now

	if err := h.instanceRepo.UpdateInstance(c.Request.Context(), instance); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"instance_id": instance.ID,
		"status":      instance.Status,
	})
}

// GetVariables godoc
// @Summary Get instance variables
// @Description Get the variables of a process instance
// @Tags instances
// @Produce json
// @Param id path string true "Instance ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/instances/{id}/variables [get]
func (h *InstanceHandler) GetVariables(c *gin.Context) {
	id := c.Param("id")

	instance, err := h.instanceRepo.GetInstanceByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, instance.Variables)
}

// UpdateVariablesRequest represents a variable update request
type UpdateVariablesRequest struct {
	Variables map[string]interface{} `json:"variables" binding:"required"`
}

// UpdateVariables godoc
// @Summary Update instance variables
// @Description Update the variables of a process instance
// @Tags instances
// @Accept json
// @Produce json
// @Param id path string true "Instance ID"
// @Param request body UpdateVariablesRequest true "Variables"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/instances/{id}/variables [put]
func (h *InstanceHandler) UpdateVariables(c *gin.Context) {
	id := c.Param("id")

	var req UpdateVariablesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	instance, err := h.instanceRepo.GetInstanceByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// Merge variables
	for k, v := range req.Variables {
		instance.Variables[k] = v
	}

	if err := h.instanceRepo.UpdateInstance(c.Request.Context(), instance); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, instance.Variables)
}
