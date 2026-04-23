package entities

import (
	"strings"
	"testing"
)

func TestNewPersonalityTrait_Defaults(t *testing.T) {
	trait := NewPersonalityTrait("analytical", TraitCognitive, 0.8)

	if trait.Name != "analytical" {
		t.Errorf("Name: got %q, want %q", trait.Name, "analytical")
	}
	if trait.Category != TraitCognitive {
		t.Errorf("Category: got %q, want %q", trait.Category, TraitCognitive)
	}
	if trait.Intensity != 0.8 {
		t.Errorf("Intensity: got %f, want 0.8", trait.Intensity)
	}
	if trait.Confidence != 0.3 {
		t.Errorf("Initial Confidence: got %f, want 0.3", trait.Confidence)
	}
	if trait.EvidenceCount != 1 {
		t.Errorf("EvidenceCount: got %d, want 1", trait.EvidenceCount)
	}
}

func TestNewPersonalityTrait_ClampIntensity(t *testing.T) {
	over := NewPersonalityTrait("x", TraitCognitive, 1.5)
	if over.Intensity != 1.0 {
		t.Errorf("Intensity clamped to 1.0: got %f", over.Intensity)
	}

	under := NewPersonalityTrait("x", TraitCognitive, -0.5)
	if under.Intensity != 0.0 {
		t.Errorf("Intensity clamped to 0.0: got %f", under.Intensity)
	}
}

func TestWithEvidence_IncreasesConfidence(t *testing.T) {
	trait := NewPersonalityTrait("analytical", TraitCognitive, 0.8)
	initial := trait.Confidence

	for i := 0; i < 5; i++ {
		trait.WithEvidence("some evidence text", "code_review")
	}

	if trait.Confidence <= initial {
		t.Errorf("Confidence should increase with evidence: initial=%f final=%f", initial, trait.Confidence)
	}
	if trait.EvidenceCount < 6 {
		t.Errorf("EvidenceCount should be >= 6, got %d", trait.EvidenceCount)
	}
}

func TestWithEvidence_UniqueContexts(t *testing.T) {
	trait := NewPersonalityTrait("analytical", TraitCognitive, 0.8)

	trait.WithEvidence("text1", "ctx1")
	trait.WithEvidence("text2", "ctx1") // Same context, should not duplicate
	trait.WithEvidence("text3", "ctx2")

	if len(trait.Contexts) != 2 {
		t.Errorf("Contexts should have 2 unique entries, got %d: %v", len(trait.Contexts), trait.Contexts)
	}
}

func TestMerge_SameNameMerges(t *testing.T) {
	t1 := NewPersonalityTrait("analytical", TraitCognitive, 0.6)
	t1.EvidenceCount = 3

	t2 := NewPersonalityTrait("analytical", TraitCognitive, 0.8)
	t2.EvidenceCount = 7

	merged := t1.Merge(t2)

	expectedEvidence := 10
	if merged.EvidenceCount != expectedEvidence {
		t.Errorf("EvidenceCount after merge: got %d, want %d", merged.EvidenceCount, expectedEvidence)
	}

	// Weighted average: (0.6*3 + 0.8*7) / 10 = (1.8+5.6)/10 = 0.74
	expectedIntensity := (0.6*3 + 0.8*7) / 10.0
	if merged.Intensity < expectedIntensity-0.01 || merged.Intensity > expectedIntensity+0.01 {
		t.Errorf("Intensity after merge: got %f, want ~%f", merged.Intensity, expectedIntensity)
	}
}

func TestMerge_DifferentNameNoOp(t *testing.T) {
	t1 := NewPersonalityTrait("analytical", TraitCognitive, 0.6)
	t2 := NewPersonalityTrait("empathetic", TraitEmotional, 0.8)

	result := t1.Merge(t2)

	if result.Name != "analytical" {
		t.Errorf("Merge with different name should return receiver unchanged, Name=%q", result.Name)
	}
}

func TestIsWellEstablished(t *testing.T) {
	trait := NewPersonalityTrait("analytical", TraitCognitive, 0.8)

	if trait.IsWellEstablished() {
		t.Error("fresh trait should not be well established")
	}

	// Need: Confidence > 0.7, EvidenceCount >= 5, Consistency > 0.5
	// Consistency = len(Contexts)/10.0 → need > 5 distinct contexts
	contexts := []string{"code_review", "problem_solving", "debugging", "planning", "design", "testing", "refactoring"}
	for i, ctx := range contexts {
		trait.WithEvidence("evidence "+string(rune('a'+i)), ctx)
	}
	// Also add extra evidence with mixed contexts to push confidence up
	for i := 0; i < 10; i++ {
		trait.WithEvidence("evidence extra", contexts[i%len(contexts)])
	}

	if !trait.IsWellEstablished() {
		t.Errorf("trait should be well established: conf=%f count=%d consistency=%f",
			trait.Confidence, trait.EvidenceCount, trait.Consistency)
	}
}

func TestToNaturalDescription_IntensityPhrases(t *testing.T) {
	cases := []struct {
		intensity float64
		keyword   string
	}{
		{0.95, "strongly"},
		{0.75, "noticeably"},
		{0.55, "moderately"},
		{0.35, "somewhat"},
		{0.1, "slightly"},
	}

	for _, tc := range cases {
		trait := NewPersonalityTrait("curious", TraitEpistemic, tc.intensity)
		desc := trait.ToNaturalDescription()
		if !strings.Contains(desc, tc.keyword) {
			t.Errorf("intensity=%.2f: description %q should contain %q", tc.intensity, desc, tc.keyword)
		}
	}
}

func TestNewTraitObservation(t *testing.T) {
	obs := NewTraitObservation("agent-1", "analytical", TraitCognitive, "saw evidence", "ctx", 0.7)

	if obs.AgentID != "agent-1" {
		t.Errorf("AgentID: got %q", obs.AgentID)
	}
	if obs.TraitName != "analytical" {
		t.Errorf("TraitName: got %q", obs.TraitName)
	}
	if obs.Intensity != 0.7 {
		t.Errorf("Intensity: got %f", obs.Intensity)
	}
}
