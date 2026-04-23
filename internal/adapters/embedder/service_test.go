package embedder

import (
	"context"
	"math"
	"testing"

	"github.com/benoitpetit/soul/internal/domain/entities"
)

// --- Helpers ---

func makeSnapshot(agentID string) *entities.IdentitySnapshot {
	snap := entities.NewIdentitySnapshot(agentID, "gpt-4")
	snap.VoiceProfile = *entities.NewVoiceProfile()
	snap.VoiceProfile.FormalityLevel = 0.7
	snap.VoiceProfile.EmpathyLevel = 0.6
	snap.VoiceProfile.TechnicalDepth = 0.8
	snap.ValueSystem = *entities.NewValueSystem()
	snap.BehavioralSignature = *entities.NewBehavioralSignature()
	return snap
}

// --- Constructor ---

func TestNewSoulEmbedderService(t *testing.T) {
	e := NewSoulEmbedderService(nil)
	if e == nil {
		t.Fatal("NewSoulEmbedderService should not return nil")
	}
}

// --- ModelHash / Dimension ---

func TestModelHash(t *testing.T) {
	e := NewSoulEmbedderService(nil)
	if e.ModelHash() != ModelHash {
		t.Errorf("ModelHash: got %q, want %q", e.ModelHash(), ModelHash)
	}
}

func TestDimension(t *testing.T) {
	e := NewSoulEmbedderService(nil)
	if e.Dimension() != Dimension {
		t.Errorf("Dimension: got %d, want %d", e.Dimension(), Dimension)
	}
}

// --- EncodeIdentity ---

func TestEncodeIdentity_NilSnapshot(t *testing.T) {
	e := NewSoulEmbedderService(nil)
	_, err := e.EncodeIdentity(context.Background(), nil)
	if err == nil {
		t.Error("EncodeIdentity(nil) should return error")
	}
}

func TestEncodeIdentity_CorrectLength(t *testing.T) {
	e := NewSoulEmbedderService(nil)
	snap := makeSnapshot("agent-1")

	vec, err := e.EncodeIdentity(context.Background(), snap)
	if err != nil {
		t.Fatalf("EncodeIdentity error: %v", err)
	}
	if len(vec) != Dimension {
		t.Errorf("vector length: got %d, want %d", len(vec), Dimension)
	}
}

func TestEncodeIdentity_NormalisedOrZero(t *testing.T) {
	e := NewSoulEmbedderService(nil)
	snap := makeSnapshot("agent-1")

	vec, err := e.EncodeIdentity(context.Background(), snap)
	if err != nil {
		t.Fatalf("EncodeIdentity error: %v", err)
	}

	// Either the zero vector (all VoiceProfile/ValueSystem fields are 0) or unit-normalised.
	norm := float64(0)
	for _, v := range vec {
		norm += float64(v) * float64(v)
	}
	if norm > 0 {
		// Expect norm ≈ 1.0
		if math.Abs(math.Sqrt(norm)-1.0) > 0.01 {
			t.Errorf("normalised vector: L2 norm = %f, want ~1.0", math.Sqrt(norm))
		}
	}
}

func TestEncodeIdentity_Deterministic(t *testing.T) {
	e := NewSoulEmbedderService(nil)
	snap := makeSnapshot("agent-1")

	v1, _ := e.EncodeIdentity(context.Background(), snap)
	v2, _ := e.EncodeIdentity(context.Background(), snap)

	for i := range v1 {
		if v1[i] != v2[i] {
			t.Errorf("EncodeIdentity is not deterministic at index %d", i)
			break
		}
	}
}

func TestEncodeIdentity_DifferentSnapshots_DifferentVectors(t *testing.T) {
	e := NewSoulEmbedderService(nil)

	snapA := makeSnapshot("agent-a")
	snapA.VoiceProfile.FormalityLevel = 0.1
	snapA.VoiceProfile.TechnicalDepth = 0.1

	snapB := makeSnapshot("agent-b")
	snapB.VoiceProfile.FormalityLevel = 0.9
	snapB.VoiceProfile.TechnicalDepth = 0.9

	vA, _ := e.EncodeIdentity(context.Background(), snapA)
	vB, _ := e.EncodeIdentity(context.Background(), snapB)

	identical := true
	for i := range vA {
		if vA[i] != vB[i] {
			identical = false
			break
		}
	}
	if identical {
		t.Error("different snapshots should produce different vectors")
	}
}

// --- EncodeTrait ---

func TestEncodeTrait_NilTrait(t *testing.T) {
	e := NewSoulEmbedderService(nil)
	_, err := e.EncodeTrait(context.Background(), nil)
	if err == nil {
		t.Error("EncodeTrait(nil) should return error")
	}
}

func TestEncodeTrait_CorrectLength(t *testing.T) {
	e := NewSoulEmbedderService(nil)
	trait := entities.NewPersonalityTrait("analytical", entities.TraitCognitive, 0.8)

	vec, err := e.EncodeTrait(context.Background(), trait)
	if err != nil {
		t.Fatalf("EncodeTrait error: %v", err)
	}
	if len(vec) != Dimension {
		t.Errorf("vector length: got %d, want %d", len(vec), Dimension)
	}
}

func TestEncodeTrait_AllCategories(t *testing.T) {
	e := NewSoulEmbedderService(nil)
	ctx := context.Background()

	categories := []entities.TraitCategory{
		entities.TraitCognitive,
		entities.TraitEmotional,
		entities.TraitSocial,
		entities.TraitEpistemic,
		entities.TraitExpressive,
		entities.TraitEthical,
	}
	for _, cat := range categories {
		trait := entities.NewPersonalityTrait("test", cat, 0.7)
		vec, err := e.EncodeTrait(ctx, trait)
		if err != nil {
			t.Errorf("EncodeTrait(%s) error: %v", cat, err)
			continue
		}
		if len(vec) != Dimension {
			t.Errorf("EncodeTrait(%s) wrong dimension: %d", cat, len(vec))
		}
	}
}

func TestEncodeTrait_ZeroIntensity(t *testing.T) {
	e := NewSoulEmbedderService(nil)
	trait := entities.NewPersonalityTrait("analytical", entities.TraitCognitive, 0.0)

	vec, err := e.EncodeTrait(context.Background(), trait)
	if err != nil {
		t.Fatalf("EncodeTrait error: %v", err)
	}
	if len(vec) != Dimension {
		t.Errorf("vector length: got %d, want %d", len(vec), Dimension)
	}
}

// --- FindSimilarIdentities ---

func TestFindSimilarIdentities_NoStorage(t *testing.T) {
	e := NewSoulEmbedderService(nil)
	vec := make([]float32, Dimension)

	_, err := e.FindSimilarIdentities(context.Background(), vec, 5)
	if err == nil {
		t.Error("FindSimilarIdentities without storage should return error")
	}
}

func TestFindSimilarIdentities_WrongDimension(t *testing.T) {
	// Even with nil storage the dimension check fires first.
	e := NewSoulEmbedderService(nil)
	vec := make([]float32, 5) // wrong size

	_, err := e.FindSimilarIdentities(context.Background(), vec, 5)
	if err == nil {
		t.Error("FindSimilarIdentities with wrong dimension should return error")
	}
}

// --- Internal helpers ---

func TestNormalize_ZeroVector(t *testing.T) {
	v := []float64{0, 0, 0}
	result := normalize(v)
	for i, x := range result {
		if x != 0 {
			t.Errorf("normalize(zero)[%d] = %f, want 0", i, x)
		}
	}
}

func TestNormalize_UnitVector(t *testing.T) {
	v := []float64{3, 4} // norm = 5
	result := normalize(v)
	norm := 0.0
	for _, x := range result {
		norm += x * x
	}
	if math.Abs(math.Sqrt(norm)-1.0) > 1e-9 {
		t.Errorf("normalised vector norm = %f, want 1.0", math.Sqrt(norm))
	}
}

func TestCosineSimilarity_Identical(t *testing.T) {
	v := []float64{1, 0, 0}
	sim := cosineSimilarity(v, v)
	if math.Abs(sim-1.0) > 1e-9 {
		t.Errorf("cosineSimilarity identical: got %f, want 1.0", sim)
	}
}

func TestCosineSimilarity_Orthogonal(t *testing.T) {
	a := []float64{1, 0}
	b := []float64{0, 1}
	sim := cosineSimilarity(a, b)
	if math.Abs(sim) > 1e-9 {
		t.Errorf("cosineSimilarity orthogonal: got %f, want 0.0", sim)
	}
}

func TestCosineSimilarity_ZeroVector(t *testing.T) {
	a := []float64{0, 0}
	b := []float64{1, 0}
	sim := cosineSimilarity(a, b)
	if sim != 0.0 {
		t.Errorf("cosineSimilarity zero vector: got %f, want 0.0", sim)
	}
}

func TestToFloat32AndBack(t *testing.T) {
	original := []float64{0.1, 0.5, 0.9}
	f32 := toFloat32(original)
	f64 := toFloat64(f32)
	for i, v := range f64 {
		if math.Abs(v-original[i]) > 1e-6 {
			t.Errorf("round-trip [%d]: got %f, want %f", i, v, original[i])
		}
	}
}
