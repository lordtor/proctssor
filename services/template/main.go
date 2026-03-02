// BPMN Workflow Platform - Microservice Template
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	serviceName    = "Service Template"
	serviceVersion = "1.0.0"
	defaultPort    = "8081"
)

func main() {
	port := getEnv("PORT", defaultPort)

	log.Printf("Starting %s v%s", serviceName, serviceVersion)
	log.Printf("Listening on port %s", port)

	router := gin.Default()
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": serviceName,
			"version": serviceVersion,
			"time":    time.Now().UTC(),
		})
	})

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	if err := srv.Shutdown(nil); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	fmt.Println("Service stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
