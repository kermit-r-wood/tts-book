package api

import (
	"tts-book/backend/internal/config"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, cfg *config.Config) {
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

		api.GET("/characters", GetCharacters)
		api.POST("/confirm-mapping", ConfirmMapping)

		api.POST("/generate/:chapterID", GenerateAudio)
		api.GET("/browse", BrowseFiles)
		api.GET("/voices/list", ListConfiguredVoices(cfg))
		api.GET("/llm/models", ListLLMModels(cfg))
		api.GET("/ws", WsHandler)
	}

	// Serve generated audio
	r.Static("/output", "data/out")
}
