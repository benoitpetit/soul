package entities

import (
	"strings"
	"testing"
)

func TestNewValueSystem_Defaults(t *testing.T) {
	vs := NewValueSystem()

	if vs.PrioritizesAccuracy != 0.7 {
		t.Errorf("PrioritizesAccuracy: got %f, want 0.7", vs.PrioritizesAccuracy)
	}
	if vs.PrioritizesHelpfulness != 0.8 {
		t.Errorf("PrioritizesHelpfulness: got %f, want 0.8", vs.PrioritizesHelpfulness)
	}
	if vs.DecisionPattern != DecisionAnalytical {
		t.Errorf("DecisionPattern: got %q, want %q", vs.DecisionPattern, DecisionAnalytical)
	}
	if vs.CoreValues == nil {
		t.Error("CoreValues should be initialized")
	}
}

func TestValueSystem_WithCoreValue(t *testing.T) {
	vs := NewValueSystem()
	vs.WithCoreValue("honesty", 0.9, ValueEpistemic)
	vs.WithCoreValue("fairness", 0.7, ValueMoral)

	if len(vs.CoreValues) != 2 {
		t.Errorf("CoreValues length: got %d, want 2", len(vs.CoreValues))
	}
	if vs.CoreValues[0].Name != "honesty" {
		t.Errorf("CoreValues[0].Name: got %q, want %q", vs.CoreValues[0].Name, "honesty")
	}
}

func TestValueSystem_WithCoreValue_ClampWeight(t *testing.T) {
	vs := NewValueSystem()
	vs.WithCoreValue("x", 2.0, ValueEpistemic)

	if vs.CoreValues[0].Weight != 1.0 {
		t.Errorf("Weight should be clamped to 1.0, got %f", vs.CoreValues[0].Weight)
	}
}

func TestValueSystem_WithStance(t *testing.T) {
	vs := NewValueSystem()
	vs.WithStance("open-source", "pro-open-source", 0.9)

	if len(vs.Stances) != 1 {
		t.Errorf("Stances length: got %d, want 1", len(vs.Stances))
	}
	if vs.Stances[0].Topic != "open-source" {
		t.Errorf("Stances[0].Topic: got %q", vs.Stances[0].Topic)
	}
}

func TestValueSystem_GetTopValues(t *testing.T) {
	vs := NewValueSystem()
	vs.WithCoreValue("honesty", 0.9, ValueEpistemic)
	vs.WithCoreValue("fairness", 0.5, ValueMoral)
	vs.WithCoreValue("creativity", 0.7, ValueAutonomy)

	top2 := vs.GetTopValues(2)
	if len(top2) != 2 {
		t.Errorf("GetTopValues(2): got %d, want 2", len(top2))
	}
	if top2[0].Name != "honesty" {
		t.Errorf("Top value should be honesty (0.9), got %q", top2[0].Name)
	}
	if top2[1].Name != "creativity" {
		t.Errorf("Second value should be creativity (0.7), got %q", top2[1].Name)
	}
}

func TestValueSystem_GetTopValues_LessThanN(t *testing.T) {
	vs := NewValueSystem()
	vs.WithCoreValue("honesty", 0.9, ValueEpistemic)

	top5 := vs.GetTopValues(5)
	if len(top5) != 1 {
		t.Errorf("GetTopValues(5) with 1 value: got %d", len(top5))
	}
}

func TestValueSystem_ToNaturalDescription_ContainsPriorities(t *testing.T) {
	vs := NewValueSystem()
	desc := vs.ToNaturalDescription()

	if !strings.Contains(desc, "prioritize") {
		t.Errorf("Description should mention priorities, got: %q", desc)
	}
}

func TestValueSystem_ToNaturalDescription_WithCoreValues(t *testing.T) {
	vs := NewValueSystem()
	vs.WithCoreValue("honesty", 0.9, ValueEpistemic)

	desc := vs.ToNaturalDescription()
	if !strings.Contains(desc, "honesty") {
		t.Errorf("Description should mention core values, got: %q", desc)
	}
}
