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

	// 5. Store/merge traits (batched: 2 queries total instead of 2N+1)
	if len(extraction.Traits) > 0 {
		traitNames := make([]string, len(extraction.Traits))
		for i, t := range extraction.Traits {
			traitNames[i] = t.Name
		}
		existingTraits, err := uc.storage.GetTraitsByNames(ctx, request.AgentID, traitNames)
		if err != nil {
			slog.Warn("failed to batch-fetch traits", "error", err)
		}
		existingMap := make(map[string]*entities.PersonalityTrait, len(existingTraits))
		for _, et := range existingTraits {
			existingMap[et.Name] = et
		}

		var traitsToUpsert []*entities.PersonalityTrait
		for _, trait := range extraction.Traits {
			if existing, ok := existingMap[trait.Name]; ok {
				existing.Merge(trait)
				traitsToUpsert = append(traitsToUpsert, existing)
			} else {
				traitsToUpsert = append(traitsToUpsert, trait)
			}
		}
		if err := uc.storage.UpsertTraits(ctx, request.AgentID, traitsToUpsert); err != nil {
			slog.Warn("failed to batch-upsert traits", "error", err)
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
