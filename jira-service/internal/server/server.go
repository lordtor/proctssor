package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Server represents the HTTP server
type Server struct {
	addr   string
	server *http.Server
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status        string                 `json:"status"`
	ServiceName   string                 `json:"service_name"`
	Timestamp     time.Time              `json:"timestamp"`
	UptimeSeconds int64                  `json:"uptime_seconds"`
	Checks        map[string]interface{} `json:"checks"`
}

// NewServer creates a new HTTP server
func NewServer(addr string) *Server {
	return &Server{
		addr: addr,
	}
}

// Start starts the HTTP server
func (s *Server) Start(startTime time.Time, natsConnected, jiraConnected bool) {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		checks := map[string]interface{}{
			"nats": natsConnected,
			"jira": jiraConnected,
		}

		// Determine overall status
		status := "healthy"
		if !natsConnected {
			status = "unhealthy"
		}

		response := HealthResponse{
			Status:        status,
			ServiceName:   "jira-service",
			Timestamp:     time.Now(),
			UptimeSeconds: int64(time.Since(startTime).Seconds()),
			Checks:        checks,
		}

		w.Header().Set("Content-Type", "application/json")
		if status == "unhealthy" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		json.NewEncoder(w).Encode(response)
	})

	// Readiness endpoint
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if natsConnected && jiraConnected {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "not ready"})
		}
	})

	// Liveness endpoint
	mux.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
	})

	// Service info endpoint
	mux.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		info := map[string]interface{}{
			"name":        "jira-service",
			"version":     "1.0.0",
			"description": "Jira integration service for workflow automation",
			"endpoints": []string{
				"GET /health - Health check",
				"GET /ready - Readiness check",
				"GET /live - Liveness check",
				"GET /info - Service information",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info)
	})

	s.server = &http.Server{
		Addr:         s.addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("Starting HTTP server on %s", s.addr)
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}

// Stop stops the HTTP server
func (s *Server) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

// NewMux creates a new HTTP mux with default handlers
func NewMux() *http.ServeMux {
	mux := http.NewServeMux()

	// Default handlers
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "jira-service is running")
	})

	return mux
}
