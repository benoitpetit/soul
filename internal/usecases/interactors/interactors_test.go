// Package interactors_test tests use cases with real in-memory SQLite storage.
package interactors_test

import (
	"context"
	"testing"

	"github.com/benoitpetit/soul/internal/adapters/composition"
	"github.com/benoitpetit/soul/internal/adapters/drift"
	"github.com/benoitpetit/soul/internal/adapters/extraction"
	"github.com/benoitpetit/soul/internal/adapters/modelswap"
	"github.com/benoitpetit/soul/internal/domain/entities"
	"github.com/benoitpetit/soul/internal/domain/valueobjects"
	"github.com/benoitpetit/soul/internal/testhelper"
	"github.com/benoitpetit/soul/internal/usecases/interactors"
)

// ========== IdentityCaptureUseCase ==========

func TestCaptureFromConversation_CreatesNewSnapshot(t *testing.T) {
	s := testhelper.NewStorage(t)
	extractor := extraction.NewSoulExtractorService()
	uc := interactors.NewIdentityCaptureUseCase(s, extractor)
	ctx := context.Background()

	req := &valueobjects.SoulCaptureRequest{
		AgentID:      "agent-capture-1",
		ModelID:      "gpt-4",
		Conversation: "Let me analyze this carefully. I want to help you.",
		AgentResponses: []string{
			"Let me analyze this carefully.",
			"I want to help you understand.",
		},
		UserFeedback: map[string]string{},
	}

	snapshot, err := uc.CaptureFromConversation(ctx, req)
	if err != nil {
		t.Fatalf("CaptureFromConversation error: %v", err)
	}
	if snapshot == nil {
		t.Fatal("snapshot should not be nil")
	}
	if snapshot.AgentID != "agent-capture-1" {
		t.Errorf("AgentID: got %q, want %q", snapshot.AgentID, "agent-capture-1")
	}

	stored, err := s.GetLatestIdentity(ctx, "agent-capture-1")
	if err != nil {
		t.Fatalf("GetLatestIdentity error: %v", err)
	}
	if stored == nil {
		t.Fatal("stored identity should not be nil")
	}
	if stored.ID != snapshot.ID {
		t.Errorf("stored ID: got %v, want %v", stored.ID, snapshot.ID)
	}
}

func TestCaptureFromConversation_BuildsOnExisting(t *testing.T) {
	s := testhelper.NewStorage(t)
	extractor := extraction.NewSoulExtractorService()
	uc := interactors.NewIdentityCaptureUseCase(s, extractor)
	ctx := context.Background()

	initial := testhelper.MakeFullSnapshot("agent-v2", "gpt-4")
	if err := s.StoreIdentity(ctx, initial); err != nil {
		t.Fatalf("StoreIdentity error: %v", err)
	}

	req := &valueobjects.SoulCaptureRequest{
		AgentID:        "agent-v2",
		ModelID:        "gpt-4",
		Conversation:   "I analyze carefully.",
		AgentResponses: []string{"I analyze carefully."},
		UserFeedback:   map[string]string{},
	}

	snapshot2, err := uc.CaptureFromConversation(ctx, req)
	if err != nil {
		t.Fatalf("Second capture error: %v", err)
	}
	if snapshot2.DerivedFromID == nil {
		t.Error("Second snapshot should have a parent (DerivedFromID)")
	}
	if *snapshot2.DerivedFromID != initial.ID {
		t.Errorf("DerivedFromID: got %v, want %v", *snapshot2.DerivedFromID, initial.ID)
	}
}

func TestCaptureFromSingleInteraction(t *testing.T) {
	s := testhelper.NewStorage(t)
	extractor := extraction.NewSoulExtractorService()
	uc := interactors.NewIdentityCaptureUseCase(s, extractor)
	ctx := context.Background()

	err := uc.CaptureFromSingleInteraction(ctx, "agent-single", "I will help you.", "Help me please.", "gpt-4")
	if err != nil {
		t.Fatalf("CaptureFromSingleInteraction error: %v", err)
	}

	stored, err := s.GetLatestIdentity(ctx, "agent-single")
	if err != nil {
		t.Fatalf("GetLatestIdentity error: %v", err)
	}
	if stored == nil {
		t.Fatal("stored identity should not be nil after capture")
	}
}

// ========== Capture + Drift integration ==========

func TestCaptureThenDrift_Integration(t *testing.T) {
	s := testhelper.NewStorage(t)
	extractor := extraction.NewSoulExtractorService()
	detector := drift.NewSoulDriftDetector(0.3)
	composer := composition.NewSoulComposerService()
	captureUC := interactors.NewIdentityCaptureUseCase(s, extractor)
	driftUC := interactors.NewDriftDetectionUseCase(s, detector, composer)
	ctx := context.Background()

	// Capture initial identity
	initial, err := captureUC.CaptureFromConversation(ctx, &valueobjects.SoulCaptureRequest{
		AgentID:        "agent-capture-drift",
		ModelID:        "gpt-4",
		Conversation:   "Let me analyze this carefully and help you.",
		AgentResponses: []string{"Let me analyze this carefully.", "I will help you."},
		UserFeedback:   map[string]string{},
	})
	if err != nil {
		t.Fatalf("initial capture error: %v", err)
	}

	// Capture a second time (builds on existing)
	second, err := captureUC.CaptureFromConversation(ctx, &valueobjects.SoulCaptureRequest{
		AgentID:        "agent-capture-drift",
		ModelID:        "gpt-4",
		Conversation:   "Actually, let me be more casual about this.",
		AgentResponses: []string{"Let me be more casual.", "Sure, no problem!"},
		UserFeedback:   map[string]string{},
	})
	if err != nil {
		t.Fatalf("second capture error: %v", err)
	}

	// Check drift between initial and second snapshot
	report, err := driftUC.CheckDrift(ctx, "agent-capture-drift", second)
	if err != nil {
		t.Fatalf("CheckDrift error: %v", err)
	}
	if report == nil {
		t.Fatal("drift report should not be nil")
	}
	if report.DriftScore < 0 || report.DriftScore > 1 {
		t.Errorf("DriftScore should be in [0,1], got %f", report.DriftScore)
	}
	_ = initial
}

func TestCaptureThenCheckDiffusion(t *testing.T) {
	s := testhelper.NewStorage(t)
	extractor := extraction.NewSoulExtractorService()
	detector := drift.NewSoulDriftDetector(0.3)
	composer := composition.NewSoulComposerService()
	captureUC := interactors.NewIdentityCaptureUseCase(s, extractor)
	driftUC := interactors.NewDriftDetectionUseCase(s, detector, composer)
	ctx := context.Background()

	_, err := captureUC.CaptureFromConversation(ctx, &valueobjects.SoulCaptureRequest{
		AgentID:        "agent-diffusion",
		ModelID:        "gpt-4",
		Conversation:   "I will help you with your problem.",
		AgentResponses: []string{"I will help you."},
		UserFeedback:   map[string]string{},
	})
	if err != nil {
		t.Fatalf("capture error: %v", err)
	}

	_, score, err := driftUC.CheckDiffusion(ctx, "agent-diffusion")
	if err != nil {
		t.Fatalf("CheckDiffusion error: %v", err)
	}
	if score < 0 || score > 1 {
		t.Errorf("Diffusion score should be in [0,1], got %f", score)
	}
}

// ========== IdentityRecallUseCase ==========

func TestRecallIdentity_Success(t *testing.T) {
	s := testhelper.NewStorage(t)
	composer := composition.NewSoulComposerService()
	uc := interactors.NewIdentityRecallUseCase(s, composer)
	ctx := context.Background()

	snap := testhelper.MakeFullSnapshot("agent-recall", "gpt-4")
	if err := s.StoreIdentity(ctx, snap); err != nil {
		t.Fatalf("StoreIdentity error: %v", err)
	}

	query := &valueobjects.SoulQuery{
		AgentID:      "agent-recall",
		BudgetTokens: 1000,
	}

	prompt, err := uc.RecallIdentity(ctx, query)
	if err != nil {
		t.Fatalf("RecallIdentity error: %v", err)
	}
	if prompt == nil {
		t.Fatal("prompt should not be nil")
	}
	if prompt.Content == "" {
		t.Error("prompt content should not be empty")
	}
}

func TestRecallIdentity_NoIdentityFound(t *testing.T) {
	s := testhelper.NewStorage(t)
	composer := composition.NewSoulComposerService()
	uc := interactors.NewIdentityRecallUseCase(s, composer)
	ctx := context.Background()

	query := &valueobjects.SoulQuery{
		AgentID:      "no-such-agent",
		BudgetTokens: 500,
	}

	_, err := uc.RecallIdentity(ctx, query)
	if err == nil {
		t.Error("RecallIdentity for unknown agent should return error")
	}
}

func TestGetIdentitySummary_NoIdentity(t *testing.T) {
	s := testhelper.NewStorage(t)
	composer := composition.NewSoulComposerService()
	uc := interactors.NewIdentityRecallUseCase(s, composer)
	ctx := context.Background()

	summary, err := uc.GetIdentitySummary(ctx, "no-agent")
	if err != nil {
		t.Fatalf("GetIdentitySummary error: %v", err)
	}
	if summary == "" {
		t.Error("Summary should return a non-empty string even for unknown agent")
	}
}

func TestGetIdentitySummary_WithIdentity(t *testing.T) {
	s := testhelper.NewStorage(t)
	composer := composition.NewSoulComposerService()
	uc := interactors.NewIdentityRecallUseCase(s, composer)
	ctx := context.Background()

	snap := testhelper.MakeFullSnapshot("agent-sum", "gpt-4")
	s.StoreIdentity(ctx, snap)

	summary, err := uc.GetIdentitySummary(ctx, "agent-sum")
	if err != nil {
		t.Fatalf("GetIdentitySummary error: %v", err)
	}
	if summary == "" {
		t.Error("Summary should not be empty for existing identity")
	}
}

func TestGetIdentityTraits(t *testing.T) {
	s := testhelper.NewStorage(t)
	composer := composition.NewSoulComposerService()
	uc := interactors.NewIdentityRecallUseCase(s, composer)
	ctx := context.Background()

	trait := entities.NewPersonalityTrait("curious", entities.TraitEpistemic, 0.7)
	trait.AgentID = "agent-traits"
	s.StoreTrait(ctx, trait)

	traits, err := uc.GetIdentityTraits(ctx, "agent-traits", false)
	if err != nil {
		t.Fatalf("GetIdentityTraits error: %v", err)
	}
	if len(traits) != 1 {
		t.Errorf("Expected 1 trait, got %d", len(traits))
	}
}

func TestGetIdentityHistory(t *testing.T) {
	s := testhelper.NewStorage(t)
	composer := composition.NewSoulComposerService()
	uc := interactors.NewIdentityRecallUseCase(s, composer)
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		snap := testhelper.MakeFullSnapshot("agent-hist-uc", "gpt-4")
		snap.Version = i
		s.StoreIdentity(ctx, snap)
	}

	history, err := uc.GetIdentityHistory(ctx, "agent-hist-uc", 10)
	if err != nil {
		t.Fatalf("GetIdentityHistory error: %v", err)
	}
	if len(history) != 3 {
		t.Errorf("Expected 3 history entries, got %d", len(history))
	}
}

// ========== DriftDetectionUseCase ==========

func TestCheckDrift_Success(t *testing.T) {
	s := testhelper.NewStorage(t)
	detector := drift.NewSoulDriftDetector(0.3)
	composer := composition.NewSoulComposerService()
	uc := interactors.NewDriftDetectionUseCase(s, detector, composer)
	ctx := context.Background()

	reference := testhelper.MakeFullSnapshot("agent-drift-uc", "gpt-4")
	s.StoreIdentity(ctx, reference)

	current := testhelper.MakeFullSnapshot("agent-drift-uc", "gpt-4")
	current.VoiceProfile.FormalityLevel = 0.9

	report, err := uc.CheckDrift(ctx, "agent-drift-uc", current)
	if err != nil {
		t.Fatalf("CheckDrift error: %v", err)
	}
	if report == nil {
		t.Fatal("report should not be nil")
	}
	if report.DriftScore < 0 || report.DriftScore > 1 {
		t.Errorf("DriftScore should be in [0,1], got %f", report.DriftScore)
	}
}

func TestCheckDrift_NoReferenceIdentity(t *testing.T) {
	s := testhelper.NewStorage(t)
	detector := drift.NewSoulDriftDetector(0.3)
	composer := composition.NewSoulComposerService()
	uc := interactors.NewDriftDetectionUseCase(s, detector, composer)
	ctx := context.Background()

	current := testhelper.MakeFullSnapshot("no-agent", "gpt-4")
	_, err := uc.CheckDrift(ctx, "no-agent", current)
	if err == nil {
		t.Error("CheckDrift without reference identity should return error")
	}
}

func TestCheckDiffusion(t *testing.T) {
	s := testhelper.NewStorage(t)
	detector := drift.NewSoulDriftDetector(0.3)
	composer := composition.NewSoulComposerService()
	uc := interactors.NewDriftDetectionUseCase(s, detector, composer)
	ctx := context.Background()

	snap := entities.NewIdentitySnapshot("diffuse-agent", "gpt-4")
	snap.VoiceProfile = *entities.NewVoiceProfile()
	snap.CommunicationStyle = *entities.NewCommunicationStyle()
	snap.BehavioralSignature = *entities.NewBehavioralSignature()
	snap.ValueSystem = *entities.NewValueSystem()
	snap.EmotionalTone = *entities.NewEmotionalTone()
	s.StoreIdentity(ctx, snap)

	isDiffused, score, err := uc.CheckDiffusion(ctx, "diffuse-agent")
	if err != nil {
		t.Fatalf("CheckDiffusion error: %v", err)
	}
	if !isDiffused {
		t.Error("Agent with no traits should be considered diffused")
	}
	if score < 0 || score > 1 {
		t.Errorf("Diffusion score should be in [0,1], got %f", score)
	}
}

func TestGetDiffHistory(t *testing.T) {
	s := testhelper.NewStorage(t)
	detector := drift.NewSoulDriftDetector(0.3)
	composer := composition.NewSoulComposerService()
	uc := interactors.NewDriftDetectionUseCase(s, detector, composer)
	ctx := context.Background()

	history, err := uc.GetDiffHistory(ctx, "no-agent", 10)
	if err != nil {
		t.Fatalf("GetDiffHistory error: %v", err)
	}
	_ = history
}

func TestRestoreIdentity(t *testing.T) {
	s := testhelper.NewStorage(t)
	detector := drift.NewSoulDriftDetector(0.3)
	composer := composition.NewSoulComposerService()
	uc := interactors.NewDriftDetectionUseCase(s, detector, composer)
	ctx := context.Background()

	snap := testhelper.MakeFullSnapshot("agent-restore", "gpt-4")
	snap.Version = 3
	s.StoreIdentity(ctx, snap)

	restored, err := uc.RestoreIdentity(ctx, "agent-restore", 3)
	if err != nil {
		t.Fatalf("RestoreIdentity error: %v", err)
	}
	if restored == nil {
		t.Fatal("restored should not be nil")
	}
	if restored.DerivedFromID == nil {
		t.Error("restored should have parent ID set")
	}
}

func TestGetDriftReport(t *testing.T) {
	s := testhelper.NewStorage(t)
	detector := drift.NewSoulDriftDetector(0.3)
	composer := composition.NewSoulComposerService()
	uc := interactors.NewDriftDetectionUseCase(s, detector, composer)
	ctx := context.Background()

	snap := testhelper.MakeFullSnapshot("agent-report", "gpt-4")
	s.StoreIdentity(ctx, snap)

	report, err := uc.GetDriftReport(ctx, "agent-report", 10)
	if err != nil {
		t.Fatalf("GetDriftReport error: %v", err)
	}
	if report == nil {
		t.Fatal("drift report should not be nil")
	}
}

// ========== IdentityUpdateUseCase ==========

func TestUpdateFromDirective_Enthusiasm(t *testing.T) {
	s := testhelper.NewStorage(t)
	uc := interactors.NewIdentityUpdateUseCase(s)
	ctx := context.Background()

	snap := testhelper.MakeFullSnapshot("agent-upd-enth", "gpt-4")
	s.StoreIdentity(ctx, snap)

	updated, result, err := uc.UpdateFromDirective(ctx, "agent-upd-enth", "sois plus enthousiaste", "test")
	if err != nil {
		t.Fatalf("UpdateFromDirective error: %v", err)
	}
	if updated == nil {
		t.Fatal("updated snapshot should not be nil")
	}
	if result == nil {
		t.Fatal("result should not be nil")
	}
	if len(result.ChangesApplied) == 0 {
		t.Error("expected at least one change to be applied")
	}
	if updated.Version != snap.Version+1 {
		t.Errorf("expected version %d, got %d", snap.Version+1, updated.Version)
	}
}

func TestUpdateFromDirective_Concise(t *testing.T) {
	s := testhelper.NewStorage(t)
	uc := interactors.NewIdentityUpdateUseCase(s)
	ctx := context.Background()

	snap := testhelper.MakeFullSnapshot("agent-upd-concise", "gpt-4")
	s.StoreIdentity(ctx, snap)

	updated, result, err := uc.UpdateFromDirective(ctx, "agent-upd-concise", "sois plus concis", "testing")
	if err != nil {
		t.Fatalf("UpdateFromDirective error: %v", err)
	}
	if updated == nil {
		t.Fatal("updated snapshot should not be nil")
	}
	if len(result.ChangesApplied) == 0 {
		t.Error("expected changes from concise directive")
	}
}

func TestPatchIdentity_Success(t *testing.T) {
	s := testhelper.NewStorage(t)
	uc := interactors.NewIdentityUpdateUseCase(s)
	ctx := context.Background()

	snap := testhelper.MakeFullSnapshot("agent-patch", "gpt-4")
	s.StoreIdentity(ctx, snap)

	formality := 0.9
	patch := &valueobjects.IdentityPatch{
		FormalityLevel: &formality,
	}

	updated, result, err := uc.PatchIdentity(ctx, "agent-patch", patch)
	if err != nil {
		t.Fatalf("PatchIdentity error: %v", err)
	}
	if updated == nil {
		t.Fatal("updated snapshot should not be nil")
	}
	if len(result.ChangesApplied) == 0 {
		t.Error("expected changes from patch")
	}
	if updated.Version != snap.Version+1 {
		t.Errorf("expected version %d, got %d", snap.Version+1, updated.Version)
	}
}

func TestPatchIdentity_NewAgent(t *testing.T) {
	s := testhelper.NewStorage(t)
	uc := interactors.NewIdentityUpdateUseCase(s)
	ctx := context.Background()

	warmth := 0.8
	patch := &valueobjects.IdentityPatch{
		Warmth: &warmth,
	}

	updated, result, err := uc.PatchIdentity(ctx, "brand-new-agent", patch)
	if err != nil {
		t.Fatalf("PatchIdentity on new agent error: %v", err)
	}
	if updated == nil {
		t.Fatal("updated snapshot should not be nil")
	}
	if len(result.ChangesApplied) == 0 {
		t.Error("expected changes from patch on new agent")
	}
}

func TestPatchIdentity_EmptyPatch(t *testing.T) {
	s := testhelper.NewStorage(t)
	uc := interactors.NewIdentityUpdateUseCase(s)
	ctx := context.Background()

	_, _, err := uc.PatchIdentity(ctx, "agent-empty", &valueobjects.IdentityPatch{})
	if err == nil {
		t.Error("expected error for empty patch")
	}
}

// ========== IdentityEvolutionUseCase ==========

func TestTrackSnapshot_WithDerivedFrom(t *testing.T) {
	s := testhelper.NewStorage(t)
	uc := interactors.NewIdentityEvolutionUseCase(s, nil)
	ctx := context.Background()

	first := testhelper.MakeFullSnapshot("agent-evo", "gpt-4")
	s.StoreIdentity(ctx, first)

	second := testhelper.MakeFullSnapshot("agent-evo", "gpt-4")
	second.WithParentSnapshot(first.ID)

	diff, err := uc.TrackSnapshot(ctx, second)
	if err != nil {
		t.Fatalf("TrackSnapshot error: %v", err)
	}
	if diff == nil {
		t.Fatal("diff should not be nil when tracking evolution from previous snapshot")
	}
	if diff.AgentID != "agent-evo" {
		t.Errorf("expected agent-evo, got %s", diff.AgentID)
	}
}

func TestTrackSnapshot_FirstSnapshot(t *testing.T) {
	s := testhelper.NewStorage(t)
	uc := interactors.NewIdentityEvolutionUseCase(s, nil)
	ctx := context.Background()

	first := testhelper.MakeFullSnapshot("agent-evo-first", "gpt-4")

	// First snapshot has no DerivedFromID → should return nil diff
	diff, err := uc.TrackSnapshot(ctx, first)
	if err != nil {
		t.Fatalf("TrackSnapshot error: %v", err)
	}
	if diff != nil {
		t.Error("expected nil diff for first snapshot")
	}
}

func TestGetEvolutionTimeline_Empty(t *testing.T) {
	s := testhelper.NewStorage(t)
	uc := interactors.NewIdentityEvolutionUseCase(s, nil)
	ctx := context.Background()

	timeline, err := uc.GetEvolutionTimeline(ctx, "no-such-agent")
	if err != nil {
		t.Fatalf("GetEvolutionTimeline error: %v", err)
	}
	if len(timeline) != 0 {
		t.Errorf("expected empty timeline, got %d entries", len(timeline))
	}
}

func TestGetEvolutionSummary_NoIdentity(t *testing.T) {
	s := testhelper.NewStorage(t)
	uc := interactors.NewIdentityEvolutionUseCase(s, nil)
	ctx := context.Background()

	summary, err := uc.GetEvolutionSummary(ctx, "no-agent")
	if err != nil {
		t.Fatalf("GetEvolutionSummary error: %v", err)
	}
	if summary == "" {
		t.Error("expected non-empty summary even for unknown agent")
	}
}

// ========== IdentityMergeUseCase ==========

func TestMergeIdentities_Success(t *testing.T) {
	s := testhelper.NewStorage(t)
	merger := modelswap.NewSoulMergerService()
	uc := interactors.NewIdentityMergeUseCase(s, merger)
	ctx := context.Background()

	agentA := testhelper.MakeFullSnapshot("agent-merge-a", "gpt-4")
	agentB := testhelper.MakeFullSnapshot("agent-merge-b", "gpt-4")
	agentB.AgentID = "agent-merge-b"
	s.StoreIdentity(ctx, agentA)
	s.StoreIdentity(ctx, agentB)

	merged, err := uc.MergeIdentities(ctx, "agent-merge-a", "agent-merge-b", valueobjects.MergePreserveDominant)
	if err != nil {
		t.Fatalf("MergeIdentities error: %v", err)
	}
	if merged == nil {
		t.Fatal("merged identity should not be nil")
	}
	if merged.AgentID != "agent-merge-a" {
		t.Errorf("expected agent-merge-a, got %s", merged.AgentID)
	}
}

func TestCalculateCompatibility_Success(t *testing.T) {
	s := testhelper.NewStorage(t)
	merger := modelswap.NewSoulMergerService()
	uc := interactors.NewIdentityMergeUseCase(s, merger)
	ctx := context.Background()

	agentA := testhelper.MakeFullSnapshot("agent-comp-a", "gpt-4")
	agentB := testhelper.MakeFullSnapshot("agent-comp-b", "gpt-4")
	agentB.AgentID = "agent-comp-b"
	s.StoreIdentity(ctx, agentA)
	s.StoreIdentity(ctx, agentB)

	compatibility, err := uc.CalculateCompatibility(ctx, "agent-comp-a", "agent-comp-b")
	if err != nil {
		t.Fatalf("CalculateCompatibility error: %v", err)
	}
	if compatibility < 0 || compatibility > 1 {
		t.Errorf("compatibility should be in [0,1], got %f", compatibility)
	}
}

func TestMergeIdentities_WithDifferentStrategies(t *testing.T) {
	s := testhelper.NewStorage(t)
	merger := modelswap.NewSoulMergerService()
	uc := interactors.NewIdentityMergeUseCase(s, merger)
	ctx := context.Background()

	agentA := testhelper.MakeFullSnapshot("agent-merge-strat-a", "gpt-4")
	agentB := testhelper.MakeFullSnapshot("agent-merge-strat-b", "gpt-4")
	agentB.AgentID = "agent-merge-strat-b"
	s.StoreIdentity(ctx, agentA)
	s.StoreIdentity(ctx, agentB)

	for _, strategy := range []valueobjects.MergeStrategy{
		valueobjects.MergePreserveDominant,
		valueobjects.MergeWeightedAverage,
		valueobjects.MergeLatestWins,
		valueobjects.MergeSynthesize,
	} {
		merged, err := uc.MergeIdentities(ctx, "agent-merge-strat-a", "agent-merge-strat-b", strategy)
		if err != nil {
			t.Errorf("MergeIdentities with strategy %s error: %v", strategy, err)
		}
		if merged == nil {
			t.Errorf("merged identity should not be nil for strategy %s", strategy)
		}
	}
}

// ========== ModelSwapUseCase ==========

func TestHandleModelSwap_Success(t *testing.T) {
	s := testhelper.NewStorage(t)
	handler := modelswap.NewSoulModelSwapHandler()
	composer := composition.NewSoulComposerService()
	uc := interactors.NewModelSwapUseCase(s, handler, composer)
	ctx := context.Background()

	snap := testhelper.MakeFullSnapshot("agent-swap", "gpt-4")
	s.StoreIdentity(ctx, snap)

	swapCtx, err := uc.HandleModelSwap(ctx, "agent-swap", "gpt-4", "gpt-5")
	if err != nil {
		t.Fatalf("HandleModelSwap error: %v", err)
	}
	if swapCtx == nil {
		t.Fatal("swap context should not be nil")
	}
	if swapCtx.PreviousModel != "gpt-4" {
		t.Errorf("expected previous model gpt-4, got %s", swapCtx.PreviousModel)
	}
	if swapCtx.NewModel != "gpt-5" {
		t.Errorf("expected new model gpt-5, got %s", swapCtx.NewModel)
	}
}

func TestHandleModelSwap_NoExistingIdentity(t *testing.T) {
	s := testhelper.NewStorage(t)
	handler := modelswap.NewSoulModelSwapHandler()
	composer := composition.NewSoulComposerService()
	uc := interactors.NewModelSwapUseCase(s, handler, composer)
	ctx := context.Background()

	swapCtx, err := uc.HandleModelSwap(ctx, "unknown-agent", "gpt-3", "gpt-4")
	if err != nil {
		t.Fatalf("HandleModelSwap for unknown agent error: %v", err)
	}
	if swapCtx == nil {
		t.Fatal("swap context should not be nil even for unknown agent")
	}
}

func TestGetReinforcementPrompt(t *testing.T) {
	s := testhelper.NewStorage(t)
	handler := modelswap.NewSoulModelSwapHandler()
	composer := composition.NewSoulComposerService()
	uc := interactors.NewModelSwapUseCase(s, handler, composer)
	ctx := context.Background()

	snap := testhelper.MakeFullSnapshot("agent-reinf", "gpt-4")
	s.StoreIdentity(ctx, snap)

	prompt, err := uc.GetReinforcementPrompt(ctx, "agent-reinf")
	if err != nil {
		t.Fatalf("GetReinforcementPrompt error: %v", err)
	}
	if prompt == nil {
		t.Fatal("reinforcement prompt should not be nil")
	}
}

func TestValidateIdentityPreserved_NoSwap(t *testing.T) {
	s := testhelper.NewStorage(t)
	handler := modelswap.NewSoulModelSwapHandler()
	composer := composition.NewSoulComposerService()
	uc := interactors.NewModelSwapUseCase(s, handler, composer)
	ctx := context.Background()

	preserved, err := uc.ValidateIdentityPreserved(ctx, "unknown-agent")
	if err != nil {
		t.Fatalf("ValidateIdentityPreserved error: %v", err)
	}
	if !preserved {
		t.Error("expected preserved=true for agent with no swap history")
	}
}

func TestGetSwapHistory_Empty(t *testing.T) {
	s := testhelper.NewStorage(t)
	handler := modelswap.NewSoulModelSwapHandler()
	composer := composition.NewSoulComposerService()
	uc := interactors.NewModelSwapUseCase(s, handler, composer)
	ctx := context.Background()

	history, err := uc.GetSwapHistory(ctx, "no-swaps-agent")
	if err != nil {
		t.Fatalf("GetSwapHistory error: %v", err)
	}
	if len(history) != 0 {
		t.Errorf("expected empty history, got %d entries", len(history))
	}
}

func TestHandleModelSwap_ReinforceAndValidate(t *testing.T) {
	s := testhelper.NewStorage(t)
	handler := modelswap.NewSoulModelSwapHandler()
	composer := composition.NewSoulComposerService()
	uc := interactors.NewModelSwapUseCase(s, handler, composer)
	ctx := context.Background()

	snap := testhelper.MakeFullSnapshot("agent-swap-full", "gpt-4")
	s.StoreIdentity(ctx, snap)

	// Swap from gpt-4 to gpt-5
	swapCtx, err := uc.HandleModelSwap(ctx, "agent-swap-full", "gpt-4", "gpt-5")
	if err != nil {
		t.Fatalf("HandleModelSwap error: %v", err)
	}
	if swapCtx == nil {
		t.Fatal("swap context should not be nil")
	}

	// Get reinforcement prompt after swap
	prompt, err := uc.GetReinforcementPrompt(ctx, "agent-swap-full")
	if err != nil {
		t.Fatalf("GetReinforcementPrompt error: %v", err)
	}
	if prompt == nil || prompt.Content == "" {
		t.Error("reinforcement prompt should have content")
	}

	// Check swap history
	history, err := uc.GetSwapHistory(ctx, "agent-swap-full")
	if err != nil {
		t.Fatalf("GetSwapHistory error: %v", err)
	}
	if len(history) == 0 {
		t.Error("expected at least 1 swap history entry")
	}
}

func TestMeasurePostSwapDrift(t *testing.T) {
	s := testhelper.NewStorage(t)
	handler := modelswap.NewSoulModelSwapHandler()
	composer := composition.NewSoulComposerService()
	uc := interactors.NewModelSwapUseCase(s, handler, composer)
	ctx := context.Background()

	snap := testhelper.MakeFullSnapshot("agent-msd", "gpt-4")
	s.StoreIdentity(ctx, snap)

	uc.HandleModelSwap(ctx, "agent-msd", "gpt-4", "gpt-5")

	drift, err := uc.MeasurePostSwapDrift(ctx, "agent-msd")
	if err != nil {
		t.Fatalf("MeasurePostSwapDrift error: %v", err)
	}
	if drift < 0 || drift > 1 {
		t.Errorf("drift should be in [0,1], got %f", drift)
	}
}
