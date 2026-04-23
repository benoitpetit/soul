// Recall - Récupération de l'identité pour injection dans le contexte LLM
package interactors

import (
	"context"
	"fmt"

	"github.com/benoitpetit/soul/internal/domain/entities"
	"github.com/benoitpetit/soul/internal/domain/valueobjects"
	"github.com/benoitpetit/soul/internal/usecases/ports"
)

// IdentityRecallUseCase implémente le use case de récupération d'identité
// C'est le "cœur battant" de SOUL : fournir au LLM son "souvenir de soi".
type IdentityRecallUseCase struct {
	storage   ports.SoulStorage
	composer  ports.IdentityComposer
}

// NewIdentityRecallUseCase crée un nouveau use case de recall
func NewIdentityRecallUseCase(storage ports.SoulStorage, composer ports.IdentityComposer) *IdentityRecallUseCase {
	return &IdentityRecallUseCase{
		storage:  storage,
		composer: composer,
	}
}

// RecallIdentity récupère l'identité complète d'un agent et génère un prompt
// d'injection identitaire prêt à être inséré dans le context window du LLM.
func (uc *IdentityRecallUseCase) RecallIdentity(ctx context.Context, query *valueobjects.SoulQuery) (*valueobjects.IdentityContextPrompt, error) {
	// 1. Récupérer le dernier snapshot
	identity, err := uc.storage.GetLatestIdentity(ctx, query.AgentID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve identity for agent %s: %w", query.AgentID, err)
	}
	
	if identity == nil {
		return nil, fmt.Errorf("no identity found for agent %s", query.AgentID)
	}
	
	// 2. Vérifier si diffusion identitaire détectée
	if identity.DerivedFromID != nil {
		previous, _ := uc.storage.GetIdentityByID(ctx, *identity.DerivedFromID)
		if previous != nil && identity.IsIdentityDiffusionDetected(previous) {
			// Générer une alerte de diffusion
			fmt.Printf("WARNING: Identity diffusion detected for agent %s\n", query.AgentID)
		}
	}
	
	// 3. Composer le prompt d'identité
	budget := query.BudgetTokens
	if budget <= 0 {
		budget = 1000 // Budget par défaut : 1000 tokens
	}
	
	prompt, err := uc.composer.ComposeIdentityPrompt(ctx, identity, budget)
	if err != nil {
		return nil, fmt.Errorf("failed to compose identity prompt: %w", err)
	}
	
	return prompt, nil
}

// RecallIdentityWithContext récupère l'identité en tenant compte du contexte conversationnel
// Cette méthode enrichit l'identité avec les mémoires factuelles pertinentes de MIRA.
func (uc *IdentityRecallUseCase) RecallIdentityWithContext(ctx context.Context, query *valueobjects.SoulQuery) (*valueobjects.IdentityContextPrompt, error) {
	// 1. Récupérer l'identité de base
	basePrompt, err := uc.RecallIdentity(ctx, query)
	if err != nil {
		return nil, err
	}
	
	// 2. Récupérer les mémoires MIRA pertinentes
	miraMemories, err := uc.storage.GetMiraMemories(ctx, query.AgentID, query.Context, 5)
	if err != nil {
		// Si MIRA n'est pas disponible, retourner le prompt de base
		return basePrompt, nil
	}
	
	// 3. Enrichir le prompt avec les mémoires liées
	enrichedContent := basePrompt.Content
	if len(miraMemories) > 0 {
		enrichedContent += "\n\n### Memories That Shape Your Perspective\n"
		for _, mem := range miraMemories {
			enrichedContent += fmt.Sprintf("- %s\n", mem.Content)
		}
	}
	
	// 4. Recalculer l'estimation de tokens
	tokenEstimate := len(enrichedContent) / 4 // Approximation grossière
	
	return &valueobjects.IdentityContextPrompt{
		Content:         enrichedContent,
		TokenEstimate:   tokenEstimate,
		Priority:        basePrompt.Priority,
		GeneratedAt:     basePrompt.GeneratedAt,
		SnapshotVersion: basePrompt.SnapshotVersion,
	}, nil
}

// GetIdentitySummary retourne un résumé lisible de l'identité
func (uc *IdentityRecallUseCase) GetIdentitySummary(ctx context.Context, agentID string) (string, error) {
	identity, err := uc.storage.GetLatestIdentity(ctx, agentID)
	if err != nil {
		return "", err
	}
	if identity == nil {
		return "No identity captured yet.", nil
	}
	
	return identity.GenerateIdentityPrompt(), nil
}

// GetIdentityTraits retourne les traits de personnalité d'un agent
func (uc *IdentityRecallUseCase) GetIdentityTraits(ctx context.Context, agentID string, wellEstablishedOnly bool) ([]*entities.PersonalityTrait, error) {
	if wellEstablishedOnly {
		return uc.storage.GetWellEstablishedTraits(ctx, agentID, 0.7)
	}
	return uc.storage.GetAllTraits(ctx, agentID)
}

// GetIdentityHistory retourne l'historique des snapshots
func (uc *IdentityRecallUseCase) GetIdentityHistory(ctx context.Context, agentID string, limit int) ([]*entities.IdentitySnapshot, error) {
	return uc.storage.GetIdentityHistory(ctx, agentID, limit)
}
