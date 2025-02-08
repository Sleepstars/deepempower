package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/codeium/deepempower/internal/clients"
	"github.com/codeium/deepempower/internal/config"
	"github.com/codeium/deepempower/internal/logger"
	"github.com/codeium/deepempower/internal/modelbridge"
	"github.com/codeium/deepempower/internal/models"
)

// Payload represents the data passed between pipeline stages
type Payload struct {
	OriginalRequest *models.ChatCompletionRequest
	ReasoningChain  []string
	IntermContent   string
	FinalContent    string
	Error           error
	mux             sync.RWMutex
}

// PipelineStage defines the interface for a stage in the processing pipeline
type PipelineStage interface {
	Execute(ctx context.Context, data *Payload) error
	Name() string
}

// HybridPipeline implements a pipeline that combines Normal and Reasoner models
type HybridPipeline struct {
	stages []PipelineStage
	config *config.PipelineConfig
	bridge *modelbridge.ModelBridge
	logger *logger.Logger
}

// NewHybridPipeline creates a new hybrid pipeline with the specified configuration
func NewHybridPipeline(cfg *config.PipelineConfig) *HybridPipeline {
	// Initialize logger
	log := logger.GetLogger().WithComponent("pipeline")
	log.Info("Creating new hybrid pipeline")

	// Create model bridge
	bridge := modelbridge.NewModelBridge(
		clients.ModelClientConfig{
			APIBase:       cfg.Models.Normal.APIBase,
			DefaultParams: cfg.Models.Normal.DefaultParams,
		},
		clients.ModelClientConfig{
			APIBase:        cfg.Models.Reasoner.APIBase,
			DisabledParams: cfg.Models.Reasoner.DisabledParams,
		},
	)

	log.Debug("Model bridge created with configs: normal=%v, reasoner=%v", 
		cfg.Models.Normal, cfg.Models.Reasoner)

	return &HybridPipeline{
		config: cfg,
		bridge: bridge,
		logger: log,
		stages: []PipelineStage{
			newNormalPreprocessor(cfg.Prompts.PreProcess, bridge),
			newReasonerEngine(cfg.Prompts.Reasoning, bridge),
			newNormalPostprocessor(cfg.Prompts.PostProcess, bridge),
		},
	}
}

// Execute runs the pipeline stages in sequence
func (p *HybridPipeline) Execute(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
	// Generate request ID if not provided
	if req.RequestID == "" {
		req.RequestID = fmt.Sprintf("req_%d", time.Now().UnixNano())
	}

	p.logger.Info("Starting pipeline execution for request id: %s", req.RequestID)
	p.logger.Debug("Request details: model=%s, stream=%v", req.Model, req.Stream)

	payload := &Payload{
		OriginalRequest: req,
		ReasoningChain:  make([]string, 0),
	}

	for _, stage := range p.stages {
		stageName := stage.Name()
		p.logger.Debug("Executing stage: %s", stageName)

		select {
		case <-ctx.Done():
			p.logger.Warn("Pipeline execution cancelled for request id: %s", req.RequestID)
			return nil, ctx.Err()
		default:
			if err := stage.Execute(ctx, payload); err != nil {
				p.logger.WithError(err).Error("Stage %s failed for request id: %s", stageName, req.RequestID)
				return nil, fmt.Errorf("stage %s failed: %w", stageName, err)
			}
			p.logger.Debug("Stage %s completed successfully", stageName)
		}
	}

	p.logger.Info("Pipeline execution completed successfully for request id: %s", req.RequestID)
	return p.buildResponse(payload), nil
}

// buildResponse creates the final API response
func (p *HybridPipeline) buildResponse(payload *Payload) *models.ChatCompletionResponse {
	p.logger.Debug("Building final response with content length: %d", len(payload.FinalContent))

	return &models.ChatCompletionResponse{
		Choices: []models.ChatCompletionChoice{
			{
				Message: models.ChatCompletionMessage{
					Role:             "assistant",
					Content:          payload.FinalContent,
					ReasoningContent: payload.ReasoningChain,
				},
				FinishReason: "stop",
			},
		},
	}
}
