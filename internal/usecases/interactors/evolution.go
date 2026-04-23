// Evolution Tracker - Suivi de l'évolution identitaire
package interactors

import (
	"context"
	"fmt"

	"github.com/benoitpetit/soul/internal/domain/entities"
	"github.com/benoitpetit/soul/internal/usecases/ports"
	"github.com/google/uuid"
)

// IdentityEvolutionUseCase implémente le suivi de l'évolution identitaire
// Permet de comprendre comment l'agent "grandit" et change au fil du temps.
type IdentityEvolutionUseCase struct {
	storage  ports.SoulStorage
	tracker  ports.IdentityEvolutionTracker
}

// NewIdentityEvolutionUseCase crée un nouveau use case
func NewIdentityEvolutionUseCase(storage ports.SoulStorage, tracker ports.IdentityEvolutionTracker) *IdentityEvolutionUseCase {
	return &IdentityEvolutionUseCase{
		storage: storage,
		tracker: tracker,
	}
}

// TrackSnapshot enregistre un nouveau snapshot et calcule l'évolution
func (uc *IdentityEvolutionUseCase) TrackSnapshot(ctx context.Context, newSnapshot *entities.IdentitySnapshot) (*entities.IdentityDiff, error) {
	// 1. Récupérer le snapshot précédent
	var previous *entities.IdentitySnapshot
	if newSnapshot.DerivedFromID != nil {
		prev, err := uc.storage.GetIdentityByID(ctx, *newSnapshot.DerivedFromID)
		if err == nil {
			previous = prev
		}
	}
	
	if previous == nil {
		// Premier snapshot, pas de diff
		return nil, nil
	}
	
	// 2. Tracker l'évolution
	if uc.tracker == nil {
		// No tracker wired: compute diff inline and persist it
		d := entities.CalculateDiff(previous, newSnapshot)
		d.AgentID = newSnapshot.AgentID
		if err := uc.storage.RecordDiff(ctx, d); err != nil {
			fmt.Printf("Warning: failed to record diff: %v\n", err)
		}
		return d, nil
	}
	diff, err := uc.tracker.TrackEvolution(ctx, previous, newSnapshot)
	if err != nil {
		return nil, fmt.Errorf("evolution tracking failed: %w", err)
	}
	
	// 3. Enregistrer le diff
	if diff != nil {
		if err := uc.storage.RecordDiff(ctx, diff); err != nil {
			fmt.Printf("Warning: failed to record diff: %v\n", err)
		}
	}
	
	return diff, nil
}

// GetEvolutionTimeline retourne la timeline complète d'évolution
func (uc *IdentityEvolutionUseCase) GetEvolutionTimeline(ctx context.Context, agentID string) ([]*entities.IdentityDiff, error) {
	if uc.tracker == nil {
		return uc.storage.GetDiffsForAgent(ctx, agentID, 100)
	}
	return uc.tracker.GetEvolutionTimeline(ctx, agentID)
}

// GetEvolutionSummary retourne un résumé lisible de l'évolution
func (uc *IdentityEvolutionUseCase) GetEvolutionSummary(ctx context.Context, agentID string) (string, error) {
	timeline, err := uc.GetEvolutionTimeline(ctx, agentID)
	if err != nil {
		return "", err
	}
	
	if len(timeline) == 0 {
		return "No evolution recorded yet.", nil
	}
	
	summary := fmt.Sprintf("## Identity Evolution for Agent %s\n\n", agentID)
	summary += fmt.Sprintf("Total changes tracked: %d\n\n", len(timeline))
	
	// Compter les changements par type
	added := 0
	removed := 0
	strengthened := 0
	weakened := 0
	
	for _, diff := range timeline {
		added += len(diff.AddedTraits)
		removed += len(diff.RemovedTraits)
		strengthened += len(diff.StrengthenedTraits)
		weakened += len(diff.WeakenedTraits)
	}
	
	summary += fmt.Sprintf("- Traits added: %d\n", added)
	summary += fmt.Sprintf("- Traits removed: %d\n", removed)
	summary += fmt.Sprintf("- Traits strengthened: %d\n", strengthened)
	summary += fmt.Sprintf("- Traits weakened: %d\n\n", weakened)
	
	// Derniers changements
	latest := timeline[len(timeline)-1]
	summary += fmt.Sprintf("### Latest Changes (v%d → v%d)\n", latest.FromVersion, latest.ToVersion)
	
	if len(latest.AddedTraits) > 0 {
		summary += "\n**New traits:**\n"
		for _, t := range latest.AddedTraits {
			summary += fmt.Sprintf("- %s (%.0f%% confidence)\n", t.Name, t.Confidence*100)
		}
	}
	
	if len(latest.StrengthenedTraits) > 0 {
		summary += "\n**Strengthened traits:**\n"
		for _, t := range latest.StrengthenedTraits {
			summary += fmt.Sprintf("- %s (now %.0f%% confidence)\n", t.Name, t.Confidence*100)
		}
	}
	
	if len(latest.WeakenedTraits) > 0 {
		summary += "\n**Weakened traits:**\n"
		for _, t := range latest.WeakenedTraits {
			summary += fmt.Sprintf("- %s (now %.0f%% confidence)\n", t.Name, t.Confidence*100)
		}
	}
	
	return summary, nil
}

// PredictEmergingTraits prédit les traits qui pourraient émerger
func (uc *IdentityEvolutionUseCase) PredictEmergingTraits(ctx context.Context, agentID string) ([]*entities.PersonalityTrait, error) {
	if uc.tracker == nil {
		return nil, nil
	}
	return uc.tracker.PredictNextTraits(ctx, agentID)
}

// SuggestAdjustments suggère des ajustements d'identité
func (uc *IdentityEvolutionUseCase) SuggestAdjustments(ctx context.Context, agentID string) ([]string, error) {
	if uc.tracker == nil {
		return nil, nil
	}
	return uc.tracker.SuggestIdentityAdjustments(ctx, agentID)
}

// GetIdentityLineage retourne la lignée complète d'un snapshot
func (uc *IdentityEvolutionUseCase) GetIdentityLineage(ctx context.Context, snapshotID string) (*ports.IdentityLineage, error) {
	id, err := uuid.Parse(snapshotID)
	if err != nil {
		return nil, fmt.Errorf("invalid snapshot ID %q: %w", snapshotID, err)
	}
	return uc.storage.GetIdentityLineage(ctx, id)
}
