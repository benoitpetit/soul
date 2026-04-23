// Package interactors_test tests use cases with real in-memory SQLite storage.
package interactors_test

import (
	"context"
	"testing"

	"github.com/benoitpetit/soul/internal/adapters/composition"
	"github.com/benoitpetit/soul/internal/adapters/drift"
	"github.com/benoitpetit/soul/internal/adapters/extraction"
	"github.com/benoitpetit/soul/internal/adapters/sqlite"
	"github.com/benoitpetit/soul/internal/domain/entities"
	"github.com/benoitpetit/soul/internal/domain/valueobjects"
	"github.com/benoitpetit/soul/internal/usecases/interactors"
)

// newStorage creates a fresh in-memory SQLite for testing.
func newStorage(t *testing.T) *sqlite.SoulSQLiteStorage {
	t.Helper()
	s, err := sqlite.NewSoulSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// makeFullSnapshot creates a populated snapshot for testing.
func makeFullSnapshot(agentID, modelID string) *entities.IdentitySnapshot {
	snap := entities.NewIdentitySnapshot(agentID, modelID)
	snap.VoiceProfile = *entities.NewVoiceProfile()
	snap.CommunicationStyle = *entities.NewCommunicationStyle()
	snap.BehavioralSignature = *entities.NewBehavioralSignature()
	snap.ValueSystem = *entities.NewValueSystem()
	snap.EmotionalTone = *entities.NewEmotionalTone()
	tr := entities.NewPersonalityTrait("analytical", entities.TraitCognitive, 0.8)
	tr.Confidence = 0.75
	snap.WithTraits(*tr)
	return snap
}

// ========== IdentityCaptureUseCase ==========

func TestCaptureFromConversation_CreatesNewSnapshot(t *testing.T) {
	s := newStorage(t)
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

	// Verify it was persisted
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
	s := newStorage(t)
	extractor := extraction.NewSoulExtractorService()
	uc := interactors.NewIdentityCaptureUseCase(s, extractor)
	ctx := context.Background()

	// Store an initial snapshot
	initial := makeFullSnapshot("agent-v2", "gpt-4")
	if err := s.StoreIdentity(ctx, initial); err != nil {
		t.Fatalf("StoreIdentity error: %v", err)
	}

	// Capture a second time → should create a derived snapshot
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
	s := newStorage(t)
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

// ========== IdentityRecallUseCase ==========

func TestRecallIdentity_Success(t *testing.T) {
	s := newStorage(t)
	composer := composition.NewSoulComposerService()
	uc := interactors.NewIdentityRecallUseCase(s, composer)
	ctx := context.Background()

	snap := makeFullSnapshot("agent-recall", "gpt-4")
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
	s := newStorage(t)
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
	s := newStorage(t)
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
	s := newStorage(t)
	composer := composition.NewSoulComposerService()
	uc := interactors.NewIdentityRecallUseCase(s, composer)
	ctx := context.Background()

	snap := makeFullSnapshot("agent-sum", "gpt-4")
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
	s := newStorage(t)
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
	s := newStorage(t)
	composer := composition.NewSoulComposerService()
	uc := interactors.NewIdentityRecallUseCase(s, composer)
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		snap := makeFullSnapshot("agent-hist-uc", "gpt-4")
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
	s := newStorage(t)
	detector := drift.NewSoulDriftDetector(0.3)
	composer := composition.NewSoulComposerService()
	uc := interactors.NewDriftDetectionUseCase(s, detector, composer)
	ctx := context.Background()

	// Store reference identity
	reference := makeFullSnapshot("agent-drift-uc", "gpt-4")
	s.StoreIdentity(ctx, reference)

	// Create a "current" observed identity (slightly different)
	current := makeFullSnapshot("agent-drift-uc", "gpt-4")
	current.VoiceProfile.FormalityLevel = 0.9 // Changed

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
	s := newStorage(t)
	detector := drift.NewSoulDriftDetector(0.3)
	composer := composition.NewSoulComposerService()
	uc := interactors.NewDriftDetectionUseCase(s, detector, composer)
	ctx := context.Background()

	current := makeFullSnapshot("no-agent", "gpt-4")
	_, err := uc.CheckDrift(ctx, "no-agent", current)
	if err == nil {
		t.Error("CheckDrift without reference identity should return error")
	}
}

func TestCheckDiffusion(t *testing.T) {
	s := newStorage(t)
	detector := drift.NewSoulDriftDetector(0.3)
	composer := composition.NewSoulComposerService()
	uc := interactors.NewDriftDetectionUseCase(s, detector, composer)
	ctx := context.Background()

	// Agent with no identity → should be diffused
	snap := entities.NewIdentitySnapshot("diffuse-agent", "gpt-4")
	snap.VoiceProfile = *entities.NewVoiceProfile()
	snap.CommunicationStyle = *entities.NewCommunicationStyle()
	snap.BehavioralSignature = *entities.NewBehavioralSignature()
	snap.ValueSystem = *entities.NewValueSystem()
	snap.EmotionalTone = *entities.NewEmotionalTone()
	// No traits → should be diffused
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
	s := newStorage(t)
	detector := drift.NewSoulDriftDetector(0.3)
	composer := composition.NewSoulComposerService()
	uc := interactors.NewDriftDetectionUseCase(s, detector, composer)
	ctx := context.Background()

	history, err := uc.GetDiffHistory(ctx, "no-agent", 10)
	if err != nil {
		t.Fatalf("GetDiffHistory error: %v", err)
	}
	_ = history // Should be empty but not error
}

func TestRestoreIdentity(t *testing.T) {
	s := newStorage(t)
	detector := drift.NewSoulDriftDetector(0.3)
	composer := composition.NewSoulComposerService()
	uc := interactors.NewDriftDetectionUseCase(s, detector, composer)
	ctx := context.Background()

	snap := makeFullSnapshot("agent-restore", "gpt-4")
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
