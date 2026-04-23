// Package valueobjects définit les objets de valeur du domaine SOUL
// Ces objets sont immuables et identifiés par leurs attributs, pas par un ID.
package valueobjects

import (
	"fmt"
	"time"
)

// IdentityVersion représente une version d'identité (immutable)
type IdentityVersion struct {
	Major     int       `json:"major"`     // Changement majeur de personnalité
	Minor     int       `json:"minor"`     // Ajustement mineur
	Patch     int       `json:"patch"`     // Correction micro
	Timestamp time.Time `json:"timestamp"`
}

// NewIdentityVersion crée une nouvelle version
func NewIdentityVersion(major, minor, patch int) IdentityVersion {
	return IdentityVersion{
		Major:     major,
		Minor:     minor,
		Patch:     patch,
		Timestamp: time.Now(),
	}
}

// String retourne la version au format semver
func (iv IdentityVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", iv.Major, iv.Minor, iv.Patch)
}

// IdentitySource représente la source d'une observation identitaire
type IdentitySource struct {
	Type        SourceType `json:"type"`         // Type de source
	Content     string     `json:"content"`      // Contenu source
	Timestamp   time.Time  `json:"timestamp"`
	Context     string     `json:"context"`      // Contexte de l'interaction
	UserID      string     `json:"user_id"`      // ID de l'utilisateur
	SessionID   string     `json:"session_id"`   // ID de session
}

// SourceType énumère les types de sources possibles
type SourceType string

const (
	SourceConversation  SourceType = "conversation"   // Dialogue utilisateur-agent
	SourceFeedback      SourceType = "feedback"       // Feedback explicite de l'utilisateur
	SourceSelfReflection SourceType = "self_reflection" // Auto-réflexion de l'agent
	SourceObservation   SourceType = "observation"    // Observation tierce
	SourceMemoryMira    SourceType = "mira_memory"    // Mémoire factuelle de MIRA
)

// ExtractedTrait représente un trait brut extrait d'une source
type ExtractedTrait struct {
	Name        string        `json:"name"`
	Category    string        `json:"category"`
	Intensity   float64       `json:"intensity"`
	Evidence    string        `json:"evidence"`     // Texte justificatif
	Confidence  float64       `json:"confidence"`
	Source      IdentitySource `json:"source"`
}

// IdentityContextPrompt représente le prompt d'identité généré pour injection
type IdentityContextPrompt struct {
	Content        string    `json:"content"`         // Le prompt textuel
	TokenEstimate  int       `json:"token_estimate"`  // Estimation du nombre de tokens
	Priority       int       `json:"priority"`        // Priorité dans le context window
	GeneratedAt    time.Time `json:"generated_at"`
	SnapshotVersion int      `json:"snapshot_version"`
}

// IdentityDriftReport représente un rapport de dérive identitaire
type IdentityDriftReport struct {
	Timestamp        time.Time              `json:"timestamp"`
	PreviousVersion  int                    `json:"previous_version"`
	CurrentVersion   int                    `json:"current_version"`
	DriftScore       float64                `json:"drift_score"`       // 0-1, score global de dérive
	DriftDimensions  []DimensionDrift       `json:"drift_dimensions"`  // Dérive par dimension
	IsSignificant    bool                   `json:"is_significant"`    // Dérive significative ?
	Recommendations  []string               `json:"recommendations"`   // Actions recommandées
}

// DimensionDrift représente la dérive sur une dimension spécifique
type DimensionDrift struct {
	Dimension    string  `json:"dimension"`    // ex: "voice", "personality", "values"
	PreviousValue float64 `json:"previous_value"`
	CurrentValue  float64 `json:"current_value"`
	Change       float64 `json:"change"`       // Delta absolu
	IsSignificant bool    `json:"is_significant"`
}

// ModelSwapContext représente le contexte lors d'un changement de modèle
// C'est le moment critique où l'identité risque d'être perdue
type ModelSwapContext struct {
	AgentID            string    `json:"agent_id"`
	PreviousModel      string    `json:"previous_model"`
	NewModel           string    `json:"new_model"`
	Timestamp          time.Time `json:"timestamp"`
	IdentityPreserved  bool      `json:"identity_preserved"`
	IdentityDrift      float64   `json:"identity_drift"`       // Dérive mesurée post-swap
	ReinforcementApplied bool    `json:"reinforcement_applied"` // Renforcement appliqué ?
}

// SoulQuery représente une requête pour récupérer l'identité
type SoulQuery struct {
	AgentID         string    `json:"agent_id"`
	Context         string    `json:"context"`           // Contexte de la conversation actuelle
	BudgetTokens    int       `json:"budget_tokens"`     // Budget de tokens pour le prompt d'identité
	PrioritizeRecent bool     `json:"prioritize_recent"` // Prioriser les observations récentes
	IncludeTraits   []string  `json:"include_traits,omitempty"` // Traits spécifiques à inclure
	ExcludeTraits   []string  `json:"exclude_traits,omitempty"` // Traits à exclure
}

// SoulCaptureRequest représente une demande de capture d'identité
type SoulCaptureRequest struct {
	AgentID           string                 `json:"agent_id"`
	Conversation      string                 `json:"conversation"`     // Texte de la conversation
	AgentResponses    []string               `json:"agent_responses"`  // Réponses spécifiques de l'agent
	UserFeedback      map[string]string      `json:"user_feedback"`    // Feedback utilisateur (optionnel)
	ModelID           string                 `json:"model_id"`         // Identifiant du modèle
	SessionID         string                 `json:"session_id"`
	Timestamp         time.Time              `json:"timestamp"`
	BehavioralMetrics map[string]interface{} `json:"behavioral_metrics,omitempty"`
}

// MergeStrategy définit comment fusionner deux identités
type MergeStrategy string

const (
	MergePreserveDominant MergeStrategy = "preserve_dominant" // Garde l'identité dominante
	MergeWeightedAverage  MergeStrategy = "weighted_average"  // Moyenne pondérée
	MergeLatestWins       MergeStrategy = "latest_wins"       // La plus récente gagne
	MergeSynthesize       MergeStrategy = "synthesize"        // Synthèse intelligente
)
