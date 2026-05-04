package interactors

import (
	"encoding/json"
	"testing"
)

func TestGetStringSlice_InterfaceSlice(t *testing.T) {
	metrics := map[string]interface{}{
		"preferred_tools": []interface{}{"bash", "read", "write"},
	}
	result := getStringSlice(metrics, "preferred_tools")
	if len(result) != 3 {
		t.Errorf("expected 3 items, got %d", len(result))
	}
	if result[0] != "bash" {
		t.Errorf("expected 'bash', got %s", result[0])
	}
}

func TestGetStringSlice_StringSlice(t *testing.T) {
	metrics := map[string]interface{}{
		"preferred_tools": []string{"bash", "read"},
	}
	result := getStringSlice(metrics, "preferred_tools")
	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}
}

func TestGetStringSlice_Missing(t *testing.T) {
	metrics := map[string]interface{}{}
	result := getStringSlice(metrics, "preferred_tools")
	if len(result) != 0 {
		t.Errorf("expected 0 items, got %d", len(result))
	}
}

func TestGetFloat64_Float64(t *testing.T) {
	metrics := map[string]interface{}{
		"success_rate": float64(0.85),
	}
	result := getFloat64(metrics, "success_rate")
	if result != 0.85 {
		t.Errorf("expected 0.85, got %f", result)
	}
}

func TestGetFloat64_JsonNumber(t *testing.T) {
	metrics := map[string]interface{}{
		"success_rate": json.Number("0.75"),
	}
	result := getFloat64(metrics, "success_rate")
	if result != 0.75 {
		t.Errorf("expected 0.75, got %f", result)
	}
}
