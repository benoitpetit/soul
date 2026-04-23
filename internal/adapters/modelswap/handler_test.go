package modelswap

import (
	"context"
	"testing"

	"github.com/benoitpetit/soul/internal/domain/entities"
	"github.com/benoitpetit/soul/internal/domain/valueobjects"
)

// --- SoulModelSwapHandler tests ---

func TestNewSoulModelSwapHandler(t *testing.T) {
	h := NewSoulModelSwapHandler()
	if h == nil {
		t.Fatal("NewSoulModelSwapHandler should not return nil")
	}
	if h.maxAcceptableDrift != 0.3 {
		t.Errorf("maxAcceptableDrift: got %f, want 0.3", h.maxAcceptableDrift)
	}
}

func TestHandleModelSwap(t *testing.T) {
	h := NewSoulModelSwapHandler()
	ctx := context.Background()

	swap, err := h.HandleModelSwap(ctx, "agent-1", "gpt-3.5", "gpt-4")
	if err != nil {
		t.Fatalf("HandleModelSwap error: %v", err)
	}
	if swap == nil {
		t.Fatal("swap should not be nil")
	}
	if swap.PreviousModel != "gpt-3.5" {
		t.Errorf("PreviousModel: got %q, want %q", swap.PreviousModel, "gpt-3.5")
	}
	if swap.NewModel != "gpt-4" {
		t.Errorf("NewModel: got %q, want %q", swap.NewModel, "gpt-4")
	}
}

func TestReinforceIdentity_NilReturnsError(t *testing.T) {
	h := NewSoulModelSwapHandler()
	ctx := context.Background()

	_, err := h.ReinforceIdentity(ctx, nil)
	if err == nil {
		t.Error("ReinforceIdentity with nil identity should return error")
	}
}

func TestReinforceIdentity_CopiesAllDimensions(t *testing.T) {
	h := NewSoulModelSwapHandler()
	ctx := context.Background()

	orig := entities.NewIdentitySnapshot("agent-1", "gpt-4")
	orig.VoiceProfile = *entities.NewVoiceProfile()
	orig.CommunicationStyle = *entities.NewCommunicationStyle()
	orig.BehavioralSignature = *entities.NewBehavioralSignature()
	orig.ValueSystem = *entities.NewValueSystem()
	orig.EmotionalTone = *entities.NewEmotionalTone()

	trait := entities.NewPersonalityTrait("analytical", entities.TraitCognitive, 0.9)
	trait.Confidence = 0.8
	orig.WithTraits(*trait)

	reinforced, err := h.ReinforceIdentity(ctx, orig)
	if err != nil {
		t.Fatalf("ReinforceIdentity error: %v", err)
	}

	if reinforced == nil {
		t.Fatal("reinforced should not be nil")
	}
	if reinforced.AgentID != orig.AgentID {
		t.Errorf("AgentID: got %q, want %q", reinforced.AgentID, orig.AgentID)
	}
	if reinforced.DerivedFromID == nil {
		t.Error("reinforced should have parent ID set")
	}
	if len(reinforced.PersonalityTraits) != len(orig.PersonalityTraits) {
		t.Errorf("PersonalityTraits: got %d, want %d", len(reinforced.PersonalityTraits), len(orig.PersonalityTraits))
	}
}

func TestReinforceIdentity_BoostsHighConfidenceTraits(t *testing.T) {
	h := NewSoulModelSwapHandler()
	ctx := context.Background()

	orig := entities.NewIdentitySnapshot("a", "m")
	orig.VoiceProfile = *entities.NewVoiceProfile()
	orig.CommunicationStyle = *entities.NewCommunicationStyle()
	orig.BehavioralSignature = *entities.NewBehavioralSignature()
	orig.ValueSystem = *entities.NewValueSystem()
	orig.EmotionalTone = *entities.NewEmotionalTone()

	trait := entities.NewPersonalityTrait("analytical", entities.TraitCognitive, 0.9)
	trait.Confidence = 0.8 // > 0.7, should be boosted
	orig.WithTraits(*trait)
	origConfidence := orig.PersonalityTraits[0].Confidence

	reinforced, err := h.ReinforceIdentity(ctx, orig)
	if err != nil {
		t.Fatalf("ReinforceIdentity error: %v", err)
	}

	if reinforced.PersonalityTraits[0].Confidence <= origConfidence {
		t.Errorf("High-confidence trait should be boosted: original=%f, reinforced=%f",
			origConfidence, reinforced.PersonalityTraits[0].Confidence)
	}
}

func TestMeasurePostSwapDrift_Nil(t *testing.T) {
	h := NewSoulModelSwapHandler()
	ctx := context.Background()

	_, err := h.MeasurePostSwapDrift(ctx, nil)
	if err == nil {
		t.Error("MeasurePostSwapDrift with nil swap should return error")
	}
}

func TestMeasurePostSwapDrift_ReturnsDrift(t *testing.T) {
	h := NewSoulModelSwapHandler()
	ctx := context.Background()

	swap := &valueobjects.ModelSwapContext{
		PreviousModel: "gpt-3.5",
		NewModel:      "gpt-4",
		IdentityDrift: 0.25,
	}

	drift, err := h.MeasurePostSwapDrift(ctx, swap)
	if err != nil {
		t.Fatalf("MeasurePostSwapDrift error: %v", err)
	}
	if drift != 0.25 {
		t.Errorf("Drift: got %f, want 0.25", drift)
	}
}

// --- SoulMergerService tests ---

func TestNewSoulMergerService(t *testing.T) {
	m := NewSoulMergerService()
	if m == nil {
		t.Fatal("NewSoulMergerService should not return nil")
	}
}

func TestMergeIdentities_NilReturnsError(t *testing.T) {
	m := NewSoulMergerService()
	ctx := context.Background()

	_, err := m.MergeIdentities(ctx, nil, entities.NewIdentitySnapshot("a", "m"), valueobjects.MergeWeightedAverage)
	if err == nil {
		t.Error("MergeIdentities with nil A should return error")
	}

	_, err = m.MergeIdentities(ctx, entities.NewIdentitySnapshot("a", "m"), nil, valueobjects.MergeWeightedAverage)
	if err == nil {
		t.Error("MergeIdentities with nil B should return error")
	}
}

func TestMergeIdentities_AllStrategies(t *testing.T) {
	m := NewSoulMergerService()
	ctx := context.Background()

	strategies := []valueobjects.MergeStrategy{
		valueobjects.MergePreserveDominant,
		valueobjects.MergeWeightedAverage,
		valueobjects.MergeLatestWins,
		valueobjects.MergeSynthesize,
	}

	for _, strategy := range strategies {
		a := entities.NewIdentitySnapshot("agent-1", "gpt-4")
		a.VoiceProfile = *entities.NewVoiceProfile()
		a.CommunicationStyle = *entities.NewCommunicationStyle()
		a.BehavioralSignature = *entities.NewBehavioralSignature()
		a.ValueSystem = *entities.NewValueSystem()
		a.EmotionalTone = *entities.NewEmotionalTone()

		b := entities.NewIdentitySnapshot("agent-1", "gpt-4")
		b.VoiceProfile = *entities.NewVoiceProfile()
		b.CommunicationStyle = *entities.NewCommunicationStyle()
		b.BehavioralSignature = *entities.NewBehavioralSignature()
		b.ValueSystem = *entities.NewValueSystem()
		b.EmotionalTone = *entities.NewEmotionalTone()

		merged, err := m.MergeIdentities(ctx, a, b, strategy)
		if err != nil {
			t.Errorf("MergeIdentities strategy=%q error: %v", strategy, err)
			continue
		}
		if merged == nil {
			t.Errorf("MergeIdentities strategy=%q: result should not be nil", strategy)
		}
	}
}

func TestCalculateMergeCompatibility_Nil(t *testing.T) {
	m := NewSoulMergerService()
	ctx := context.Background()

	_, err := m.CalculateMergeCompatibility(ctx, nil, entities.NewIdentitySnapshot("a", "m"))
	if err == nil {
		t.Error("CalculateMergeCompatibility with nil A should return error")
	}
}

func TestCalculateMergeCompatibility_Identical(t *testing.T) {
	m := NewSoulMergerService()
	ctx := context.Background()

	snap := entities.NewIdentitySnapshot("a", "m")
	snap.VoiceProfile = *entities.NewVoiceProfile()
	snap.EmotionalTone = *entities.NewEmotionalTone()
	snap.ValueSystem = *entities.NewValueSystem()

	compatibility, err := m.CalculateMergeCompatibility(ctx, snap, snap)
	if err != nil {
		t.Fatalf("CalculateMergeCompatibility error: %v", err)
	}
	if compatibility < 0.9 {
		t.Errorf("Identical snapshots should have compatibility >= 0.9, got %f", compatibility)
	}
}
