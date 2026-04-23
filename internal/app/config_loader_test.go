package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_NoFile(t *testing.T) {
	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("LoadConfig(\"\") error: %v", err)
	}
	if cfg == nil {
		t.Fatal("config must not be nil")
	}
	// Should equal defaults.
	def := DefaultConfig()
	if cfg.StoragePath != def.StoragePath {
		t.Errorf("StoragePath: got %q, want %q", cfg.StoragePath, def.StoragePath)
	}
	if cfg.DriftThreshold != def.DriftThreshold {
		t.Errorf("DriftThreshold: got %f, want %f", cfg.DriftThreshold, def.DriftThreshold)
	}
}

func TestLoadConfig_NonExistentFile(t *testing.T) {
	cfg, err := LoadConfig("/tmp/this_file_does_not_exist_soul_test.yaml")
	if err != nil {
		t.Fatalf("LoadConfig with missing file should not error: %v", err)
	}
	if cfg == nil {
		t.Fatal("config must not be nil (should fall back to defaults)")
	}
}

func TestLoadConfig_ValidYAML(t *testing.T) {
	yaml := `
soul:
  storage:
    path: "/tmp/test_soul.db"
  drift_detection:
    threshold: 0.42
  recall:
    default_budget_tokens: 500
`
	f, err := os.CreateTemp("", "soul_config_*.yaml")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	defer os.Remove(f.Name())
	f.WriteString(yaml)
	f.Close()

	cfg, err := LoadConfig(f.Name())
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if cfg.StoragePath != "/tmp/test_soul.db" {
		t.Errorf("StoragePath: got %q, want /tmp/test_soul.db", cfg.StoragePath)
	}
	if cfg.DriftThreshold != 0.42 {
		t.Errorf("DriftThreshold: got %f, want 0.42", cfg.DriftThreshold)
	}
	if cfg.MaxContextTokens != 500 {
		t.Errorf("MaxContextTokens: got %d, want 500", cfg.MaxContextTokens)
	}
}

func TestLoadConfig_PartialYAML(t *testing.T) {
	// Only storage path is set — other fields should keep defaults.
	yaml := `
soul:
  storage:
    path: "/custom/path.db"
`
	f, err := os.CreateTemp("", "soul_config_partial_*.yaml")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	defer os.Remove(f.Name())
	f.WriteString(yaml)
	f.Close()

	def := DefaultConfig()
	cfg, err := LoadConfig(f.Name())
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if cfg.StoragePath != "/custom/path.db" {
		t.Errorf("StoragePath: got %q", cfg.StoragePath)
	}
	if cfg.DriftThreshold != def.DriftThreshold {
		t.Errorf("DriftThreshold should keep default %f, got %f", def.DriftThreshold, cfg.DriftThreshold)
	}
	if cfg.MaxContextTokens != def.MaxContextTokens {
		t.Errorf("MaxContextTokens should keep default %d, got %d", def.MaxContextTokens, cfg.MaxContextTokens)
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	f, err := os.CreateTemp("", "soul_config_bad_*.yaml")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	defer os.Remove(f.Name())
	f.WriteString(": invalid: yaml: [unclosed")
	f.Close()

	_, err = LoadConfig(f.Name())
	if err == nil {
		t.Error("LoadConfig with invalid YAML should return error")
	}
}

func TestLoadConfigFile_EmptyYAML(t *testing.T) {
	f, err := os.CreateTemp("", "soul_config_empty_*.yaml")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	defer os.Remove(f.Name())
	f.Close()

	cfg, err := LoadConfigFile(f.Name())
	if err != nil {
		t.Fatalf("LoadConfigFile empty error: %v", err)
	}
	if cfg == nil {
		t.Fatal("empty YAML should still return a config struct")
	}
	// All fields are zero — nothing was set.
	if cfg.StoragePath != "" {
		t.Errorf("expected empty StoragePath, got %q", cfg.StoragePath)
	}
}

// TestLoadConfig_ExampleFile verifies the project's config.example.yaml is valid.
func TestLoadConfig_ExampleFile(t *testing.T) {
	// Walk up to repo root to find config.example.yaml.
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	examplePath := filepath.Join(root, "config.example.yaml")
	if _, err := os.Stat(examplePath); os.IsNotExist(err) {
		t.Skip("config.example.yaml not found, skipping")
	}

	cfg, err := LoadConfig(examplePath)
	if err != nil {
		t.Fatalf("LoadConfig(config.example.yaml) error: %v", err)
	}
	if cfg == nil {
		t.Fatal("config must not be nil")
	}
	if cfg.StoragePath == "" {
		t.Error("StoragePath should be set from example file")
	}
}
