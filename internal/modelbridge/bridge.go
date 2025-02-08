package modelbridge

import (
	"context"
	"fmt"
	"sync"

	"github.com/codeium/deepempower/internal/clients"
	"github.com/codeium/deepempower/internal/logger"
	"github.com/codeium/deepempower/internal/models"
)

// ModelBridge handles model-specific logic and provides a unified interface
type ModelBridge struct {
	normalClient   clients.ModelClient
	reasonerClient clients.ModelClient
	logger         *logger.Logger
	mu             sync.RWMutex
}

// NewModelBridge creates a new model bridge instance
func NewModelBridge(normalCfg, reasonerCfg clients.ModelClientConfig) *ModelBridge {
	log := logger.GetLogger().WithComponent("model_bridge")
	log.Info("Creating new model bridge")

	return &ModelBridge{
		normalClient:   clients.NewNormalClient(normalCfg),
		reasonerClient: clients.NewReasonerClient(reasonerCfg),
		logger:         log,
	}
}

// CallNormal sends a request to the Normal model
func (b *ModelBridge) CallNormal(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Ensure model name is correct
	req.Model = "Normal"
	b.logger.Debug("Calling Normal model with %d messages", len(req.Messages))
	
	resp, err := b.normalClient.Complete(ctx, req)
	if err != nil {
		b.logger.WithError(err).Error("Normal model call failed")
		return nil, err
	}

	b.logger.Debug("Normal model call completed successfully")
	return resp, nil
}

// CallReasoner sends a request to the Reasoner model
func (b *ModelBridge) CallReasoner(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Ensure model name is correct
	req.Model = "Reasoner"
	b.logger.Debug("Calling Reasoner model with %d messages", len(req.Messages))
	
	resp, err := b.reasonerClient.Complete(ctx, req)
	if err != nil {
		b.logger.WithError(err).Error("Reasoner model call failed")
		return nil, err
	}

	b.logger.Debug("Reasoner model call completed successfully")
	return resp, nil
}

// CallReasonerStream sends a streaming request to the Reasoner model
func (b *ModelBridge) CallReasonerStream(ctx context.Context, req *models.ChatCompletionRequest) (<-chan *models.ChatCompletionResponse, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Ensure model name and streaming flag are correct
	req.Model = "Reasoner"
	req.Stream = true

	b.logger.Debug("Starting streaming call to Reasoner model with %d messages", len(req.Messages))
	respChan, err := b.reasonerClient.CompleteStream(ctx, req)
	if err != nil {
		b.logger.WithError(err).Error("Failed to start Reasoner model streaming")
		return nil, fmt.Errorf("start stream: %w", err)
	}

	// Create a new channel for filtered responses
	filteredChan := make(chan *models.ChatCompletionResponse)

	// Start goroutine to process responses
	go func() {
		defer close(filteredChan)
		responseCount := 0
		contentCount := 0
		reasoningCount := 0

		for resp := range respChan {
			responseCount++
			// Only forward responses that have content
			if resp != nil && len(resp.Choices) > 0 {
				hasContent := len(resp.Choices[0].Message.Content) > 0
				hasReasoning := len(resp.Choices[0].Message.ReasoningContent) > 0

				if hasContent {
					contentCount++
				}
				if hasReasoning {
					reasoningCount++
				}

				if hasContent || hasReasoning {
					filteredChan <- resp
				}
			}
		}

		b.logger.Debug("Streaming completed: total=%d, content=%d, reasoning=%d",
			responseCount, contentCount, reasoningCount)
	}()

	return filteredChan, nil
}
