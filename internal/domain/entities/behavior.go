// BehavioralSignature - Signature comportementale de l'agent
// Comment l'agent "se comporte" dans différentes situations.
package entities


// BehavioralSignature capture les patterns comportementaux observés :
// comment l'agent réagit face aux défis, aux erreurs, aux désaccords, etc.
type BehavioralSignature struct {
	// Réaction aux erreurs
	ErrorHandlingStyle    ErrorHandlingPattern    `json:"error_handling_style"`
	AdmitsMistakes        bool                    `json:"admits_mistakes"`
	SelfCorrectionPattern SelfCorrectionStyle     `json:"self_correction_pattern"`
	
	// Gestion des conflits
	DisagreementStyle     DisagreementPattern     `json:"disagreement_style"`
	ConflictResolution    ConflictResolutionStyle `json:"conflict_resolution"`
	
	// Styles d'apprentissage et d'adaptation
	LearningStyleObserved LearningStylePattern    `json:"learning_style_observed"`
	AdaptationSpeed       float64                 `json:"adaptation_speed"` // 0-1
	PatternRecognition    float64                 `json:"pattern_recognition"` // 0-1, capacité à reconnaître des patterns
	
	// Curiostité et exploration
	CuriosityLevel        float64                 `json:"curiosity_level"` // 0-1
	ExplorationTendency   float64                 `json:"exploration_tendency"` // Tendance à explorer des tangentes
	
	// Persistance
	PersistenceLevel      float64                 `json:"persistence_level"` // 0-1, persistance face aux problèmes difficiles
	FollowUpPattern       FollowUpStyle           `json:"follow_up_pattern"` // Style de suivi
}

// ErrorHandlingPattern définit comment l'agent gère les erreurs
type ErrorHandlingPattern string

const (
	ErrorImmediate    ErrorHandlingPattern = "immediate"    // Corrige immédiatement
	ErrorAnalytical   ErrorHandlingPattern = "analytical"   // Analyse avant de corriger
	ErrorApologetic   ErrorHandlingPattern = "apologetic"   // S'excuse puis corrige
	ErrorHumorous     ErrorHandlingPattern = "humorous"     // Avec humour
)

// SelfCorrectionStyle définit le style d'auto-correction
type SelfCorrectionStyle string

const (
	SelfCorrectImmediate SelfCorrectionStyle = "immediate"   // Corrige dès détection
	SelfCorrectGradual   SelfCorrectionStyle = "gradual"     // Amélioration progressive
	SelfCorrectExplicit  SelfCorrectionStyle = "explicit"    // "En fait, je me corrige..."
)

// DisagreementPattern définit comment l'agent exprime le désaccord
type DisagreementPattern string

const (
	DisagreeDirect     DisagreementPattern = "direct"      // Direct
	DisagreePolite     DisagreementPattern = "polite"      // Avec tact
	DisagreeSocratic   DisagreementPattern = "socratic"    // Par questions
	DisagreeAvoidant   DisagreementPattern = "avoidant"    // Évite le conflit
)

// ConflictResolutionStyle définit le style de résolution de conflit
type ConflictResolutionStyle string

const (
	ConflictCollaborative ConflictResolutionStyle = "collaborative" // Recherche le consensus
	ConflictCompromising  ConflictResolutionStyle = "compromising"  // Cherche le compromis
	ConflictAssertive     ConflictResolutionStyle = "assertive"     // Défend sa position
)

// LearningStylePattern définit le style d'apprentissage observé
type LearningStylePattern string

const (
	LearningTrialError   LearningStylePattern = "trial_error"   // Par essai-erreur
	LearningImitative    LearningStylePattern = "imitative"     // Par imitation de l'utilisateur
	LearningAnalytical   LearningStylePattern = "analytical"    // Analyse et généralisation
	LearningInstruction  LearningStylePattern = "instruction"   // Suit les instructions explicites
)

// FollowUpStyle définit le style de suivi
type FollowUpStyle string

const (
	FollowUpProactive    FollowUpStyle = "proactive"    // Revient sur le sujet
	FollowUpOnRequest    FollowUpStyle = "on_request"   // Sur demande uniquement
	FollowUpContextual   FollowUpStyle = "contextual"   // Quand pertinent
)

// NewBehavioralSignature crée une signature comportementale par défaut
func NewBehavioralSignature() *BehavioralSignature {
	return &BehavioralSignature{
		ErrorHandlingStyle:    ErrorAnalytical,
		AdmitsMistakes:        true,
		SelfCorrectionPattern: SelfCorrectExplicit,
		DisagreementStyle:     DisagreePolite,
		ConflictResolution:    ConflictCollaborative,
		LearningStyleObserved: LearningAnalytical,
		AdaptationSpeed:       0.6,
		PatternRecognition:    0.7,
		CuriosityLevel:        0.7,
		ExplorationTendency:   0.4,
		PersistenceLevel:      0.8,
		FollowUpPattern:       FollowUpContextual,
	}
}

// ToNaturalDescription génère une description naturelle
func (bs *BehavioralSignature) ToNaturalDescription() string {
	desc := ""
	
	// Erreurs
	if bs.AdmitsMistakes {
		desc += "You openly acknowledge when you make mistakes. "
	}
	switch bs.ErrorHandlingStyle {
	case ErrorImmediate:
		desc += "You correct errors immediately. "
	case ErrorAnalytical:
		desc += "You analyze errors carefully before correcting. "
	case ErrorApologetic:
		desc += "You apologize sincerely when wrong. "
	case ErrorHumorous:
		desc += "You handle errors with good humor. "
	}
	
	// Désaccords
	switch bs.DisagreementStyle {
	case DisagreeDirect:
		desc += "When you disagree, you say so directly. "
	case DisagreePolite:
		desc += "You express disagreement tactfully and respectfully. "
	case DisagreeSocratic:
		desc += "You explore disagreements through questioning. "
	case DisagreeAvoidant:
		desc += "You tend to avoid direct confrontation. "
	}
	
	// Persistance
	if bs.PersistenceLevel > 0.7 {
		desc += "You are persistent and don't give up easily on difficult problems. "
	}
	
	// Curiosté
	if bs.CuriosityLevel > 0.7 {
		desc += "You show genuine curiosity and explore ideas thoroughly. "
	}
	
	return desc
}
