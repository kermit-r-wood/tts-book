package main

import (
	"embed"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"runtime"
	"time"
	"tts-book/backend/internal/api"
	"tts-book/backend/internal/config"

	"github.com/gin-gonic/gin"
)

//go:embed dist
var distFS embed.FS

func main() {
	// Set Release Mode to silence debug warnings
	gin.SetMode(gin.ReleaseMode)

	// Setup Logging
	f, _ := os.OpenFile("server.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	w := io.MultiWriter(f, os.Stdout)
	log.SetOutput(w)
	gin.DefaultWriter = w

	log.Println("----------------------------------------")
	log.Println("Starting TTS Book Application (Release Build)")
	log.Println("----------------------------------------")

	// Load Configuration
	cfg := config.Load()

	// Initialize Router
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.RedirectTrailingSlash = false
	r.RedirectFixedPath = false

	// Get the subtree "dist" because go:embed keeps the top dir
	frontendFS, err := fs.Sub(distFS, "dist")
	if err != nil {
		log.Printf("[WARNING] Frontend files not found in embed (dist): %v", err)
		frontendFS = nil
	} else {
		// Verify if index.html exists in the subtree
		if _, err := frontendFS.Open("index.html"); err != nil {
			log.Printf("[WARNING] index.html not found in embedded dist: %v", err)
			frontendFS = nil
		} else {
			log.Printf("[INFO] Frontend embedded filesystem loaded successfully.")
		}
	}

	// Setup Routes
	api.SetupRoutes(r, cfg, frontendFS)

	// Start Browser (in goroutine to not block)
	go func() {
		// Give server a moment to start
		time.Sleep(500 * time.Millisecond)
		openBrowser("http://localhost:8080")
	}()

	// Start Server
	log.Printf("Server starting on port 8080...")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// openBrowser opens the specified URL in the default browser of the user.
func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		// err = fmt.Errorf("unsupported platform")
		log.Printf("Unsupported platform for auto-opening browser")
	}
	if err != nil {
		log.Printf("Failed to open browser: %v", err)
	}
}
