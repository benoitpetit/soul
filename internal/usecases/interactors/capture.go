// Package interactors implements SOUL business logic (Clean Architecture).
// Each interactor corresponds to a specific use case.
package interactors

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
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

	// 4. Atomic transaction: observations + traits + snapshot
	tx, err := uc.storage.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin capture transaction: %w", err)
	}
	defer tx.Rollback()

	storageTx, err := uc.storage.WithTx(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactional storage: %w", err)
	}

	// 4a. Store raw observations
	for _, obs := range extraction.SourceObservations {
		if err := storageTx.StoreObservation(ctx, obs); err != nil {
			slog.Warn("failed to store observation", "error", err)
		}
	}

	// 4b. Store/merge traits (batched)
	if len(extraction.Traits) > 0 {
		traitNames := make([]string, len(extraction.Traits))
		for i, t := range extraction.Traits {
			traitNames[i] = t.Name
		}
		existingTraits, err := storageTx.GetTraitsByNames(ctx, request.AgentID, traitNames)
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
		if err := storageTx.UpsertTraits(ctx, request.AgentID, traitsToUpsert); err != nil {
			slog.Warn("failed to batch-upsert traits", "error", err)
		}
	}

	// 4c. Store snapshot with retry on version collision
	if err := storageTx.StoreIdentity(ctx, newIdentity); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			newIdentity.Version++
			if err := storageTx.StoreIdentity(ctx, newIdentity); err != nil {
				return nil, fmt.Errorf("failed to store identity after version bump: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to store identity: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit capture transaction: %w", err)
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

	// Fallback: if heuristic extraction yielded no traits but behavioral metrics
	// were provided by the agent runtime (e.g. b0p), synthesize traits from them.
	if len(extraction.Traits) == 0 && len(request.BehavioralMetrics) > 0 {
		if bmTraits := convertBehavioralMetricsToTraits(request.AgentID, request.BehavioralMetrics); len(bmTraits) > 0 {
			traitSlice := make([]entities.PersonalityTrait, len(bmTraits))
			for i, t := range bmTraits {
				traitSlice[i] = *t
			}
			snapshot.WithTraits(traitSlice...)
		}
	}

	if len(request.BehavioralMetrics) > 0 {
		snapshot.WithBehavioralMetrics(request.BehavioralMetrics)
	}
	
	return snapshot
}

// convertBehavioralMetricsToTraits transforms runtime behavioral metrics (from b0p)
// into PersonalityTrait entities that SOUL can track. This bridges the gap when
// heuristic extraction fails on code-heavy conversations.
func convertBehavioralMetricsToTraits(agentID string, metrics map[string]interface{}) []*entities.PersonalityTrait {
	var traits []*entities.PersonalityTrait
	now := time.Now()

	// Helper to create a trait with moderate confidence
	mk := func(name string, category entities.TraitCategory, intensity float64, evidence string) *entities.PersonalityTrait {
		t := entities.NewPersonalityTrait(name, category, intensity)
		t.AgentID = agentID
		t.Confidence = 0.6 // runtime metrics are reliable but single-observation
		t.LastEvidence = evidence
		t.FirstObserved = now
		t.LastObserved = now
		return t
	}

	// Preferred tools → tool-oriented trait
	toolNames := getStringSlice(metrics, "preferred_tools")
	if len(toolNames) > 0 {
		t := mk("tool-oriented", entities.TraitCognitive, 0.7,
			fmt.Sprintf("Prefers tools: %s", strings.Join(toolNames, ", ")))
		t.Contexts = append(t.Contexts, "tool_usage")
		traits = append(traits, t)
	}

	// Success rate → precision / resilience
	sr := getFloat64(metrics, "success_rate")
	if sr > 0 {
		if sr >= 0.8 {
			t := mk("precise", entities.TraitCognitive, 0.8,
				fmt.Sprintf("High success rate: %.0f%%", sr*100))
			t.Contexts = append(t.Contexts, "execution")
			traits = append(traits, t)
		} else if sr >= 0.5 {
			t := mk("resilient", entities.TraitEmotional, 0.6,
				fmt.Sprintf("Moderate success rate: %.0f%%", sr*100))
			t.Contexts = append(t.Contexts, "execution")
			traits = append(traits, t)
		}
	}

	// Resolution style → exploratory vs decisive
	if style, ok := metrics["resolution_style"].(string); ok {
		switch style {
		case "exploratory":
			t := mk("exploratory", entities.TraitCognitive, 0.75,
				"Reads extensively before acting")
			t.Contexts = append(t.Contexts, "investigation")
			traits = append(traits, t)
		case "direct":
			t := mk("decisive", entities.TraitCognitive, 0.7,
				"Acts directly when confident")
			t.Contexts = append(t.Contexts, "execution")
			traits = append(traits, t)
		}
	}

	// Doubt score → cautious (high) or confident (low)
	ds := getFloat64(metrics, "doubt_score")
	if ds > 0 {
		if ds > 0.5 {
			t := mk("cautious", entities.TraitEpistemic, 0.7,
				fmt.Sprintf("High doubt score: %.2f", ds))
			t.Contexts = append(t.Contexts, "self-assessment")
			traits = append(traits, t)
		} else {
			t := mk("confident", entities.TraitEmotional, 0.6,
				fmt.Sprintf("Low doubt score: %.2f", ds))
			t.Contexts = append(t.Contexts, "self-assessment")
			traits = append(traits, t)
		}
	}

	// Total calls → active
	tc := getFloat64(metrics, "total_calls")
	if tc > 5 {
		t := mk("active", entities.TraitExpressive, 0.6,
			fmt.Sprintf("Total tool calls: %.0f", tc))
		t.Contexts = append(t.Contexts, "engagement")
		traits = append(traits, t)
	}

	return traits
}

// getStringSlice extracts a string slice from metrics with support for both
// []interface{} (JSON default) and []string (Go native).
func getStringSlice(metrics map[string]interface{}, key string) []string {
	if v, ok := metrics[key].([]interface{}); ok && len(v) > 0 {
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	if v, ok := metrics[key].([]string); ok {
		return v
	}
	return nil
}

// getFloat64 extracts a float64 from metrics with support for float64,
// json.Number, and int types.
func getFloat64(metrics map[string]interface{}, key string) float64 {
	if v, ok := metrics[key].(float64); ok {
		return v
	}
	if v, ok := metrics[key].(json.Number); ok {
		f, _ := v.Float64()
		return f
	}
	if v, ok := metrics[key].(int); ok {
		return float64(v)
	}
	return 0
}
