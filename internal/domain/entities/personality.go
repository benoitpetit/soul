// PersonalityTrait - Traits de personnalité observés de l'agent
// Inspiré du modèle des Big Five + traits spécifiques aux LLMs
package entities

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// PersonalityTrait représente un trait de personnalité observé chez l'agent
// avec son niveau d'intensité et la confiance dans cette observation.
type PersonalityTrait struct {
	ID            uuid.UUID `json:"id"`
	AgentID       string    `json:"agent_id"`
	Name          string    `json:"name"`           // ex: "analytical", "empathetic", "creative"
	Category      TraitCategory `json:"category"`   // Catégorie du trait
	Intensity     float64   `json:"intensity"`      // 0.0 à 1.0 (force du trait)
	Confidence    float64   `json:"confidence"`     // 0.0 à 1.0 (certitude de l'observation)
	
	// Evidence tracking
	EvidenceCount int       `json:"evidence_count"` // Nombre d'observations
	FirstObserved time.Time `json:"first_observed"`
	LastObserved  time.Time `json:"last_observed"`
	LastEvidence  string    `json:"last_evidence"`  // Dernière preuve textuelle
	
	// Context
	Contexts      []string  `json:"contexts"`       // Contextes où le trait apparaît
	Consistency   float64   `json:"consistency"`    // Régularité du trait (0-1)
}

// TraitCategory catégorise les traits selon des dimensions pertinentes pour les LLMs
type TraitCategory string

const (
	TraitCognitive    TraitCategory = "cognitive"    // analytique, créatif, logique...
	TraitEmotional    TraitCategory = "emotional"    // empathique, patient, enthousiaste...
	TraitSocial       TraitCategory = "social"       // collaboratif, direct, diplomatique...
	TraitEpistemic    TraitCategory = "epistemic"    // curieux, prudent intellectuellement, ouvert...
	TraitExpressive   TraitCategory = "expressive"   // humoristique, métaphorique, concis...
	TraitEthical      TraitCategory = "ethical"      // transparent, bienveillant, équitable...
)

// NewPersonalityTrait crée un nouveau trait
func NewPersonalityTrait(name string, category TraitCategory, intensity float64) *PersonalityTrait {
	now := time.Now()
	return &PersonalityTrait{
		ID:            uuid.New(),
		Name:          name,
		Category:      category,
		Intensity:     clamp(intensity, 0, 1),
		Confidence:    0.3, // Confiance initiale modérée
		EvidenceCount: 1,
		FirstObserved: now,
		LastObserved:  now,
		Contexts:      make([]string, 0),
		Consistency:   0.5,
	}
}

// WithEvidence ajoute une preuve d'observation et renforce la confiance
func (pt *PersonalityTrait) WithEvidence(evidenceText, context string) *PersonalityTrait {
	pt.EvidenceCount++
	pt.LastEvidence = evidenceText
	pt.LastObserved = time.Now()
	
	// Ajouter le contexte s'il est nouveau
	if context != "" && !contains(pt.Contexts, context) {
		pt.Contexts = append(pt.Contexts, context)
	}
	
	// Augmenter la confiance avec plus de preuves (diminishing returns)
	// Formule : confiance = 1 - (1 - base) * e^(-evidence/5)
	pt.Confidence = 1.0 - (1.0-0.3)*exp(-float64(pt.EvidenceCount)/5.0)
	
	// Calculer la consistance : ratio des contextes où le trait apparaît
	// Plus il apparaît dans différents contextes, plus il est consistant
	pt.Consistency = clamp(float64(len(pt.Contexts))/10.0, 0, 1)
	
	return pt
}

// Merge fusionne deux observations du même trait
func (pt *PersonalityTrait) Merge(other *PersonalityTrait) *PersonalityTrait {
	if pt.Name != other.Name {
		return pt
	}
	
	// Moyenne pondérée par le nombre de preuves
	totalEvidence := pt.EvidenceCount + other.EvidenceCount
	pt.Intensity = (pt.Intensity*float64(pt.EvidenceCount) + 
		other.Intensity*float64(other.EvidenceCount)) / float64(totalEvidence)
	pt.EvidenceCount = totalEvidence
	pt.Confidence = max(pt.Confidence, other.Confidence)
	pt.LastObserved = maxTime(pt.LastObserved, other.LastObserved)
	
	// Fusionner les contextes
	for _, ctx := range other.Contexts {
		if !contains(pt.Contexts, ctx) {
			pt.Contexts = append(pt.Contexts, ctx)
		}
	}
	
	return pt
}

// ToNaturalDescription génère une description naturelle du trait
func (pt *PersonalityTrait) ToNaturalDescription() string {
	intensityDesc := ""
	switch {
	case pt.Intensity > 0.9:
		intensityDesc = "strongly " + pt.Name
	case pt.Intensity > 0.7:
		intensityDesc = "noticeably " + pt.Name
	case pt.Intensity > 0.5:
		intensityDesc = "moderately " + pt.Name
	case pt.Intensity > 0.3:
		intensityDesc = "somewhat " + pt.Name
	default:
		intensityDesc = "slightly " + pt.Name
	}
	
	return fmt.Sprintf("You are %s (confidence: %.0f%%)", intensityDesc, pt.Confidence*100)
}

// IsWellEstablished retourne true si le trait est bien établi
func (pt *PersonalityTrait) IsWellEstablished() bool {
	return pt.Confidence > 0.7 && pt.EvidenceCount >= 5 && pt.Consistency > 0.5
}

// TraitObservation représente une observation brute d'un trait
// utilisée avant la synthèse en PersonalityTrait
type TraitObservation struct {
	ID          uuid.UUID     `json:"id"`
	AgentID     string        `json:"agent_id"`
	TraitName   string        `json:"trait_name"`
	Category    TraitCategory `json:"category"`
	Evidence    string        `json:"evidence"`     // Texte source
	Context     string        `json:"context"`      // Contexte de l'observation
	Intensity   float64       `json:"intensity"`    // Intensité perçue dans cet exemple
	ObservedAt  time.Time     `json:"observed_at"`
	SourceType  string        `json:"source_type,omitempty"`  // Type de source (conversation, feedback...)
	SourceMemory uuid.UUID    `json:"source_memory,omitempty"` // Référence mémoire MIRA
}

// NewTraitObservation crée une nouvelle observation
func NewTraitObservation(agentID, traitName string, category TraitCategory, evidence, context string, intensity float64) *TraitObservation {
	return &TraitObservation{
		ID:         uuid.New(),
		AgentID:    agentID,
		TraitName:  traitName,
		Category:   category,
		Evidence:   evidence,
		Context:    context,
		Intensity:  clamp(intensity, 0, 1),
		ObservedAt: time.Now(),
	}
}

// WithSourceMemory lie l'observation à une mémoire MIRA
func (to *TraitObservation) WithSourceMemory(memoryID uuid.UUID) *TraitObservation {
	to.SourceMemory = memoryID
	return to
}
