package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/workflow-engine/v2/internal/integration/postgres"
	"github.com/workflow-engine/v2/internal/service"
)

// InstanceHandler handles instance-related HTTP requests
type InstanceHandler struct {
	instanceService *service.InstanceService
}

// NewInstanceHandler creates a new instance handler with Service Layer
func NewInstanceHandler(instanceService *service.InstanceService) *InstanceHandler {
	return &InstanceHandler{
		instanceService: instanceService,
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

	// Вызываем Service Layer - он делает всю работу
	instance, err := h.instanceService.StartInstance(
		c.Request.Context(),
		processKey,
		req.Variables,
		"", // business key
		req.Initiator,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, StartResponse{
		InstanceID: instance.ID,
		ProcessID:  processKey,
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
// @Success 200 {array} service.InstanceInfo
// @Router /api/v1/instances [get]
func (h *InstanceHandler) List(c *gin.Context) {
	filter := service.InstanceFilter{
		ProcessKey: c.Query("process_key"),
		Status:     c.Query("status"),
	}

	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			filter.Limit = l
		}
	}

	// Используем Service Layer
	instances, err := h.instanceService.ListInstances(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if instances == nil {
		instances = []*service.InstanceInfo{}
	}

	c.JSON(http.StatusOK, instances)
}

// ListTasks godoc
// @Summary List user tasks
// @Description Get list of user tasks for the current user
// @Tags tasks
// @Produce json
// @Param assignee query string false "Filter by assignee"
// @Success 200 {array} postgres.UserTask
// @Router /api/v1/tasks [get]
func (h *InstanceHandler) ListTasks(c *gin.Context) {
	filter := postgres.TaskFilter{
		Assignee: c.Query("assignee"),
		Status:   c.Query("status"),
	}

	tasks, err := h.instanceService.GetTasks(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tasks)
}

// GetByID godoc
// @Summary Get instance by ID
// @Description Get a specific process instance by ID
// @Tags instances
// @Produce json
// @Param id path string true "Instance ID"
// @Success 200 {object} service.InstanceInfo
// @Router /api/v1/instances/{id} [get]
func (h *InstanceHandler) GetByID(c *gin.Context) {
	id := c.Param("id")

	// Используем Service Layer
	instance, err := h.instanceService.GetInstance(c.Request.Context(), id)
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

	// Используем Service Layer
	if err := h.instanceService.SuspendInstance(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"instance_id": id,
		"status":      "suspended",
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

	// Используем Service Layer
	if err := h.instanceService.ResumeInstance(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"instance_id": id,
		"status":      "running",
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

	// Используем Service Layer
	if err := h.instanceService.TerminateInstance(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"instance_id": id,
		"status":      "terminated",
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

	// Используем Service Layer
	variables, err := h.instanceService.GetVariables(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, variables)
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

	// Используем Service Layer
	variables, err := h.instanceService.UpdateVariables(c.Request.Context(), id, req.Variables)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, variables)
}

// CompleteTaskRequest represents a task completion request
type CompleteTaskRequest struct {
	Variables map[string]interface{} `json:"variables"`
	UserID    string                 `json:"user_id"`
}

// CompleteTask godoc
// @Summary Complete a user task
// @Description Complete a user task in a process instance
// @Tags instances
// @Accept json
// @Produce json
// @Param id path string true "Instance ID"
// @Param taskId path string true "Task ID"
// @Param request body CompleteTaskRequest true "Task completion request"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/instances/{id}/tasks/{taskId}/complete [post]
func (h *InstanceHandler) CompleteTask(c *gin.Context) {
	instanceID := c.Param("id")
	taskID := c.Param("taskId")

	var req CompleteTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Variables = make(map[string]interface{})
	}

	err := h.instanceService.CompleteUserTask(
		c.Request.Context(),
		instanceID,
		taskID,
		req.Variables,
		req.UserID,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task completed"})
}

// GetTaskForm godoc
// @Summary Get task form
// @Description Get the form schema for a user task
// @Tags instances
// @Produce json
// @Param id path string true "Instance ID"
// @Param taskId path string true "Task ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/instances/{id}/tasks/{taskId}/form [get]
func (h *InstanceHandler) GetTaskForm(c *gin.Context) {
	instanceID := c.Param("id")
	taskID := c.Param("taskId")

	form, err := h.instanceService.GetTaskForm(c.Request.Context(), instanceID, taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, form)
}

// ClaimTaskRequest represents a task claim request
type ClaimTaskRequest struct {
	UserID string `json:"user_id"`
}

// ClaimTask godoc
// @Summary Claim a task
// @Description Assign a task to a user
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path string true "Task ID"
// @Param request body ClaimTaskRequest true "Claim request"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tasks/{id}/claim [post]
func (h *InstanceHandler) ClaimTask(c *gin.Context) {
	taskID := c.Param("id")

	var req ClaimTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Try to get user from header or use default
		req.UserID = c.GetHeader("X-User-ID")
		if req.UserID == "" {
			req.UserID = "anonymous"
		}
	}

	err := h.instanceService.ClaimTask(c.Request.Context(), taskID, req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task claimed", "task_id": taskID})
}

// UnclaimTask godoc
// @Summary Unclaim a task
// @Description Remove assignment from a task
// @Tags tasks
// @Produce json
// @Param id path string true "Task ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tasks/{id}/unclaim [post]
func (h *InstanceHandler) UnclaimTask(c *gin.Context) {
	taskID := c.Param("id")

	err := h.instanceService.UnclaimTask(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task unclaimed", "task_id": taskID})
}

// DelegateTaskRequest represents a task delegation request
type DelegateTaskRequest struct {
	UserID string `json:"user_id"`
}

// DelegateTask godoc
// @Summary Delegate a task
// @Description Delegate a task to another user
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path string true "Task ID"
// @Param request body DelegateTaskRequest true "Delegation request"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/tasks/{id}/delegate [post]
func (h *InstanceHandler) DelegateTask(c *gin.Context) {
	taskID := c.Param("id")

	var req DelegateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	err := h.instanceService.DelegateTask(c.Request.Context(), taskID, req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Task delegated", "task_id": taskID, "delegate_to": req.UserID})
}

// GetTaskHistory godoc
// @Summary Get task history
// @Description Get history of completed tasks
// @Tags tasks
// @Produce json
// @Param limit query int false "Limit results"
// @Success 200 {array} postgres.UserTask
// @Router /api/v1/tasks/history [get]
func (h *InstanceHandler) GetTaskHistory(c *gin.Context) {
	limit := 100
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	tasks, err := h.instanceService.GetTaskHistory(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if tasks == nil {
		tasks = []postgres.UserTask{}
	}

	c.JSON(http.StatusOK, tasks)
}

// GetTokens godoc
// @Summary Get instance tokens
// @Description Get tokens for a process instance
// @Tags instances
// @Produce json
// @Param id path string true "Instance ID"
// @Success 200 {array} map[string]interface{}
// @Router /api/v1/instances/{id}/tokens [get]
func (h *InstanceHandler) GetTokens(c *gin.Context) {
	instanceID := c.Param("id")

	tokens, err := h.instanceService.GetTokens(c.Request.Context(), instanceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if tokens == nil {
		tokens = []map[string]interface{}{}
	}

	c.JSON(http.StatusOK, tokens)
}

// GetEvents godoc
// @Summary Get instance events
// @Description Get events for a process instance
// @Tags instances
// @Produce json
// @Param id path string true "Instance ID"
// @Success 200 {array} map[string]interface{}
// @Router /api/v1/instances/{id}/events [get]
func (h *InstanceHandler) GetEvents(c *gin.Context) {
	instanceID := c.Param("id")

	events, err := h.instanceService.GetEvents(c.Request.Context(), instanceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if events == nil {
		events = []map[string]interface{}{}
	}

	c.JSON(http.StatusOK, events)
}
