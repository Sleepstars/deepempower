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
	Logger *logger.Logger
}

// NewHybridPipeline creates a new hybrid pipeline with the specified configuration
func NewHybridPipeline(cfg *config.PipelineConfig) *HybridPipeline {
	// Initialize logger with default level
	logger.InitLogger(logger.INFO, "pipeline")
	log := logger.GetLogger().WithComponent("pipeline")
	log.Info("Creating new hybrid pipeline")

	// Create pipeline instance
	p := &HybridPipeline{
		config: cfg,
		Logger: log,
	}

	// Create model bridge if config is provided
	if cfg != nil {
		p.bridge = modelbridge.NewModelBridge(
			clients.ModelClientConfig{
				APIBase:       cfg.Models.Normal.APIBase,
				Model:         cfg.Models.Normal.Model,
				DefaultParams: cfg.Models.Normal.DefaultParams,
			},
			clients.ModelClientConfig{
				APIBase:        cfg.Models.Reasoner.APIBase,
				Model:          cfg.Models.Reasoner.Model,
				DisabledParams: cfg.Models.Reasoner.DisabledParams,
			},
		)

		// Initialize pipeline stages
		p.stages = []PipelineStage{
			newNormalPreprocessor(cfg.Prompts.PreProcess, p.bridge),
			newReasonerEngine(cfg.Prompts.Reasoning, p.bridge),
			newNormalPostprocessor(cfg.Prompts.PostProcess, p.bridge),
		}
	}

	return p
}

// SetBridge replaces the current model bridge with a new one (mainly for testing)
func (p *HybridPipeline) SetBridge(bridge *modelbridge.ModelBridge) {
	p.bridge = bridge
	if p.stages == nil {
		// Initialize stages for testing if they don't exist
		p.stages = []PipelineStage{
			newNormalPreprocessor("test_pre_process", bridge),
			newReasonerEngine("test_reasoning", bridge),
			newNormalPostprocessor("test_post_process", bridge),
		}
	} else {
		// Update bridge in existing stages
		for _, stage := range p.stages {
			if preprocessor, ok := stage.(*NormalPreprocessor); ok {
				preprocessor.bridge = bridge
			}
			if engine, ok := stage.(*ReasonerEngine); ok {
				engine.bridge = bridge
			}
			if postprocessor, ok := stage.(*NormalPostprocessor); ok {
				postprocessor.bridge = bridge
			}
		}
	}
}

// Execute runs the pipeline stages in sequence
func (p *HybridPipeline) Execute(ctx context.Context, req *models.ChatCompletionRequest) (*models.ChatCompletionResponse, error) {
	// Generate request ID if not provided
	if req.RequestID == "" {
		req.RequestID = fmt.Sprintf("req_%d", time.Now().UnixNano())
	}

	p.Logger.Info("Starting pipeline execution for request id: %s", req.RequestID)
	p.Logger.Debug("Request details: model=%s, stream=%v", req.Model, req.Stream)

	payload := &Payload{
		OriginalRequest: req,
		ReasoningChain:  make([]string, 0),
	}

	for _, stage := range p.stages {
		stageName := stage.Name()
		p.Logger.Debug("Executing stage: %s", stageName)

		select {
		case <-ctx.Done():
			p.Logger.Warn("Pipeline execution cancelled for request id: %s", req.RequestID)
			return nil, ctx.Err()
		default:
			if err := stage.Execute(ctx, payload); err != nil {
				p.Logger.WithError(err).Error("Stage %s failed for request id: %s", stageName, req.RequestID)
				return nil, fmt.Errorf("stage %s failed: %w", stageName, err)
			}
			p.Logger.Debug("Stage %s completed successfully", stageName)
		}
	}

	p.Logger.Info("Pipeline execution completed successfully for request id: %s", req.RequestID)
	return p.buildResponse(payload), nil
}

// buildResponse creates the final API response
func (p *HybridPipeline) buildResponse(payload *Payload) *models.ChatCompletionResponse {
	p.Logger.Debug("Building final response with content length: %d", len(payload.FinalContent))

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
