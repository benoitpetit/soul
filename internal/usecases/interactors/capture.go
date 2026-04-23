// Package interactors implements SOUL business logic (Clean Architecture).
// Each interactor corresponds to a specific use case.
package interactors

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/benoitpetit/soul/internal/domain/entities"
	"github.com/benoitpetit/soul/internal/domain/valueobjects"
	"github.com/benoitpetit/soul/internal/usecases/ports"
)

// IdentityCaptureUseCase implements identity capture from conversations.
type IdentityCaptureUseCase struct {
	storage   ports.SoulStorage
	extractor ports.IdentityExtractor
}

// NewIdentityCaptureUseCase creates a new capture use case.
func NewIdentityCaptureUseCase(storage ports.SoulStorage, extractor ports.IdentityExtractor) *IdentityCaptureUseCase {
	return &IdentityCaptureUseCase{
		storage:   storage,
		extractor: extractor,
	}
}

// CaptureFromConversation captures identity from a complete conversation.
func (uc *IdentityCaptureUseCase) CaptureFromConversation(ctx context.Context, request *valueobjects.SoulCaptureRequest) (*entities.IdentitySnapshot, error) {
	// 1. Extract all identity elements from the conversation
	extraction, err := uc.extractor.ExtractFromConversation(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	// 2. Get existing identity if any
	existingIdentity, err := uc.storage.GetLatestIdentity(ctx, request.AgentID)
	if err != nil {
		existingIdentity = nil
	}

	// 3. Build new snapshot
	newIdentity := uc.buildSnapshotFromExtraction(request, extraction, existingIdentity)

	// 4. Store raw observations
	for _, obs := range extraction.SourceObservations {
		if err := uc.storage.StoreObservation(ctx, obs); err != nil {
			slog.Warn("failed to store observation", "error", err)
		}
	}

	// 5. Store/merge traits
	for _, trait := range extraction.Traits {
		existingTrait, _ := uc.storage.GetTraitByName(ctx, request.AgentID, trait.Name)
		if existingTrait != nil {
			existingTrait.Merge(trait)
			if err := uc.storage.UpdateTrait(ctx, existingTrait); err != nil {
				slog.Warn("failed to update trait", "error", err)
			}
		} else {
			if err := uc.storage.StoreTrait(ctx, trait); err != nil {
				slog.Warn("failed to store trait", "error", err)
			}
		}
	}

	// 6. Store snapshot
	if err := uc.storage.StoreIdentity(ctx, newIdentity); err != nil {
		return nil, fmt.Errorf("failed to store identity: %w", err)
	}

	return newIdentity, nil
}

// CaptureFromSingleInteraction captures identity from a single interaction.
// Used for incremental updates.
func (uc *IdentityCaptureUseCase) CaptureFromSingleInteraction(ctx context.Context, agentID, agentResponse, userMessage, modelID string) error {
	request := &valueobjects.SoulCaptureRequest{
		AgentID:        agentID,
		Conversation:   userMessage + "\n" + agentResponse,
		AgentResponses: []string{agentResponse},
		ModelID:        modelID,
		Timestamp:      time.Now(),
	}
	
	_, err := uc.CaptureFromConversation(ctx, request)
	return err
}

// buildSnapshotFromExtraction construit un snapshot à partir du résultat d'extraction
func (uc *IdentityCaptureUseCase) buildSnapshotFromExtraction(
	request *valueobjects.SoulCaptureRequest,
	extraction *ports.ExtractionResult,
	existing *entities.IdentitySnapshot,
) *entities.IdentitySnapshot {
	var snapshot *entities.IdentitySnapshot
	
	if existing != nil {
		// Créer une nouvelle version basée sur l'existant
		snapshot = entities.NewIdentitySnapshot(request.AgentID, request.ModelID)
		snapshot.WithParentSnapshot(existing.ID)
		
		// Hériter les dimensions qui n'ont pas été extraites
		snapshot.VoiceProfile = existing.VoiceProfile
		snapshot.CommunicationStyle = existing.CommunicationStyle
		snapshot.BehavioralSignature = existing.BehavioralSignature
		snapshot.ValueSystem = existing.ValueSystem
		snapshot.EmotionalTone = existing.EmotionalTone
		snapshot.PersonalityTraits = existing.PersonalityTraits
	} else {
		snapshot = entities.NewIdentitySnapshot(request.AgentID, request.ModelID)
	}
	
	// Mettre à jour avec les nouvelles extractions
	if extraction.VoiceProfile != nil {
		snapshot.WithVoiceProfile(*extraction.VoiceProfile)
	}
	if extraction.CommunicationStyle != nil {
		snapshot.WithCommunicationStyle(*extraction.CommunicationStyle)
	}
	if extraction.BehavioralSignature != nil {
		snapshot.WithBehavioralSignature(*extraction.BehavioralSignature)
	}
	if extraction.ValueSystem != nil {
		snapshot.WithValueSystem(*extraction.ValueSystem)
	}
	if extraction.EmotionalTone != nil {
		snapshot.WithEmotionalTone(*extraction.EmotionalTone)
	}
	if len(extraction.Traits) > 0 {
		traitSlice := make([]entities.PersonalityTrait, len(extraction.Traits))
		for i, t := range extraction.Traits {
			traitSlice[i] = *t
		}
		snapshot.WithTraits(traitSlice...)
	}

	if len(request.BehavioralMetrics) > 0 {
		snapshot.WithBehavioralMetrics(request.BehavioralMetrics)
	}
	
	return snapshot
}
