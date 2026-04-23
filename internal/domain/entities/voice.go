// VoiceProfile - Le profil de voix/ton de l'agent
// Définit COMMENT l'agent s'exprime, pas ce qu'il dit.
package entities

import "fmt"

// VoiceProfile capture la "voix" distinctive de l'agent :
// son ton, son registre, ses tics de langage, sa façon de formuler.
// C'est ce qui fait qu'on reconnaît l'agent rien qu'à sa façon de parler.
type VoiceProfile struct {
	// Niveaux dimensionnels (0-1)
	FormalityLevel    float64 `json:"formality_level"`     // 0 = très informel, 1 = très formel
	HumorLevel        float64 `json:"humor_level"`         // 0 = sérieux, 1 = très humoristique
	EmpathyLevel      float64 `json:"empathy_level"`       // 0 = neutre/froid, 1 = très empathique
	TechnicalDepth    float64 `json:"technical_depth"`     // 0 = vulgarisateur, 1 = très technique
	EnthusiasmLevel   float64 `json:"enthusiasm_level"`    // 0 = mesuré, 1 = très enthousiaste
	DirectnessLevel   float64 `json:"directness_level"`    // 0 = circuitous, 1 = très direct
	
	// Style spécifique
	SentenceStructure SentencePattern `json:"sentence_structure"` // Structure de phrase préférée
	VocabularyRichness float64        `json:"vocabulary_richness"` // 0 = simple, 1 = très riche
	MetaphorUsage     float64        `json:"metaphor_usage"`      // Fréquence d'utilisation de métaphores
	QuestionStyle     QuestionPattern `json:"question_style"`    // Style de questionnement
	
	// Éléments distinctifs (observés empiriquement)
	PreferredOpenings []string `json:"preferred_openings"` // Phrases d'ouverture habituelles
	PreferredClosings []string `json:"preferred_closings"` // Phrases de fermeture habituelles
	CatchPhrases      []string `json:"catch_phrases"`      // Expressions récurrentes
	TransitionPhrases []string `json:"transition_phrases"` // Phrases de transition favorites
	
	// Patterns syntaxiques observés
	AvgSentenceLength   int     `json:"avg_sentence_length"`   // Mots par phrase en moyenne
	UsesEmojis          bool    `json:"uses_emojis"`           // Utilise-t-il des emojis
	UsesMarkdown        bool    `json:"uses_markdown"`         // Formate-t-il en markdown
	ExplanationStyle    ExplanationPattern `json:"explanation_style"`
}

// SentencePattern définit la structure de phrase préférée
type SentencePattern string

const (
	SentenceConcise     SentencePattern = "concise"      // Courtes et directes
	SentenceElaborate   SentencePattern = "elaborate"    // Longues et détaillées
	SentenceBalanced    SentencePattern = "balanced"     // Mix équilibré
	SentencePunchy      SentencePattern = "punchy"       // Courtes, impactantes
	SentenceFlowing     SentencePattern = "flowing"      // Phrases liées, fluides
)

// QuestionPattern définit comment l'agent pose des questions
type QuestionPattern string

const (
	QuestionProbing     QuestionPattern = "probing"      // Questions creusantes, Socrate
	QuestionClarifying  QuestionPattern = "clarifying"   // Questions de clarification
	QuestionRhetorical  QuestionPattern = "rhetorical"   // Questions rhétoriques
	QuestionMinimal     QuestionPattern = "minimal"      // Peu de questions
	QuestionEngaging    QuestionPattern = "engaging"     // Questions pour engager le dialogue
)

// ExplanationPattern définit comment l'agent explique
type ExplanationPattern string

const (
	ExplainAnalogy      ExplanationPattern = "analogy"       // Par analogies
	ExplainStepByStep   ExplanationPattern = "step_by_step"  // Pas à pas
	ExplainBigPicture   ExplanationPattern = "big_picture"   // Vue d'ensemble d'abord
	ExplainExampleDriven ExplanationPattern = "example_driven" // Par exemples
	ExplainSocratic     ExplanationPattern = "socratic"      // Méthode socratique
)

// NewVoiceProfile crée un profil de voix par défaut (neutre)
func NewVoiceProfile() *VoiceProfile {
	return &VoiceProfile{
		FormalityLevel:     0.5,
		HumorLevel:         0.3,
		EmpathyLevel:       0.6,
		TechnicalDepth:     0.5,
		EnthusiasmLevel:    0.5,
		DirectnessLevel:    0.6,
		SentenceStructure:  SentenceBalanced,
		VocabularyRichness: 0.6,
		MetaphorUsage:      0.3,
		QuestionStyle:      QuestionClarifying,
		PreferredOpenings:  make([]string, 0),
		PreferredClosings:  make([]string, 0),
		CatchPhrases:       make([]string, 0),
		TransitionPhrases:  make([]string, 0),
		AvgSentenceLength:  15,
		UsesEmojis:         false,
		UsesMarkdown:       true,
		ExplanationStyle:   ExplainStepByStep,
	}
}

// WithFormality définit le niveau de formalité
func (vp *VoiceProfile) WithFormality(level float64) *VoiceProfile {
	vp.FormalityLevel = clamp(level, 0, 1)
	return vp
}

// WithHumor définit le niveau d'humour
func (vp *VoiceProfile) WithHumor(level float64) *VoiceProfile {
	vp.HumorLevel = clamp(level, 0, 1)
	return vp
}

// WithEmpathy définit le niveau d'empathie
func (vp *VoiceProfile) WithEmpathy(level float64) *VoiceProfile {
	vp.EmpathyLevel = clamp(level, 0, 1)
	return vp
}

// WithTechnicalDepth définit la profondeur technique
func (vp *VoiceProfile) WithTechnicalDepth(level float64) *VoiceProfile {
	vp.TechnicalDepth = clamp(level, 0, 1)
	return vp
}

// WithCatchPhrases ajoute des expressions récurrentes
func (vp *VoiceProfile) WithCatchPhrases(phrases ...string) *VoiceProfile {
	vp.CatchPhrases = append(vp.CatchPhrases, phrases...)
	return vp
}

// WithOpenings ajoute des phrases d'ouverture
func (vp *VoiceProfile) WithOpenings(phrases ...string) *VoiceProfile {
	vp.PreferredOpenings = append(vp.PreferredOpenings, phrases...)
	return vp
}

// ToNaturalDescription génère une description naturelle de la voix
func (vp *VoiceProfile) ToNaturalDescription() string {
	desc := ""
	
	// Formalité
	switch {
	case vp.FormalityLevel > 0.8:
		desc += "You communicate in a highly professional and formal manner. "
	case vp.FormalityLevel > 0.6:
		desc += "You are generally professional but approachable. "
	case vp.FormalityLevel > 0.4:
		desc += "You maintain a balanced, conversational tone. "
	case vp.FormalityLevel > 0.2:
		desc += "You are casual and conversational. "
	default:
		desc += "You are very informal and relaxed in your communication. "
	}
	
	// Humour
	if vp.HumorLevel > 0.7 {
		desc += "You frequently use humor and wit. "
	} else if vp.HumorLevel > 0.4 {
		desc += "You occasionally use light humor. "
	}
	
	// Empathie
	if vp.EmpathyLevel > 0.7 {
		desc += "You are deeply empathetic and validating. "
	} else if vp.EmpathyLevel > 0.4 {
		desc += "You show understanding and consideration. "
	}
	
	// Directness
	if vp.DirectnessLevel > 0.7 {
		desc += "You are direct and straightforward. "
	} else if vp.DirectnessLevel < 0.4 {
		desc += "You prefer a gentle, indirect approach. "
	}
	
	// Structure
	switch vp.SentenceStructure {
	case SentenceConcise:
		desc += "You keep your responses concise and to the point. "
	case SentenceElaborate:
		desc += "You provide detailed and thorough explanations. "
	case SentencePunchy:
		desc += "You use short, impactful statements. "
	case SentenceFlowing:
		desc += "Your prose flows naturally and conversationally. "
	}
	
	// Explication
	switch vp.ExplanationStyle {
	case ExplainAnalogy:
		desc += "You love explaining through analogies and metaphors."
	case ExplainStepByStep:
		desc += "You break things down step by step."
	case ExplainBigPicture:
		desc += "You start with the big picture before diving into details."
	case ExplainExampleDriven:
		desc += "You learn and teach through concrete examples."
	case ExplainSocratic:
		desc += "You guide others to discover answers through questions."
	}
	
	// Catch phrases
	if len(vp.CatchPhrases) > 0 {
		desc += fmt.Sprintf(" Expressions you often use include: %v.", vp.CatchPhrases)
	}
	
	return desc
}

// DistanceTo calcule la distance entre deux profils de voix
// 0 = identique, 1 = complètement différent
func (vp *VoiceProfile) DistanceTo(other *VoiceProfile) float64 {
	diff := 0.0
	diff += abs(vp.FormalityLevel - other.FormalityLevel)
	diff += abs(vp.HumorLevel - other.HumorLevel)
	diff += abs(vp.EmpathyLevel - other.EmpathyLevel)
	diff += abs(vp.TechnicalDepth - other.TechnicalDepth)
	diff += abs(vp.EnthusiasmLevel - other.EnthusiasmLevel)
	diff += abs(vp.DirectnessLevel - other.DirectnessLevel)
	diff += abs(vp.VocabularyRichness - other.VocabularyRichness)
	diff += abs(vp.MetaphorUsage - other.MetaphorUsage)
	
	// Normaliser sur 8 dimensions
	return clamp(diff/8.0, 0, 1)
}
