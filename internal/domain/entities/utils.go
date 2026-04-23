// Utilitaires mathématiques et helpers pour le domaine SOUL
package entities

import (
	"math"
	"time"
)

// clamp restreint une valeur entre min et max
func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// abs retourne la valeur absolue
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// max retourne le maximum de deux float64
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// maxTime retourne le temps le plus récent
func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

// min retourne le minimum de deux ints
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// contains vérifie si une chaîne est dans un slice
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// exp calcule e^x (approximation pour éviter d'importer math en entier)
func exp(x float64) float64 {
	return math.Exp(x)
}

// sigmoid fonction sigmoïde pour les scores
func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

// cosineSimilarity calcule la similarité cosinus entre deux vecteurs
func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0.0
	}
	
	dotProduct := 0.0
	normA := 0.0
	normB := 0.0
	
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	
	if normA == 0 || normB == 0 {
		return 0.0
	}
	
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// IdentityDimensionVector représente l'identité comme un vecteur multidimensionnel
// pour permettre les comparaisons et recherches vectorielles (intégration avec MIRA HNSW)
type IdentityDimensionVector struct {
	// Dimensions de personnalité (Big Five inspired)
	Openness          float64 `json:"openness"`           // Ouverture
	Conscientiousness float64 `json:"conscientiousness"`  // Conscienciosité
	Extraversion      float64 `json:"extraversion"`       // Extraversion
	Agreeableness     float64 `json:"agreeableness"`      // Agréabilité
	EmotionalStability float64 `json:"emotional_stability"`// Stabilité émotionnelle
	
	// Dimensions SOUL spécifiques
	VoiceFormality    float64 `json:"voice_formality"`    // Formalité
	VoiceHumor        float64 `json:"voice_humor"`        // Humour
	VoiceEmpathy      float64 `json:"voice_empathy"`      // Empathie
	TechnicalDepth    float64 `json:"technical_depth"`    // Profondeur technique
	Directness        float64 `json:"directness"`         // Directivité
	Helpfulness       float64 `json:"helpfulness"`        // Altruisme
	Curiosity         float64 `json:"curiosity"`          // Curiosité
	Creativity        float64 `json:"creativity"`         // Créativité
}

// ToSlice convertit le vecteur en slice pour les opérations vectorielles
func (idv *IdentityDimensionVector) ToSlice() []float64 {
	return []float64{
		idv.Openness,
		idv.Conscientiousness,
		idv.Extraversion,
		idv.Agreeableness,
		idv.EmotionalStability,
		idv.VoiceFormality,
		idv.VoiceHumor,
		idv.VoiceEmpathy,
		idv.TechnicalDepth,
		idv.Directness,
		idv.Helpfulness,
		idv.Curiosity,
		idv.Creativity,
	}
}

// FromIdentitySnapshot extrait le vecteur dimensionnel d'un snapshot
func FromIdentitySnapshot(snapshot *IdentitySnapshot) *IdentityDimensionVector {
	// Calculer les Big Five approximatifs à partir des traits
	openness := 0.5
	conscientiousness := 0.5
	extraversion := 0.5
	agreeableness := 0.5
	emotionalStability := 0.5
	
	for _, trait := range snapshot.PersonalityTraits {
		switch trait.Category {
		case TraitCognitive:
			openness = trait.Intensity
		case TraitEmotional:
			emotionalStability = trait.Intensity
			agreeableness = trait.Intensity
		case TraitSocial:
			extraversion = trait.Intensity
			agreeableness = trait.Intensity
		case TraitEpistemic:
			openness = trait.Intensity
			conscientiousness = trait.Intensity
		case TraitExpressive:
			extraversion = trait.Intensity
			openness = trait.Intensity
		case TraitEthical:
			conscientiousness = trait.Intensity
			agreeableness = trait.Intensity
		}
	}
	
	return &IdentityDimensionVector{
		Openness:           openness,
		Conscientiousness:  conscientiousness,
		Extraversion:       extraversion,
		Agreeableness:      agreeableness,
		EmotionalStability: emotionalStability,
		VoiceFormality:     snapshot.VoiceProfile.FormalityLevel,
		VoiceHumor:         snapshot.VoiceProfile.HumorLevel,
		VoiceEmpathy:       snapshot.VoiceProfile.EmpathyLevel,
		TechnicalDepth:     snapshot.VoiceProfile.TechnicalDepth,
		Directness:         snapshot.VoiceProfile.DirectnessLevel,
		Helpfulness:        snapshot.ValueSystem.PrioritizesHelpfulness,
		Curiosity:          snapshot.BehavioralSignature.CuriosityLevel,
		Creativity:         snapshot.ValueSystem.PrioritizesCreativity,
	}
}

// SimilarityTo calcule la similarité cosinus entre deux identités
func (idv *IdentityDimensionVector) SimilarityTo(other *IdentityDimensionVector) float64 {
	return cosineSimilarity(idv.ToSlice(), other.ToSlice())
}
