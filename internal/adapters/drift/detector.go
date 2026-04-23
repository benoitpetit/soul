// Package drift implémente la détection de dérive identitaire
// Surveille si l'agent "oublie qui il est" au fil du temps.
package drift

import (
	"context"
	"fmt"
	"time"

	"github.com/benoitpetit/soul/internal/domain/entities"
	"github.com/benoitpetit/soul/internal/domain/valueobjects"
)

// SoulDriftDetector implémente ports.IdentityDriftDetector
// Utilise des algorithmes de comparaison pour détecter la dérive identitaire.
type SoulDriftDetector struct {
	threshold float64 // Seuil de détection
}

// NewSoulDriftDetector crée un nouveau détecteur de dérive
func NewSoulDriftDetector(threshold float64) *SoulDriftDetector {
	if threshold <= 0 || threshold > 1 {
		threshold = 0.3 // Valeur par défaut
	}
	return &SoulDriftDetector{
		threshold: threshold,
	}
}

// DetectDrift compare deux snapshots et détecte la dérive
func (d *SoulDriftDetector) DetectDrift(ctx context.Context, previous, current *entities.IdentitySnapshot) (*valueobjects.IdentityDriftReport, error) {
	if previous == nil || current == nil {
		return nil, fmt.Errorf("both snapshots must be non-nil")
	}
	
	report := &valueobjects.IdentityDriftReport{
		Timestamp:       time.Now(),
		PreviousVersion: previous.Version,
		CurrentVersion:  current.Version,
		DriftDimensions: make([]valueobjects.DimensionDrift, 0),
		Recommendations: make([]string, 0),
	}
	
	totalDrift := 0.0
	dimensionCount := 0
	
	// 1. Comparer le profil de voix
	if previous.VoiceProfile.FormalityLevel > 0 || current.VoiceProfile.FormalityLevel > 0 {
		voiceDistance := previous.VoiceProfile.DistanceTo(&current.VoiceProfile)
		drift := valueobjects.DimensionDrift{
			Dimension:     "voice_profile",
			PreviousValue: 0.5, // Valeur composite
			CurrentValue:  0.5,
			Change:        voiceDistance,
			IsSignificant: voiceDistance > d.threshold,
		}
		report.DriftDimensions = append(report.DriftDimensions, drift)
		totalDrift += voiceDistance
		dimensionCount++
		
		if drift.IsSignificant {
			report.Recommendations = append(report.Recommendations,
				"Voice profile has drifted. Consider voice reinforcement.")
		}
	}
	
	// 2. Comparer les traits de personnalité
	traitDrift := d.compareTraits(previous.PersonalityTraits, current.PersonalityTraits)
	if traitDrift > 0 {
		drift := valueobjects.DimensionDrift{
			Dimension:     "personality_traits",
			PreviousValue: 1.0,
			CurrentValue:  1.0 - traitDrift,
			Change:        traitDrift,
			IsSignificant: traitDrift > d.threshold,
		}
		report.DriftDimensions = append(report.DriftDimensions, drift)
		totalDrift += traitDrift
		dimensionCount++
		
		if drift.IsSignificant {
			report.Recommendations = append(report.Recommendations,
				"Personality traits have drifted. Review trait consistency.")
		}
	}
	
	// 3. Comparer le système de valeurs
	valuesDrift := d.compareValueSystems(&previous.ValueSystem, &current.ValueSystem)
	if valuesDrift > 0 {
		drift := valueobjects.DimensionDrift{
			Dimension:     "value_system",
			PreviousValue: 1.0,
			CurrentValue:  1.0 - valuesDrift,
			Change:        valuesDrift,
			IsSignificant: valuesDrift > d.threshold,
		}
		report.DriftDimensions = append(report.DriftDimensions, drift)
		totalDrift += valuesDrift
		dimensionCount++
		
		if drift.IsSignificant {
			report.Recommendations = append(report.Recommendations,
				"Value system has drifted. Consider value reinforcement.")
		}
	}
	
	// 4. Comparer le ton émotionnel
	if previous.EmotionalTone.Warmth > 0 || current.EmotionalTone.Warmth > 0 {
		emotionalDistance := previous.EmotionalTone.DistanceTo(&current.EmotionalTone)
		drift := valueobjects.DimensionDrift{
			Dimension:     "emotional_tone",
			PreviousValue: 1.0,
			CurrentValue:  1.0 - emotionalDistance,
			Change:        emotionalDistance,
			IsSignificant: emotionalDistance > d.threshold,
		}
		report.DriftDimensions = append(report.DriftDimensions, drift)
		totalDrift += emotionalDistance
		dimensionCount++
		
		if drift.IsSignificant {
			report.Recommendations = append(report.Recommendations,
				"Emotional tone has drifted. Consider tone reinforcement.")
		}
	}
	
	// Calculer le drift global
	if dimensionCount > 0 {
		report.DriftScore = totalDrift / float64(dimensionCount)
	}
	
	report.IsSignificant = report.DriftScore > d.threshold
	
	if report.IsSignificant {
		report.Recommendations = append(report.Recommendations,
			fmt.Sprintf("Overall drift score: %.2f. Identity reinforcement recommended.", report.DriftScore))
	}
	
	return report, nil
}

// DetectDiffusion détecte si l'identité s'est "diffusée" (perdue)
func (d *SoulDriftDetector) DetectDiffusion(ctx context.Context, identity *entities.IdentitySnapshot) (bool, float64, error) {
	// L'identité est "diffusée" si :
	// 1. Peu de traits bien établis
	// 2. Score de confiance global bas
	// 3. Beaucoup de traits récents avec faible confiance
	
	wellEstablishedCount := 0
	lowConfidenceCount := 0
	
	for _, trait := range identity.PersonalityTraits {
		if trait.IsWellEstablished() {
			wellEstablishedCount++
		}
		if trait.Confidence < 0.4 {
			lowConfidenceCount++
		}
	}
	
	totalTraits := len(identity.PersonalityTraits)
	if totalTraits == 0 {
		return true, 1.0, nil // Aucun trait = identité complètement diffusée
	}
	
	// Ratio de traits bien établis
	establishedRatio := float64(wellEstablishedCount) / float64(totalTraits)
	
	// Ratio de traits faibles
	weakRatio := float64(lowConfidenceCount) / float64(totalTraits)
	
	// Score de diffusion
	diffusionScore := (1.0 - establishedRatio)*0.6 + weakRatio*0.4
	
	// Diffusion détectée si :
	// - Moins de 30% de traits bien établis
	// OU
	// - Score de diffusion > 0.6
	isDiffused := establishedRatio < 0.3 || diffusionScore > 0.6
	
	return isDiffused, diffusionScore, nil
}

// MonitorContinuously surveille en continu la dérive identitaire.
// Lance une goroutine qui tourne jusqu'à ce que le contexte soit annulé.
// Les rapports de dérive sont envoyés sur le channel retourné.
//
// NOTE: This is a best-effort monitoring stub. For production use, integrate
// with the interactor layer to fetch snapshots periodically and call DetectDrift.
// Currently logs periodic ticks but does not perform actual drift detection.
func (d *SoulDriftDetector) MonitorContinuously(ctx context.Context, agentID string, threshold float64) (<-chan valueobjects.IdentityDriftReport, error) {
	if threshold <= 0 {
		threshold = d.threshold
	}

	reports := make(chan valueobjects.IdentityDriftReport, 10)

	go func() {
		defer close(reports)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// TODO: Integrate with storage to fetch snapshots and perform detection
				// For now, this is a stub that fires on each tick without sending reports
				// until the full integration is implemented.
				//
				// Full implementation would:
				// 1. Fetch latest identity snapshot from storage
				// 2. Compare with previous snapshot
				// 3. Send report on channel if drift detected
			}
		}
	}()

	return reports, nil
}

// CalculateIdentityVector calcule le vecteur dimensionnel de l'identité
func (d *SoulDriftDetector) CalculateIdentityVector(ctx context.Context, identity *entities.IdentitySnapshot) (*entities.IdentityDimensionVector, error) {
	vector := entities.FromIdentitySnapshot(identity)
	return vector, nil
}

// --- Helpers de comparaison ---

func (d *SoulDriftDetector) compareTraits(previous, current []entities.PersonalityTrait) float64 {
	if len(previous) == 0 && len(current) == 0 {
		return 0
	}
	if len(previous) == 0 || len(current) == 0 {
		return 1.0 // Tous les traits ont disparu ou apparu
	}
	
	// Créer un map des traits précédents
	prevMap := make(map[string]entities.PersonalityTrait)
	for _, trait := range previous {
		prevMap[trait.Name] = trait
	}
	
	// Calculer la dérive
	matchingTraits := 0
	totalDistance := 0.0
	
	for _, currTrait := range current {
		if prevTrait, exists := prevMap[currTrait.Name]; exists {
			matchingTraits++
			// Distance = différence d'intensité + différence de confiance
			intensityDiff := abs(currTrait.Intensity - prevTrait.Intensity)
			confidenceDiff := abs(currTrait.Confidence - prevTrait.Confidence)
			totalDistance += (intensityDiff + confidenceDiff) / 2.0
		}
	}
	
	// Traits disparus
	missingTraits := len(previous) - matchingTraits
	missingPenalty := float64(missingTraits) / float64(len(previous))
	
	// Traits nouveaux
	newTraits := len(current) - matchingTraits
	newPenalty := float64(newTraits) / float64(len(current))
	
	// Moyenne des distances pour les traits correspondants
	avgDistance := 0.0
	if matchingTraits > 0 {
		avgDistance = totalDistance / float64(matchingTraits)
	}
	
	// Score composite
	drift := avgDistance*0.4 + missingPenalty*0.3 + newPenalty*0.3
	
	return min(drift, 1.0)
}

func (d *SoulDriftDetector) compareValueSystems(previous, current *entities.ValueSystem) float64 {
	diff := 0.0
	diff += abs(previous.PrioritizesAccuracy - current.PrioritizesAccuracy)
	diff += abs(previous.PrioritizesHelpfulness - current.PrioritizesHelpfulness)
	diff += abs(previous.PrioritizesEfficiency - current.PrioritizesEfficiency)
	diff += abs(previous.PrioritizesClarity - current.PrioritizesClarity)
	diff += abs(previous.PrioritizesSafety - current.PrioritizesSafety)
	diff += abs(previous.PrioritizesCreativity - current.PrioritizesCreativity)
	
	return min(diff/6.0, 1.0)
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
