package drift

import (
	"context"
	"testing"
	"time"

	"github.com/benoitpetit/soul/internal/domain/entities"
)

func TestNewSoulDriftDetector_DefaultThreshold(t *testing.T) {
	d := NewSoulDriftDetector(0)
	if d.threshold != 0.3 {
		t.Errorf("Default threshold should be 0.3, got %f", d.threshold)
	}

	d2 := NewSoulDriftDetector(1.5)
	if d2.threshold != 0.3 {
		t.Errorf("Out-of-range threshold should default to 0.3, got %f", d2.threshold)
	}
}

func TestNewSoulDriftDetector_ValidThreshold(t *testing.T) {
	d := NewSoulDriftDetector(0.5)
	if d.threshold != 0.5 {
		t.Errorf("Threshold: got %f, want 0.5", d.threshold)
	}
}

func TestDetectDrift_NilSnapshots(t *testing.T) {
	d := NewSoulDriftDetector(0.3)
	ctx := context.Background()

	_, err := d.DetectDrift(ctx, nil, entities.NewIdentitySnapshot("a", "m"))
	if err == nil {
		t.Error("DetectDrift with nil previous should return error")
	}

	_, err = d.DetectDrift(ctx, entities.NewIdentitySnapshot("a", "m"), nil)
	if err == nil {
		t.Error("DetectDrift with nil current should return error")
	}
}

func TestDetectDrift_IdenticalSnapshots_LowDrift(t *testing.T) {
	d := NewSoulDriftDetector(0.3)
	ctx := context.Background()

	snap := entities.NewIdentitySnapshot("agent-1", "gpt-4")
	snap.VoiceProfile = *entities.NewVoiceProfile()
	snap.EmotionalTone = *entities.NewEmotionalTone()
	snap.ValueSystem = *entities.NewValueSystem()

	trait := entities.NewPersonalityTrait("analytical", entities.TraitCognitive, 0.8)
	snap.WithTraits(*trait)

	report, err := d.DetectDrift(ctx, snap, snap)
	if err != nil {
		t.Fatalf("DetectDrift error: %v", err)
	}

	if report.DriftScore != 0.0 {
		t.Errorf("DriftScore for identical snapshots should be 0.0, got %f", report.DriftScore)
	}
	if report.IsSignificant {
		t.Error("Identical snapshots should not produce significant drift")
	}
}

func TestDetectDrift_VeryDifferentSnapshots(t *testing.T) {
	d := NewSoulDriftDetector(0.1) // Low threshold to ensure detection
	ctx := context.Background()

	prev := entities.NewIdentitySnapshot("agent-1", "gpt-4")
	prev.VoiceProfile = *entities.NewVoiceProfile()
	prev.VoiceProfile.FormalityLevel = 0.0
	prev.VoiceProfile.HumorLevel = 0.0
	prev.VoiceProfile.EmpathyLevel = 0.0
	prev.VoiceProfile.TechnicalDepth = 0.0
	prev.VoiceProfile.EnthusiasmLevel = 0.0
	prev.VoiceProfile.DirectnessLevel = 0.0
	prev.VoiceProfile.VocabularyRichness = 0.0
	prev.VoiceProfile.MetaphorUsage = 0.0
	prev.EmotionalTone = *entities.NewEmotionalTone()
	prev.ValueSystem = *entities.NewValueSystem()

	curr := entities.NewIdentitySnapshot("agent-1", "gpt-4")
	curr.VoiceProfile = *entities.NewVoiceProfile()
	curr.VoiceProfile.FormalityLevel = 1.0
	curr.VoiceProfile.HumorLevel = 1.0
	curr.VoiceProfile.EmpathyLevel = 1.0
	curr.VoiceProfile.TechnicalDepth = 1.0
	curr.VoiceProfile.EnthusiasmLevel = 1.0
	curr.VoiceProfile.DirectnessLevel = 1.0
	curr.VoiceProfile.VocabularyRichness = 1.0
	curr.VoiceProfile.MetaphorUsage = 1.0
	curr.EmotionalTone = *entities.NewEmotionalTone()
	curr.ValueSystem = *entities.NewValueSystem()

	report, err := d.DetectDrift(ctx, prev, curr)
	if err != nil {
		t.Fatalf("DetectDrift error: %v", err)
	}

	if report.DriftScore <= 0 {
		t.Errorf("DriftScore should be > 0 for very different snapshots, got %f", report.DriftScore)
	}
	if !report.IsSignificant {
		t.Error("Very different snapshots should produce significant drift (threshold=0.1)")
	}
}

func TestDetectDrift_RecommendationsOnSignificantDrift(t *testing.T) {
	d := NewSoulDriftDetector(0.0001) // Extremely low threshold
	ctx := context.Background()

	prev := entities.NewIdentitySnapshot("a", "m")
	prev.VoiceProfile = *entities.NewVoiceProfile()
	prev.VoiceProfile.FormalityLevel = 0.0
	prev.EmotionalTone = *entities.NewEmotionalTone()
	prev.ValueSystem = *entities.NewValueSystem()

	curr := entities.NewIdentitySnapshot("a", "m")
	curr.VoiceProfile = *entities.NewVoiceProfile()
	curr.VoiceProfile.FormalityLevel = 1.0
	curr.EmotionalTone = *entities.NewEmotionalTone()
	curr.ValueSystem = *entities.NewValueSystem()

	report, err := d.DetectDrift(ctx, prev, curr)
	if err != nil {
		t.Fatalf("DetectDrift error: %v", err)
	}

	if len(report.Recommendations) == 0 {
		t.Error("Significant drift should produce recommendations")
	}
}

func TestDetectDiffusion_NoTraits(t *testing.T) {
	d := NewSoulDriftDetector(0.3)
	ctx := context.Background()

	snap := entities.NewIdentitySnapshot("a", "m")

	isDiffused, score, err := d.DetectDiffusion(ctx, snap)
	if err != nil {
		t.Fatalf("DetectDiffusion error: %v", err)
	}
	if !isDiffused {
		t.Error("No-trait identity should be considered diffused")
	}
	if score != 1.0 {
		t.Errorf("Diffusion score for no traits should be 1.0, got %f", score)
	}
}

func TestDetectDiffusion_StrongTraits(t *testing.T) {
	d := NewSoulDriftDetector(0.3)
	ctx := context.Background()

	snap := entities.NewIdentitySnapshot("a", "m")

	// Add traits with many different contexts so they are well-established
	contexts := []string{"code_review", "problem_solving", "debugging", "planning", "design", "testing", "refactoring"}
	for i := 0; i < 3; i++ {
		trait := entities.NewPersonalityTrait("analytical", entities.TraitCognitive, 0.9)
		for j, c := range contexts {
			trait.WithEvidence("evidence "+string(rune('a'+j)), c)
		}
		// Duplicate name – just test with 1 well-established trait
		_ = trait
		snap.WithTraits(*trait)
		break
	}

	isDiffused, _, err := d.DetectDiffusion(ctx, snap)
	if err != nil {
		t.Fatalf("DetectDiffusion error: %v", err)
	}
	if isDiffused {
		t.Error("Identity with well-established trait should not be considered diffused")
	}
}

func TestMonitorContinuously_ClosesOnContextCancel(t *testing.T) {
	d := NewSoulDriftDetector(0.3)
	ctx, cancel := context.WithCancel(context.Background())

	ch, err := d.MonitorContinuously(ctx, "agent-1", 0.3)
	if err != nil {
		t.Fatalf("MonitorContinuously error: %v", err)
	}
	if ch == nil {
		t.Fatal("returned channel must not be nil")
	}

	// Cancel the context — the goroutine should close the channel promptly.
	cancel()

	select {
	case _, ok := <-ch:
		if ok {
			// A report was emitted before close — that is valid behaviour.
		}
		// Channel closed: success.
	case <-time.After(time.Second):
		t.Error("channel should be closed within 1 second after context cancellation")
	}
}

func TestMonitorContinuously_DefaultThresholdWhenInvalid(t *testing.T) {
	d := NewSoulDriftDetector(0.3)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// threshold <= 0 should fall back to detector's own threshold — no error expected.
	ch, err := d.MonitorContinuously(ctx, "agent-1", -1.0)
	if err != nil {
		t.Fatalf("MonitorContinuously error: %v", err)
	}
	if ch == nil {
		t.Error("channel must not be nil")
	}
}

func TestCalculateIdentityVector(t *testing.T) {
	d := NewSoulDriftDetector(0.3)
	ctx := context.Background()

	snap := entities.NewIdentitySnapshot("a", "m")
	snap.VoiceProfile = *entities.NewVoiceProfile()
	snap.ValueSystem = *entities.NewValueSystem()
	snap.BehavioralSignature = *entities.NewBehavioralSignature()

	v, err := d.CalculateIdentityVector(ctx, snap)
	if err != nil {
		t.Fatalf("CalculateIdentityVector error: %v", err)
	}
	if v == nil {
		t.Fatal("IdentityDimensionVector should not be nil")
	}
}

func TestCompareTraits_BothEmpty(t *testing.T) {
	d := NewSoulDriftDetector(0.3)
	drift := d.compareTraits(nil, nil)
	if drift != 0 {
		t.Errorf("compareTraits with both empty should return 0, got %f", drift)
	}
}

func TestCompareTraits_OneSide(t *testing.T) {
	d := NewSoulDriftDetector(0.3)

	trait := entities.NewPersonalityTrait("analytical", entities.TraitCognitive, 0.8)
	traits := []entities.PersonalityTrait{*trait}

	drift := d.compareTraits(traits, nil)
	if drift != 1.0 {
		t.Errorf("compareTraits with all previous traits gone should return 1.0, got %f", drift)
	}
}
