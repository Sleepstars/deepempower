package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sleepstars/deepempower/internal/config"
	"github.com/sleepstars/deepempower/internal/models"
	"github.com/sleepstars/deepempower/internal/orchestrator"
)

func main() {
	// 解析命令行标志
	configPath := flag.String("config", "/app/config.yaml", "Path to the configuration file")
	flag.Parse()

	// Load configuration from file
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	// Create pipeline
	pipeline := orchestrator.NewHybridPipeline(cfg)

	// Setup router
	r := gin.Default()

	// Middleware to check API key
	r.Use(func(c *gin.Context) {
		apiKey := c.GetHeader("Authorization")
		if apiKey != cfg.APIKey {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	})

	// Chat completions endpoint
	r.POST("/v1/chat/completions", func(c *gin.Context) {
		var req models.ChatCompletionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		resp, err := pipeline.Execute(c.Request.Context(), &req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, resp)
	})

	// Start server
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
