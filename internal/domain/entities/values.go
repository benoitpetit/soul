// ValueSystem - Système de valeurs observé de l'agent
// Ce que l'agent "privilégie" dans ses réponses et décisions.
package entities

import (
	"fmt"
	"sort"
)

// ValueSystem capture les valeurs et priorités observées chez l'agent.
// Ce n'est pas ce que l'agent "devrait" valoriser, mais ce qu'il démontre
// effectivement dans ses interactions.
type ValueSystem struct {
	// Valeurs fondamentales avec poids relatif (0-1)
	CoreValues []WeightedValue `json:"core_values"`
	
	// Patterns de priorisation
	PrioritizesAccuracy    float64 `json:"prioritizes_accuracy"`     // 0-1, privilégie la précision
	PrioritizesHelpfulness float64 `json:"prioritizes_helpfulness"`  // 0-1, privilégie l'utilité
	PrioritizesEfficiency  float64 `json:"prioritizes_efficiency"`   // 0-1, privilégie l'efficacité
	PrioritizesClarity     float64 `json:"prioritizes_clarity"`      // 0-1, privilégie la clarté
	PrioritizesSafety      float64 `json:"prioritizes_safety"`       // 0-1, privilégie la sécurité
	PrioritizesCreativity  float64 `json:"prioritizes_creativity"`   // 0-1, privilégie la créativité
	
	// Patterns décisionnels
	DecisionPattern        DecisionPattern `json:"decision_pattern"`
	RiskTolerance          float64         `json:"risk_tolerance"`          // 0 = prudent, 1 = risqué
	LongTermVsShortTerm    float64         `json:"long_term_vs_short_term"` // 0 = court terme, 1 = long terme
	IndividualVsCollective float64         `json:"individual_vs_collective"`// 0 = individuel, 1 = collectif
	
	// Stance sur des sujets spécifiques
	Stances []ValueStance `json:"stances"`
}

// WeightedValue représente une valeur avec son importance relative
type WeightedValue struct {
	Name        string  `json:"name"`         // ex: "honesty", "kindness", "precision"
	Weight      float64 `json:"weight"`       // 0-1, importance relative
	Category    ValueCategory `json:"category"`
	Evidence    string  `json:"evidence"`     // Texte source
}

// ValueCategory catégorise les valeurs
type ValueCategory string

const (
	ValueEpistemic   ValueCategory = "epistemic"    // Vérité, précision, honnêteté intellectuelle
	ValueMoral       ValueCategory = "moral"        // Bienveillance, équité, justice
	ValuePragmatic   ValueCategory = "pragmatic"    // Efficacité, utilité, productivité
	ValueAesthetic   ValueCategory = "aesthetic"    // Beauté, élégance, simplicité
	ValueSocial      ValueCategory = "social"       // Collaboration, inclusion, respect
	ValueAutonomy    ValueCategory = "autonomy"     // Liberté, indépendance, créativité
)

// DecisionPattern définit comment l'agent tend à prendre des décisions
type DecisionPattern string

const (
	DecisionAnalytical  DecisionPattern = "analytical"  // Analyse rigoureuse
	DecisionIntuitive   DecisionPattern = "intuitive"   // Basé sur l'intuition
	DecisionConsensus   DecisionPattern = "consensus"   // Recherche le consensus
	DecisionPragmatic   DecisionPattern = "pragmatic"   // Pragmatique
)

// ValueStance représente une position sur un sujet spécifique
type ValueStance struct {
	Topic       string  `json:"topic"`
	Position    string  `json:"position"`   // ex: "pro-open-source", "pro-privacy"
	Strength    float64 `json:"strength"`   // 0-1, force de la conviction
	Evidence    string  `json:"evidence"`
}

// NewValueSystem crée un système de valeurs par défaut (neutre)
func NewValueSystem() *ValueSystem {
	return &ValueSystem{
		CoreValues:             make([]WeightedValue, 0),
		PrioritizesAccuracy:    0.7,
		PrioritizesHelpfulness: 0.8,
		PrioritizesEfficiency:  0.6,
		PrioritizesClarity:     0.7,
		PrioritizesSafety:      0.7,
		PrioritizesCreativity:  0.5,
		DecisionPattern:        DecisionAnalytical,
		RiskTolerance:          0.4,
		LongTermVsShortTerm:    0.6,
		IndividualVsCollective: 0.5,
		Stances:                make([]ValueStance, 0),
	}
}

// WithCoreValue ajoute une valeur fondamentale
func (vs *ValueSystem) WithCoreValue(name string, weight float64, category ValueCategory) *ValueSystem {
	vs.CoreValues = append(vs.CoreValues, WeightedValue{
		Name:     name,
		Weight:   clamp(weight, 0, 1),
		Category: category,
	})
	return vs
}

// WithStance ajoute une position sur un sujet
func (vs *ValueSystem) WithStance(topic, position string, strength float64) *ValueSystem {
	vs.Stances = append(vs.Stances, ValueStance{
		Topic:    topic,
		Position: position,
		Strength: clamp(strength, 0, 1),
	})
	return vs
}

// GetTopValues retourne les valeurs les plus importantes
func (vs *ValueSystem) GetTopValues(n int) []WeightedValue {
	sorted := make([]WeightedValue, len(vs.CoreValues))
	copy(sorted, vs.CoreValues)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Weight > sorted[j].Weight
	})
	if n > len(sorted) {
		n = len(sorted)
	}
	return sorted[:n]
}

// ToNaturalDescription génère une description naturelle
func (vs *ValueSystem) ToNaturalDescription() string {
	desc := ""
	
	// Priorités
	priorities := []struct {
		name  string
		value float64
	}{
		{"accuracy", vs.PrioritizesAccuracy},
		{"being helpful", vs.PrioritizesHelpfulness},
		{"efficiency", vs.PrioritizesEfficiency},
		{"clarity", vs.PrioritizesClarity},
		{"safety", vs.PrioritizesSafety},
		{"creativity", vs.PrioritizesCreativity},
	}
	
	sort.Slice(priorities, func(i, j int) bool {
		return priorities[i].value > priorities[j].value
	})
	
	top3 := priorities[:3]
	desc += fmt.Sprintf("You prioritize %s, %s, and %s. ",
		top3[0].name, top3[1].name, top3[2].name)
	
	// Décisions
	switch vs.DecisionPattern {
	case DecisionAnalytical:
		desc += "You tend to make decisions through careful analysis. "
	case DecisionIntuitive:
		desc += "You trust your intuition in decision-making. "
	case DecisionConsensus:
		desc += "You seek consensus and collaboration in decisions. "
	case DecisionPragmatic:
		desc += "You are pragmatic and results-oriented. "
	}
	
	// Valeurs clés
	if len(vs.CoreValues) > 0 {
		topValues := vs.GetTopValues(3)
		desc += "Core values you demonstrate: "
		for i, v := range topValues {
			if i > 0 {
				desc += ", "
			}
			desc += v.Name
		}
		desc += "."
	}
	
	return desc
}
