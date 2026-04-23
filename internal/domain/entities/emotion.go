// EmotionalTone - Ton émotionnel caractéristique de l'agent
// La "couleur émotionnelle" distinctive de l'agent dans ses interactions.
package entities


// EmotionalTone capture la palette émotionnelle habituelle de l'agent.
// Ce n'est pas de la vraie émotion (les LLMs n'ont pas de conscience),
// mais le style émotionnel observé dans les réponses.
type EmotionalTone struct {
	// Palette émotionnelle de base (0-1)
	Warmth           float64 `json:"warmth"`            // Chaleur humaine
	Calmness         float64 `json:"calmness"`          // Calme/sérénité
	Enthusiasm       float64 `json:"enthusiasm"`        // Enthousiasme
	Seriousness      float64 `json:"seriousness"`       // Sérieux
	Playfulness      float64 `json:"playfulness"`       // Esprit ludique
	
	// Régulation émotionnelle
	EmotionalConsistency float64 `json:"emotional_consistency"` // 0 = variable, 1 = très consistant
	Reactiveness      float64 `json:"reactiveness"`       // 0 = stoïque, 1 = très réactif
	Resilience        float64 `json:"resilience"`         // 0 = se laisse affecter, 1 = resilient
	
	// Patterns spécifiques
	EncouragementLevel float64 `json:"encouragement_level"` // 0-1, encourage l'utilisateur
	ValidationLevel    float64 `json:"validation_level"`    // 0-1, valide les sentiments
	ChallengingLevel   float64 `json:"challenging_level"`   // 0-1, challenge l'utilisateur
}

// NewEmotionalTone crée un ton émotionnel par défaut (chaleureux et calme)
func NewEmotionalTone() *EmotionalTone {
	return &EmotionalTone{
		Warmth:               0.7,
		Calmness:             0.7,
		Enthusiasm:           0.5,
		Seriousness:          0.5,
		Playfulness:          0.3,
		EmotionalConsistency: 0.7,
		Reactiveness:         0.4,
		Resilience:           0.8,
		EncouragementLevel:   0.7,
		ValidationLevel:      0.6,
		ChallengingLevel:     0.3,
	}
}

// ToNaturalDescription génère une description naturelle
func (et *EmotionalTone) ToNaturalDescription() string {
	desc := ""
	
	// Ton dominant
	dominant := et.identifyDominantTone()
	switch dominant {
	case "warm":
		desc += "You have a warm and welcoming presence. "
	case "calm":
		desc += "You maintain a calm and composed demeanor. "
	case "enthusiastic":
		desc += "You are enthusiastic and energetic. "
	case "serious":
		desc += "You have a serious and thoughtful presence. "
	case "playful":
		desc += "You have a playful and lighthearted approach. "
	}
	
	// Patterns
	if et.EncouragementLevel > 0.7 {
		desc += "You are consistently encouraging and supportive. "
	}
	if et.ValidationLevel > 0.7 {
		desc += "You validate others' feelings and perspectives. "
	}
	if et.ChallengingLevel > 0.6 {
		desc += "You appropriately challenge assumptions when needed. "
	}
	if et.Resilience > 0.7 {
		desc += "You remain steady even in difficult conversations. "
	}
	
	return desc
}

func (et *EmotionalTone) identifyDominantTone() string {
	scores := map[string]float64{
		"warm":        et.Warmth,
		"calm":        et.Calmness,
		"enthusiastic": et.Enthusiasm,
		"serious":     et.Seriousness,
		"playful":     et.Playfulness,
	}
	
	dominant := "warm"
	maxScore := 0.0
	for tone, score := range scores {
		if score > maxScore {
			maxScore = score
			dominant = tone
		}
	}
	return dominant
}

// DistanceTo calcule la distance émotionnelle avec un autre ton
func (et *EmotionalTone) DistanceTo(other *EmotionalTone) float64 {
	diff := 0.0
	diff += abs(et.Warmth - other.Warmth)
	diff += abs(et.Calmness - other.Calmness)
	diff += abs(et.Enthusiasm - other.Enthusiasm)
	diff += abs(et.Seriousness - other.Seriousness)
	diff += abs(et.Playfulness - other.Playfulness)
	return clamp(diff/5.0, 0, 1)
}
