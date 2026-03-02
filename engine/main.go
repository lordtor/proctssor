// BPMN Workflow Engine - Main Entry Point
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
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

// Response helpers
func jsonResponse(w http.ResponseWriter, data map[string]interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// Handlers
func healthHandler(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, map[string]interface{}{
		"status":    "healthy",
		"service":   appName,
		"version":   appVersion,
		"timestamp": time.Now().UTC(),
	}, http.StatusOK)
}

func readyHandler(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, map[string]interface{}{"ready": true}, http.StatusOK)
}

// Placeholder handlers
func listWorkflows(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, map[string]interface{}{"workflows": []interface{}{}}, http.StatusOK)
}

func createWorkflow(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, map[string]interface{}{
		"id":      "new-workflow-id",
		"message": "Workflow created",
	}, http.StatusCreated)
}

func getWorkflow(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		id = "workflow-1"
	}
	jsonResponse(w, map[string]interface{}{
		"id":          id,
		"name":        "Example Workflow",
		"description": "Example BPMN workflow",
	}, http.StatusOK)
}

func listProcesses(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, map[string]interface{}{"processes": []interface{}{}}, http.StatusOK)
}

func startProcess(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, map[string]interface{}{
		"id":     "new-process-id",
		"status": "started",
	}, http.StatusCreated)
}

func getProcess(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		id = "process-1"
	}
	jsonResponse(w, map[string]interface{}{
		"id":     id,
		"status": "running",
	}, http.StatusOK)
}

func completeTask(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, map[string]interface{}{"message": "Task completed"}, http.StatusOK)
}

func listTasks(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, map[string]interface{}{"tasks": []interface{}{}}, http.StatusOK)
}

func getTask(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		id = "task-1"
	}
	jsonResponse(w, map[string]interface{}{
		"id":     id,
		"status": "pending",
	}, http.StatusOK)
}

func claimTask(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, map[string]interface{}{"message": "Task claimed"}, http.StatusOK)
}

func main() {
	cfg := NewConfig()

	log.Printf("Starting %s v%s", appName, appVersion)
	log.Printf("Listening on port %s", cfg.Port)

	// Simple router using http.ServeMux
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/ready", readyHandler)
	mux.HandleFunc("/api/v1/workflows", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			listWorkflows(w, r)
		} else if r.Method == "POST" {
			createWorkflow(w, r)
		}
	})
	mux.HandleFunc("/api/v1/workflows/", getWorkflow)
	mux.HandleFunc("/api/v1/processes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			listProcesses(w, r)
		} else if r.Method == "POST" {
			startProcess(w, r)
		}
	})
	mux.HandleFunc("/api/v1/processes/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			getProcess(w, r)
		} else if r.Method == "POST" {
			completeTask(w, r)
		}
	})
	mux.HandleFunc("/api/v1/tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			listTasks(w, r)
		}
	})
	mux.HandleFunc("/api/v1/tasks/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			getTask(w, r)
		} else if r.Method == "POST" {
			claimTask(w, r)
		}
	})

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: mux,
	}

	// Start server in goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
	fmt.Println("BPMN Workflow Engine stopped")
}
