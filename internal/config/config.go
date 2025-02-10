package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// PipelineConfig represents the configuration for a processing pipeline
type PipelineConfig struct {
	Prompts PromptsConfig `yaml:"prompts"`
	Models  ModelsConfig  `yaml:"models"`
	APIKey  string        `yaml:"api_key"`
}

// PromptsConfig contains prompt templates for different stages
type PromptsConfig struct {
	PreProcess  string `yaml:"pre_process"`
	Reasoning   string `yaml:"reasoning"`
	PostProcess string `yaml:"post_process"`
}

// ModelsConfig contains configurations for different models
type ModelsConfig struct {
	Normal   ModelConfig `yaml:"Normal"`
	Reasoner ModelConfig `yaml:"Reasoner"`
}

// ModelConfig contains configuration for a specific model
type ModelConfig struct {
	APIBase        string                 `yaml:"api_base"`
	Model          string                 `yaml:"model"`
	DefaultParams  map[string]interface{} `yaml:"default_params,omitempty"`
	DisabledParams []string               `yaml:"disabled_params,omitempty"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*PipelineConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg PipelineConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
