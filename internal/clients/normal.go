package clients

import (
	"context"
	"fmt"
	"strings"
	openai "github.com/sashabaranov/go-openai"
	"github.com/sleepstars/deepempower/internal/models"
)

// NormalClient implements ModelClient for the Normal (Claude) model
type NormalClient struct {
	config ModelClientConfig
	client *openai.Client
}

// NewNormalClient creates a new Normal model client
func NewNormalClient(config ModelClientConfig) *NormalClient {
	clientConfig := openai.DefaultConfig("")
	clientConfig.BaseURL = config.APIBase
	
	// Ensure URL has scheme
	if !strings.HasPrefix(clientConfig.BaseURL, "http://") && !strings.HasPrefix(clientConfig.BaseURL, "https://") {
		clientConfig.BaseURL = "http://" + clientConfig.BaseURL
	}
	
	return &NormalClient{
		config: config,
		client: openai.NewClientWithConfig(clientConfig),
	}
}

// Complete sends a non-streaming completion request
func (c *NormalClient) Complete(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
	// Prepare OpenAI request
	openaiReq, err := c.prepareRequest(req)
	if err != nil {
		return nil, err
	}

	// Call OpenAI API
	resp, err := c.client.CreateChatCompletion(ctx, openaiReq)
	if err != nil {
		return nil, fmt.Errorf("create chat completion: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	// Convert response
	return convertResponse(resp), nil
}

// CompleteStream sends a streaming completion request
func (c *NormalClient) CompleteStream(ctx context.Context, req *models.ChatCompletionRequest) (<-chan *models.ChatCompletionResponse, error) {
	// Prepare OpenAI request
	openaiReq, err := c.prepareRequest(req)
	if err != nil {
		return nil, err
	}
	openaiReq.Stream = true

	// Create stream
	stream, err := c.client.CreateChatCompletionStream(ctx, openaiReq)
	if err != nil {
		return nil, fmt.Errorf("create chat completion stream: %w", err)
	}

	resultChan := make(chan *models.ChatCompletionResponse)

	// Start streaming goroutine
	go func() {
		defer close(resultChan)
		defer stream.Close()

		var contentBuilder strings.Builder
		var partialContent []string

		for {
			select {
			case <-ctx.Done():
				return
			default:
				chunk, err := stream.Recv()
				if err != nil {
					return
				}

				if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
					content := chunk.Choices[0].Delta.Content
					contentBuilder.WriteString(content)
					partialContent = append(partialContent, content)

					resultChan <- &models.ChatCompletionResponse{
						Choices: []models.ChatCompletionChoice{
							{
								Message: models.ChatCompletionMessage{
									Role:    chunk.Choices[0].Delta.Role,
									Content: content,
								},
								FinishReason: string(chunk.Choices[0].FinishReason),
							},
						},
					}
				}
			}
		}
	}()

	return resultChan, nil
}

// Helper functions

// prepareRequest prepares an OpenAI request from our internal request format
func (c *NormalClient) prepareRequest(req *models.ChatCompletionRequest) (openai.ChatCompletionRequest, error) {
	// Set model from config if not specified
	if req.Model == "" {
		req.Model = c.config.Model
	}

	// Create OpenAI request
	openaiReq := openai.ChatCompletionRequest{
		Model:    req.Model,
		Messages: convertMessages(req.Messages),
	}

	// Apply default parameters
	applyDefaultParams(&openaiReq, c.config.DefaultParams)

	// Override with request parameters if provided
	if req.Temperature != 0 {
		openaiReq.Temperature = req.Temperature
	}
	if req.MaxTokens != 0 {
		openaiReq.MaxTokens = req.MaxTokens
	}

	return openaiReq, nil
}

// convertMessages converts our message format to OpenAI's format
func convertMessages(msgs []models.ChatCompletionMessage) []openai.ChatCompletionMessage {
	result := make([]openai.ChatCompletionMessage, len(msgs))
	for i, msg := range msgs {
		result[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return result
}

// convertResponse converts OpenAI's response to our format
func convertResponse(resp openai.ChatCompletionResponse) *models.ChatCompletionResponse {
	return &models.ChatCompletionResponse{
		Choices: []models.ChatCompletionChoice{
			{
				Message: models.ChatCompletionMessage{
					Role:    resp.Choices[0].Message.Role,
					Content: resp.Choices[0].Message.Content,
				},
				FinishReason: string(resp.Choices[0].FinishReason),
			},
		},
	}
}

// applyDefaultParams applies default parameters from config
func applyDefaultParams(req *openai.ChatCompletionRequest, params map[string]interface{}) {
	for k, v := range params {
		switch k {
		case "temperature":
			if v, ok := v.(float64); ok {
				req.Temperature = float32(v)
			}
		case "max_tokens":
			if v, ok := v.(int); ok {
				req.MaxTokens = v
			}
		}
	}
}
