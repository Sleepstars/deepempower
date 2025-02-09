package main

import (
	"log"
	"net/http"

	"github.com/codeium/deepempower/internal/config"
	"github.com/codeium/deepempower/internal/models"
	"github.com/codeium/deepempower/internal/orchestrator"
	"github.com/gin-gonic/gin"
)

func main() {
	// TODO: Load configuration from file
	cfg := &config.PipelineConfig{}

	// Create pipeline
	pipeline := orchestrator.NewHybridPipeline(cfg)

	// Setup router
	r := gin.Default()

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
