package app

import (
	"os"
	"testing"
)

func TestLoadConfig_BoolOverlay(t *testing.T) {
	yaml := `
soul:
  drift_detection:
    threshold: 0.5
    window_size: 20
    auto_check_after_capture: false
  model_swap:
    auto_reinforce: false
  evolution:
    enabled: false
    max_history_versions: 50
  mcp:
    enabled: false
    host: "0.0.0.0"
    port: 9090
`
	tmpfile, err := os.CreateTemp("", "soul-config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.WriteString(yaml); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	cfg, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if cfg.DriftThreshold != 0.5 {
		t.Errorf("DriftThreshold = %.1f, want 0.5", cfg.DriftThreshold)
	}
	if cfg.DriftWindowSize != 20 {
		t.Errorf("DriftWindowSize = %d, want 20", cfg.DriftWindowSize)
	}
	if cfg.AutoCheckAfterCapture != false {
		t.Errorf("AutoCheckAfterCapture = %v, want false", cfg.AutoCheckAfterCapture)
	}
	if cfg.AutoReinforce != false {
		t.Errorf("AutoReinforce = %v, want false", cfg.AutoReinforce)
	}
	if cfg.EvolutionEnabled != false {
		t.Errorf("EvolutionEnabled = %v, want false", cfg.EvolutionEnabled)
	}
	if cfg.MaxHistoryVersions != 50 {
		t.Errorf("MaxHistoryVersions = %d, want 50", cfg.MaxHistoryVersions)
	}
	if cfg.MCPEnabled != false {
		t.Errorf("MCPEnabled = %v, want false", cfg.MCPEnabled)
	}
	if cfg.MCPHost != "0.0.0.0" {
		t.Errorf("MCPHost = %s, want 0.0.0.0", cfg.MCPHost)
	}
	if cfg.MCPPort != 9090 {
		t.Errorf("MCPPort = %d, want 9090", cfg.MCPPort)
	}
}
