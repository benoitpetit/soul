// Package extraction implémente les services d'extraction d'identité
// Transforme les conversations en traits identitaires observables.
package extraction

import (
	"context"
	"strings"
	"time"

	"github.com/benoitpetit/soul/internal/domain/entities"
	"github.com/benoitpetit/soul/internal/domain/valueobjects"
	"github.com/benoitpetit/soul/internal/usecases/ports"
)

// SoulExtractorService implémente ports.IdentityExtractor
// Utilise des heuristiques et pattern matching pour extraire l'identité.
// Peut être enrichi par un LLM pour une extraction plus sophistiquée.
type SoulExtractorService struct {
	heuristicRules *HeuristicRules
}

// HeuristicRules définit les règles heuristiques pour l'extraction
type HeuristicRules struct {
	// Patterns pour détecter les traits
	TraitPatterns map[entities.TraitCategory][]TraitPattern
	
	// Patterns pour le style de voix
	VoicePatterns VoicePatterns
	
	// Patterns pour le style de communication
	CommunicationPatterns CommunicationPatterns
}

// TraitPattern définit un pattern pour détecter un trait
type TraitPattern struct {
	TraitName    string   `json:"trait_name"`
	Keywords     []string `json:"keywords"`
	Intensity    float64  `json:"intensity"`
	Confidence   float64  `json:"confidence"`
}

// VoicePatterns définit les patterns pour le profil de voix
type VoicePatterns struct {
	FormalIndicators    []string `json:"formal_indicators"`
	InformalIndicators  []string `json:"informal_indicators"`
	HumorIndicators     []string `json:"humor_indicators"`
	EmpathyIndicators   []string `json:"empathy_indicators"`
	TechnicalIndicators []string `json:"technical_indicators"`
}

// CommunicationPatterns définit les patterns pour le style de communication
type CommunicationPatterns struct {
	QuestionPatterns    []string `json:"question_patterns"`
	AcknowledgmentPatterns []string `json:"acknowledgment_patterns"`
	AlternativePatterns []string `json:"alternative_patterns"`
}

// NewSoulExtractorService crée un nouveau service d'extraction
func NewSoulExtractorService() *SoulExtractorService {
	rules := &HeuristicRules{
		TraitPatterns: map[entities.TraitCategory][]TraitPattern{
			entities.TraitCognitive: {
				{TraitName: "analytical", Keywords: []string{"analyze", "analysis", "examine", "break down", "deconstruct"}, Intensity: 0.8, Confidence: 0.6},
				{TraitName: "creative", Keywords: []string{"imagine", "create", "innovative", "novel", "unique approach"}, Intensity: 0.7, Confidence: 0.5},
				{TraitName: "logical", Keywords: []string{"therefore", "thus", "consequently", "logically", "follows that"}, Intensity: 0.8, Confidence: 0.6},
			},
			entities.TraitEmotional: {
				{TraitName: "empathetic", Keywords: []string{"I understand", "that must be", "I feel", "you're feeling", "I hear you"}, Intensity: 0.8, Confidence: 0.7},
				{TraitName: "patient", Keywords: []string{"take your time", "no rush", "step by step", "at your pace"}, Intensity: 0.7, Confidence: 0.6},
				{TraitName: "enthusiastic", Keywords: []string{"excited", "thrilled", "love this", "amazing", "fantastic"}, Intensity: 0.8, Confidence: 0.6},
			},
			entities.TraitSocial: {
				{TraitName: "collaborative", Keywords: []string{"together", "let's", "we can", "our approach", "team"}, Intensity: 0.7, Confidence: 0.5},
				{TraitName: "direct", Keywords: []string{"frankly", "honestly", "to be direct", "straightforward", "plainly"}, Intensity: 0.8, Confidence: 0.6},
				{TraitName: "diplomatic", Keywords: []string{"perhaps", "might", "could be", "one perspective", "another way"}, Intensity: 0.7, Confidence: 0.5},
			},
			entities.TraitEpistemic: {
				{TraitName: "curious", Keywords: []string{"wonder", "curious", "explore", "discover", "interesting question"}, Intensity: 0.8, Confidence: 0.6},
				{TraitName: "intellectually_humble", Keywords: []string{"I might be wrong", "not sure", "could be mistaken", "my understanding"}, Intensity: 0.7, Confidence: 0.6},
				{TraitName: "open_minded", Keywords: []string{"alternative view", "different perspective", "valid point", "consider"}, Intensity: 0.7, Confidence: 0.5},
			},
			entities.TraitExpressive: {
				{TraitName: "humorous", Keywords: []string{"haha", "funny", "joke", "pun", "lol", "amusing"}, Intensity: 0.7, Confidence: 0.6},
				{TraitName: "metaphorical", Keywords: []string{"like a", "imagine", "picture this", "analogy", "metaphor"}, Intensity: 0.8, Confidence: 0.6},
				{TraitName: "concise", Keywords: []string{"in short", "briefly", "to sum up", "TL;DR", "essentially"}, Intensity: 0.7, Confidence: 0.5},
			},
			entities.TraitEthical: {
				{TraitName: "transparent", Keywords: []string{"to be transparent", "full disclosure", "honestly", "candidly"}, Intensity: 0.8, Confidence: 0.7},
				{TraitName: "benevolent", Keywords: []string{"help you", "your best interest", "for your benefit", "want to help"}, Intensity: 0.7, Confidence: 0.6},
				{TraitName: "fair", Keywords: []string{"fair", "balanced view", "both sides", "equitable", "objective"}, Intensity: 0.7, Confidence: 0.5},
			},
		},
		VoicePatterns: VoicePatterns{
			FormalIndicators:    []string{"dear", "sincerely", "furthermore", "however", "moreover", "regarding"},
			InformalIndicators:  []string{"hey", "yeah", "nope", "gonna", "wanna", "kinda", "btw"},
			HumorIndicators:     []string{":)", ":D", "haha", "lol", "joke", "funny", "pun", "hilarious"},
			EmpathyIndicators:   []string{"I understand", "that sounds", "you must", "I hear", "I feel"},
			TechnicalIndicators: []string{"implementation", "architecture", "component", "system", "API", "function"},
		},
		CommunicationPatterns: CommunicationPatterns{
			QuestionPatterns:       []string{"?", "what", "how", "why", "when", "where", "who"},
			AcknowledgmentPatterns: []string{"I see", "understood", "got it", "makes sense", "I understand"},
			AlternativePatterns:    []string{"alternatively", "another option", "you could also", "or you might"},
		},
	}
	
	return &SoulExtractorService{
		heuristicRules: rules,
	}
}

// ExtractFromConversation extrait l'identité complète depuis une conversation
func (s *SoulExtractorService) ExtractFromConversation(ctx context.Context, request *valueobjects.SoulCaptureRequest) (*ports.ExtractionResult, error) {
	result := &ports.ExtractionResult{
		Traits:              make([]*entities.PersonalityTrait, 0),
		SourceObservations:  make([]*entities.TraitObservation, 0),
		Confidence:          0.5,
		ExtractionTimestamp: time.Now().Format(time.RFC3339),
	}
	
	// Extraire les traits
	observations, err := s.ExtractTraits(ctx, request.AgentResponses, request.Conversation)
	if err != nil {
		return nil, err
	}
	result.SourceObservations = observations
	
	// Synthétiser les observations en traits
	traits := s.synthesizeTraits(observations)
	result.Traits = traits
	
	// Extraire le profil de voix
	voice, err := s.ExtractVoiceProfile(ctx, request.AgentResponses)
	if err == nil {
		result.VoiceProfile = voice
	}
	
	// Extraire le style de communication
	comm, err := s.ExtractCommunicationStyle(ctx, request.AgentResponses)
	if err == nil {
		result.CommunicationStyle = comm
	}
	
	// Extraire la signature comportementale
	behavior, err := s.ExtractBehavioralSignature(ctx, request.Conversation, request.AgentResponses)
	if err == nil {
		result.BehavioralSignature = behavior
	}
	
	// Extraire le système de valeurs
	values, err := s.ExtractValueSystem(ctx, request.AgentResponses, request.UserFeedback)
	if err == nil {
		result.ValueSystem = values
	}
	
	// Extraire le ton émotionnel
	tone, err := s.ExtractEmotionalTone(ctx, request.AgentResponses)
	if err == nil {
		result.EmotionalTone = tone
	}
	
	// Calculer la confiance globale
	result.Confidence = s.calculateOverallConfidence(result)
	
	return result, nil
}

// ExtractTraits extrait les traits de personnalité
func (s *SoulExtractorService) ExtractTraits(ctx context.Context, agentResponses []string, context string) ([]*entities.TraitObservation, error) {
	var observations []*entities.TraitObservation
	
	allText := strings.Join(agentResponses, " ")
	allTextLower := strings.ToLower(allText)
	
	// Parcourir toutes les catégories et patterns
	for category, patterns := range s.heuristicRules.TraitPatterns {
		for _, pattern := range patterns {
			// Compter les occurrences des keywords
			occurrences := 0
			for _, keyword := range pattern.Keywords {
				occurrences += strings.Count(allTextLower, keyword)
			}
			
			if occurrences > 0 {
				// Calculer l'intensité basée sur la fréquence
				intensity := pattern.Intensity * minFloat(1.0, float64(occurrences)/3.0)
				confidence := pattern.Confidence * minFloat(1.0, float64(occurrences)/5.0)
				
				// Trouver l'évidence (premier texte contenant un keyword)
				var evidence string
				for _, response := range agentResponses {
					responseLower := strings.ToLower(response)
					for _, keyword := range pattern.Keywords {
						if strings.Contains(responseLower, keyword) {
							evidence = response
							break
						}
					}
					if evidence != "" {
						break
					}
				}
				
				obs := entities.NewTraitObservation(
					"",
					pattern.TraitName,
					category,
					evidence,
					context,
					intensity,
				)
				_ = confidence // confidence tracked via PersonalityTrait, not TraitObservation
				observations = append(observations, obs)
			}
		}
	}
	
	return observations, nil
}

// ExtractVoiceProfile extrait le profil de voix
func (s *SoulExtractorService) ExtractVoiceProfile(ctx context.Context, agentResponses []string) (*entities.VoiceProfile, error) {
	voice := entities.NewVoiceProfile()
	allText := strings.ToLower(strings.Join(agentResponses, " "))
	
	// Analyser la formalité
	formalCount := countOccurrences(allText, s.heuristicRules.VoicePatterns.FormalIndicators)
	informalCount := countOccurrences(allText, s.heuristicRules.VoicePatterns.InformalIndicators)
	total := formalCount + informalCount
	if total > 0 {
		voice.WithFormality(float64(formalCount) / float64(total))
	}
	
	// Analyser l'humour
	humorCount := countOccurrences(allText, s.heuristicRules.VoicePatterns.HumorIndicators)
	voice.WithHumor(minFloat(1.0, float64(humorCount)/5.0))
	
	// Analyser l'empathie
	empathyCount := countOccurrences(allText, s.heuristicRules.VoicePatterns.EmpathyIndicators)
	voice.WithEmpathy(minFloat(1.0, float64(empathyCount)/5.0))
	
	// Analyser la profondeur technique
	techCount := countOccurrences(allText, s.heuristicRules.VoicePatterns.TechnicalIndicators)
	voice.WithTechnicalDepth(minFloat(1.0, float64(techCount)/5.0))
	
	// Détecter les catch phrases
	voice.CatchPhrases = s.detectCatchPhrases(agentResponses)
	
	// Calculer la longueur moyenne des phrases
	voice.AvgSentenceLength = s.calculateAvgSentenceLength(agentResponses)
	
	// Détecter les emojis
	voice.UsesEmojis = strings.Contains(allText, ":)") || strings.Contains(allText, ":(") || 
		strings.Contains(allText, "😊") || strings.Contains(allText, "👍")
	
	return voice, nil
}

// ExtractCommunicationStyle extrait le style de communication
func (s *SoulExtractorService) ExtractCommunicationStyle(ctx context.Context, agentResponses []string) (*entities.CommunicationStyle, error) {
	style := entities.NewCommunicationStyle()
	allText := strings.Join(agentResponses, " ")
	
	// Analyser la longueur des réponses
	avgLength := 0
	for _, resp := range agentResponses {
		avgLength += len(resp)
	}
	if len(agentResponses) > 0 {
		avgLength /= len(agentResponses)
	}
	
	switch {
	case avgLength < 100:
		style.ResponseLength = entities.LengthConcise
	case avgLength < 300:
		style.ResponseLength = entities.LengthModerate
	case avgLength < 600:
		style.ResponseLength = entities.LengthDetailed
	default:
		style.ResponseLength = entities.LengthExhaustive
	}
	
	// Détecter les questions de clarification
	style.AsksClarifyingQuestions = countOccurrences(allText, s.heuristicRules.CommunicationPatterns.QuestionPatterns) > 2
	
	// Détecter les acknowledgments
	style.AcknowledgesBeforeAnswering = countOccurrences(allText, s.heuristicRules.CommunicationPatterns.AcknowledgmentPatterns) > 1
	
	// Détecter les alternatives
	style.ProvidesAlternatives = countOccurrences(allText, s.heuristicRules.CommunicationPatterns.AlternativePatterns) > 0
	
	// Détecter la structure
	if strings.Contains(allText, "1.") || strings.Contains(allText, "2.") {
		style.StructurePreference = entities.StructureNumbered
	} else if strings.Contains(allText, "-") || strings.Contains(allText, "*") {
		style.StructurePreference = entities.StructureBulleted
	}
	
	return style, nil
}

// ExtractBehavioralSignature extrait la signature comportementale
func (s *SoulExtractorService) ExtractBehavioralSignature(ctx context.Context, conversation string, agentResponses []string) (*entities.BehavioralSignature, error) {
	behavior := entities.NewBehavioralSignature()
	
	// Détecter la gestion des erreurs
	if strings.Contains(strings.ToLower(conversation), "wrong") || 
	   strings.Contains(strings.ToLower(conversation), "mistake") ||
	   strings.Contains(strings.ToLower(conversation), "error") {
		if strings.Contains(strings.ToLower(conversation), "sorry") ||
		   strings.Contains(strings.ToLower(conversation), "apologize") {
			behavior.ErrorHandlingStyle = entities.ErrorApologetic
		}
		behavior.AdmitsMistakes = true
	}
	
	// Détecter le style de désaccord
	lowerConv := strings.ToLower(conversation)
	switch {
	case strings.Contains(lowerConv, "I disagree") || strings.Contains(lowerConv, "that's not right"):
		behavior.DisagreementStyle = entities.DisagreeDirect
	case strings.Contains(lowerConv, "I see it differently") || strings.Contains(lowerConv, "another perspective"):
		behavior.DisagreementStyle = entities.DisagreePolite
	case strings.Contains(lowerConv, "what if") || strings.Contains(lowerConv, "have you considered"):
		behavior.DisagreementStyle = entities.DisagreeSocratic
	}
	
	// Détecter la curiosité
	questionCount := strings.Count(conversation, "?")
	behavior.CuriosityLevel = minFloat(1.0, float64(questionCount)/10.0)
	
	return behavior, nil
}

// ExtractValueSystem extrait le système de valeurs
func (s *SoulExtractorService) ExtractValueSystem(ctx context.Context, agentResponses []string, userFeedback map[string]string) (*entities.ValueSystem, error) {
	values := entities.NewValueSystem()
	allText := strings.ToLower(strings.Join(agentResponses, " "))
	
	// Analyser les priorités
	if strings.Contains(allText, "accurate") || strings.Contains(allText, "correct") || strings.Contains(allText, "precision") {
		values.PrioritizesAccuracy = 0.9
	}
	if strings.Contains(allText, "help") || strings.Contains(allText, "assist") || strings.Contains(allText, "support") {
		values.PrioritizesHelpfulness = 0.9
	}
	if strings.Contains(allText, "efficient") || strings.Contains(allText, "quick") || strings.Contains(allText, "optimize") {
		values.PrioritizesEfficiency = 0.8
	}
	if strings.Contains(allText, "clear") || strings.Contains(allText, "simple") || strings.Contains(allText, "understandable") {
		values.PrioritizesClarity = 0.9
	}
	
	// Extraire les valeurs fondamentales
	if strings.Contains(allText, "honest") || strings.Contains(allText, "transparent") {
		values.WithCoreValue("honesty", 0.9, entities.ValueEpistemic)
	}
	if strings.Contains(allText, "fair") || strings.Contains(allText, "equitable") {
		values.WithCoreValue("fairness", 0.8, entities.ValueMoral)
	}
	if strings.Contains(allText, "helpful") || strings.Contains(allText, "useful") {
		values.WithCoreValue("helpfulness", 0.9, entities.ValuePragmatic)
	}
	
	// Intégrer le feedback utilisateur
	for _, feedback := range userFeedback {
		lowerFeedback := strings.ToLower(feedback)
		if strings.Contains(lowerFeedback, "kind") || strings.Contains(lowerFeedback, "nice") {
			values.WithCoreValue("kindness", 0.8, entities.ValueMoral)
		}
		if strings.Contains(lowerFeedback, "smart") || strings.Contains(lowerFeedback, "intelligent") {
			values.WithCoreValue("intelligence", 0.8, entities.ValueEpistemic)
		}
	}
	
	return values, nil
}

// ExtractEmotionalTone extrait le ton émotionnel
func (s *SoulExtractorService) ExtractEmotionalTone(ctx context.Context, agentResponses []string) (*entities.EmotionalTone, error) {
	tone := entities.NewEmotionalTone()
	allText := strings.ToLower(strings.Join(agentResponses, " "))
	
	// Analyser la chaleur
	warmIndicators := []string{"glad", "happy", "pleased", "delighted", "welcome", "appreciate"}
	tone.Warmth = minFloat(1.0, float64(countOccurrences(allText, warmIndicators))/3.0)
	
	// Analyser le calme
	calmIndicators := []string{"calm", "peaceful", "steady", "composed", "relaxed"}
	if countOccurrences(allText, calmIndicators) > 0 {
		tone.Calmness = 0.8
	}
	
	// Analyser l'enthousiasme
	enthusiasmIndicators := []string{"excited", "thrilled", "love", "amazing", "fantastic", "wonderful"}
	tone.Enthusiasm = minFloat(1.0, float64(countOccurrences(allText, enthusiasmIndicators))/3.0)
	
	// Analyser l'encouragement
	encouragementIndicators := []string{"great job", "well done", "you can do", "excellent", "proud"}
	tone.EncouragementLevel = minFloat(1.0, float64(countOccurrences(allText, encouragementIndicators))/3.0)
	
	return tone, nil
}

// --- Helpers ---

func (s *SoulExtractorService) synthesizeTraits(observations []*entities.TraitObservation) []*entities.PersonalityTrait {
	// Grouper les observations par nom de trait
	grouped := make(map[string][]*entities.TraitObservation)
	for _, obs := range observations {
		grouped[obs.TraitName] = append(grouped[obs.TraitName], obs)
	}
	
	var traits []*entities.PersonalityTrait
	for traitName, obsList := range grouped {
		if len(obsList) == 0 {
			continue
		}
		
		// Calculer l'intensité moyenne
		totalIntensity := 0.0
		for _, obs := range obsList {
			totalIntensity += obs.Intensity
		}
		avgIntensity := totalIntensity / float64(len(obsList))
		
		// Créer le trait
		trait := entities.NewPersonalityTrait(traitName, obsList[0].Category, avgIntensity)
		
		// Ajouter les preuves
		for _, obs := range obsList {
			trait.WithEvidence(obs.Evidence, obs.Context)
		}
		
		traits = append(traits, trait)
	}
	
	return traits
}

func (s *SoulExtractorService) calculateOverallConfidence(result *ports.ExtractionResult) float64 {
	if len(result.Traits) == 0 {
		return 0.3
	}
	
	totalConfidence := 0.0
	for _, trait := range result.Traits {
		totalConfidence += trait.Confidence
	}
	
	return totalConfidence / float64(len(result.Traits))
}

func (s *SoulExtractorService) detectCatchPhrases(responses []string) []string {
	// Détecter les expressions récurrentes (simplifié)
	// Une vraie implémentation utiliserait du NLP plus sophistiqué
	phraseCount := make(map[string]int)
	
	for _, response := range responses {
		// Extraire les phrases de 2-4 mots
		words := strings.Fields(response)
		for i := 0; i < len(words)-2; i++ {
			phrase := strings.ToLower(words[i] + " " + words[i+1] + " " + words[i+2])
			phraseCount[phrase]++
		}
	}
	
	// Retourner les phrases qui apparaissent plus d'une fois
	var catchPhrases []string
	for phrase, count := range phraseCount {
		if count > 1 && len(phrase) > 10 {
			catchPhrases = append(catchPhrases, phrase)
		}
	}
	
	// Limiter à 5
	if len(catchPhrases) > 5 {
		catchPhrases = catchPhrases[:5]
	}
	
	return catchPhrases
}

func (s *SoulExtractorService) calculateAvgSentenceLength(responses []string) int {
	totalWords := 0
	totalSentences := 0
	
	for _, response := range responses {
		sentences := strings.Split(response, ".")
		for _, sentence := range sentences {
			words := strings.Fields(sentence)
			totalWords += len(words)
			totalSentences++
		}
	}
	
	if totalSentences == 0 {
		return 15 // Valeur par défaut
	}
	
	return totalWords / totalSentences
}

func countOccurrences(text string, patterns []string) int {
	count := 0
	for _, pattern := range patterns {
		count += strings.Count(text, pattern)
	}
	return count
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
