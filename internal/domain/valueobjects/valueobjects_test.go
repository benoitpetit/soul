package valueobjects

import (
	"testing"
	"time"
)

func TestIdentityVersion(t *testing.T) {
	v := NewIdentityVersion(1, 2, 3)
	if v.Major != 1 || v.Minor != 2 || v.Patch != 3 {
		t.Errorf("expected 1.2.3, got %d.%d.%d", v.Major, v.Minor, v.Patch)
	}
	if v.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

func TestIdentityVersionString(t *testing.T) {
	v := IdentityVersion{Major: 2, Minor: 1, Patch: 0}
	if v.String() != "2.1.0" {
		t.Errorf("expected 2.1.0, got %s", v.String())
	}
}

func TestSourceTypeConstants(t *testing.T) {
	tests := []struct {
		src  SourceType
		want string
	}{
		{SourceConversation, "conversation"},
		{SourceFeedback, "feedback"},
		{SourceSelfReflection, "self_reflection"},
		{SourceObservation, "observation"},
		{SourceMemoryMira, "mira_memory"},
	}
	for _, tt := range tests {
		if string(tt.src) != tt.want {
			t.Errorf("got %s, want %s", tt.src, tt.want)
		}
	}
}

func TestExtractedTrait(t *testing.T) {
	trait := ExtractedTrait{
		Name:       "curious",
		Category:   "cognition",
		Intensity:  0.8,
		Evidence:   "asks many questions",
		Confidence: 0.75,
		Source: IdentitySource{
			Type:      SourceConversation,
			Content:   "test content",
			Timestamp: time.Now(),
		},
	}
	if trait.Name != "curious" {
		t.Errorf("expected curious, got %s", trait.Name)
	}
	if trait.Intensity != 0.8 {
		t.Errorf("expected 0.8, got %f", trait.Intensity)
	}
}

func TestIdentityContextPrompt(t *testing.T) {
	prompt := IdentityContextPrompt{
		Content:         "You are a curious AI assistant",
		TokenEstimate:   150,
		Priority:        1,
		GeneratedAt:     time.Now(),
		SnapshotVersion: 3,
	}
	if prompt.TokenEstimate != 150 {
		t.Errorf("expected 150, got %d", prompt.TokenEstimate)
	}
	if prompt.Priority != 1 {
		t.Errorf("expected 1, got %d", prompt.Priority)
	}
}

func TestIdentityDriftReport(t *testing.T) {
	report := IdentityDriftReport{
		Timestamp:       time.Now(),
		PreviousVersion: 2,
		CurrentVersion:  3,
		DriftScore:      0.35,
		DriftDimensions: []DimensionDrift{
			{
				Dimension:     "voice",
				PreviousValue: 0.7,
				CurrentValue:   0.5,
				Change:         0.2,
				IsSignificant: true,
			},
		},
		IsSignificant:   true,
		Recommendations:  []string{"reinfuse identity"},
	}
	if report.DriftScore != 0.35 {
		t.Errorf("expected 0.35, got %f", report.DriftScore)
	}
	if len(report.DriftDimensions) != 1 {
		t.Errorf("expected 1 dimension, got %d", len(report.DriftDimensions))
	}
}

func TestDimensionDrift(t *testing.T) {
	d := DimensionDrift{
		Dimension:     "personality",
		PreviousValue: 0.6,
		CurrentValue:   0.8,
		Change:         0.2,
		IsSignificant: true,
	}
	if d.Change != 0.2 {
		t.Errorf("expected 0.2, got %f", d.Change)
	}
}

func TestModelSwapContext(t *testing.T) {
	ctx := ModelSwapContext{
		AgentID:             "agent-123",
		PreviousModel:      "gpt-4",
		NewModel:           "gpt-4o",
		Timestamp:          time.Now(),
		IdentityPreserved:  true,
		IdentityDrift:      0.15,
		ReinforcementApplied: true,
	}
	if !ctx.IdentityPreserved {
		t.Error("expected IdentityPreserved to be true")
	}
	if ctx.IdentityDrift != 0.15 {
		t.Errorf("expected 0.15, got %f", ctx.IdentityDrift)
	}
}

func TestSoulQuery(t *testing.T) {
	q := SoulQuery{
		AgentID:         "agent-456",
		Context:         "coding assistance",
		BudgetTokens:    500,
		PrioritizeRecent: true,
		IncludeTraits:   []string{"curious", "helpful"},
		ExcludeTraits:   []string{"lazy"},
	}
	if q.BudgetTokens != 500 {
		t.Errorf("expected 500, got %d", q.BudgetTokens)
	}
	if len(q.IncludeTraits) != 2 {
		t.Errorf("expected 2 traits, got %d", len(q.IncludeTraits))
	}
}

func TestSoulCaptureRequest(t *testing.T) {
	req := SoulCaptureRequest{
		AgentID:        "agent-789",
		Conversation:   "Hello, how are you?",
		AgentResponses: []string{"I'm doing great!"},
		UserFeedback:  map[string]string{"tone": "friendly"},
		ModelID:       "gpt-4",
		SessionID:     "sess-abc",
		Timestamp:     time.Now(),
	}
	if req.AgentID != "agent-789" {
		t.Errorf("expected agent-789, got %s", req.AgentID)
	}
	if len(req.AgentResponses) != 1 {
		t.Errorf("expected 1 response, got %d", len(req.AgentResponses))
	}
}

func TestMergeStrategyConstants(t *testing.T) {
	tests := []struct {
		strat MergeStrategy
		want  string
	}{
		{MergePreserveDominant, "preserve_dominant"},
		{MergeWeightedAverage, "weighted_average"},
		{MergeLatestWins, "latest_wins"},
		{MergeSynthesize, "synthesize"},
	}
	for _, tt := range tests {
		if string(tt.strat) != tt.want {
			t.Errorf("got %s, want %s", tt.strat, tt.want)
		}
	}
}