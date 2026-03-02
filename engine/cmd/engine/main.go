// BPMN Workflow Engine - Main Entry Point
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

const (
	// App constants
	appName    = "BPMN Workflow Engine"
	appVersion = "1.0.0"

	// Default port
	defaultPort = "8080"
)

// Config holds application configuration
type Config struct {
	Port        string
	DatabaseURL string
	NATSURL     string
}

// NewConfig creates configuration from environment
func NewConfig() *Config {
	return &Config{
		Port:        getEnv("PORT", defaultPort),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		NATSURL:     getEnv("NATS_URL", "nats://localhost:4222"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// @title BPMN Workflow Engine API
// @version 1.0.0
// @description REST API for BPMN Workflow Engine
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.example.com/support
// @contact.email support@example.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

func main() {
	// Load .env file if exists
	_ = godotenv.Load()

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Create config
	cfg := NewConfig()

	logger.Info("Starting BPMN Workflow Engine",
		zap.String("version", appVersion),
		zap.String("port", cfg.Port),
	)

	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)

	// Create router
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   appName,
			"version":   appVersion,
			"timestamp": time.Now().UTC(),
		})
	})

	// Readiness endpoint
	router.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ready": true})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Workflows
		workflows := v1.Group("/workflows")
		{
			workflows.GET("", listWorkflows)
			workflows.POST("", createWorkflow)
			workflows.GET("/:id", getWorkflow)
		}

		// Processes
		processes := v1.Group("/processes")
		{
			processes.GET("", listProcesses)
			processes.POST("", startProcess)
			processes.GET("/:id", getProcess)
			processes.POST("/:id/complete", completeTask)
		}

		// Tasks
		tasks := v1.Group("/tasks")
		{
			tasks.GET("", listTasks)
			tasks.GET("/:id", getTask)
			tasks.POST("/:id/claim", claimTask)
		}
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Server starting", zap.String("address", ":"+cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited")
	fmt.Println("BPMN Workflow Engine stopped")
}

// Handlers

// @Summary List all workflows
// @Description Get all workflow definitions
// @Tags workflows
// @Produce json
// @Success 200 {array} interface{}
func listWorkflows(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"workflows": []interface{}{}})
}

// @Summary Create a new workflow
// @Description Create a new workflow definition
// @Tags workflows
// @Accept json
// @Produce json
// @Success 201 {object} interface{}
func createWorkflow(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{
		"id":      "new-workflow-id",
		"message": "Workflow created",
	})
}

// @Summary Get workflow by ID
// @Description Get a specific workflow by ID
// @Tags workflows
// @Produce json
// @Param id path string true "Workflow ID"
// @Success 200 {object} interface{}
func getWorkflow(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":          id,
		"name":        "Example Workflow",
		"description": "Example BPMN workflow",
	})
}

// @Summary List all processes
// @Description Get all running processes
// @Tags processes
// @Produce json
// @Success 200 {array} interface{}
func listProcesses(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"processes": []interface{}{}})
}

// @Summary Start a new process
// @Description Start a new process from a workflow
// @Tags processes
// @Accept json
// @Produce json
// @Success 201 {object} interface{}
func startProcess(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{
		"id":     "new-process-id",
		"status": "started",
	})
}

// @Summary Get process by ID
// @Description Get a specific process by ID
// @Tags processes
// @Produce json
// @Param id path string true "Process ID"
// @Success 200 {object} interface{}
func getProcess(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":     id,
		"status": "running",
	})
}

// @Summary Complete a task
// @Description Complete a task in a process
// @Tags processes
// @Accept json
// @Produce json
// @Param id path string true "Process ID"
// @Success 200 {object} interface{}
func completeTask(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Task completed"})
}

// @Summary List all tasks
// @Description Get all available tasks
// @Tags tasks
// @Produce json
// @Success 200 {array} interface{}
func listTasks(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"tasks": []interface{}{}})
}

// @Summary Get task by ID
// @Description Get a specific task by ID
// @Tags tasks
// @Produce json
// @Param id path string true "Task ID"
// @Success 200 {object} interface{}
func getTask(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":     id,
		"status": "pending",
	})
}

// @Summary Claim a task
// @Description Claim a task for a user
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path string true "Task ID"
// @Success 200 {object} interface{}
func claimTask(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Task claimed"})
}
