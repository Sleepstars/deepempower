package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/codeium/deepempower/internal/models"
)

func main() {
	port := flag.String("port", "8001", "Port to run the server on")
	flag.Parse()

	r := gin.Default()

	// Chat completions endpoint
	r.POST("/v1/chat/completions", func(c *gin.Context) {
		var req models.ChatCompletionRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Mock response based on the model
		var content string
		switch req.Model {
		case "Normal":
			content = "This is a response from the Normal model"
		case "Reasoner":
			content = "This is a response from the Reasoner model with reasoning steps"
		default:
			content = "Unknown model"
		}

		resp := &models.ChatCompletionResponse{
			Choices: []models.ChatCompletionChoice{
				{
					Message: models.ChatCompletionMessage{
						Role:    "assistant",
						Content: content,
					},
					FinishReason: "stop",
				},
			},
		}

		c.JSON(http.StatusOK, resp)
	})

	// Start server
	if err := r.Run(":" + *port); err != nil {
		log.Fatal(err)
	}
}
