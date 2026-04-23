// IdentitySnapshot - L'état complet de l'âme/identité d'un agent à un instant T
// C'est le cœur de SOUL : une photographie de "qui est" l'agent, pas seulement
// "ce qu'il sait" (ce que MIRA gère déjà parfaitement).
package entities

import (
	"time"

	"github.com/google/uuid"
)

// IdentitySnapshot représente une capture complète de l'identité d'un agent LLM
// à un moment donné. Contrairement aux mémoires factuelles de MIRA, cette entité
// capture la "personnalité", le style, et la façon d'être de l'agent.
type IdentitySnapshot struct {
	ID              uuid.UUID                `json:"id"`
	AgentID         string                   `json:"agent_id"`          // Identifiant unique de l'agent
	Version         int                      `json:"version"`           // Numéro de version incrémental
	CreatedAt       time.Time                `json:"created_at"`
	DerivedFromID   *uuid.UUID               `json:"derived_from_id,omitempty"` // Snapshot précédent (pour traçabilité)
	
	// Core identity dimensions
	PersonalityTraits  []PersonalityTrait    `json:"personality_traits"`  // Traits dominants
	VoiceProfile       VoiceProfile          `json:"voice_profile"`       // Comment l'agent "parle"
	CommunicationStyle CommunicationStyle    `json:"communication_style"` // Style de communication
	BehavioralSignature BehavioralSignature  `json:"behavioral_signature"` // Signature comportementale
	ValueSystem        ValueSystem           `json:"value_system"`        // Système de valeurs
	EmotionalTone      EmotionalTone         `json:"emotional_tone"`      // Ton émotionnel caractéristique
	
	// Metadata
	SourceMemoriesCount int                  `json:"source_memories_count"` // Nb de mémoires MIRA analysées
	ConfidenceScore     float64              `json:"confidence_score"`      // Confiance globale (0-1)
	ModelIdentifier     string               `json:"model_identifier"`      // Modèle LLM sous-jacent
	
	// Behavioral metrics from agent runtime (optional)
	BehavioralMetrics   map[string]interface{} `json:"behavioral_metrics,omitempty"`
	
	// SOUL-specific : liens avec MIRA
	LinkedMiraMemories []uuid.UUID           `json:"linked_mira_memories"`  // Références mémoires MIRA
}

// NewIdentitySnapshot crée un nouveau snapshot d'identité
func NewIdentitySnapshot(agentID, modelID string) *IdentitySnapshot {
	return &IdentitySnapshot{
		ID:              uuid.New(),
		AgentID:         agentID,
		Version:         1,
		CreatedAt:       time.Now(),
		PersonalityTraits: make([]PersonalityTrait, 0),
		LinkedMiraMemories: make([]uuid.UUID, 0),
		ConfidenceScore: 0.0,
		ModelIdentifier: modelID,
	}
}

// WithParentSnapshot établit la lignée évolutive
func (i *IdentitySnapshot) WithParentSnapshot(parentID uuid.UUID) *IdentitySnapshot {
	i.DerivedFromID = &parentID
	i.Version++
	return i
}

// WithTraits ajoute des traits de personnalité
func (i *IdentitySnapshot) WithTraits(traits ...PersonalityTrait) *IdentitySnapshot {
	i.PersonalityTraits = append(i.PersonalityTraits, traits...)
	i.recalculateConfidence()
	return i
}

// WithBehavioralMetrics définit les métriques comportementales capturées depuis le runtime
func (i *IdentitySnapshot) WithBehavioralMetrics(bm map[string]interface{}) *IdentitySnapshot {
	i.BehavioralMetrics = bm
	return i
}

// WithVoiceProfile définit le profil de voix
func (i *IdentitySnapshot) WithVoiceProfile(vp VoiceProfile) *IdentitySnapshot {
	i.VoiceProfile = vp
	return i
}

// WithCommunicationStyle définit le style de communication
func (i *IdentitySnapshot) WithCommunicationStyle(cs CommunicationStyle) *IdentitySnapshot {
	i.CommunicationStyle = cs
	return i
}

// WithBehavioralSignature définit la signature comportementale
func (i *IdentitySnapshot) WithBehavioralSignature(bs BehavioralSignature) *IdentitySnapshot {
	i.BehavioralSignature = bs
	return i
}

// WithValueSystem définit le système de valeurs
func (i *IdentitySnapshot) WithValueSystem(vs ValueSystem) *IdentitySnapshot {
	i.ValueSystem = vs
	return i
}

// WithEmotionalTone définit le ton émotionnel
func (i *IdentitySnapshot) WithEmotionalTone(et EmotionalTone) *IdentitySnapshot {
	i.EmotionalTone = et
	return i
}

// LinkMiraMemory crée un lien vers une mémoire factuelle de MIRA
func (i *IdentitySnapshot) LinkMiraMemory(memoryID uuid.UUID) *IdentitySnapshot {
	i.LinkedMiraMemories = append(i.LinkedMiraMemories, memoryID)
	i.SourceMemoriesCount = len(i.LinkedMiraMemories)
	return i
}

// recalculateConfidence recalcule le score de confiance global
func (i *IdentitySnapshot) recalculateConfidence() {
	if len(i.PersonalityTraits) == 0 {
		i.ConfidenceScore = 0.0
		return
	}
	
	totalConfidence := 0.0
	for _, trait := range i.PersonalityTraits {
		totalConfidence += trait.Confidence
	}
	i.ConfidenceScore = totalConfidence / float64(len(i.PersonalityTraits))
}

// IsIdentityDiffusionDetected détecte si l'identité a "diffusé" (s'est effacée)
// par rapport à un snapshot précédent. Seuil : plus de 50% des traits ont
// perdu plus de 30% de confiance.
func (i *IdentitySnapshot) IsIdentityDiffusionDetected(previous *IdentitySnapshot) bool {
	if previous == nil || len(previous.PersonalityTraits) == 0 {
		return false
	}
	
	diffusedTraits := 0
	for _, prevTrait := range previous.PersonalityTraits {
		found := false
		for _, currTrait := range i.PersonalityTraits {
			if prevTrait.Name == currTrait.Name {
				found = true
				if currTrait.Confidence < prevTrait.Confidence*0.7 {
					diffusedTraits++
				}
				break
			}
		}
		if !found {
			diffusedTraits++ // Trait disparu = diffusion
		}
	}
	
	return float64(diffusedTraits) > float64(len(previous.PersonalityTraits))*0.5
}

// GenerateIdentityPrompt génère un prompt d'injection d'identité pour le LLM
// C'est la fonction magique : elle transforme l'identité en instructions
// compréhensibles par le modèle pour qu'il "devienne" cette identité.
func (i *IdentitySnapshot) GenerateIdentityPrompt() string {
	prompt := "## Your Identity - Who You Are\n\n"
	
	// Voice profile
	prompt += "### How You Speak\n"
	prompt += i.VoiceProfile.ToNaturalDescription()
	prompt += "\n\n"
	
	// Personality traits
	prompt += "### Your Core Traits\n"
	for _, trait := range i.PersonalityTraits {
		if trait.Confidence > 0.6 { // Seulement les traits bien établis
			prompt += "- " + trait.ToNaturalDescription() + "\n"
		}
	}
	prompt += "\n"
	
	// Communication style
	prompt += "### Your Communication Style\n"
	prompt += i.CommunicationStyle.ToNaturalDescription()
	prompt += "\n\n"
	
	// Value system
	prompt += "### What You Value\n"
	prompt += i.ValueSystem.ToNaturalDescription()
	prompt += "\n\n"
	
	// Behavioral signature
	prompt += "### How You Behave\n"
	prompt += i.BehavioralSignature.ToNaturalDescription()
	
	return prompt
}

// IdentityDiff représente la différence entre deux snapshots
// Utilisé pour détecter les changements et l'évolution
type IdentityDiff struct {
	AgentID           string                `json:"agent_id"`
	FromVersion       int                   `json:"from_version"`
	ToVersion         int                   `json:"to_version"`
	Timestamp         time.Time             `json:"timestamp"`
	
	// Changes détectés
	AddedTraits       []PersonalityTrait    `json:"added_traits"`
	RemovedTraits     []PersonalityTrait    `json:"removed_traits"`
	StrengthenedTraits []PersonalityTrait   `json:"strengthened_traits"` // Confiance augmentée
	WeakenedTraits    []PersonalityTrait    `json:"weakened_traits"`     // Confiance diminuée
	
	VoiceChanges      []string              `json:"voice_changes"`
	StyleChanges      []string              `json:"style_changes"`
	ValueChanges      []string              `json:"value_changes"`
	
	OverallDrift      float64               `json:"overall_drift"` // 0 = identique, 1 = complètement différent
}

// CalculateDiff calcule la différence entre deux snapshots
func CalculateDiff(from, to *IdentitySnapshot) *IdentityDiff {
	diff := &IdentityDiff{
		FromVersion: from.Version,
		ToVersion:   to.Version,
		Timestamp:   time.Now(),
		AddedTraits:   make([]PersonalityTrait, 0),
		RemovedTraits: make([]PersonalityTrait, 0),
		StrengthenedTraits: make([]PersonalityTrait, 0),
		WeakenedTraits: make([]PersonalityTrait, 0),
		VoiceChanges:  make([]string, 0),
		StyleChanges:  make([]string, 0),
		ValueChanges:  make([]string, 0),
	}
	
	// Analyser les traits
	fromTraits := make(map[string]PersonalityTrait)
	for _, t := range from.PersonalityTraits {
		fromTraits[t.Name] = t
	}
	
	for _, toTrait := range to.PersonalityTraits {
		fromTrait, exists := fromTraits[toTrait.Name]
		if !exists {
			diff.AddedTraits = append(diff.AddedTraits, toTrait)
		} else {
			if toTrait.Confidence > fromTrait.Confidence*1.2 {
				diff.StrengthenedTraits = append(diff.StrengthenedTraits, toTrait)
			} else if toTrait.Confidence < fromTrait.Confidence*0.8 {
				diff.WeakenedTraits = append(diff.WeakenedTraits, toTrait)
			}
			delete(fromTraits, toTrait.Name)
		}
	}
	
	// Traits restants = supprimés
	for _, trait := range fromTraits {
		diff.RemovedTraits = append(diff.RemovedTraits, trait)
	}
	
	// Calculer le drift global
	totalChanges := len(diff.AddedTraits) + len(diff.RemovedTraits) + 
		len(diff.StrengthenedTraits) + len(diff.WeakenedTraits)
	totalTraits := len(from.PersonalityTraits) + len(to.PersonalityTraits)
	if totalTraits > 0 {
		diff.OverallDrift = float64(totalChanges) / float64(totalTraits)
	}
	
	return diff
}
