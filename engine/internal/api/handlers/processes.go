package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/workflow-engine/v2/internal/core/bpmn"
	"github.com/workflow-engine/v2/internal/integration/postgres"
	"go.uber.org/zap"
)

// ProcessHandler handles process-related HTTP requests
type ProcessHandler struct {
	repo   *postgres.PostgresProcessRepository
	logger *zap.Logger
}

// NewProcessHandler creates a new process handler
func NewProcessHandler(repo *postgres.PostgresProcessRepository, logger *zap.Logger) *ProcessHandler {
	return &ProcessHandler{repo: repo, logger: logger}
}

// DeployRequest represents a process deployment request
type DeployRequest struct {
	ProcessKey string `json:"process_key" binding:"required"`
	Name       string `json:"name"`
	XML        string `json:"xml" binding:"required"`
	Version    int    `json:"version"`
}

// DeployResponse represents a deployment response
type DeployResponse struct {
	ID         string `json:"id"`
	ProcessKey string `json:"process_key"`
	Version    int    `json:"version"`
	Name       string `json:"name"`
	DeployedAt string `json:"deployed_at"`
}

// Deploy godoc
// @Summary Deploy a new process
// @Description Deploy a new BPMN process definition
// @Tags processes
// @Accept json
// @Produce json
// @Param request body DeployRequest true "Process deployment request"
// @Success 201 {object} DeployResponse
// @Router /api/v1/processes/deploy [post]
func (h *ProcessHandler) Deploy(c *gin.Context) {
	var req DeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse BPMN XML
	process, err := bpmn.Parse([]byte(req.XML))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid BPMN XML: " + err.Error()})
		return
	}

	// Set process metadata
	version := req.Version
	if version == 0 {
		version = 1
	}
	process.ID = req.ProcessKey + "_" + strconv.Itoa(version)
	if req.Name != "" {
		process.Name = req.Name
	}

	// Deploy process
	if err := h.repo.DeployProcess(c.Request.Context(), process, []byte(req.XML)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, DeployResponse{
		ID:         process.ID,
		ProcessKey: req.ProcessKey,
		Version:    version,
		Name:       process.Name,
		DeployedAt: "now",
	})
}

// List godoc
// @Summary List all processes
// @Description Get list of deployed processes
// @Tags processes
// @Produce json
// @Success 200 {array} postgres.ProcessInfo
// @Router /api/v1/processes [get]
func (h *ProcessHandler) List(c *gin.Context) {
	processes, err := h.repo.ListProcesses(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if processes == nil {
		processes = []postgres.ProcessInfo{}
	}

	c.JSON(http.StatusOK, processes)
}

// Get godoc
// @Summary Get process by ID
// @Description Get a specific process definition by ID
// @Tags processes
// @Produce json
// @Param id path string true "Process ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/processes/{id} [get]
func (h *ProcessHandler) Get(c *gin.Context) {
	id := c.Param("id")

	process, definition, err := h.repo.GetProcessByID(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to get process", zap.String("id", id), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "process not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":            process.ID,
		"name":          process.Name,
		"is_executable": process.IsExecutable,
		"elements":      len(process.FlowElement),
		"flows":         len(process.SequenceFlow),
		"definition":    string(definition),
	})
}

// GetByID godoc
// @Summary Get process by ID (alias for Get)
// @Description Get a specific process definition by ID
// @Tags processes
// @Produce json
// @Param id path string true "Process ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/processes/{id} [get]
func (h *ProcessHandler) GetByID(c *gin.Context) {
	h.Get(c)
}

// GetXML godoc
// @Summary Get process XML
// @Description Get the raw BPMN XML for a process
// @Tags processes
// @Produce application/xml
// @Param id path string true "Process ID"
// @Success 200 {string} string
// @Router /api/v1/processes/{id}/xml [get]
func (h *ProcessHandler) GetXML(c *gin.Context) {
	id := c.Param("id")

	_, definition, err := h.repo.GetProcessByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.Data(http.StatusOK, "application/xml", definition)
}

// GetXMLByKey godoc
// @Summary Get process XML by key
// @Description Get the raw BPMN XML for a process by key
// @Tags processes
// @Produce application/xml
// @Param key path string true "Process Key"
// @Success 200 {string} string
// @Router /api/v1/processes/key/{key}/xml [get]
func (h *ProcessHandler) GetXMLByKey(c *gin.Context) {
	key := c.Param("key")

	_, definition, err := h.repo.GetProcessByKey(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.Data(http.StatusOK, "application/xml", definition)
}

// GetByKey godoc
// @Summary Get process by key
// @Description Get the latest version of a process by key
// @Tags processes
// @Produce json
// @Param key path string true "Process Key"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/processes/key/{key} [get]
func (h *ProcessHandler) GetByKey(c *gin.Context) {
	key := c.Param("key")

	process, definition, err := h.repo.GetProcessByKey(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":            process.ID,
		"name":          process.Name,
		"process_key":   key,
		"is_executable": process.IsExecutable,
		"elements":      len(process.FlowElement),
		"flows":         len(process.SequenceFlow),
		"definition":    string(definition),
	})
}

// Delete godoc
// @Summary Delete a process
// @Description Delete a process definition
// @Tags processes
// @Param id path string true "Process ID"
// @Success 204
// @Router /api/v1/processes/{id} [delete]
func (h *ProcessHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.repo.DeleteProcess(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// ValidateRequest represents a validation request
type ValidateRequest struct {
	XML string `json:"xml" binding:"required"`
}

// Validate godoc
// @Summary Validate BPMN XML
// @Description Validate BPMN XML without deploying
// @Tags processes
// @Accept json
// @Produce json
// @Param request body ValidateRequest true "BPMN XML"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/processes/validate [post]
func (h *ProcessHandler) Validate(c *gin.Context) {
	var req ValidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse BPMN XML
	process, err := bpmn.Parse([]byte(req.XML))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	// Validate process
	errors := bpmn.Validate(process)

	c.JSON(http.StatusOK, gin.H{
		"valid":      len(errors) == 0,
		"errors":     errors,
		"process_id": process.ID,
		"name":       process.Name,
	})
}
