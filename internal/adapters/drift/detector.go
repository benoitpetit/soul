package drift

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/benoitpetit/soul/internal/domain/entities"
	"github.com/benoitpetit/soul/internal/domain/valueobjects"
	"github.com/benoitpetit/soul/internal/usecases/ports"
)

// SoulDriftDetector implémente ports.IdentityDriftDetector
// Utilise des algorithmes de comparaison pour détecter la dérive identitaire.
type SoulDriftDetector struct {
	threshold float64 // Seuil de détection
	storage   ports.IdentityRepository // Stockage optionnel pour le monitoring continu
	mu        sync.RWMutex
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

// WithStorage injecte un repository d'identité pour activer le monitoring continu réel.
// Sans stockage, MonitorContinuously fonctionne en mode stub.
func (d *SoulDriftDetector) WithStorage(storage ports.IdentityRepository) *SoulDriftDetector {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.storage = storage
	return d
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

	// Seuil adaptatif basé sur la distribution des scores de chaque dimension
	// (porté depuis MIRA : IQR / Elbow / MeanStddev)
	adaptiveThreshold := d.threshold
	if dimensionCount >= 2 {
		dimensionScores := make([]float64, 0, dimensionCount)
		for _, dim := range report.DriftDimensions {
			dimensionScores = append(dimensionScores, dim.Change)
		}
		adaptiveThreshold = computeAdaptiveThreshold(dimensionScores, "iqr")
	}
	report.IsSignificant = report.DriftScore > adaptiveThreshold
	
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

// MonitorContinuously surveille en continu la dérive identitaire d'un agent.
// Si un stockage est configuré (via WithStorage), la boucle compare périodiquement
// le dernier snapshot avec la version précédente et émet un rapport quand une dérive
// significative est détectée. Sinon, elle fonctionne en mode stub (logs uniquement).
func (d *SoulDriftDetector) MonitorContinuously(ctx context.Context, agentID string, threshold float64) (<-chan valueobjects.IdentityDriftReport, error) {
	if threshold <= 0 {
		threshold = d.threshold
	}

	d.mu.RLock()
	storage := d.storage
	d.mu.RUnlock()

	slog.Info("starting continuous drift monitoring",
		"agent_id", agentID,
		"threshold", threshold,
		"interval", "30s",
		"storage_enabled", storage != nil)

	reports := make(chan valueobjects.IdentityDriftReport, 10)

	go func() {
		defer close(reports)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		// Cache du dernier snapshot connu pour comparer
		var lastSnapshot *entities.IdentitySnapshot

		if storage != nil {
			// Récupération initiale
			snap, err := storage.GetLatestIdentity(ctx, agentID)
			if err == nil && snap != nil {
				lastSnapshot = snap
			}
		}

		for {
			select {
			case <-ctx.Done():
				slog.Info("stopping continuous drift monitoring", "agent_id", agentID)
				return
			case <-ticker.C:
				if storage == nil || lastSnapshot == nil {
					slog.Debug("drift monitoring tick (no storage or baseline)", "agent_id", agentID)
					continue
				}

				current, err := storage.GetLatestIdentity(ctx, agentID)
				if err != nil {
					slog.Warn("monitoring failed to fetch latest identity", "agent_id", agentID, "error", err)
					continue
				}
				if current == nil {
					continue
				}

				// Éviter de comparer un snapshot avec lui-même
				if current.ID == lastSnapshot.ID {
					continue
				}

				report, err := d.DetectDrift(ctx, lastSnapshot, current)
				if err != nil {
					slog.Warn("monitoring drift detection failed", "agent_id", agentID, "error", err)
					continue
				}

				// Mettre à jour la baseline même si la dérive n'est pas significative
				lastSnapshot = current

				if report.IsSignificant {
					slog.Info("significant drift detected in monitoring",
						"agent_id", agentID,
						"drift_score", report.DriftScore,
						"from_version", report.PreviousVersion,
						"to_version", report.CurrentVersion)
					select {
					case reports <- *report:
					case <-ctx.Done():
						return
					}
				}
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

// --- Adaptive Threshold Helpers (ported from MIRA) ---

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	index := p / 100.0 * float64(len(sorted)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))
	if lower == upper {
		return sorted[lower]
	}
	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

func adaptiveThresholdIQR(scores []float64) float64 {
	if len(scores) < 3 {
		return 0.3
	}
	sorted := make([]float64, len(scores))
	copy(sorted, scores)
	sort.Float64s(sorted)
	q1 := percentile(sorted, 25)
	q3 := percentile(sorted, 75)
	iqr := q3 - q1
	threshold := q1 - 1.5*iqr
	if threshold < 0.15 {
		threshold = 0.15
	}
	if threshold > 0.75 {
		threshold = 0.75
	}
	return threshold
}

func adaptiveThresholdElbow(scores []float64) float64 {
	if len(scores) < 3 {
		return 0.3
	}
	sorted := make([]float64, len(scores))
	copy(sorted, scores)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] > sorted[j] })
	derivatives := make([]float64, len(sorted)-1)
	for i := 0; i < len(sorted)-1; i++ {
		derivatives[i] = sorted[i] - sorted[i+1]
	}
	mean := 0.0
	for _, d := range derivatives {
		mean += d
	}
	mean /= float64(len(derivatives))
	variance := 0.0
	for _, d := range derivatives {
		variance += (d - mean) * (d - mean)
	}
	stddev := math.Sqrt(variance / float64(len(derivatives)))
	cutoff := mean + stddev
	for i, d := range derivatives {
		if d > cutoff {
			if i+1 < len(sorted) {
				return sorted[i+1]
			}
			return sorted[i]
		}
	}
	return sorted[len(sorted)-1]
}

func adaptiveThresholdMeanStddev(scores []float64) float64 {
	if len(scores) == 0 {
		return 0.3
	}
	mean := 0.0
	for _, s := range scores {
		mean += s
	}
	mean /= float64(len(scores))
	variance := 0.0
	for _, s := range scores {
		variance += (s - mean) * (s - mean)
	}
	stddev := math.Sqrt(variance / float64(len(scores)))
	threshold := mean - stddev
	if threshold < 0.15 {
		threshold = 0.15
	}
	if threshold > 0.75 {
		threshold = 0.75
	}
	return threshold
}

func computeAdaptiveThreshold(scores []float64, method string) float64 {
	switch method {
	case "iqr":
		return adaptiveThresholdIQR(scores)
	case "elbow":
		return adaptiveThresholdElbow(scores)
	case "mean_stddev":
		return adaptiveThresholdMeanStddev(scores)
	default:
		return adaptiveThresholdIQR(scores)
	}
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
