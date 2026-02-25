package api

import (
	"io/fs"
	"net/http"
	"strings"
	"tts-book/backend/internal/config"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, cfg *config.Config, staticFS fs.FS) {
	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	api := r.Group("/api")
	{
		api.GET("/config", GetConfig(cfg))
		api.POST("/config", UpdateConfig(cfg))
		api.POST("/upload", UploadEPUB)
		api.POST("/analyze/:chapterID", AnalyzeChapter)
		api.POST("/analyze-all", AnalyzeAllChapters)

		api.GET("/characters", GetCharacters)
		api.POST("/characters/merge", MergeCharacters)
		api.POST("/characters/update", UpdateCharacter)
		api.POST("/confirm-mapping", ConfirmMapping)

		api.POST("/generate/:chapterID", GenerateAudio)
		api.POST("/generate-all", GenerateAllAudio)
		api.GET("/audio-status/:chapterID", GetAudioStatus)
		api.GET("/browse", BrowseFiles)
		api.GET("/voices/list", ListConfiguredVoices(cfg))
		api.GET("/voices/preview", PreviewVoice)
		api.GET("/llm/models", ListLLMModels(cfg))
		api.GET("/ws", WsHandler)
	}

	// Serve generated audio
	r.Static("/output", "data/out")

	// Serve Frontend Static Files (SPA Support)
	if staticFS != nil {
		// Pre-read index.html to serve directly (avoids any FileServer redirect magic)
		indexData, err := fs.ReadFile(staticFS, "index.html")
		if err != nil {
			// This should be caught by main.go checks, but just in case
			indexData = []byte("<h1>Error: index.html not found</h1>")
		}

		// Helper to serve index.html
		serveIndex := func(c *gin.Context) {
			c.Data(200, "text/html; charset=utf-8", indexData)
		}

		// Explicitly serve index.html at root to prevent 301 redirects
		r.GET("/", serveIndex)
		// Also serve /index.html as the same thing
		r.GET("/index.html", serveIndex)

		// Serve index.html for root and unknown routes (SPA fallback)
		// We use NoRoute but exclude API paths
		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			// Don't fallback for API or output
			if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/output") {
				c.JSON(404, gin.H{"error": "Not Found"})
				return
			}

			// Try to serve the exact file first (e.g. favicon.ico, vite.svg)
			// Note: Gin's StaticFS at /assets handles assets, but root files need help or a root StaticFS.
			// Since we want SPA fallback, we can't just StaticFS("/") because it would verify file existence for every route.
			// However, http.FileServer does that.

			// Simple approach: Use http.FileServer for everything else?
			// If we use NoRoute only, valid files at root (vite.svg) won't be served unless we catch them.

			// Let's manually check if the file exists in the FS
			file, err := staticFS.Open(strings.TrimPrefix(path, "/"))
			if err == nil {
				file.Close()
				c.FileFromFS(path, http.FS(staticFS))
				return
			}

			// Fallback to index.html
			c.FileFromFS("index.html", http.FS(staticFS))
		})
	}
}
