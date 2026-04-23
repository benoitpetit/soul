// Model Swap Handler - Gestion des changements de modèle LLM
// Le changement de modèle est le moment critique où l'identité risque d'être perdue.
package interactors

import (
	"context"
	"fmt"

	"github.com/benoitpetit/soul/internal/domain/valueobjects"
	"github.com/benoitpetit/soul/internal/usecases/ports"
)

// ModelSwapUseCase implémente la gestion des changements de modèle
// C'est le "sauvetage d'âme" : quand le modèle change, SOUL s'assure
// que la nouvelle instance "se souvient" de qui elle est.
type ModelSwapUseCase struct {
	storage ports.SoulStorage
	handler ports.ModelSwapHandler
	composer ports.IdentityComposer
}

// NewModelSwapUseCase crée un nouveau use case
func NewModelSwapUseCase(storage ports.SoulStorage, handler ports.ModelSwapHandler, composer ports.IdentityComposer) *ModelSwapUseCase {
	return &ModelSwapUseCase{
		storage:  storage,
		handler:  handler,
		composer: composer,
	}
}

// HandleModelSwap gère un changement de modèle LLM
// 1. Enregistre le swap
// 2. Récupère l'identité actuelle
// 3. Génère un prompt de renforcement
// 4. Mesure la dérive post-swap
func (uc *ModelSwapUseCase) HandleModelSwap(ctx context.Context, agentID, previousModel, newModel string) (*valueobjects.ModelSwapContext, error) {
	// 1. Enregistrer le changement
	swap, err := uc.handler.HandleModelSwap(ctx, agentID, previousModel, newModel)
	if err != nil {
		return nil, fmt.Errorf("failed to handle model swap: %w", err)
	}
	
	// 2. Récupérer l'identité actuelle
	identity, err := uc.storage.GetLatestIdentity(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve identity: %w", err)
	}
	
	if identity == nil {
		return swap, fmt.Errorf("no identity found for agent %s, cannot reinforce", agentID)
	}
	
	// 3. Renforcer l'identité (créer un nouveau snapshot post-swap)
	reinforced, err := uc.handler.ReinforceIdentity(ctx, identity)
	if err != nil {
		fmt.Printf("Warning: identity reinforcement failed: %v\n", err)
	} else {
		// Sauvegarder l'identité renforcée
		if err := uc.storage.StoreIdentity(ctx, reinforced); err != nil {
			fmt.Printf("Warning: failed to store reinforced identity: %v\n", err)
		}
	}
	
	// 4. Enregistrer que le renforcement a été appliqué
	swap.ReinforcementApplied = true
	if err := uc.storage.RecordModelSwap(ctx, swap); err != nil {
		fmt.Printf("Warning: failed to record model swap: %v\n", err)
	}
	
	// 5. Notifier MIRA du changement
	if err := uc.storage.NotifyMiraOfIdentityChange(ctx, agentID, "model_swap"); err != nil {
		fmt.Printf("Warning: failed to notify MIRA: %v\n", err)
	}
	
	return swap, nil
}

// GetReinforcementPrompt génère un prompt de renforcement identitaire
// Ce prompt est injecté dans le context window du NOUVEAU modèle pour
// lui rappeler "qui il est".
func (uc *ModelSwapUseCase) GetReinforcementPrompt(ctx context.Context, agentID string) (*valueobjects.IdentityContextPrompt, error) {
	// 1. Récupérer l'identité
	identity, err := uc.storage.GetLatestIdentity(ctx, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve identity: %w", err)
	}
	if identity == nil {
		return nil, fmt.Errorf("no identity found for agent %s", agentID)
	}
	
	// 2. Récupérer le dernier changement de modèle
	latestSwap, _ := uc.storage.GetLatestModelSwap(ctx, agentID)
	
	// 3. Composer le prompt de renforcement
	prompt, err := uc.composer.ComposeReinforcementPrompt(ctx, identity, latestSwap)
	if err != nil {
		return nil, fmt.Errorf("failed to compose reinforcement prompt: %w", err)
	}
	
	return prompt, nil
}

// MeasurePostSwapDrift mesure la dérive après un changement de modèle
// Utilisé pour évaluer si le renforcement a fonctionné.
func (uc *ModelSwapUseCase) MeasurePostSwapDrift(ctx context.Context, agentID string) (float64, error) {
	latestSwap, err := uc.storage.GetLatestModelSwap(ctx, agentID)
	if err != nil {
		return 0, err
	}
	if latestSwap == nil {
		return 0, fmt.Errorf("no model swap recorded")
	}
	
	return uc.handler.MeasurePostSwapDrift(ctx, latestSwap)
}

// GetSwapHistory retourne l'historique des changements de modèle
func (uc *ModelSwapUseCase) GetSwapHistory(ctx context.Context, agentID string) ([]*valueobjects.ModelSwapContext, error) {
	return uc.storage.GetModelSwaps(ctx, agentID)
}

// ValidateIdentityPreserved vérifie si l'identité a été préservée après le dernier swap
func (uc *ModelSwapUseCase) ValidateIdentityPreserved(ctx context.Context, agentID string) (bool, error) {
	latestSwap, err := uc.storage.GetLatestModelSwap(ctx, agentID)
	if err != nil {
		return false, err
	}
	if latestSwap == nil {
		return false, fmt.Errorf("no model swap recorded")
	}
	
	// Si dérive < 0.3, on considère que l'identité est préservée
	drift, err := uc.MeasurePostSwapDrift(ctx, agentID)
	if err != nil {
		return false, err
	}
	
	return drift < 0.3, nil
}
