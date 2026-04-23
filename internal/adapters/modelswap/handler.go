// Package modelswap implémente la gestion des changements de modèle
// Le moment critique où l'âme de l'agent risque d'être perdue.
package modelswap

import (
	"context"
	"fmt"
	"time"

	"github.com/benoitpetit/soul/internal/domain/entities"
	"github.com/benoitpetit/soul/internal/domain/valueobjects"
)

// SoulModelSwapHandler implémente ports.ModelSwapHandler
// Gère la transition identitaire lors des changements de modèle LLM.
type SoulModelSwapHandler struct {
	maxAcceptableDrift float64 // Dérive maximale acceptable post-swap
}

// NewSoulModelSwapHandler crée un nouveau handler
func NewSoulModelSwapHandler() *SoulModelSwapHandler {
	return &SoulModelSwapHandler{
		maxAcceptableDrift: 0.3,
	}
}

// HandleModelSwap enregistre et gère le changement de modèle
func (h *SoulModelSwapHandler) HandleModelSwap(ctx context.Context, agentID, previousModel, newModel string) (*valueobjects.ModelSwapContext, error) {
	swap := &valueobjects.ModelSwapContext{
		PreviousModel:      previousModel,
		NewModel:           newModel,
		Timestamp:          time.Now(),
		IdentityPreserved:  false, // Sera mis à jour après mesure
		IdentityDrift:      0,
		ReinforcementApplied: false,
	}
	
	return swap, nil
}

// ReinforceIdentity renforce l'identité après un changement de modèle
// Crée un nouveau snapshot avec des marqueurs de renforcement
func (h *SoulModelSwapHandler) ReinforceIdentity(ctx context.Context, identity *entities.IdentitySnapshot) (*entities.IdentitySnapshot, error) {
	if identity == nil {
		return nil, fmt.Errorf("identity cannot be nil")
	}
	
	// Créer un nouveau snapshot post-swap
	reinforced := entities.NewIdentitySnapshot(identity.AgentID, identity.ModelIdentifier)
	reinforced.WithParentSnapshot(identity.ID)
	
	// Copier toutes les dimensions identitaires
	reinforced.PersonalityTraits = identity.PersonalityTraits
	reinforced.VoiceProfile = identity.VoiceProfile
	reinforced.CommunicationStyle = identity.CommunicationStyle
	reinforced.BehavioralSignature = identity.BehavioralSignature
	reinforced.ValueSystem = identity.ValueSystem
	reinforced.EmotionalTone = identity.EmotionalTone
	
	// Augmenter la confiance des traits critiques pour les "ancrer"
	for i := range reinforced.PersonalityTraits {
		if reinforced.PersonalityTraits[i].Confidence > 0.7 {
			// Renforcer les traits bien établis
			reinforced.PersonalityTraits[i].Confidence = min(reinforced.PersonalityTraits[i].Confidence*1.1, 1.0)
		}
	}
	
	// Recalculer la confiance globale
	reinforced.ConfidenceScore = identity.ConfidenceScore
	
	return reinforced, nil
}

// MeasurePostSwapDrift mesure la dérive après le changement de modèle
func (h *SoulModelSwapHandler) MeasurePostSwapDrift(ctx context.Context, swap *valueobjects.ModelSwapContext) (float64, error) {
	if swap == nil {
		return 0, fmt.Errorf("swap context cannot be nil")
	}
	
	// La dérive est stockée dans le contexte du swap
	return swap.IdentityDrift, nil
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// SoulMergerService implémente ports.SoulMerger
// Gère la fusion de deux identités.
type SoulMergerService struct{}

// NewSoulMergerService crée un nouveau service de fusion
func NewSoulMergerService() *SoulMergerService {
	return &SoulMergerService{}
}

// MergeIdentities fusionne deux identités selon une stratégie
func (m *SoulMergerService) MergeIdentities(ctx context.Context, identityA, identityB *entities.IdentitySnapshot, strategy valueobjects.MergeStrategy) (*entities.IdentitySnapshot, error) {
	if identityA == nil || identityB == nil {
		return nil, fmt.Errorf("both identities must be non-nil")
	}
	
	// Créer le snapshot fusionné
	merged := entities.NewIdentitySnapshot(identityA.AgentID, identityA.ModelIdentifier)
	
	switch strategy {
	case valueobjects.MergePreserveDominant:
		m.mergePreserveDominant(merged, identityA, identityB)
	case valueobjects.MergeWeightedAverage:
		m.mergeWeightedAverage(merged, identityA, identityB)
	case valueobjects.MergeLatestWins:
		m.mergeLatestWins(merged, identityA, identityB)
	case valueobjects.MergeSynthesize:
		m.mergeSynthesize(merged, identityA, identityB)
	default:
		m.mergeWeightedAverage(merged, identityA, identityB)
	}
	
	return merged, nil
}

// CalculateMergeCompatibility calcule la compatibilité entre deux identités
func (m *SoulMergerService) CalculateMergeCompatibility(ctx context.Context, identityA, identityB *entities.IdentitySnapshot) (float64, error) {
	if identityA == nil || identityB == nil {
		return 0, fmt.Errorf("both identities must be non-nil")
	}
	
	// Similarité du profil de voix
	voiceSim := 1.0 - identityA.VoiceProfile.DistanceTo(&identityB.VoiceProfile)
	
	// Similarité du ton émotionnel
	emotionalSim := 1.0 - identityA.EmotionalTone.DistanceTo(&identityB.EmotionalTone)
	
	// Similarité des valeurs
	valuesSim := m.calculateValuesSimilarity(&identityA.ValueSystem, &identityB.ValueSystem)
	
	// Similarité des traits
	traitsSim := m.calculateTraitsSimilarity(identityA.PersonalityTraits, identityB.PersonalityTraits)
	
	// Score composite (moyenne pondérée)
	compatibility := voiceSim*0.3 + emotionalSim*0.2 + valuesSim*0.3 + traitsSim*0.2
	
	return compatibility, nil
}

// --- Stratégies de fusion ---

func (m *SoulMergerService) mergePreserveDominant(merged, a, b *entities.IdentitySnapshot) {
	// Déterminer l'identité dominante (celle avec la plus haute confiance)
	dominant, secondary := a, b
	if b.ConfidenceScore > a.ConfidenceScore {
		dominant, secondary = b, a
	}
	
	merged.PersonalityTraits = dominant.PersonalityTraits
	merged.VoiceProfile = dominant.VoiceProfile
	merged.CommunicationStyle = dominant.CommunicationStyle
	merged.BehavioralSignature = dominant.BehavioralSignature
	merged.ValueSystem = dominant.ValueSystem
	merged.EmotionalTone = dominant.EmotionalTone
	merged.ConfidenceScore = dominant.ConfidenceScore
	
	// Ajouter les traits uniques de l'identité secondaire
	secondaryTraits := make(map[string]bool)
	for _, t := range dominant.PersonalityTraits {
		secondaryTraits[t.Name] = true
	}
	for _, t := range secondary.PersonalityTraits {
		if !secondaryTraits[t.Name] {
			merged.PersonalityTraits = append(merged.PersonalityTraits, t)
		}
	}
}

func (m *SoulMergerService) mergeWeightedAverage(merged, a, b *entities.IdentitySnapshot) {
	// Moyenne pondérée par la confiance
	totalConfidence := a.ConfidenceScore + b.ConfidenceScore
	if totalConfidence == 0 {
		totalConfidence = 1
	}
	weightA := a.ConfidenceScore / totalConfidence
	weightB := b.ConfidenceScore / totalConfidence
	
	// Fusionner les profils de voix
	merged.VoiceProfile = m.interpolateVoice(&a.VoiceProfile, &b.VoiceProfile, weightA, weightB)
	
	// Fusionner les tons émotionnels
	merged.EmotionalTone = m.interpolateEmotionalTone(&a.EmotionalTone, &b.EmotionalTone, weightA, weightB)
	
	// Fusionner les systèmes de valeurs
	merged.ValueSystem = m.interpolateValueSystem(&a.ValueSystem, &b.ValueSystem, weightA, weightB)
	
	// Fusionner les traits
	merged.PersonalityTraits = m.mergeTraits(a.PersonalityTraits, b.PersonalityTraits, weightA, weightB)
	
	// Prendre le style de communication du dominant
	if weightA > weightB {
		merged.CommunicationStyle = a.CommunicationStyle
		merged.BehavioralSignature = a.BehavioralSignature
	} else {
		merged.CommunicationStyle = b.CommunicationStyle
		merged.BehavioralSignature = b.BehavioralSignature
	}
	
	merged.ConfidenceScore = (a.ConfidenceScore + b.ConfidenceScore) / 2.0
}

func (m *SoulMergerService) mergeLatestWins(merged, a, b *entities.IdentitySnapshot) {
	// La plus récente gagne complètement
	if a.CreatedAt.After(b.CreatedAt) {
		*merged = *a
		merged.ID = [16]byte{} // Reset ID
	} else {
		*merged = *b
		merged.ID = [16]byte{} // Reset ID
	}
}

func (m *SoulMergerService) mergeSynthesize(merged, a, b *entities.IdentitySnapshot) {
	// Syntèse intelligente : prendre le meilleur des deux
	merged.VoiceProfile = m.selectBetterVoice(&a.VoiceProfile, &b.VoiceProfile)
	merged.EmotionalTone = m.selectBetterTone(&a.EmotionalTone, &b.EmotionalTone)
	merged.ValueSystem = m.selectBetterValues(&a.ValueSystem, &b.ValueSystem)
	merged.PersonalityTraits = m.selectBetterTraits(a.PersonalityTraits, b.PersonalityTraits)
	
	// Styles : prendre le plus récent
	if a.CreatedAt.After(b.CreatedAt) {
		merged.CommunicationStyle = a.CommunicationStyle
		merged.BehavioralSignature = a.BehavioralSignature
	} else {
		merged.CommunicationStyle = b.CommunicationStyle
		merged.BehavioralSignature = b.BehavioralSignature
	}
	
	merged.ConfidenceScore = maxFloat(a.ConfidenceScore, b.ConfidenceScore)
}

// --- Helpers d'interpolation ---

func (m *SoulMergerService) interpolateVoice(a, b *entities.VoiceProfile, weightA, weightB float64) entities.VoiceProfile {
	return entities.VoiceProfile{
		FormalityLevel:     a.FormalityLevel*weightA + b.FormalityLevel*weightB,
		HumorLevel:         a.HumorLevel*weightA + b.HumorLevel*weightB,
		EmpathyLevel:       a.EmpathyLevel*weightA + b.EmpathyLevel*weightB,
		TechnicalDepth:     a.TechnicalDepth*weightA + b.TechnicalDepth*weightB,
		EnthusiasmLevel:    a.EnthusiasmLevel*weightA + b.EnthusiasmLevel*weightB,
		DirectnessLevel:    a.DirectnessLevel*weightA + b.DirectnessLevel*weightB,
		SentenceStructure:  a.SentenceStructure, // Prendre celui de A
		VocabularyRichness: a.VocabularyRichness*weightA + b.VocabularyRichness*weightB,
		MetaphorUsage:      a.MetaphorUsage*weightA + b.MetaphorUsage*weightB,
		QuestionStyle:      a.QuestionStyle,
		AvgSentenceLength:  a.AvgSentenceLength,
		UsesEmojis:         a.UsesEmojis || b.UsesEmojis,
		UsesMarkdown:       a.UsesMarkdown || b.UsesMarkdown,
		ExplanationStyle:   a.ExplanationStyle,
	}
}

func (m *SoulMergerService) interpolateEmotionalTone(a, b *entities.EmotionalTone, weightA, weightB float64) entities.EmotionalTone {
	return entities.EmotionalTone{
		Warmth:               a.Warmth*weightA + b.Warmth*weightB,
		Calmness:             a.Calmness*weightA + b.Calmness*weightB,
		Enthusiasm:           a.Enthusiasm*weightA + b.Enthusiasm*weightB,
		Seriousness:          a.Seriousness*weightA + b.Seriousness*weightB,
		Playfulness:          a.Playfulness*weightA + b.Playfulness*weightB,
		EmotionalConsistency: a.EmotionalConsistency*weightA + b.EmotionalConsistency*weightB,
		Reactiveness:         a.Reactiveness*weightA + b.Reactiveness*weightB,
		Resilience:           a.Resilience*weightA + b.Resilience*weightB,
		EncouragementLevel:   a.EncouragementLevel*weightA + b.EncouragementLevel*weightB,
		ValidationLevel:      a.ValidationLevel*weightA + b.ValidationLevel*weightB,
		ChallengingLevel:     a.ChallengingLevel*weightA + b.ChallengingLevel*weightB,
	}
}

func (m *SoulMergerService) interpolateValueSystem(a, b *entities.ValueSystem, weightA, weightB float64) entities.ValueSystem {
	vs := entities.NewValueSystem()
	vs.PrioritizesAccuracy = a.PrioritizesAccuracy*weightA + b.PrioritizesAccuracy*weightB
	vs.PrioritizesHelpfulness = a.PrioritizesHelpfulness*weightA + b.PrioritizesHelpfulness*weightB
	vs.PrioritizesEfficiency = a.PrioritizesEfficiency*weightA + b.PrioritizesEfficiency*weightB
	vs.PrioritizesClarity = a.PrioritizesClarity*weightA + b.PrioritizesClarity*weightB
	vs.PrioritizesSafety = a.PrioritizesSafety*weightA + b.PrioritizesSafety*weightB
	vs.PrioritizesCreativity = a.PrioritizesCreativity*weightA + b.PrioritizesCreativity*weightB
	vs.RiskTolerance = a.RiskTolerance*weightA + b.RiskTolerance*weightB
	vs.LongTermVsShortTerm = a.LongTermVsShortTerm*weightA + b.LongTermVsShortTerm*weightB
	vs.IndividualVsCollective = a.IndividualVsCollective*weightA + b.IndividualVsCollective*weightB
	return *vs
}

func (m *SoulMergerService) mergeTraits(a, b []entities.PersonalityTrait, weightA, weightB float64) []entities.PersonalityTrait {
	merged := make([]entities.PersonalityTrait, 0)
	seen := make(map[string]bool)
	
	// Ajouter les traits de A
	for _, trait := range a {
		merged = append(merged, trait)
		seen[trait.Name] = true
	}
	
	// Fusionner ou ajouter les traits de B
	for _, traitB := range b {
		found := false
		for i, traitA := range merged {
			if traitA.Name == traitB.Name {
				// Fusionner
				merged[i].Intensity = traitA.Intensity*weightA + traitB.Intensity*weightB
				merged[i].Confidence = maxFloat(traitA.Confidence, traitB.Confidence)
				found = true
				break
			}
		}
		if !found && !seen[traitB.Name] {
			merged = append(merged, traitB)
		}
	}
	
	return merged
}

func (m *SoulMergerService) selectBetterVoice(a, b *entities.VoiceProfile) entities.VoiceProfile {
	// Sélectionner le plus équilibré (proche de 0.5 sur toutes les dimensions)
	balanceA := abs(a.FormalityLevel-0.5) + abs(a.HumorLevel-0.5) + abs(a.EmpathyLevel-0.5)
	balanceB := abs(b.FormalityLevel-0.5) + abs(b.HumorLevel-0.5) + abs(b.EmpathyLevel-0.5)
	if balanceA < balanceB {
		return *a
	}
	return *b
}

func (m *SoulMergerService) selectBetterTone(a, b *entities.EmotionalTone) entities.EmotionalTone {
	if a.Warmth+a.Enthusiasm > b.Warmth+b.Enthusiasm {
		return *a
	}
	return *b
}

func (m *SoulMergerService) selectBetterValues(a, b *entities.ValueSystem) entities.ValueSystem {
	if a.PrioritizesHelpfulness+a.PrioritizesAccuracy > b.PrioritizesHelpfulness+b.PrioritizesAccuracy {
		return *a
	}
	return *b
}

func (m *SoulMergerService) selectBetterTraits(a, b []entities.PersonalityTrait) []entities.PersonalityTrait {
	if len(a) >= len(b) {
		return a
	}
	return b
}

func (m *SoulMergerService) calculateValuesSimilarity(a, b *entities.ValueSystem) float64 {
	diff := abs(a.PrioritizesAccuracy - b.PrioritizesAccuracy)
	diff += abs(a.PrioritizesHelpfulness - b.PrioritizesHelpfulness)
	diff += abs(a.PrioritizesEfficiency - b.PrioritizesEfficiency)
	diff += abs(a.PrioritizesClarity - b.PrioritizesClarity)
	return 1.0 - min(diff/4.0, 1.0)
}

func (m *SoulMergerService) calculateTraitsSimilarity(a, b []entities.PersonalityTrait) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0.5
	}
	
	matches := 0
	for _, ta := range a {
		for _, tb := range b {
			if ta.Name == tb.Name {
				matches++
				break
			}
		}
	}
	
	return float64(matches) / float64(maxInt(len(a), len(b)))
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
