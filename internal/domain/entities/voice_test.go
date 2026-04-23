package entities

import (
	"strings"
	"testing"
)

func TestNewVoiceProfile_Defaults(t *testing.T) {
	vp := NewVoiceProfile()

	if vp.FormalityLevel != 0.5 {
		t.Errorf("FormalityLevel: got %f, want 0.5", vp.FormalityLevel)
	}
	if vp.EmpathyLevel != 0.6 {
		t.Errorf("EmpathyLevel: got %f, want 0.6", vp.EmpathyLevel)
	}
	if vp.UsesMarkdown != true {
		t.Error("UsesMarkdown should default to true")
	}
}

func TestVoiceProfile_WithFormality_Clamped(t *testing.T) {
	vp := NewVoiceProfile()

	vp.WithFormality(1.5)
	if vp.FormalityLevel != 1.0 {
		t.Errorf("Formality should be clamped to 1.0, got %f", vp.FormalityLevel)
	}

	vp.WithFormality(-0.5)
	if vp.FormalityLevel != 0.0 {
		t.Errorf("Formality should be clamped to 0.0, got %f", vp.FormalityLevel)
	}
}

func TestVoiceProfile_WithCatchPhrases(t *testing.T) {
	vp := NewVoiceProfile()
	vp.WithCatchPhrases("to be honest", "let me think")

	if len(vp.CatchPhrases) != 2 {
		t.Errorf("CatchPhrases length: got %d, want 2", len(vp.CatchPhrases))
	}
}

func TestVoiceProfile_DistanceTo_Identical(t *testing.T) {
	vp := NewVoiceProfile()
	dist := vp.DistanceTo(vp)

	if dist != 0.0 {
		t.Errorf("Distance to itself should be 0.0, got %f", dist)
	}
}

func TestVoiceProfile_DistanceTo_Opposite(t *testing.T) {
	vp1 := NewVoiceProfile()
	vp1.FormalityLevel = 0.0
	vp1.HumorLevel = 0.0
	vp1.EmpathyLevel = 0.0
	vp1.TechnicalDepth = 0.0
	vp1.EnthusiasmLevel = 0.0
	vp1.DirectnessLevel = 0.0
	vp1.VocabularyRichness = 0.0
	vp1.MetaphorUsage = 0.0

	vp2 := NewVoiceProfile()
	vp2.FormalityLevel = 1.0
	vp2.HumorLevel = 1.0
	vp2.EmpathyLevel = 1.0
	vp2.TechnicalDepth = 1.0
	vp2.EnthusiasmLevel = 1.0
	vp2.DirectnessLevel = 1.0
	vp2.VocabularyRichness = 1.0
	vp2.MetaphorUsage = 1.0

	dist := vp1.DistanceTo(vp2)
	if dist != 1.0 {
		t.Errorf("Max distance should be 1.0, got %f", dist)
	}
}

func TestVoiceProfile_ToNaturalDescription_HighFormality(t *testing.T) {
	vp := NewVoiceProfile()
	vp.FormalityLevel = 0.9

	desc := vp.ToNaturalDescription()
	if !strings.Contains(desc, "professional") {
		t.Errorf("High formality description should mention 'professional', got: %q", desc)
	}
}

func TestVoiceProfile_ToNaturalDescription_WithCatchPhrases(t *testing.T) {
	vp := NewVoiceProfile()
	vp.WithCatchPhrases("for instance")

	desc := vp.ToNaturalDescription()
	if !strings.Contains(desc, "for instance") {
		t.Errorf("Description should mention catch phrases, got: %q", desc)
	}
}
