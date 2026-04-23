// Drift Detection - Détection de dérive identitaire
package interactors

import (
	"context"
	"fmt"

	"github.com/benoitpetit/soul/internal/domain/entities"
	"github.com/benoitpetit/soul/internal/domain/valueobjects"
	"github.com/benoitpetit/soul/internal/usecases/ports"
)

// DriftDetectionUseCase implémente la détection de dérive identitaire
// La dérive identitaire est le phénomène où l'agent "oublie qui il est"
// au fil des conversations ou après des changements de modèle.
type DriftDetectionUseCase struct {
	storage  ports.SoulStorage
	detector ports.IdentityDriftDetector
	composer ports.IdentityComposer
}

// NewDriftDetectionUseCase crée un nouveau use case de détection
func NewDriftDetectionUseCase(storage ports.SoulStorage, detector ports.IdentityDriftDetector, composer ports.IdentityComposer) *DriftDetectionUseCase {
	return &DriftDetectionUseCase{
		storage:  storage,
		detector: detector,
		composer: composer,
	}
}

// CheckDrift vérifie la dérive entre l'identité stockée et l'identité actuelle observée
func (uc *DriftDetectionUseCase) CheckDrift(ctx context.Context, agentID string, currentObserved *entities.IdentitySnapshot) (*valueobjects.IdentityDriftReport, error) {
	// 1. Récupérer l'identité de référence (dernier snapshot validé)
	reference, err := uc.storage.GetLatestIdentity(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve reference identity: %w", err)
	}
	if reference == nil {
		return nil, fmt.Errorf("no reference identity found for agent %s", agentID)
	}
	
	// 2. Détecter la dérive
	report, err := uc.detector.DetectDrift(ctx, reference, currentObserved)
	if err != nil {
		return nil, fmt.Errorf("drift detection failed: %w", err)
	}
	
	// 3. Enregistrer le diff
	if report != nil {
		diff := entities.CalculateDiff(reference, currentObserved)
		if diff != nil {
			if err := uc.storage.RecordDiff(ctx, diff); err != nil {
				fmt.Printf("Warning: failed to record diff: %v\n", err)
			}
		}
		
		// 4. Si dérive significative, générer une alerte
		if report.IsSignificant {
			alert, err := uc.composer.ComposeDiffusionAlert(ctx, report)
			if err == nil && alert != nil {
				fmt.Printf("ALERT: Significant identity drift detected for agent %s (score: %.2f)\n", 
					agentID, report.DriftScore)
				fmt.Printf("Alert prompt (%d tokens): %s\n", alert.TokenEstimate, alert.Content)
			}
		}
	}
	
	return report, nil
}

// CheckDiffusion vérifie si l'identité s'est diffusée (perdue)
func (uc *DriftDetectionUseCase) CheckDiffusion(ctx context.Context, agentID string) (bool, float64, error) {
	identity, err := uc.storage.GetLatestIdentity(ctx, agentID)
	if err != nil {
		return false, 0, err
	}
	if identity == nil {
		return false, 0, fmt.Errorf("no identity found")
	}
	
	return uc.detector.DetectDiffusion(ctx, identity)
}

// GetDriftReport génère un rapport de dérive complet
func (uc *DriftDetectionUseCase) GetDriftReport(ctx context.Context, agentID string, windowSize int) (*valueobjects.IdentityDriftReport, error) {
	return uc.storage.GetDriftReport(ctx, agentID, windowSize)
}

// MonitorAgent démarre la surveillance continue d'un agent
func (uc *DriftDetectionUseCase) MonitorAgent(ctx context.Context, agentID string, threshold float64) (<-chan valueobjects.IdentityDriftReport, error) {
	return uc.detector.MonitorContinuously(ctx, agentID, threshold)
}

// GetDiffHistory retourne l'historique des diffs
func (uc *DriftDetectionUseCase) GetDiffHistory(ctx context.Context, agentID string, limit int) ([]*entities.IdentityDiff, error) {
	return uc.storage.GetDiffsForAgent(ctx, agentID, limit)
}

// RestoreIdentity restaure l'identité depuis une version précédente
// Utilisé quand l'identité actuelle a trop dérivé.
func (uc *DriftDetectionUseCase) RestoreIdentity(ctx context.Context, agentID string, targetVersion int) (*entities.IdentitySnapshot, error) {
	// 1. Récupérer la version cible
	target, err := uc.storage.GetIdentityAtVersion(ctx, agentID, targetVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve target version: %w", err)
	}
	
	// 2. Créer une nouvelle version basée sur la cible
	restored := entities.NewIdentitySnapshot(target.AgentID, target.ModelIdentifier)
	restored.WithParentSnapshot(target.ID)
	restored.PersonalityTraits = target.PersonalityTraits
	restored.VoiceProfile = target.VoiceProfile
	restored.CommunicationStyle = target.CommunicationStyle
	restored.BehavioralSignature = target.BehavioralSignature
	restored.ValueSystem = target.ValueSystem
	restored.EmotionalTone = target.EmotionalTone
	
	// 3. Sauvegarder
	if err := uc.storage.StoreIdentity(ctx, restored); err != nil {
		return nil, fmt.Errorf("failed to store restored identity: %w", err)
	}
	
	return restored, nil
}
