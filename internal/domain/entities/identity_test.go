package entities

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewIdentitySnapshot(t *testing.T) {
	snap := NewIdentitySnapshot("agent-1", "gpt-4")

	if snap.AgentID != "agent-1" {
		t.Errorf("AgentID: got %q, want %q", snap.AgentID, "agent-1")
	}
	if snap.ModelIdentifier != "gpt-4" {
		t.Errorf("ModelIdentifier: got %q, want %q", snap.ModelIdentifier, "gpt-4")
	}
	if snap.Version != 1 {
		t.Errorf("Version: got %d, want 1", snap.Version)
	}
	if snap.ID == (uuid.UUID{}) {
		t.Error("ID should not be zero")
	}
	if snap.PersonalityTraits == nil {
		t.Error("PersonalityTraits should be initialized")
	}
	if snap.ConfidenceScore != 0.0 {
		t.Errorf("ConfidenceScore: got %f, want 0.0", snap.ConfidenceScore)
	}
}

func TestWithParentSnapshot(t *testing.T) {
	parent := NewIdentitySnapshot("agent-1", "gpt-4")
	child := NewIdentitySnapshot("agent-1", "gpt-4")

	child.WithParentSnapshot(parent.ID)

	if child.DerivedFromID == nil {
		t.Fatal("DerivedFromID should not be nil")
	}
	if *child.DerivedFromID != parent.ID {
		t.Errorf("DerivedFromID: got %v, want %v", *child.DerivedFromID, parent.ID)
	}
	if child.Version != 2 {
		t.Errorf("Version after WithParentSnapshot: got %d, want 2", child.Version)
	}
}

func TestWithTraitsRecalculatesConfidence(t *testing.T) {
	snap := NewIdentitySnapshot("agent-1", "gpt-4")

	t1 := NewPersonalityTrait("analytical", TraitCognitive, 0.8)
	t1.Confidence = 0.8
	t2 := NewPersonalityTrait("empathetic", TraitEmotional, 0.6)
	t2.Confidence = 0.6

	snap.WithTraits(*t1, *t2)

	expectedConf := (0.8 + 0.6) / 2.0
	if snap.ConfidenceScore != expectedConf {
		t.Errorf("ConfidenceScore: got %f, want %f", snap.ConfidenceScore, expectedConf)
	}
}

func TestLinkMiraMemory(t *testing.T) {
	snap := NewIdentitySnapshot("agent-1", "gpt-4")
	memID := uuid.New()

	snap.LinkMiraMemory(memID)

	if snap.SourceMemoriesCount != 1 {
		t.Errorf("SourceMemoriesCount: got %d, want 1", snap.SourceMemoriesCount)
	}
	if len(snap.LinkedMiraMemories) != 1 {
		t.Errorf("LinkedMiraMemories length: got %d, want 1", len(snap.LinkedMiraMemories))
	}
	if snap.LinkedMiraMemories[0] != memID {
		t.Errorf("LinkedMiraMemories[0]: got %v, want %v", snap.LinkedMiraMemories[0], memID)
	}
}

func TestIsIdentityDiffusionDetected_NilPrevious(t *testing.T) {
	snap := NewIdentitySnapshot("agent-1", "gpt-4")
	if snap.IsIdentityDiffusionDetected(nil) {
		t.Error("should return false when previous is nil")
	}
}

func TestIsIdentityDiffusionDetected_NoDiffusion(t *testing.T) {
	previous := NewIdentitySnapshot("agent-1", "gpt-4")
	current := NewIdentitySnapshot("agent-1", "gpt-4")

	trait := NewPersonalityTrait("analytical", TraitCognitive, 0.8)
	trait.Confidence = 0.8

	previous.WithTraits(*trait)
	// Same trait, same confidence
	current.WithTraits(*trait)

	if current.IsIdentityDiffusionDetected(previous) {
		t.Error("should not detect diffusion when traits are stable")
	}
}

func TestIsIdentityDiffusionDetected_WithDiffusion(t *testing.T) {
	previous := NewIdentitySnapshot("agent-1", "gpt-4")
	current := NewIdentitySnapshot("agent-1", "gpt-4")

	// Previous has 3 strong traits
	for _, name := range []string{"analytical", "empathetic", "curious"} {
		tr := NewPersonalityTrait(name, TraitCognitive, 0.8)
		tr.Confidence = 0.8
		previous.WithTraits(*tr)
	}
	// Current has no traits → all 3 "disappear" → 3/3 > 50%

	if !current.IsIdentityDiffusionDetected(previous) {
		t.Error("should detect diffusion when all traits disappeared")
	}
}

func TestCalculateDiff_AddedRemovedTraits(t *testing.T) {
	from := NewIdentitySnapshot("agent-1", "gpt-4")
	to := NewIdentitySnapshot("agent-1", "gpt-4")

	t1 := NewPersonalityTrait("analytical", TraitCognitive, 0.8)
	t2 := NewPersonalityTrait("empathetic", TraitEmotional, 0.7)
	from.WithTraits(*t1)
	to.WithTraits(*t2)

	diff := CalculateDiff(from, to)

	if len(diff.AddedTraits) != 1 || diff.AddedTraits[0].Name != "empathetic" {
		t.Errorf("AddedTraits: expected [empathetic], got %v", diff.AddedTraits)
	}
	if len(diff.RemovedTraits) != 1 || diff.RemovedTraits[0].Name != "analytical" {
		t.Errorf("RemovedTraits: expected [analytical], got %v", diff.RemovedTraits)
	}
}

func TestCalculateDiff_StrengthenedWeakenedTraits(t *testing.T) {
	from := NewIdentitySnapshot("agent-1", "gpt-4")
	to := NewIdentitySnapshot("agent-1", "gpt-4")

	tWeak := NewPersonalityTrait("analytical", TraitCognitive, 0.5)
	tWeak.Confidence = 0.9
	from.WithTraits(*tWeak)

	// Same trait but much weaker confidence
	tStrong := NewPersonalityTrait("analytical", TraitCognitive, 0.5)
	tStrong.Confidence = 0.3 // < 0.9 * 0.8 = 0.72 → weakened
	to.WithTraits(*tStrong)

	diff := CalculateDiff(from, to)

	if len(diff.WeakenedTraits) != 1 {
		t.Errorf("WeakenedTraits: expected 1, got %d", len(diff.WeakenedTraits))
	}
}

func TestCalculateDiff_OverallDrift(t *testing.T) {
	from := NewIdentitySnapshot("agent-1", "gpt-4")
	to := NewIdentitySnapshot("agent-1", "gpt-4")

	t1 := NewPersonalityTrait("analytical", TraitCognitive, 0.8)
	from.WithTraits(*t1)

	t2 := NewPersonalityTrait("empathetic", TraitEmotional, 0.7)
	to.WithTraits(*t2)

	diff := CalculateDiff(from, to)

	if diff.OverallDrift <= 0 {
		t.Error("OverallDrift should be > 0 when traits change completely")
	}
	if diff.OverallDrift > 1 {
		t.Error("OverallDrift should be <= 1")
	}
}

func TestCalculateDiff_Timestamps(t *testing.T) {
	before := time.Now()
	from := NewIdentitySnapshot("a", "m")
	to := NewIdentitySnapshot("a", "m")

	diff := CalculateDiff(from, to)

	if diff.Timestamp.Before(before) {
		t.Error("Timestamp should be recent")
	}
}

func TestGenerateIdentityPrompt_ContainsSections(t *testing.T) {
	snap := NewIdentitySnapshot("agent-1", "gpt-4")
	snap.VoiceProfile = *NewVoiceProfile()
	snap.CommunicationStyle = *NewCommunicationStyle()
	snap.BehavioralSignature = *NewBehavioralSignature()
	snap.ValueSystem = *NewValueSystem()
	snap.EmotionalTone = *NewEmotionalTone()

	prompt := snap.GenerateIdentityPrompt()

	sections := []string{"## Your Identity", "### How You Speak", "### Your Core Traits", "### Your Communication Style", "### What You Value", "### How You Behave"}
	for _, section := range sections {
		found := false
		for i := 0; i+len(section) <= len(prompt); i++ {
			if prompt[i:i+len(section)] == section {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GenerateIdentityPrompt missing section: %q", section)
		}
	}
}
