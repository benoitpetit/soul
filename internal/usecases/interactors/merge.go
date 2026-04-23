// Identity Merger - Fusion d'identités
package interactors

import (
	"context"
	"fmt"

	"github.com/benoitpetit/soul/internal/domain/entities"
	"github.com/benoitpetit/soul/internal/domain/valueobjects"
	"github.com/benoitpetit/soul/internal/usecases/ports"
)

// IdentityMergeUseCase implémente la fusion d'identités
// Utilisé quand deux sessions ou deux instances d'agent doivent fusionner.
type IdentityMergeUseCase struct {
	storage ports.SoulStorage
	merger  ports.SoulMerger
}

// NewIdentityMergeUseCase crée un nouveau use case
func NewIdentityMergeUseCase(storage ports.SoulStorage, merger ports.SoulMerger) *IdentityMergeUseCase {
	return &IdentityMergeUseCase{
		storage: storage,
		merger:  merger,
	}
}

// MergeIdentities fusionne deux identités selon une stratégie
func (uc *IdentityMergeUseCase) MergeIdentities(ctx context.Context, agentIDA, agentIDB string, strategy valueobjects.MergeStrategy) (*entities.IdentitySnapshot, error) {
	// 1. Récupérer les deux identités
	identityA, err := uc.storage.GetLatestIdentity(ctx, agentIDA)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve identity A: %w", err)
	}
	identityB, err := uc.storage.GetLatestIdentity(ctx, agentIDB)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve identity B: %w", err)
	}
	
	if identityA == nil || identityB == nil {
		return nil, fmt.Errorf("one or both identities not found")
	}
	
	// 2. Calculer la compatibilité
	compatibility, err := uc.merger.CalculateMergeCompatibility(ctx, identityA, identityB)
	if err != nil {
		return nil, fmt.Errorf("compatibility calculation failed: %w", err)
	}
	
	fmt.Printf("Merge compatibility between %s and %s: %.2f\n", agentIDA, agentIDB, compatibility)
	
	if compatibility < 0.3 {
		return nil, fmt.Errorf("identities are incompatible (%.2f), merge aborted", compatibility)
	}
	
	// 3. Fusionner
	merged, err := uc.merger.MergeIdentities(ctx, identityA, identityB, strategy)
	if err != nil {
		return nil, fmt.Errorf("merge failed: %w", err)
	}
	
	// 4. Sauvegarder le résultat
	merged.AgentID = agentIDA // L'agent A est considéré comme le "principal"
	if err := uc.storage.StoreIdentity(ctx, merged); err != nil {
		return nil, fmt.Errorf("failed to store merged identity: %w", err)
	}
	
	return merged, nil
}

// CalculateCompatibility calcule la compatibilité entre deux identités sans les fusionner
func (uc *IdentityMergeUseCase) CalculateCompatibility(ctx context.Context, agentIDA, agentIDB string) (float64, error) {
	identityA, err := uc.storage.GetLatestIdentity(ctx, agentIDA)
	if err != nil {
		return 0, err
	}
	identityB, err := uc.storage.GetLatestIdentity(ctx, agentIDB)
	if err != nil {
		return 0, err
	}
	
	if identityA == nil || identityB == nil {
		return 0, fmt.Errorf("one or both identities not found")
	}
	
	return uc.merger.CalculateMergeCompatibility(ctx, identityA, identityB)
}
