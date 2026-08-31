package testhelper

import (
	"testing"

	"github.com/benoitpetit/soul/internal/adapters/sqlite"
	"github.com/benoitpetit/soul/internal/domain/entities"
	_ "github.com/mutecomm/go-sqlcipher/v4"
)

func NewStorage(t *testing.T) *sqlite.SoulSQLiteStorage {
	t.Helper()
	s, err := sqlite.NewSoulSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func MakeFullSnapshot(agentID, modelID string) *entities.IdentitySnapshot {
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
