package integration

import (
	"context"
	"testing"
	"time"

	"github.com/codeium/deepempower/internal/config"
	"github.com/codeium/deepempower/internal/models"
	"github.com/codeium/deepempower/internal/orchestrator"
	"github.com/stretchr/testify/assert"
)

func TestPipelineIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load test configuration
	cfg := &config.PipelineConfig{
		Models: config.ModelsConfig{
			Normal: config.ModelConfig{
				APIBase: "http://localhost:8001",
				DefaultParams: map[string]interface{}{
					"temperature": 0.7,
					"max_tokens": 1000,
				},
			},
			Reasoner: config.ModelConfig{
				APIBase: "http://localhost:8002",
				DisabledParams: []string{
					"temperature",
					"presence_penalty",
					"frequency_penalty",
				},
			},
		},
		Prompts: config.PromptsConfig{
			PreProcess: `
				You are a preprocessing agent. 
				Analyze the following user input and structure it:
				${input}
			`,
			Reasoning: `
				You are a reasoning agent.
				Break down the problem and solve it step by step:
				${input}
			`,
			PostProcess: `
				You are a postprocessing agent.
				Based on the reasoning chain and intermediate results,
				generate a clear and concise response:
				Reasoning: ${reasoning_chain}
				Result: ${intermediate_result}
			`,
		},
	}

	// Create pipeline
	pipeline := orchestrator.NewHybridPipeline(cfg)

	// Test cases
	testCases := []struct {
		name     string
		input    string
		timeout  time.Duration
		validate func(t *testing.T, resp *models.ChatCompletionResponse, err error)
	}{
		{
			name:    "Simple question",
			input:   "What is 2+2?",
			timeout: 10 * time.Second,
			validate: func(t *testing.T, resp *models.ChatCompletionResponse, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Choices)
				assert.NotEmpty(t, resp.Choices[0].Message.Content)
				assert.NotEmpty(t, resp.Choices[0].Message.ReasoningContent)
			},
		},
		{
			name:    "Complex question",
			input:   "Explain how a car engine works",
			timeout: 30 * time.Second,
			validate: func(t *testing.T, resp *models.ChatCompletionResponse, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.Choices)
				assert.NotEmpty(t, resp.Choices[0].Message.Content)
				assert.True(t, len(resp.Choices[0].Message.ReasoningContent) > 2)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()

			req := &models.ChatCompletionRequest{
				Messages: []models.ChatCompletionMessage{
					{Role: "user", Content: tc.input},
				},
			}

			resp, err := pipeline.Execute(ctx, req)
			tc.validate(t, resp, err)
		})
	}
}
