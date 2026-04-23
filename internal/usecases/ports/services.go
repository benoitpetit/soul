// Services - Interfaces pour les services externes et algorithmes de SOUL
package ports

import (
	"context"

	"github.com/benoitpetit/soul/internal/domain/entities"
	"github.com/benoitpetit/soul/internal/domain/valueobjects"
)

// IdentityExtractor définit l'interface pour l'extraction d'identité depuis du texte
// C'est le cœur algorithmique de SOUL : transformer des conversations en traits identitaires.
type IdentityExtractor interface {
	// ExtractFromConversation extrait l'identité depuis une conversation complète
	ExtractFromConversation(ctx context.Context, request *valueobjects.SoulCaptureRequest) (*ExtractionResult, error)
	
	// ExtractTraits extrait les traits de personnalité depuis des réponses d'agent
	ExtractTraits(ctx context.Context, agentResponses []string, context string) ([]*entities.TraitObservation, error)
	
	// ExtractVoiceProfile extrait le profil de voix depuis des réponses
	ExtractVoiceProfile(ctx context.Context, agentResponses []string) (*entities.VoiceProfile, error)
	
	// ExtractCommunicationStyle extrait le style de communication
	ExtractCommunicationStyle(ctx context.Context, agentResponses []string) (*entities.CommunicationStyle, error)
	
	// ExtractBehavioralSignature extrait la signature comportementale
	ExtractBehavioralSignature(ctx context.Context, conversation string, agentResponses []string) (*entities.BehavioralSignature, error)
	
	// ExtractValueSystem extrait le système de valeurs
	ExtractValueSystem(ctx context.Context, agentResponses []string, userFeedback map[string]string) (*entities.ValueSystem, error)
	
	// ExtractEmotionalTone extrait le ton émotionnel
	ExtractEmotionalTone(ctx context.Context, agentResponses []string) (*entities.EmotionalTone, error)
}

// ExtractionResult contient tous les éléments extraits d'une conversation
type ExtractionResult struct {
	Traits               []*entities.PersonalityTrait      `json:"traits"`
	VoiceProfile         *entities.VoiceProfile            `json:"voice_profile"`
	CommunicationStyle   *entities.CommunicationStyle      `json:"communication_style"`
	BehavioralSignature  *entities.BehavioralSignature     `json:"behavioral_signature"`
	ValueSystem          *entities.ValueSystem             `json:"value_system"`
	EmotionalTone        *entities.EmotionalTone           `json:"emotional_tone"`
	SourceObservations   []*entities.TraitObservation      `json:"source_observations"`
	Confidence           float64                           `json:"confidence"`
	ExtractionTimestamp  string                            `json:"extraction_timestamp"`
}

// IdentityComposer définit l'interface pour la composition du contexte identitaire
// Transforme l'identité stockée en prompt injectable dans le context window du LLM.
type IdentityComposer interface {
	// ComposeIdentityPrompt génère un prompt d'identité à partir d'un snapshot
	ComposeIdentityPrompt(ctx context.Context, identity *entities.IdentitySnapshot, budgetTokens int) (*valueobjects.IdentityContextPrompt, error)
	
	// ComposeReinforcementPrompt génère un prompt de renforcement après changement de modèle
	ComposeReinforcementPrompt(ctx context.Context, identity *entities.IdentitySnapshot, swap *valueobjects.ModelSwapContext) (*valueobjects.IdentityContextPrompt, error)
	
	// ComposeDiffusionAlert génère une alerte si diffusion identitaire détectée
	ComposeDiffusionAlert(ctx context.Context, drift *valueobjects.IdentityDriftReport) (*valueobjects.IdentityContextPrompt, error)
	
	// EstimateTokenCount estime le nombre de tokens d'un prompt d'identité
	EstimateTokenCount(ctx context.Context, prompt string) (int, error)
}

// IdentityDriftDetector définit l'interface pour la détection de dérive identitaire
// Surveille si l'identité de l'agent "s'efface" au fil du temps.
type IdentityDriftDetector interface {
	// DetectDrift compare deux snapshots et détecte la dérive
	DetectDrift(ctx context.Context, previous, current *entities.IdentitySnapshot) (*valueobjects.IdentityDriftReport, error)
	
	// DetectDiffusion détecte si l'identité s'est "diffusée" (perdue)
	DetectDiffusion(ctx context.Context, identity *entities.IdentitySnapshot) (bool, float64, error)
	
	// MonitorContinuously surveille en continu la dérive (pour usage async)
	MonitorContinuously(ctx context.Context, agentID string, threshold float64) (<-chan valueobjects.IdentityDriftReport, error)
	
	// CalculateIdentityVector calcule le vecteur dimensionnel de l'identité
	CalculateIdentityVector(ctx context.Context, identity *entities.IdentitySnapshot) (*entities.IdentityDimensionVector, error)
}

// ModelSwapHandler définit l'interface pour la gestion des changements de modèle
// Moment critique où l'identité risque d'être perdue.
type ModelSwapHandler interface {
	// HandleModelSwap gère le changement de modèle
	HandleModelSwap(ctx context.Context, agentID, previousModel, newModel string) (*valueobjects.ModelSwapContext, error)
	
	// ReinforceIdentity renforce l'identité après un changement
	ReinforceIdentity(ctx context.Context, identity *entities.IdentitySnapshot) (*entities.IdentitySnapshot, error)
	
	// MeasurePostSwapDrift mesure la dérive après changement
	MeasurePostSwapDrift(ctx context.Context, swap *valueobjects.ModelSwapContext) (float64, error)
}

// SoulEmbedder définit l'interface pour la génération d'embeddings identitaires
// Permet la recherche vectorielle d'identités (intégration avec HNSW de MIRA)
type SoulEmbedder interface {
	// EncodeIdentity encode un snapshot d'identité en vecteur
	EncodeIdentity(ctx context.Context, identity *entities.IdentitySnapshot) ([]float32, error)
	
	// EncodeTrait encode un trait en vecteur
	EncodeTrait(ctx context.Context, trait *entities.PersonalityTrait) ([]float32, error)
	
	// FindSimilarIdentities trouve les identités similaires par recherche vectorielle
	FindSimilarIdentities(ctx context.Context, vector []float32, limit int) ([]*entities.IdentitySnapshot, error)
	
	// ModelHash retourne l'identifiant du modèle d'embedding
	ModelHash() string
	
	// Dimension retourne la dimension des vecteurs
	Dimension() int
}

// IdentityEvolutionTracker définit l'interface pour le suivi de l'évolution
type IdentityEvolutionTracker interface {
	// TrackEvolution enregistre une évolution et retourne le diff
	TrackEvolution(ctx context.Context, oldSnapshot, newSnapshot *entities.IdentitySnapshot) (*entities.IdentityDiff, error)
	
	// GetEvolutionTimeline retourne la timeline d'évolution
	GetEvolutionTimeline(ctx context.Context, agentID string) ([]*entities.IdentityDiff, error)
	
	// PredictNextTraits prédit les traits qui pourraient émerger
	PredictNextTraits(ctx context.Context, agentID string) ([]*entities.PersonalityTrait, error)
	
	// SuggestIdentityAdjustments suggère des ajustements basés sur l'évolution
	SuggestIdentityAdjustments(ctx context.Context, agentID string) ([]string, error)
}

// SoulMerger définit l'interface pour la fusion d'identités
// Utilisé quand deux sessions/agent doivent fusionner.
type SoulMerger interface {
	// MergeIdentities fusionne deux identités
	MergeIdentities(ctx context.Context, identityA, identityB *entities.IdentitySnapshot, strategy valueobjects.MergeStrategy) (*entities.IdentitySnapshot, error)
	
	// CalculateMergeCompatibility calcule la compatibilité entre deux identités
	CalculateMergeCompatibility(ctx context.Context, identityA, identityB *entities.IdentitySnapshot) (float64, error)
}
