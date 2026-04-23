package composition

import (
	"context"
	"strings"
	"testing"

	"github.com/benoitpetit/soul/internal/domain/entities"
	"github.com/benoitpetit/soul/internal/domain/valueobjects"
)

// --- Helpers ---

func makeFullSnapshot(agentID string) *entities.IdentitySnapshot {
	snap := entities.NewIdentitySnapshot(agentID, "gpt-4")
	snap.VoiceProfile = *entities.NewVoiceProfile()
	snap.VoiceProfile.FormalityLevel = 0.8
	snap.VoiceProfile.HumorLevel = 0.6
	snap.VoiceProfile.EmpathyLevel = 0.7
	snap.VoiceProfile.DirectnessLevel = 0.9
	snap.VoiceProfile.TechnicalDepth = 0.8
	snap.CommunicationStyle = *entities.NewCommunicationStyle()
	snap.BehavioralSignature = *entities.NewBehavioralSignature()
	snap.ValueSystem = *entities.NewValueSystem()
	snap.EmotionalTone = *entities.NewEmotionalTone()
	return snap
}

// --- Constructor ---

func TestNewSoulComposerService(t *testing.T) {
	s := NewSoulComposerService()
	if s == nil {
		t.Fatal("NewSoulComposerService should not return nil")
	}
	if s.baseTemplate == "" {
		t.Error("baseTemplate must not be empty")
	}
	if s.reinforceTemplate == "" {
		t.Error("reinforceTemplate must not be empty")
	}
	if s.alertTemplate == "" {
		t.Error("alertTemplate must not be empty")
	}
}

// --- ComposeIdentityPrompt ---

func TestComposeIdentityPrompt_ReturnsPrompt(t *testing.T) {
	s := NewSoulComposerService()
	ctx := context.Background()
	snap := makeFullSnapshot("agent-1")

	result, err := s.ComposeIdentityPrompt(ctx, snap, 0)
	if err != nil {
		t.Fatalf("ComposeIdentityPrompt error: %v", err)
	}
	if result == nil {
		t.Fatal("result must not be nil")
	}
	if result.Content == "" {
		t.Error("content must not be empty")
	}
	if result.TokenEstimate <= 0 {
		t.Error("token estimate must be > 0")
	}
	if result.Priority != 100 {
		t.Errorf("priority: got %d, want 100", result.Priority)
	}
	if result.SnapshotVersion != snap.Version {
		t.Errorf("snapshot version: got %d, want %d", result.SnapshotVersion, snap.Version)
	}
}

func TestComposeIdentityPrompt_NoRawTemplateTags(t *testing.T) {
	s := NewSoulComposerService()
	ctx := context.Background()
	snap := makeFullSnapshot("agent-1")

	result, err := s.ComposeIdentityPrompt(ctx, snap, 0)
	if err != nil {
		t.Fatalf("ComposeIdentityPrompt error: %v", err)
	}

	// All {{...}} placeholders must have been replaced.
	if strings.Contains(result.Content, "{{") {
		t.Error("result content still contains unreplaced template tags")
	}
}

func TestComposeIdentityPrompt_WithPersonalityTraits(t *testing.T) {
	s := NewSoulComposerService()
	ctx := context.Background()
	snap := makeFullSnapshot("agent-1")

	trait := entities.NewPersonalityTrait("analytical", entities.TraitCognitive, 0.9)
	snap.WithTraits(*trait)

	result, err := s.ComposeIdentityPrompt(ctx, snap, 0)
	if err != nil {
		t.Fatalf("ComposeIdentityPrompt error: %v", err)
	}
	if result.Content == "" {
		t.Error("content must not be empty with traits")
	}
}

func TestComposeIdentityPrompt_BudgetTruncates(t *testing.T) {
	s := NewSoulComposerService()
	ctx := context.Background()
	snap := makeFullSnapshot("agent-1")

	// Full prompt is typically 200+ tokens; request 5 to force truncation.
	result, err := s.ComposeIdentityPrompt(ctx, snap, 5)
	if err != nil {
		t.Fatalf("ComposeIdentityPrompt error: %v", err)
	}
	if result == nil {
		t.Fatal("result must not be nil")
	}
	// Actual content length should be constrained.
	if len(result.Content) > 5*4+50 { // some slack
		t.Errorf("content (%d chars) should be smaller when budget=5 tokens", len(result.Content))
	}
}

func TestComposeIdentityPrompt_ModeratebudgetSuffix(t *testing.T) {
	s := NewSoulComposerService()
	ctx := context.Background()
	snap := makeFullSnapshot("agent-1")

	// Build a prompt first to know the natural size.
	full, err := s.ComposeIdentityPrompt(ctx, snap, 0)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	naturalTokens := full.TokenEstimate

	// Request half the natural size — reduction ratio ≥ 0.5, so simple truncation path.
	result, err := s.ComposeIdentityPrompt(ctx, snap, naturalTokens/2)
	if err != nil {
		t.Fatalf("ComposeIdentityPrompt error: %v", err)
	}
	if result == nil {
		t.Fatal("result must not be nil")
	}
}

// --- ComposeReinforcementPrompt ---

func TestComposeReinforcementPrompt_WithSwap(t *testing.T) {
	s := NewSoulComposerService()
	ctx := context.Background()
	snap := makeFullSnapshot("agent-1")
	swap := &valueobjects.ModelSwapContext{
		AgentID:       "agent-1",
		PreviousModel: "gpt-3.5",
		NewModel:      "gpt-4",
	}

	result, err := s.ComposeReinforcementPrompt(ctx, snap, swap)
	if err != nil {
		t.Fatalf("ComposeReinforcementPrompt error: %v", err)
	}
	if result == nil {
		t.Fatal("result must not be nil")
	}
	if !strings.Contains(result.Content, "gpt-3.5") {
		t.Error("reinforcement prompt should mention previous model")
	}
	if !strings.Contains(result.Content, "gpt-4") {
		t.Error("reinforcement prompt should mention new model")
	}
	if result.Priority != 100 {
		t.Errorf("priority: got %d, want 100", result.Priority)
	}
}

func TestComposeReinforcementPrompt_NoSwap(t *testing.T) {
	s := NewSoulComposerService()
	ctx := context.Background()
	snap := makeFullSnapshot("agent-1")

	result, err := s.ComposeReinforcementPrompt(ctx, snap, nil)
	if err != nil {
		t.Fatalf("ComposeReinforcementPrompt error: %v", err)
	}
	if result == nil {
		t.Fatal("result must not be nil")
	}
	if result.Content == "" {
		t.Error("content must not be empty")
	}
}

func TestComposeReinforcementPrompt_WithHighConfidenceTraits(t *testing.T) {
	s := NewSoulComposerService()
	ctx := context.Background()
	snap := makeFullSnapshot("agent-1")

	// Add a trait with confidence > 0.8 — should appear in critical markers.
	trait := entities.NewPersonalityTrait("empathetic", entities.TraitEmotional, 0.9)
	for i := 0; i < 10; i++ {
		trait.WithEvidence("evidence", "ctx")
	}
	snap.WithTraits(*trait)

	result, err := s.ComposeReinforcementPrompt(ctx, snap, nil)
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if result.Content == "" {
		t.Error("content must not be empty")
	}
}

// --- ComposeDiffusionAlert ---

func TestComposeDiffusionAlert_NilDrift(t *testing.T) {
	s := NewSoulComposerService()
	ctx := context.Background()

	_, err := s.ComposeDiffusionAlert(ctx, nil)
	if err == nil {
		t.Error("ComposeDiffusionAlert with nil drift should return error")
	}
}

func TestComposeDiffusionAlert_WithSignificantDimensions(t *testing.T) {
	s := NewSoulComposerService()
	ctx := context.Background()

	report := &valueobjects.IdentityDriftReport{
		DriftScore:      0.5,
		IsSignificant:   true,
		CurrentVersion:  2,
		PreviousVersion: 1,
		DriftDimensions: []valueobjects.DimensionDrift{
			{Dimension: "voice_profile", Change: 0.6, IsSignificant: true},
			{Dimension: "personality_traits", Change: 0.2, IsSignificant: false},
		},
	}

	result, err := s.ComposeDiffusionAlert(ctx, report)
	if err != nil {
		t.Fatalf("ComposeDiffusionAlert error: %v", err)
	}
	if result == nil {
		t.Fatal("result must not be nil")
	}
	if !strings.Contains(result.Content, "voice_profile") {
		t.Error("alert should mention the significant dimension")
	}
	if result.Priority != 90 {
		t.Errorf("priority: got %d, want 90", result.Priority)
	}
	if result.SnapshotVersion != 2 {
		t.Errorf("snapshot version: got %d, want 2", result.SnapshotVersion)
	}
}

func TestComposeDiffusionAlert_NoSignificantDimensions(t *testing.T) {
	s := NewSoulComposerService()
	ctx := context.Background()

	report := &valueobjects.IdentityDriftReport{
		DriftScore:     0.1,
		IsSignificant:  false,
		CurrentVersion: 1,
		DriftDimensions: []valueobjects.DimensionDrift{
			{Dimension: "voice_profile", Change: 0.05, IsSignificant: false},
		},
	}

	result, err := s.ComposeDiffusionAlert(ctx, report)
	if err != nil {
		t.Fatalf("ComposeDiffusionAlert error: %v", err)
	}
	// Fallback message should be included when no significant dimensions.
	if !strings.Contains(result.Content, "General identity drift") {
		t.Error("alert should include fallback message when no significant dimensions")
	}
}

// --- EstimateTokenCount ---

func TestEstimateTokenCount_Empty(t *testing.T) {
	s := NewSoulComposerService()
	ctx := context.Background()

	count, err := s.EstimateTokenCount(ctx, "")
	if err != nil {
		t.Fatalf("EstimateTokenCount error: %v", err)
	}
	if count != 0 {
		t.Errorf("empty string: got %d, want 0", count)
	}
}

func TestEstimateTokenCount_KnownLength(t *testing.T) {
	s := NewSoulComposerService()
	ctx := context.Background()

	text := "hello world test string"
	count, err := s.EstimateTokenCount(ctx, text)
	if err != nil {
		t.Fatalf("EstimateTokenCount error: %v", err)
	}
	if count <= 0 {
		t.Errorf("expected positive token count, got %d", count)
	}
}

// --- Private helpers (accessible from same package) ---

func TestReplaceTag_Basic(t *testing.T) {
	result := replaceTag("Hello {{NAME}}!", "{{NAME}}", "World")
	if result != "Hello World!" {
		t.Errorf("replaceTag: got %q, want %q", result, "Hello World!")
	}
}

func TestReplaceTag_MultipleOccurrences(t *testing.T) {
	result := replaceTag("{{A}} and {{A}}", "{{A}}", "X")
	if result != "X and X" {
		t.Errorf("replaceTag multi: got %q", result)
	}
}

func TestReplaceTag_NoOccurrence(t *testing.T) {
	result := replaceTag("hello world", "{{MISSING}}", "X")
	if result != "hello world" {
		t.Errorf("replaceTag no-op: got %q", result)
	}
}

func TestFormatPersonalityTraits_Empty(t *testing.T) {
	s := NewSoulComposerService()
	result := s.formatPersonalityTraits(nil)
	if result == "" {
		t.Error("formatPersonalityTraits empty should return non-empty fallback")
	}
}

func TestFormatPersonalityTraits_WithWellEstablishedAndDeveloping(t *testing.T) {
	s := NewSoulComposerService()

	// Create a well-established trait (needs Confidence > 0.7, EvidenceCount >= 5, Consistency > 0.5)
	wellEstab := entities.NewPersonalityTrait("analytical", entities.TraitCognitive, 0.9)
	for i := 0; i < 6; i++ {
		wellEstab.WithEvidence("evidence", "ctx"+string(rune('a'+i)))
	}

	// Create a developing trait (low confidence)
	developing := entities.NewPersonalityTrait("creative", entities.TraitExpressive, 0.5)
	developing.WithEvidence("some evidence", "creative_ctx")

	traits := []entities.PersonalityTrait{*wellEstab, *developing}
	result := s.formatPersonalityTraits(traits)

	if !strings.Contains(result, "Core traits") {
		t.Error("should include 'Core traits' section for well-established traits")
	}
}

func TestSummarizeVoice_Descriptors(t *testing.T) {
	s := NewSoulComposerService()

	voice := entities.NewVoiceProfile()
	voice.FormalityLevel = 0.9 // → "formal"
	voice.HumorLevel = 0.7     // → "humorous"
	voice.EmpathyLevel = 0.8   // → "empathetic"

	result := s.summarizeVoice(voice)
	if !strings.Contains(result, "formal") {
		t.Error("expected 'formal' in voice summary")
	}
	if !strings.Contains(result, "humorous") {
		t.Error("expected 'humorous' in voice summary")
	}
}

func TestSummarizeVoice_Balanced(t *testing.T) {
	s := NewSoulComposerService()
	voice := entities.NewVoiceProfile() // defaults → mid-range
	result := s.summarizeVoice(voice)
	if result == "" {
		t.Error("summarizeVoice should never be empty")
	}
}

func TestGenerateCondensedSummary(t *testing.T) {
	s := NewSoulComposerService()
	snap := makeFullSnapshot("agent-1")

	trait := entities.NewPersonalityTrait("direct", entities.TraitSocial, 0.8)
	for i := 0; i < 4; i++ {
		trait.WithEvidence("e", "ctx")
	}
	snap.WithTraits(*trait)

	summary := s.generateCondensedSummary(snap)
	if summary == "" {
		t.Error("condensed summary must not be empty")
	}
	if !strings.Contains(summary, "Voice") {
		t.Error("condensed summary should contain voice info")
	}
}
