package entities

import (
	"math"
	"testing"
)

func TestClamp(t *testing.T) {
	cases := []struct {
		val, min, max, expected float64
	}{
		{0.5, 0, 1, 0.5},
		{-1.0, 0, 1, 0.0},
		{2.0, 0, 1, 1.0},
		{0.0, 0, 1, 0.0},
		{1.0, 0, 1, 1.0},
	}
	for _, tc := range cases {
		result := clamp(tc.val, tc.min, tc.max)
		if result != tc.expected {
			t.Errorf("clamp(%f, %f, %f) = %f, want %f", tc.val, tc.min, tc.max, result, tc.expected)
		}
	}
}

func TestAbs(t *testing.T) {
	if abs(-3.5) != 3.5 {
		t.Error("abs(-3.5) should be 3.5")
	}
	if abs(3.5) != 3.5 {
		t.Error("abs(3.5) should be 3.5")
	}
	if abs(0) != 0 {
		t.Error("abs(0) should be 0")
	}
}

func TestMax(t *testing.T) {
	if max(3.0, 5.0) != 5.0 {
		t.Error("max(3,5) should be 5")
	}
	if max(5.0, 3.0) != 5.0 {
		t.Error("max(5,3) should be 5")
	}
	if max(4.0, 4.0) != 4.0 {
		t.Error("max(4,4) should be 4")
	}
}

func TestContains(t *testing.T) {
	slice := []string{"a", "b", "c"}
	if !contains(slice, "b") {
		t.Error("contains(slice, 'b') should be true")
	}
	if contains(slice, "d") {
		t.Error("contains(slice, 'd') should be false")
	}
}

func TestCosineSimilarity_Identical(t *testing.T) {
	v := []float64{1, 0, 0, 1}
	sim := cosineSimilarity(v, v)
	if math.Abs(sim-1.0) > 1e-9 {
		t.Errorf("cosine similarity of identical vectors should be 1.0, got %f", sim)
	}
}

func TestCosineSimilarity_Orthogonal(t *testing.T) {
	a := []float64{1, 0}
	b := []float64{0, 1}
	sim := cosineSimilarity(a, b)
	if math.Abs(sim) > 1e-9 {
		t.Errorf("cosine similarity of orthogonal vectors should be 0.0, got %f", sim)
	}
}

func TestCosineSimilarity_ZeroVector(t *testing.T) {
	a := []float64{0, 0, 0}
	b := []float64{1, 2, 3}
	sim := cosineSimilarity(a, b)
	if sim != 0.0 {
		t.Errorf("cosine similarity with zero vector should be 0.0, got %f", sim)
	}
}

func TestCosineSimilarity_DifferentLengths(t *testing.T) {
	a := []float64{1, 2}
	b := []float64{1, 2, 3}
	sim := cosineSimilarity(a, b)
	if sim != 0.0 {
		t.Errorf("cosine similarity of different-length vectors should be 0.0, got %f", sim)
	}
}

func TestIdentityDimensionVector_ToSlice(t *testing.T) {
	v := &IdentityDimensionVector{
		Openness: 0.5, Conscientiousness: 0.6, Extraversion: 0.4,
		Agreeableness: 0.7, EmotionalStability: 0.3,
		VoiceFormality: 0.8, VoiceHumor: 0.2, VoiceEmpathy: 0.9,
		TechnicalDepth: 0.5, Directness: 0.6, Helpfulness: 0.7,
		Curiosity: 0.4, Creativity: 0.3,
	}

	s := v.ToSlice()
	if len(s) != 13 {
		t.Errorf("ToSlice() length: got %d, want 13", len(s))
	}
	if s[0] != 0.5 {
		t.Errorf("ToSlice()[0] (Openness): got %f, want 0.5", s[0])
	}
}

func TestIdentityDimensionVector_SimilarityTo(t *testing.T) {
	v := &IdentityDimensionVector{
		Openness: 0.5, Conscientiousness: 0.5, Extraversion: 0.5,
		Agreeableness: 0.5, EmotionalStability: 0.5,
		VoiceFormality: 0.5, VoiceHumor: 0.5, VoiceEmpathy: 0.5,
		TechnicalDepth: 0.5, Directness: 0.5, Helpfulness: 0.5,
		Curiosity: 0.5, Creativity: 0.5,
	}

	sim := v.SimilarityTo(v)
	if math.Abs(sim-1.0) > 1e-9 {
		t.Errorf("SimilarityTo itself should be 1.0, got %f", sim)
	}
}

func TestFromIdentitySnapshot(t *testing.T) {
	snap := NewIdentitySnapshot("agent-1", "gpt-4")
	snap.VoiceProfile = *NewVoiceProfile()
	snap.ValueSystem = *NewValueSystem()
	snap.BehavioralSignature = *NewBehavioralSignature()

	trait := NewPersonalityTrait("analytical", TraitCognitive, 0.8)
	snap.WithTraits(*trait)

	v := FromIdentitySnapshot(snap)
	if v == nil {
		t.Fatal("FromIdentitySnapshot should not return nil")
	}
	// Openness should be set from cognitive trait
	if v.Openness != 0.8 {
		t.Errorf("Openness should be 0.8 (from cognitive trait intensity), got %f", v.Openness)
	}
}
