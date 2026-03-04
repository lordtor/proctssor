package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gitlab-service/internal/config"
	"gitlab-service/internal/nats"
	"gitlab-service/internal/registry"
	"gitlab-service/internal/server"
	"gitlab-service/internal/service"
)

func main() {
	log.Println("Starting gitlab-service...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create GitLab client
	gitlabClient, err := service.NewGitLabClient(
		cfg.GitLab.URL,
		cfg.GitLab.Token,
		cfg.GitLab.VerifyTLS,
	)
	if err != nil {
		log.Printf("Warning: Failed to create GitLab client: %v", err)
		// Continue without GitLab client - service can still run
	}

	// Create service handler
	handler := service.NewHandler(gitlabClient)

	// Create NATS consumer
	consumer, err := nats.NewConsumer(
		cfg.NATS.URL,
		cfg.NATS.Subscriber.SubjectPrefix+".>", // Subscribe to all subjects under the prefix
		cfg.NATS.Subscriber.QueueGroup,
		handler,
	)
	if err != nil {
		log.Fatalf("Failed to create NATS consumer: %v", err)
	}
	defer consumer.Close()

	// Start NATS consumer in background
	go func() {
		log.Printf("Starting NATS consumer on subject: %s.>", cfg.NATS.Subscriber.SubjectPrefix)
		if err := consumer.Start(ctx); err != nil && err != context.Canceled {
			log.Printf("NATS consumer error: %v", err)
		}
	}()

	// Wait for consumer to be ready
	time.Sleep(500 * time.Millisecond)

	// Create registry client and register service
	registryClient := registry.NewClient(
		cfg.Registry.EngineURL,
		cfg.Service.Name,
		cfg.Service.Type,
		cfg.Service.Endpoint,
		cfg.Service.Metadata,
	)

	// Register with engine
	if err := registryClient.Register(ctx); err != nil {
		log.Printf("Warning: Failed to register with engine: %v", err)
		// Continue without registration - service can still run
	}

	// Start heartbeat in background
	go registryClient.StartHeartbeat(ctx, cfg.Registry.HeartbeatInterval)

	// Track start time for uptime
	startTime := time.Now()

	// Create and start HTTP server
	httpServer := server.NewServer(fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port))

	go func() {
		httpServer.Start(startTime, consumer.IsConnected(), gitlabClient != nil && gitlabClient.IsConnected())
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Printf("Received signal: %v", sig)
	case <-ctx.Done():
		log.Println("Context cancelled")
	}

	// Graceful shutdown
	log.Println("Shutting down...")

	// Unregister from registry
	if err := registryClient.Unregister(context.Background()); err != nil {
		log.Printf("Warning: Failed to unregister: %v", err)
	}

	// Stop heartbeat
	registryClient.Stop()

	// Stop HTTP server
	if err := httpServer.Stop(); err != nil {
		log.Printf("Warning: Failed to stop HTTP server: %v", err)
	}

	// Cancel context to stop NATS consumer
	cancel()

	log.Println("gitlab-service stopped")
}
