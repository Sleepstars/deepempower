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
	NormalClient   clients.ModelClient
	ReasonerClient clients.ModelClient
	Logger         *logger.Logger // Changed to exported field
	mu             sync.RWMutex
}

// NewModelBridge creates a new model bridge instance
func NewModelBridge(normalCfg, reasonerCfg clients.ModelClientConfig) *ModelBridge {
	// Initialize logger with default level if not already initialized
	if logger.GetLogger() == nil {
		logger.InitLogger(logger.INFO, "model_bridge")
	}
	log := logger.GetLogger().WithComponent("model_bridge")
	log.Info("Creating new model bridge")

	return &ModelBridge{
		NormalClient:   clients.NewNormalClient(normalCfg),
		ReasonerClient: clients.NewReasonerClient(reasonerCfg),
		Logger:         log,
	}
}

// CallNormal sends a request to the Normal model
func (b *ModelBridge) CallNormal(ctx context.Context, req *models.ChatCompletionRequest) (resp *models.ChatCompletionResponse, err error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			b.Logger.Error("Recovered from panic in CallNormal: %v", r)
			err = fmt.Errorf("runtime error: %v", r)
			resp = nil
		}
	}()

	b.Logger.Debug("Calling Normal model with %d messages", len(req.Messages))

	resp, err = b.NormalClient.Complete(ctx, req)
	if err != nil {
		b.Logger.WithError(err).Error("Normal model call failed")
		return nil, err
	}

	b.Logger.Debug("Normal model call completed successfully")
	return resp, nil
}

// CallReasoner sends a request to the Reasoner model
func (b *ModelBridge) CallReasoner(ctx context.Context, req *models.ChatCompletionRequest) (resp *models.ChatCompletionResponse, err error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			b.Logger.Error("Recovered from panic in CallReasoner: %v", r)
			err = fmt.Errorf("runtime error: %v", r)
			resp = nil
		}
	}()

	b.Logger.Debug("Calling Reasoner model with %d messages", len(req.Messages))

	resp, err = b.ReasonerClient.Complete(ctx, req)
	if err != nil {
		b.Logger.WithError(err).Error("Reasoner model call failed")
		return nil, err
	}

	b.Logger.Debug("Reasoner model call completed successfully")
	return resp, nil
}

// CallReasonerStream sends a streaming request to the Reasoner model
func (b *ModelBridge) CallReasonerStream(ctx context.Context, req *models.ChatCompletionRequest) (<-chan *models.ChatCompletionResponse, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	b.Logger.Debug("Starting streaming call to Reasoner model with %d messages", len(req.Messages))

	// Ensure stream flag is set
	req.Stream = true

	respChan, err := b.ReasonerClient.CompleteStream(ctx, req)
	if err != nil {
		b.Logger.WithError(err).Error("Failed to start Reasoner model streaming")
		return nil, err // Don't wrap the error again
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

		b.Logger.Debug("Streaming completed: total=%d, content=%d, reasoning=%d",
			responseCount, contentCount, reasoningCount)
	}()

	return filteredChan, nil
}
