package main

import (
	"io"
	"log"
	"os"
	"tts-book/backend/internal/api"
	"tts-book/backend/internal/config"

	"github.com/gin-gonic/gin"
)

func main() {
	// Setup Logging
	f, _ := os.OpenFile("server.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	w := io.MultiWriter(f, os.Stdout)
	log.SetOutput(w)
	gin.DefaultWriter = w

	// Load Configuration
	cfg := config.Load()

	// Initialize Router
	r := gin.Default()

	// Setup Routes
	api.SetupRoutes(r, cfg)

	// Start Server
	log.Printf("Server starting on port 8080...")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
