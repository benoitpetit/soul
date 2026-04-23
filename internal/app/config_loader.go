// config_loader.go — YAML configuration loading for SOUL.
// Reads a soul config file (config.yaml or the path provided) and maps
// the relevant fields to SoulConfig.  Missing files are silently ignored;
// only parsing errors are returned.
package app

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// yamlFileConfig mirrors the relevant subset of config.example.yaml.
type yamlFileConfig struct {
	Soul struct {
		Storage struct {
			Path string `yaml:"path"`
		} `yaml:"storage"`
		DriftDetection struct {
			Threshold       float64 `yaml:"threshold"`
			WindowSize      int     `yaml:"window_size"`
			AutoCheckAfterCapture bool `yaml:"auto_check_after_capture"`
		} `yaml:"drift_detection"`
		Recall struct {
			DefaultBudgetTokens int `yaml:"default_budget_tokens"`
		} `yaml:"recall"`
		Extraction struct {
			MinTraitConfidence       float64 `yaml:"min_trait_confidence"`
			MinObservationsForTrait  int     `yaml:"min_observations_for_trait"`
		} `yaml:"extraction"`
		ModelSwap struct {
			AutoReinforce bool `yaml:"auto_reinforce"`
		} `yaml:"model_swap"`
		Evolution struct {
			Enabled          bool `yaml:"enabled"`
			MaxHistoryVersions int `yaml:"max_history_versions"`
		} `yaml:"evolution"`
		MCP struct {
			Enabled bool   `yaml:"enabled"`
			Host    string `yaml:"host"`
			Port    int    `yaml:"port"`
		} `yaml:"mcp"`
	} `yaml:"soul"`
}

// LoadConfigFile loads a YAML config file and returns a *SoulConfig.
// Fields not present in the file keep their zero value; callers should
// merge with DefaultConfig() beforehand.
// If the file does not exist, a nil config and no error are returned so
// callers can fall back to defaults gracefully.
func LoadConfigFile(path string) (*SoulConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // not found — use defaults
		}
		return nil, fmt.Errorf("reading config file %q: %w", path, err)
	}

	var fc yamlFileConfig
	if err := yaml.Unmarshal(data, &fc); err != nil {
		return nil, fmt.Errorf("parsing config file %q: %w", path, err)
	}

	cfg := &SoulConfig{}

	if fc.Soul.Storage.Path != "" {
		cfg.StoragePath = fc.Soul.Storage.Path
	}
	if fc.Soul.DriftDetection.Threshold > 0 {
		cfg.DriftThreshold = fc.Soul.DriftDetection.Threshold
	}
	if fc.Soul.DriftDetection.WindowSize > 0 {
		cfg.DriftWindowSize = fc.Soul.DriftDetection.WindowSize
	}
	if fc.Soul.DriftDetection.AutoCheckAfterCapture {
		cfg.AutoCheckAfterCapture = fc.Soul.DriftDetection.AutoCheckAfterCapture
	}
	if fc.Soul.Recall.DefaultBudgetTokens > 0 {
		cfg.MaxContextTokens = fc.Soul.Recall.DefaultBudgetTokens
	}
	if fc.Soul.Extraction.MinTraitConfidence > 0 {
		cfg.MinTraitConfidence = fc.Soul.Extraction.MinTraitConfidence
	}
	if fc.Soul.Extraction.MinObservationsForTrait > 0 {
		cfg.MinObservationsForTrait = fc.Soul.Extraction.MinObservationsForTrait
	}
	if fc.Soul.ModelSwap.AutoReinforce {
		cfg.AutoReinforce = fc.Soul.ModelSwap.AutoReinforce
	}
	if fc.Soul.Evolution.Enabled {
		cfg.EvolutionEnabled = fc.Soul.Evolution.Enabled
	}
	if fc.Soul.Evolution.MaxHistoryVersions > 0 {
		cfg.MaxHistoryVersions = fc.Soul.Evolution.MaxHistoryVersions
	}
	if fc.Soul.MCP.Enabled {
		cfg.MCPEnabled = fc.Soul.MCP.Enabled
		cfg.MCPHost = fc.Soul.MCP.Host
		cfg.MCPPort = fc.Soul.MCP.Port
	}

	return cfg, nil
}

// LoadConfig builds a SoulConfig by starting from DefaultConfig, then
// overlaying values from the YAML file (if it exists), and returning the
// result. CLI flags are applied by the caller after this returns.
func LoadConfig(filePath string) (*SoulConfig, error) {
	cfg := DefaultConfig()

	if filePath == "" {
		return cfg, nil
	}

	fileCfg, err := LoadConfigFile(filePath)
	if err != nil {
		return nil, err
	}
	if fileCfg == nil {
		// File absent — return defaults unchanged.
		return cfg, nil
	}

	// Overlay file values onto defaults.
	if fileCfg.StoragePath != "" {
		cfg.StoragePath = fileCfg.StoragePath
	}
	if fileCfg.DriftThreshold > 0 {
		cfg.DriftThreshold = fileCfg.DriftThreshold
	}
	if fileCfg.MaxContextTokens > 0 {
		cfg.MaxContextTokens = fileCfg.MaxContextTokens
	}
	if fileCfg.MinTraitConfidence > 0 {
		cfg.MinTraitConfidence = fileCfg.MinTraitConfidence
	}
	if fileCfg.MinObservationsForTrait > 0 {
		cfg.MinObservationsForTrait = fileCfg.MinObservationsForTrait
	}
	if fileCfg.DriftWindowSize > 0 {
		cfg.DriftWindowSize = fileCfg.DriftWindowSize
	}
	if fileCfg.AutoCheckAfterCapture {
		cfg.AutoCheckAfterCapture = fileCfg.AutoCheckAfterCapture
	}
	if fileCfg.AutoReinforce {
		cfg.AutoReinforce = fileCfg.AutoReinforce
	}
	if !fileCfg.EvolutionEnabled {
		cfg.EvolutionEnabled = fileCfg.EvolutionEnabled
	}
	if fileCfg.MaxHistoryVersions > 0 {
		cfg.MaxHistoryVersions = fileCfg.MaxHistoryVersions
	}
	if fileCfg.MCPEnabled {
		cfg.MCPEnabled = fileCfg.MCPEnabled
	}
	if fileCfg.MCPHost != "" {
		cfg.MCPHost = fileCfg.MCPHost
	}
	if fileCfg.MCPPort > 0 {
		cfg.MCPPort = fileCfg.MCPPort
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate checks that configuration values are within valid ranges.
func (c *SoulConfig) Validate() error {
	if c.DriftThreshold < 0 || c.DriftThreshold > 1 {
		return fmt.Errorf("drift_threshold must be between 0 and 1, got %v", c.DriftThreshold)
	}
	if c.MaxContextTokens <= 0 {
		return fmt.Errorf("max_context_tokens must be positive, got %d", c.MaxContextTokens)
	}
	if c.MinTraitConfidence < 0 || c.MinTraitConfidence > 1 {
		return fmt.Errorf("min_trait_confidence must be between 0 and 1, got %v", c.MinTraitConfidence)
	}
	if c.MinObservationsForTrait <= 0 {
		return fmt.Errorf("min_observations_for_trait must be positive, got %d", c.MinObservationsForTrait)
	}
	if c.DriftWindowSize <= 0 {
		return fmt.Errorf("drift_window_size must be positive, got %d", c.DriftWindowSize)
	}
	if c.MaxHistoryVersions <= 0 {
		return fmt.Errorf("max_history_versions must be positive, got %d", c.MaxHistoryVersions)
	}
	if c.MCPPort < 1 || c.MCPPort > 65535 {
		return fmt.Errorf("mcp_port must be between 1 and 65535, got %d", c.MCPPort)
	}
	return nil
}
