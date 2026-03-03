// Package config provides global configuration loading for ForgeX.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds all ForgeX configuration.
type Config struct {
	// General
	LogLevel string `mapstructure:"log_level"`
	DevMode  bool   `mapstructure:"dev_mode"`

	// LLM
	LLM LLMConfig `mapstructure:"llm"`

	// Sandbox
	Sandbox SandboxConfig `mapstructure:"sandbox"`

	// Governance
	Governance GovernanceConfig `mapstructure:"governance"`
}

// LLMConfig holds LLM routing configuration.
type LLMConfig struct {
	Provider    string  `mapstructure:"provider"`    // "litellm", "ollama", "claude"
	Endpoint    string  `mapstructure:"endpoint"`    // API endpoint
	APIKey      string  `mapstructure:"api_key"`     // API key (for cloud providers)
	Model       string  `mapstructure:"model"`       // Default model name
	MaxTokens   int     `mapstructure:"max_tokens"`  // Max tokens per request
	Temperature float64 `mapstructure:"temperature"` // Default temperature

	// Router enables multi-model intelligent routing.
	Router *RouterConfig `mapstructure:"router"`
}

// RouterConfig defines multi-model routing.
type RouterConfig struct {
	Strategy string        `mapstructure:"strategy"` // "gear", "cheapest", "fallback"
	Models   []ModelConfig `mapstructure:"models"`
}

// ModelConfig defines a single model endpoint.
type ModelConfig struct {
	Name     string `mapstructure:"name"`
	Endpoint string `mapstructure:"endpoint"`
	APIKey   string `mapstructure:"api_key"`
	Tier     string `mapstructure:"tier"` // "high", "low"
}

// SandboxConfig holds sandbox execution configuration.
type SandboxConfig struct {
	Backend    string `mapstructure:"backend"`     // "exec" (Phase 0~3) or "wasm" (Phase 4+)
	TimeoutSec int    `mapstructure:"timeout_sec"` // Max execution time in seconds
	MemoryMB   int    `mapstructure:"memory_mb"`   // Max memory in MB
}

// GovernanceConfig holds safety and cost configuration.
type GovernanceConfig struct {
	AutoApproveLevel string  `mapstructure:"auto_approve_level"` // "green", "yellow"
	MaxBudget        float64 `mapstructure:"max_budget"`         // Max budget in USD
}

// Load reads configuration from file, env, and flags.
func Load() (*Config, error) {
	v := viper.New()

	// Config file
	home, _ := os.UserHomeDir()
	v.SetConfigName("forgex")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath(filepath.Join(home, ".forgex"))

	// Defaults
	v.SetDefault("log_level", "info")
	v.SetDefault("dev_mode", true)
	v.SetDefault("llm.provider", "litellm")
	v.SetDefault("llm.endpoint", "http://localhost:4000")
	v.SetDefault("llm.model", "gpt-4o")
	v.SetDefault("llm.max_tokens", 4096)
	v.SetDefault("llm.temperature", 0.7)
	v.SetDefault("sandbox.backend", "exec")
	v.SetDefault("sandbox.timeout_sec", 30)
	v.SetDefault("sandbox.memory_mb", 512)
	v.SetDefault("governance.auto_approve_level", "yellow")
	v.SetDefault("governance.max_budget", 10.0)

	// Environment variables
	v.SetEnvPrefix("FORGEX")
	v.AutomaticEnv()

	// Read config file (optional)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
