package api

import (
	"github.com/gin-gonic/gin"
	"github.com/workflow-engine/v2/internal/api/handlers"
	"github.com/workflow-engine/v2/internal/integration/postgres"
	"github.com/workflow-engine/v2/internal/integration/registry"
)

// Router holds all handlers and configuration
type Router struct {
	engine          *gin.Engine
	processHandler  *handlers.ProcessHandler
	instanceHandler *handlers.InstanceHandler
	registryHandler *handlers.RegistryHandler
}

// NewRouter creates a new router with all handlers
func NewRouter(
	processRepo *postgres.PostgresProcessRepository,
	instanceRepo *postgres.PostgresInstanceRepository,
	registryRepo *registry.PostgresRegistryRepository,
) *Router {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()

	return &Router{
		engine:          engine,
		processHandler:  handlers.NewProcessHandler(processRepo),
		instanceHandler: handlers.NewInstanceHandler(instanceRepo, processRepo),
		registryHandler: handlers.NewRegistryHandler(registryRepo),
	}
}

// SetupRoutes configures all routes
func (r *Router) SetupRoutes() {
	// Apply global middleware
	r.engine.Use(gin.Logger())
	r.engine.Use(gin.Recovery())
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
			processes.GET("/validate", r.processHandler.Validate)
			processes.GET("/key/:key", r.processHandler.GetByKey)
			processes.GET("/:id", r.processHandler.GetByID)
			processes.GET("/:id/xml", r.processHandler.GetXML)
			processes.DELETE("/:id", r.processHandler.Delete)
		}

		// Instance endpoints
		instances := v1.Group("/instances")
		{
			instances.POST("", r.instanceHandler.Start)
			instances.GET("", r.instanceHandler.List)
			instances.GET("/:id", r.instanceHandler.GetByID)
			instances.POST("/:id/suspend", r.instanceHandler.Suspend)
			instances.POST("/:id/resume", r.instanceHandler.Resume)
			instances.POST("/:id/terminate", r.instanceHandler.Terminate)
			instances.GET("/:id/variables", r.instanceHandler.GetVariables)
			instances.PUT("/:id/variables", r.instanceHandler.UpdateVariables)
		}

		// Registry endpoints
		registry := v1.Group("/registry")
		{
			registry.POST("", r.registryHandler.Register)
			registry.POST("/heartbeat", r.registryHandler.Heartbeat)
			registry.GET("/services", r.registryHandler.ListServices)
			registry.GET("/services/:name", r.registryHandler.GetService)
			registry.DELETE("/services/:name", r.registryHandler.Unregister)
			registry.GET("/services/:name/actions", r.registryHandler.GetActions)
			registry.GET("/discover/:type", r.registryHandler.DiscoverServices)
		}
	}

	// Swagger documentation route
	r.engine.GET("/swagger/*any", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Swagger documentation available at /swagger/index.html",
		})
	})
}

// GetEngine returns the gin engine
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}
