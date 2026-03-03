package api

import (
	"github.com/gin-gonic/gin"
	"github.com/workflow-engine/v2/internal/api/handlers"
	"github.com/workflow-engine/v2/internal/api/websocket"
	"github.com/workflow-engine/v2/internal/integration/nats"
	"github.com/workflow-engine/v2/internal/integration/postgres"
	"github.com/workflow-engine/v2/internal/integration/registry"
	"github.com/workflow-engine/v2/internal/service"
	"go.uber.org/zap"
)

// Router holds all handlers and configuration
type Router struct {
	engine          *gin.Engine
	logger          *zap.Logger
	processHandler  *handlers.ProcessHandler
	instanceHandler *handlers.InstanceHandler
	registryHandler *handlers.RegistryHandler
	wsHandler       *handlers.WebSocketHandler
	sseHandler      *handlers.SSEHandler
}

// RouterDependencies all dependencies for creating router
type RouterDependencies struct {
	Logger          *zap.Logger
	ProcessRepo     *postgres.PostgresProcessRepository
	InstanceRepo    *postgres.PostgresInstanceRepository
	EventRepo       *postgres.PostgresEventRepository
	RegistryRepo    registry.RegistryRepository
	InstanceService *service.InstanceService
	NatsPublisher   *nats.Publisher
	ResponseHandler *nats.ResponseHandler
	WebSocketHub    *websocket.Hub
}

// NewRouter creates a new router with all handlers
func NewRouter(deps RouterDependencies) *Router {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()

	// Create handlers with correct dependencies
	processHandler := handlers.NewProcessHandler(deps.ProcessRepo, deps.Logger)
	instanceHandler := handlers.NewInstanceHandler(deps.InstanceService)
	registryHandler := handlers.NewRegistryHandler(deps.RegistryRepo)
	wsHandler := handlers.NewWebSocketHandler(deps.WebSocketHub)
	sseHandler := handlers.NewSSEHandler(deps.ResponseHandler)

	return &Router{
		engine:          engine,
		logger:          deps.Logger,
		processHandler:  processHandler,
		instanceHandler: instanceHandler,
		registryHandler: registryHandler,
		wsHandler:       wsHandler,
		sseHandler:      sseHandler,
	}
}

// SetupRoutes configures all routes
func (r *Router) SetupRoutes() {
	// Apply global middleware
	r.engine.Use(Logger(r.logger))
	r.engine.Use(Recovery(r.logger))
	r.engine.Use(CORS())
	r.engine.Use(RequestID())

	// Health check endpoints
	r.engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy"})
	})

	r.engine.GET("/ready", func(c *gin.Context) {
		c.JSON(200, gin.H{"ready": true})
	})

	// API v1 routes
	v1 := r.engine.Group("/api/v1")
	{
		// Process endpoints
		processes := v1.Group("/processes")
		{
			processes.POST("/deploy", r.processHandler.Deploy)
			processes.GET("", r.processHandler.List)
			processes.GET("/:id", r.processHandler.Get)
			processes.GET("/:id/xml", r.processHandler.GetXML)
		}

		// Instance endpoints
		instances := v1.Group("/instances")
		{
			// Start instance - POST /api/v1/instances or POST /api/v1/processes/:id/start
			instances.POST("", r.instanceHandler.Start)
			v1.POST("/processes/:id/start", r.instanceHandler.Start)

			instances.GET("", r.instanceHandler.List)
			instances.GET("/:id", r.instanceHandler.GetByID)
			instances.POST("/:id/suspend", r.instanceHandler.Suspend)
			instances.POST("/:id/resume", r.instanceHandler.Resume)
			instances.POST("/:id/terminate", r.instanceHandler.Terminate)

			// Task endpoints
			instances.POST("/:id/tasks/:taskId/complete", r.instanceHandler.CompleteTask)
			instances.GET("/:id/tasks/:taskId/form", r.instanceHandler.GetTaskForm)

			// Variables
			instances.GET("/:id/variables", r.instanceHandler.GetVariables)
			instances.PUT("/:id/variables", r.instanceHandler.UpdateVariables)
		}

		// Registry endpoints
		registryGroup := v1.Group("/registry")
		{
			registryGroup.POST("/heartbeat", r.registryHandler.Heartbeat)
			registryGroup.GET("/services", r.registryHandler.ListServices)
			registryGroup.GET("/services/:name", r.registryHandler.GetService)
			registryGroup.GET("/services/:name/actions", r.registryHandler.GetServiceActions)
		}
	}

	// WebSocket endpoint
	r.engine.GET("/ws/instances/:id", r.wsHandler.HandleWebSocket)

	// SSE endpoints
	r.engine.GET("/sse/tasks", r.sseHandler.HandleTaskSSE)
	r.engine.GET("/sse/registry", r.sseHandler.HandleRegistrySSE)

	// Swagger placeholder
	r.engine.GET("/swagger/*any", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Swagger documentation",
			"endpoints": []string{
				"/api/v1/processes",
				"/api/v1/instances",
				"/api/v1/registry",
			},
		})
	})
}

// GetEngine returns the gin engine
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}
