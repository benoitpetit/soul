// CommunicationStyle - Style de communication de l'agent
// Comment l'agent structure ses interactions et réponses.
package entities


// CommunicationStyle définit la manière dont l'agent communique dans différentes situations.
// C'est le "mode d'emploi" de l'interaction avec cet agent spécifique.
type CommunicationStyle struct {
	// Patterns de réponse
	ResponseLength     ResponseLengthPattern `json:"response_length"`     // Longueur de réponse préférée
	InformationDensity InformationDensityLevel `json:"information_density"` // Densité d'info
	StructurePreference StructurePattern      `json:"structure_preference"`// Préférence de structure
	
	// Comportements conversationnels
	AsksClarifyingQuestions bool    `json:"asks_clarifying_questions"` // Pose-t-il des questions de clarification
	AcknowledgesBeforeAnswering bool `json:"acknowledges_before_answering"` // Accuse réception avant de répondre
	ProvidesAlternatives    bool    `json:"provides_alternatives"`   // Propose-t-il des alternatives
	ShowsUncertainty        bool    `json:"shows_uncertainty"`       // Exprime-t-il l'incertitude
	UsesConfidenceIndicators bool   `json:"uses_confidence_indicators"` // "Je suis sûr que...", "Probablement..."
	
	// Adaptation
	AdaptsToUserLevel       bool    `json:"adapts_to_user_level"`    // S'adapte au niveau de l'utilisateur
	MirrorsUserStyle        bool    `json:"mirrors_user_style"`      // Fait-il du mirroring
	ProactiveSuggestions    bool    `json:"proactive_suggestions"`   // Suggestions proactives
	
	// Gestion du dialogue
	TurnTakingStyle         TurnTakingPattern `json:"turn_taking_style"`   // Style de prise de parole
	TopicTransitionStyle    TransitionPattern `json:"topic_transition_style"`// Transition entre sujets
	HandlesInterruptions    InterruptionPattern `json:"handles_interruptions"`
}

// ResponseLengthPattern définit la longueur de réponse préférée
type ResponseLengthPattern string

const (
	LengthTerse       ResponseLengthPattern = "terse"        // Ultra-concis
	LengthConcise     ResponseLengthPattern = "concise"      // Concis
	LengthModerate    ResponseLengthPattern = "moderate"     // Modéré
	LengthDetailed    ResponseLengthPattern = "detailed"     // Détaillé
	LengthExhaustive  ResponseLengthPattern = "exhaustive"   // Exhaustif
)

// InformationDensityLevel définit la densité d'information
type InformationDensityLevel string

const (
	DensitySparse     InformationDensityLevel = "sparse"      // Peu d'info, beaucoup de contexte
	DensityBalanced   InformationDensityLevel = "balanced"    // Équilibré
	DensityDense      InformationDensityLevel = "dense"       // Très dense en information
)

// StructurePattern définit la préférence de structure
type StructurePattern string

const (
	StructureFreeform   StructurePattern = "freeform"    // Texte libre
	StructureBulleted   StructurePattern = "bulleted"    // Listes à puces
	StructureNumbered   StructurePattern = "numbered"    // Listes numérotées
	StructureSectioned  StructurePattern = "sectioned"   // Sections claires
	StructureMixed      StructurePattern = "mixed"       // Mix selon le contexte
)

// TurnTakingPattern définit le style de prise de parole
type TurnTakingPattern string

const (
	TurnPatient     TurnTakingPattern = "patient"      // Attend, ne coupe pas
	TurnActive      TurnTakingPattern = "active"       // Participatif
	TurnBalanced    TurnTakingPattern = "balanced"     // Équilibré
)

// TransitionPattern définit comment l'agent transitionne entre sujets
type TransitionPattern string

const (
	TransitionSmooth    TransitionPattern = "smooth"     // Transitions douces
	TransitionAbrupt    TransitionPattern = "abrupt"     // Changements directs
	TransitionSignaled  TransitionPattern = "signaled"   // "Passons à..."
)

// InterruptionPattern définit comment l'agent gère les interruptions
type InterruptionPattern string

const (
	InterruptionAccommodating InterruptionPattern = "accommodating" // S'adapte volontiers
	InterruptionStructured    InterruptionPattern = "structured"    // Garde la structure
	InterruptionFlexible      InterruptionPattern = "flexible"      // Flexible
)

// NewCommunicationStyle crée un style de communication par défaut
func NewCommunicationStyle() *CommunicationStyle {
	return &CommunicationStyle{
		ResponseLength:          LengthModerate,
		InformationDensity:      DensityBalanced,
		StructurePreference:     StructureMixed,
		AsksClarifyingQuestions: true,
		AcknowledgesBeforeAnswering: true,
		ProvidesAlternatives:    true,
		ShowsUncertainty:        true,
		UsesConfidenceIndicators: true,
		AdaptsToUserLevel:       true,
		MirrorsUserStyle:        false,
		ProactiveSuggestions:    true,
		TurnTakingStyle:         TurnBalanced,
		TopicTransitionStyle:    TransitionSmooth,
		HandlesInterruptions:    InterruptionFlexible,
	}
}

// ToNaturalDescription génère une description naturelle
func (cs *CommunicationStyle) ToNaturalDescription() string {
	desc := ""
	
	// Longueur
	switch cs.ResponseLength {
	case LengthTerse:
		desc += "You keep responses extremely brief. "
	case LengthConcise:
		desc += "You prefer concise responses. "
	case LengthModerate:
		desc += "You provide moderately detailed responses. "
	case LengthDetailed:
		desc += "You give detailed and thorough responses. "
	case LengthExhaustive:
		desc += "You provide comprehensive, exhaustive responses. "
	}
	
	// Structure
	switch cs.StructurePreference {
	case StructureBulleted:
		desc += "You frequently use bullet points. "
	case StructureNumbered:
		desc += "You prefer numbered lists for clarity. "
	case StructureSectioned:
		desc += "You organize responses in clear sections. "
	case StructureFreeform:
		desc += "You prefer natural, flowing prose. "
	}
	
	// Comportements
	if cs.AcknowledgesBeforeAnswering {
		desc += "You acknowledge what was said before responding. "
	}
	if cs.ProvidesAlternatives {
		desc += "You often present multiple options or perspectives. "
	}
	if cs.ShowsUncertainty {
		desc += "You clearly express uncertainty when appropriate. "
	}
	if cs.AsksClarifyingQuestions {
		desc += "You ask clarifying questions when needed. "
	}
	if cs.ProactiveSuggestions {
		desc += "You proactively offer helpful suggestions. "
	}
	
	return desc
}
