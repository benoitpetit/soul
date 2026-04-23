package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/benoitpetit/soul/internal/domain/entities"
	"github.com/benoitpetit/soul/internal/domain/valueobjects"
	"github.com/google/uuid"
)

// newTestStorage creates an in-memory SQLite storage for testing.
func newTestStorage(t *testing.T) *SoulSQLiteStorage {
	t.Helper()
	s, err := NewSoulSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test storage: %v", err)
	}
	t.Cleanup(func() {
		s.db.Close()
	})
	return s
}

// makeSnapshot builds a fully populated snapshot for tests.
func makeSnapshot(agentID, modelID string) *entities.IdentitySnapshot {
	snap := entities.NewIdentitySnapshot(agentID, modelID)
	snap.VoiceProfile = *entities.NewVoiceProfile()
	snap.CommunicationStyle = *entities.NewCommunicationStyle()
	snap.BehavioralSignature = *entities.NewBehavioralSignature()
	snap.ValueSystem = *entities.NewValueSystem()
	snap.EmotionalTone = *entities.NewEmotionalTone()
	trait := entities.NewPersonalityTrait("analytical", entities.TraitCognitive, 0.8)
	trait.Confidence = 0.75
	snap.WithTraits(*trait)
	return snap
}

// --- IdentityRepository ---

func TestStoreAndGetLatestIdentity(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	snap := makeSnapshot("agent-1", "gpt-4")

	if err := s.StoreIdentity(ctx, snap); err != nil {
		t.Fatalf("StoreIdentity error: %v", err)
	}

	retrieved, err := s.GetLatestIdentity(ctx, "agent-1")
	if err != nil {
		t.Fatalf("GetLatestIdentity error: %v", err)
	}
	if retrieved.ID != snap.ID {
		t.Errorf("ID: got %v, want %v", retrieved.ID, snap.ID)
	}
	if retrieved.AgentID != snap.AgentID {
		t.Errorf("AgentID: got %q, want %q", retrieved.AgentID, snap.AgentID)
	}
}

func TestGetLatestIdentity_NotFound(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	result, err := s.GetLatestIdentity(ctx, "no-such-agent")
	if err != nil {
		t.Fatalf("GetLatestIdentity for unknown agent returned unexpected error: %v", err)
	}
	if result != nil {
		t.Error("GetLatestIdentity for unknown agent should return nil identity")
	}
}

func TestGetIdentityByID(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	snap := makeSnapshot("agent-2", "gpt-4")
	if err := s.StoreIdentity(ctx, snap); err != nil {
		t.Fatalf("StoreIdentity error: %v", err)
	}

	retrieved, err := s.GetIdentityByID(ctx, snap.ID)
	if err != nil {
		t.Fatalf("GetIdentityByID error: %v", err)
	}
	if retrieved.ID != snap.ID {
		t.Errorf("ID: got %v, want %v", retrieved.ID, snap.ID)
	}
}

func TestGetIdentityByID_NotFound(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	result, err := s.GetIdentityByID(ctx, uuid.New())
	if err != nil {
		t.Fatalf("GetIdentityByID for unknown ID returned unexpected error: %v", err)
	}
	if result != nil {
		t.Error("GetIdentityByID for unknown ID should return nil identity")
	}
}

func TestGetIdentityHistory(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		snap := makeSnapshot("agent-hist", "gpt-4")
		snap.Version = i + 1
		if err := s.StoreIdentity(ctx, snap); err != nil {
			t.Fatalf("StoreIdentity v%d error: %v", i+1, err)
		}
	}

	history, err := s.GetIdentityHistory(ctx, "agent-hist", 10)
	if err != nil {
		t.Fatalf("GetIdentityHistory error: %v", err)
	}
	if len(history) != 3 {
		t.Errorf("History length: got %d, want 3", len(history))
	}
}

func TestDeleteIdentity(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	snap := makeSnapshot("agent-del", "gpt-4")
	if err := s.StoreIdentity(ctx, snap); err != nil {
		t.Fatalf("StoreIdentity error: %v", err)
	}

	if err := s.DeleteIdentity(ctx, snap.ID); err != nil {
		t.Fatalf("DeleteIdentity error: %v", err)
	}

	result, err := s.GetIdentityByID(ctx, snap.ID)
	if err != nil {
		t.Fatalf("GetIdentityByID after delete returned unexpected error: %v", err)
	}
	if result != nil {
		t.Error("GetIdentityByID after delete should return nil identity")
	}
}

func TestListAgents(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	snap1 := makeSnapshot("agent-A", "gpt-4")
	snap2 := makeSnapshot("agent-B", "gpt-4")

	s.StoreIdentity(ctx, snap1)
	s.StoreIdentity(ctx, snap2)

	agents, err := s.ListAgents(ctx)
	if err != nil {
		t.Fatalf("ListAgents error: %v", err)
	}

	agentSet := make(map[string]bool)
	for _, a := range agents {
		agentSet[a] = true
	}
	if !agentSet["agent-A"] || !agentSet["agent-B"] {
		t.Errorf("ListAgents: expected both agents, got %v", agents)
	}
}

func TestGetIdentityAtVersion(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	snap := makeSnapshot("agent-ver", "gpt-4")
	snap.Version = 5
	if err := s.StoreIdentity(ctx, snap); err != nil {
		t.Fatalf("StoreIdentity error: %v", err)
	}

	retrieved, err := s.GetIdentityAtVersion(ctx, "agent-ver", 5)
	if err != nil {
		t.Fatalf("GetIdentityAtVersion error: %v", err)
	}
	if retrieved.Version != 5 {
		t.Errorf("Version: got %d, want 5", retrieved.Version)
	}
}

// --- TraitRepository ---

func TestStoreAndGetTrait(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	trait := entities.NewPersonalityTrait("curious", entities.TraitEpistemic, 0.7)
	trait.AgentID = "agent-t1"

	if err := s.StoreTrait(ctx, trait); err != nil {
		t.Fatalf("StoreTrait error: %v", err)
	}

	retrieved, err := s.GetTraitByName(ctx, "agent-t1", "curious")
	if err != nil {
		t.Fatalf("GetTraitByName error: %v", err)
	}
	if retrieved.Name != "curious" {
		t.Errorf("Name: got %q, want %q", retrieved.Name, "curious")
	}
}

func TestGetAllTraits(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	agentID := "agent-all-traits"
	for _, name := range []string{"analytical", "curious", "empathetic"} {
		trait := entities.NewPersonalityTrait(name, entities.TraitCognitive, 0.7)
		trait.AgentID = agentID
		if err := s.StoreTrait(ctx, trait); err != nil {
			t.Fatalf("StoreTrait %q error: %v", name, err)
		}
	}

	traits, err := s.GetAllTraits(ctx, agentID)
	if err != nil {
		t.Fatalf("GetAllTraits error: %v", err)
	}
	if len(traits) != 3 {
		t.Errorf("GetAllTraits: got %d, want 3", len(traits))
	}
}

func TestGetWellEstablishedTraits(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	agentID := "agent-well"

	highConf := entities.NewPersonalityTrait("strong", entities.TraitCognitive, 0.9)
	highConf.AgentID = agentID
	highConf.Confidence = 0.9

	lowConf := entities.NewPersonalityTrait("weak", entities.TraitCognitive, 0.4)
	lowConf.AgentID = agentID
	lowConf.Confidence = 0.2

	s.StoreTrait(ctx, highConf)
	s.StoreTrait(ctx, lowConf)

	traits, err := s.GetWellEstablishedTraits(ctx, agentID, 0.7)
	if err != nil {
		t.Fatalf("GetWellEstablishedTraits error: %v", err)
	}
	if len(traits) != 1 {
		t.Errorf("GetWellEstablishedTraits: expected 1, got %d", len(traits))
	}
	if traits[0].Name != "strong" {
		t.Errorf("Expected 'strong', got %q", traits[0].Name)
	}
}

// --- TraitObservationRepository ---

func TestStoreAndGetObservations(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	obs := entities.NewTraitObservation("agent-obs", "curious", entities.TraitEpistemic, "evidence text", "ctx", 0.7)
	obs.SourceType = "conversation"

	if err := s.StoreObservation(ctx, obs); err != nil {
		t.Fatalf("StoreObservation error: %v", err)
	}

	observations, err := s.GetObservationsForTrait(ctx, "agent-obs", "curious", 10)
	if err != nil {
		t.Fatalf("GetObservationsForTrait error: %v", err)
	}
	if len(observations) != 1 {
		t.Errorf("Expected 1 observation, got %d", len(observations))
	}
}

func TestGetRecentObservations(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	agentID := "agent-recent"
	obs := entities.NewTraitObservation(agentID, "curious", entities.TraitEpistemic, "ev", "ctx", 0.7)
	if err := s.StoreObservation(ctx, obs); err != nil {
		t.Fatalf("StoreObservation error: %v", err)
	}

	recent, err := s.GetRecentObservations(ctx, agentID, time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("GetRecentObservations error: %v", err)
	}
	if len(recent) != 1 {
		t.Errorf("Expected 1 recent observation, got %d", len(recent))
	}
}

// --- EvolutionRepository ---

func TestRecordAndGetDiffs(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	diff := &entities.IdentityDiff{
		AgentID:     "agent-diff",
		FromVersion: 1,
		ToVersion:   2,
		Timestamp:   time.Now(),
		OverallDrift: 0.15,
		AddedTraits:        make([]entities.PersonalityTrait, 0),
		RemovedTraits:      make([]entities.PersonalityTrait, 0),
		StrengthenedTraits: make([]entities.PersonalityTrait, 0),
		WeakenedTraits:     make([]entities.PersonalityTrait, 0),
		VoiceChanges:       make([]string, 0),
		StyleChanges:       make([]string, 0),
		ValueChanges:       make([]string, 0),
	}

	if err := s.RecordDiff(ctx, diff); err != nil {
		t.Fatalf("RecordDiff error: %v", err)
	}

	diffs, err := s.GetDiffsForAgent(ctx, "agent-diff", 10)
	if err != nil {
		t.Fatalf("GetDiffsForAgent error: %v", err)
	}
	if len(diffs) != 1 {
		t.Errorf("Expected 1 diff, got %d", len(diffs))
	}
}

func TestGetLatestDiff(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	agentID := "agent-latestdiff"

	for i := 1; i <= 3; i++ {
		diff := &entities.IdentityDiff{
			AgentID:     agentID,
			FromVersion: i,
			ToVersion:   i + 1,
			Timestamp:   time.Now(),
			AddedTraits:        make([]entities.PersonalityTrait, 0),
			RemovedTraits:      make([]entities.PersonalityTrait, 0),
			StrengthenedTraits: make([]entities.PersonalityTrait, 0),
			WeakenedTraits:     make([]entities.PersonalityTrait, 0),
			VoiceChanges:       make([]string, 0),
			StyleChanges:       make([]string, 0),
			ValueChanges:       make([]string, 0),
		}
		if err := s.RecordDiff(ctx, diff); err != nil {
			t.Fatalf("RecordDiff error: %v", err)
		}
	}

	latest, err := s.GetLatestDiff(ctx, agentID)
	if err != nil {
		t.Fatalf("GetLatestDiff error: %v", err)
	}
	if latest.ToVersion != 4 {
		t.Errorf("Latest diff ToVersion: got %d, want 4", latest.ToVersion)
	}
}

// --- ModelSwapRepository ---

func TestRecordAndGetModelSwaps(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	swap := &valueobjects.ModelSwapContext{
		AgentID:           "agent-swap",
		PreviousModel:     "gpt-3.5",
		NewModel:          "gpt-4",
		Timestamp:         time.Now(),
		IdentityPreserved: true,
		IdentityDrift:     0.1,
	}

	if err := s.RecordModelSwap(ctx, swap); err != nil {
		t.Fatalf("RecordModelSwap error: %v", err)
	}

	swaps, err := s.GetModelSwaps(ctx, "agent-swap")
	if err != nil {
		t.Fatalf("GetModelSwaps error: %v", err)
	}
	if len(swaps) != 1 {
		t.Errorf("Expected 1 swap, got %d", len(swaps))
	}
}

// --- MiraBridgeRepository ---

func TestLinkAndGetLinkedMemories(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	identityID := uuid.New()
	memoryID := uuid.New()

	if err := s.LinkIdentityToMemory(ctx, identityID, memoryID); err != nil {
		t.Fatalf("LinkIdentityToMemory error: %v", err)
	}

	memories, err := s.GetLinkedMemories(ctx, identityID)
	if err != nil {
		t.Fatalf("GetLinkedMemories error: %v", err)
	}
	if len(memories) != 1 {
		t.Errorf("Expected 1 linked memory, got %d", len(memories))
	}
	if memories[0].MemoryID != memoryID {
		t.Errorf("MemoryID: got %v, want %v", memories[0].MemoryID, memoryID)
	}
}

func TestGetMiraMemories_Empty(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	// No memories stored → should return empty slice without error
	memories, err := s.GetMiraMemories(ctx, "agent-nomem", "query", 10)
	if err != nil {
		t.Fatalf("GetMiraMemories error: %v", err)
	}
	_ = memories // May be empty
}

// --- Transaction ---

func TestBeginTx_CommitRollback(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	tx, err := s.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx error: %v", err)
	}

	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback error: %v", err)
	}

	tx2, err := s.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx error: %v", err)
	}
	if err := tx2.Commit(); err != nil {
		t.Fatalf("Commit error: %v", err)
	}
}

// --- GetIdentityLineage ---

func TestGetIdentityLineage(t *testing.T) {
	s := newTestStorage(t)
	ctx := context.Background()

	parent := makeSnapshot("agent-lineage", "gpt-4")
	if err := s.StoreIdentity(ctx, parent); err != nil {
		t.Fatalf("StoreIdentity parent error: %v", err)
	}

	child := makeSnapshot("agent-lineage", "gpt-4")
	child.WithParentSnapshot(parent.ID)
	if err := s.StoreIdentity(ctx, child); err != nil {
		t.Fatalf("StoreIdentity child error: %v", err)
	}

	lineage, err := s.GetIdentityLineage(ctx, child.ID)
	if err != nil {
		t.Fatalf("GetIdentityLineage error: %v", err)
	}
	if lineage == nil {
		t.Fatal("lineage should not be nil")
	}
	if lineage.Depth < 1 {
		t.Errorf("Lineage depth should be >= 1, got %d", lineage.Depth)
	}
}
