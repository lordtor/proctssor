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

	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"github.com/workflow-engine/v2/internal/api"
	"github.com/workflow-engine/v2/internal/api/websocket"
	"github.com/workflow-engine/v2/internal/core/executor"
	"github.com/workflow-engine/v2/internal/integration/nats"
	"github.com/workflow-engine/v2/internal/integration/postgres"
	"github.com/workflow-engine/v2/internal/integration/registry"
	"github.com/workflow-engine/v2/internal/service"
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
	DBHost      string
	DBPort      int
	DBUser      string
	DBPassword  string
	DBName      string
	DBSSLMode   string
}

// NewConfig creates configuration from environment
func NewConfig() *Config {
	return &Config{
		Port:        getEnv("PORT", defaultPort),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		NATSURL:     getEnv("NATS_URL", "nats://nats:4222"),
		DBHost:      getEnv("DB_HOST", "postgres"),
		DBPort:      5432,
		DBUser:      getEnv("DB_USER", "bpmn"),
		DBPassword:  getEnv("DB_PASSWORD", "bpmn_secret"),
		DBName:      getEnv("DB_NAME", "bpmn"),
		DBSSLMode:   getEnv("DB_SSLMODE", "disable"),
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

	// Initialize database
	pgCfg := postgres.Config{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
		DBName:   cfg.DBName,
		SSLMode:  cfg.DBSSLMode,
	}
	db, err := postgres.NewDB(pgCfg)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize repositories
	processRepo := postgres.NewProcessRepository(db)
	instanceRepo := postgres.NewInstanceRepository(db)
	eventRepo := postgres.NewEventRepository(db)
	registryRepo := registry.NewRegistryRepository(db)

	// Initialize NATS publisher
	var natsPublisher *nats.Publisher
	natsPublisher, err = nats.NewPublisher(nats.PublisherConfig{
		URL: cfg.NATSURL,
	})
	if err != nil {
		logger.Warn("Failed to connect to NATS", zap.Error(err))
		// Continue without NATS
	}
	if natsPublisher != nil {
		defer natsPublisher.Close()
	}

	// Initialize WebSocket hub
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// Initialize registry cache with LRU (1000 entries, 5 min TTL)
	registryCache := registry.NewLRUCache(1000, 5*time.Minute)
	registryCacheWarmer := registry.NewCacheWarmer(registryCache, registryRepo)

	// Warmup cache at startup
	if err := registryCacheWarmer.Warmup(context.Background()); err != nil {
		logger.Warn("Failed to warmup registry cache", zap.Error(err))
	}

	// Initialize registry change listener for PostgreSQL NOTIFY
	registryListener := registry.NewRegistryChangeListener(registryCache)
	// Note: In production, you would start the listener with the database connection
	// Use pgx or lib/pq with proper notification handling
	// if err := registryListener.Start(context.Background(), db.GetDB()); err != nil {
	// 	logger.Warn("Failed to start registry change listener", zap.Error(err))
	// }
	defer registryListener.Stop()

	logger.Info("Registry cache initialized",
		zap.Int("maxEntries", 1000),
		zap.String("ttl", "5m"))

	// Initialize executor
	exec := executor.NewExecutor(registryCache, natsPublisher, logger)

	// Initialize instance service
	instanceService := service.NewInstanceService(
		processRepo,
		instanceRepo,
		eventRepo,
		exec,
		natsPublisher,
		wsHub,
		logger,
	)

	// Create router with all dependencies
	router := api.NewRouter(api.RouterDependencies{
		Logger:          logger,
		ProcessRepo:     processRepo,
		InstanceRepo:    instanceRepo,
		EventRepo:       eventRepo,
		RegistryRepo:    registryRepo,
		InstanceService: instanceService,
		NatsPublisher:   natsPublisher,
		WebSocketHub:    wsHub,
	})
	router.SetupRoutes()

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: router.GetEngine(),
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
