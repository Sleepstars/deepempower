package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary test config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.yaml")

	testConfig := `models:
  Normal:
    api_base: "http://localhost:8001"
    model: "gpt-3.5-turbo"
    default_params:
      temperature: 0.7
      max_tokens: 1000
  Reasoner:
    api_base: "http://localhost:8002"
    model: "gpt-4"
    disabled_params:
      - "temperature"
      - "presence_penalty"
      - "frequency_penalty"

prompts:
  pre_process: "Analyze the following request: {{.UserInput}}"
  reasoning: "Think step by step about: {{.StructuredInput}}"
  post_process: "Summarize the reasoning: {{.ReasoningChain}}"
`

	err := os.WriteFile(configPath, []byte(testConfig), 0644)
	assert.NoError(t, err)

	// Test successful config loading
	cfg, err := LoadConfig(configPath)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	// Verify Normal model config
	assert.Equal(t, "http://localhost:8001", cfg.Models.Normal.APIBase, "Normal APIBase mismatch")
	assert.Equal(t, "gpt-3.5-turbo", cfg.Models.Normal.Model, "Normal Model mismatch")
	assert.Equal(t, 0.7, cfg.Models.Normal.DefaultParams["temperature"], "Normal temperature param mismatch")
	if maxTokens, ok := cfg.Models.Normal.DefaultParams["max_tokens"].(float64); ok {
		assert.Equal(t, 1000, int(maxTokens), "Normal max_tokens param mismatch")
	} else if maxTokens, ok := cfg.Models.Normal.DefaultParams["max_tokens"].(int); ok {
		assert.Equal(t, 1000, maxTokens, "Normal max_tokens param mismatch")
	} else {
		assert.Fail(t, "max_tokens param type mismatch")
	}

	// Verify Reasoner model config
	assert.Equal(t, "http://localhost:8002", cfg.Models.Reasoner.APIBase, "Reasoner APIBase mismatch")
	assert.Equal(t, "gpt-4", cfg.Models.Reasoner.Model, "Reasoner Model mismatch")
	assert.Contains(t, cfg.Models.Reasoner.DisabledParams, "temperature", "Missing temperature in disabled params")
	assert.Contains(t, cfg.Models.Reasoner.DisabledParams, "presence_penalty", "Missing presence_penalty in disabled params")

	// Verify prompts config
	assert.Contains(t, cfg.Prompts.PreProcess, "{{.UserInput}}", "PreProcess template mismatch")
	assert.Contains(t, cfg.Prompts.Reasoning, "{{.StructuredInput}}", "Reasoning template mismatch")
	assert.Contains(t, cfg.Prompts.PostProcess, "{{.ReasoningChain}}", "PostProcess template mismatch")

	// Test error cases
	t.Run("NonexistentFile", func(t *testing.T) {
		_, err := LoadConfig("nonexistent.yaml")
		assert.Error(t, err)
	})

	t.Run("InvalidYAML", func(t *testing.T) {
		invalidPath := filepath.Join(tmpDir, "invalid.yaml")
		err := os.WriteFile(invalidPath, []byte("invalid: yaml: {content"), 0644)
		assert.NoError(t, err)

		_, err = LoadConfig(invalidPath)
		assert.Error(t, err)
	})

	t.Run("EmptyFile", func(t *testing.T) {
		emptyPath := filepath.Join(tmpDir, "empty.yaml")
		err := os.WriteFile(emptyPath, []byte{}, 0644)
		assert.NoError(t, err)

		cfg, err := LoadConfig(emptyPath)
		assert.NoError(t, err)
		assert.Empty(t, cfg.Models.Normal.APIBase)
	})
}

func TestModelConfig(t *testing.T) {
	cfg := ModelConfig{
		APIBase: "http://test",
		Model:   "test-model",
		DefaultParams: map[string]interface{}{
			"temperature": 0.8,
			"top_p":       1.0,
		},
		DisabledParams: []string{"presence_penalty"},
	}

	assert.Equal(t, "http://test", cfg.APIBase)
	assert.Equal(t, "test-model", cfg.Model)
	assert.Equal(t, 0.8, cfg.DefaultParams["temperature"])
	assert.Equal(t, 1.0, cfg.DefaultParams["top_p"])
	assert.Contains(t, cfg.DisabledParams, "presence_penalty")
}

func TestPromptsConfig(t *testing.T) {
	cfg := PromptsConfig{
		PreProcess:  "test pre {{.Var}}",
		Reasoning:   "test reasoning {{.Var}}",
		PostProcess: "test post {{.Var}}",
	}

	assert.Contains(t, cfg.PreProcess, "{{.Var}}")
	assert.Contains(t, cfg.Reasoning, "{{.Var}}")
	assert.Contains(t, cfg.PostProcess, "{{.Var}}")
}

func TestModelsConfig(t *testing.T) {
	cfg := ModelsConfig{
		Normal: ModelConfig{
			APIBase: "http://normal",
			Model:   "normal-model",
		},
		Reasoner: ModelConfig{
			APIBase: "http://reasoner",
			Model:   "reasoner-model",
		},
	}

	assert.Equal(t, "http://normal", cfg.Normal.APIBase)
	assert.Equal(t, "normal-model", cfg.Normal.Model)
	assert.Equal(t, "http://reasoner", cfg.Reasoner.APIBase)
	assert.Equal(t, "reasoner-model", cfg.Reasoner.Model)
}
