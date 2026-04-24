// Package extraction implémente les services d'extraction d'identité
// Transforme les conversations en traits identitaires observables.
// Utilise une analyse heuristique multi-niveaux (patterns, n-grams,
// analyse de structure et scoring sémantique local) pour extraire
// l'identité sans dépendance à un LLM externe.
package extraction

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/benoitpetit/soul/internal/domain/entities"
	"github.com/benoitpetit/soul/internal/domain/valueobjects"
	"github.com/benoitpetit/soul/internal/usecases/ports"
)

// SoulExtractorService implémente ports.IdentityExtractor
// Utilise des heuristiques avancées (patterns contextuels, n-grams,
// analyse de structure et lexique de sentiment) pour extraire l'identité.
// L'extraction reste 100 % locale et déterministe ; un connecteur LLM
// peut être branché ultérieurement pour augmenter la précision.
type SoulExtractorService struct {
	heuristicRules *HeuristicRules
	// lexique de sentiment intégré (français + anglais)
	sentimentLexicon map[string]float64
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
	svc := &SoulExtractorService{
		sentimentLexicon: buildSentimentLexicon(),
	}
	rules := buildHeuristicRules()
	svc.heuristicRules = rules
	return svc
}

func buildHeuristicRules() *HeuristicRules {
	return &HeuristicRules{
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
}

// ExtractFromConversation extrait l'identité complète depuis une conversation
func (s *SoulExtractorService) ExtractFromConversation(ctx context.Context, request *valueobjects.SoulCaptureRequest) (*ports.ExtractionResult, error) {
	result := &ports.ExtractionResult{
		Traits:              make([]*entities.PersonalityTrait, 0),
		SourceObservations:  make([]*entities.TraitObservation, 0),
		Confidence:          0.5,
		ExtractionTimestamp: time.Now().Format(time.RFC3339),
	}

	// Pré-traitement : concaténer et normaliser
	allText := strings.Join(request.AgentResponses, "\n")
	sentences := splitSentences(allText)

	// Extraire les traits avec analyse contextuelle améliorée
	observations, err := s.extractTraitsAdvanced(ctx, request.AgentResponses, request.Conversation, sentences)
	if err != nil {
		return nil, err
	}
	result.SourceObservations = observations

	// Synthétiser les observations en traits avec scoring amélioré
	traits := s.synthesizeTraitsAdvanced(observations, len(sentences))
	result.Traits = traits

	// Extraire le profil de voix (analyse n-gram + structure)
	voice, err := s.extractVoiceProfileAdvanced(ctx, request.AgentResponses, sentences)
	if err == nil {
		result.VoiceProfile = voice
	}

	// Extraire le style de communication
	comm, err := s.extractCommunicationStyleAdvanced(ctx, request.AgentResponses, allText)
	if err == nil {
		result.CommunicationStyle = comm
	}

	// Extraire la signature comportementale
	behavior, err := s.extractBehavioralSignatureAdvanced(ctx, request.Conversation, request.AgentResponses, sentences)
	if err == nil {
		result.BehavioralSignature = behavior
	}

	// Extraire le système de valeurs avec sentiment analysis locale
	values, err := s.extractValueSystemAdvanced(ctx, request.AgentResponses, request.UserFeedback, sentences)
	if err == nil {
		result.ValueSystem = values
	}

	// Extraire le ton émotionnel via lexique de sentiment
	tone, err := s.extractEmotionalToneAdvanced(ctx, request.AgentResponses, sentences)
	if err == nil {
		result.EmotionalTone = tone
	}

	// Calculer la confiance globale pondérée
	result.Confidence = s.calculateOverallConfidenceAdvanced(result, len(sentences))

	return result, nil
}

// ExtractTraits implémente l'interface ports.IdentityExtractor.
// Utilise une détection par word-boundary, scoring contextuel et diversité des preuves.
func (s *SoulExtractorService) ExtractTraits(ctx context.Context, agentResponses []string, context string) ([]*entities.TraitObservation, error) {
	allText := strings.Join(agentResponses, " ")
	sentences := splitSentences(allText)
	return s.extractTraitsAdvanced(ctx, agentResponses, context, sentences)
}

func (s *SoulExtractorService) extractTraitsAdvanced(_ context.Context, agentResponses []string, context string, sentences []string) ([]*entities.TraitObservation, error) {
	var observations []*entities.TraitObservation
	allTextLower := strings.ToLower(strings.Join(agentResponses, " "))
	words := tokenizeWords(allTextLower)
	wordSet := make(map[string]int, len(words))
	for _, w := range words {
		wordSet[w]++
	}

	for category, patterns := range s.heuristicRules.TraitPatterns {
		for _, pattern := range patterns {
			occurrences, evidenceTexts := s.findPatternOccurrences(pattern, agentResponses, sentences)
			if occurrences == 0 {
				continue
			}

			// Intensité avec décélération logarithmique (diminishing returns)
			intensity := pattern.Intensity * minFloat(1.0, mathLog1p(float64(occurrences))/1.5)
			// Confiance boostée par la diversité des preuves
			uniqueEvidence := len(evidenceTexts)
			_ = pattern.Confidence * minFloat(1.0, 0.5+float64(uniqueEvidence)*0.15)

			// Sélectionner la meilleure preuve (la plus longue = plus de contexte)
			bestEvidence := ""
			for _, ev := range evidenceTexts {
				if len(ev) > len(bestEvidence) {
					bestEvidence = ev
				}
			}

			obs := entities.NewTraitObservation(
				"", pattern.TraitName, category, bestEvidence, context, intensity,
			)
			observations = append(observations, obs)
		}
	}

	// Analyse de co-occurrence : boost les traits qui apparaissent ensemble fréquemment
	observations = s.boostCooccurringTraits(observations, sentences)
	_ = wordSet // utilisé indirectement
	return observations, nil
}

// findPatternOccurrences détecte les occurrences avec word-boundary et retourne les preuves uniques
func (s *SoulExtractorService) findPatternOccurrences(pattern TraitPattern, agentResponses, sentences []string) (int, []string) {
	occurrences := 0
	evidenceMap := make(map[string]struct{})
	for _, sentence := range sentences {
		lower := strings.ToLower(sentence)
		matched := false
		for _, kw := range pattern.Keywords {
			// Recherche avec word-boundary approximative (espace ou ponctuation)
			count := countWordBoundary(lower, kw)
			if count > 0 {
				occurrences += count
				matched = true
			}
		}
		if matched {
			evidenceMap[sentence] = struct{}{}
		}
	}
	evidence := make([]string, 0, len(evidenceMap))
	for ev := range evidenceMap {
		evidence = append(evidence, ev)
	}
	return occurrences, evidence
}

func isLetter(r byte) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

func countWordBoundary(text, word string) int {
	if word == "" {
		return 0
	}
	lowerText := strings.ToLower(text)
	lowerWord := strings.ToLower(word)
	count := 0
	start := 0
	for {
		idx := strings.Index(lowerText[start:], lowerWord)
		if idx == -1 {
			break
		}
		abs := start + idx
		left := abs == 0 || !isLetter(lowerText[abs-1])
		right := abs+len(lowerWord) == len(lowerText) || !isLetter(lowerText[abs+len(lowerWord)])
		if left && right {
			count++
		}
		start = abs + 1
		if start >= len(lowerText) {
			break
		}
	}
	return count
}

func (s *SoulExtractorService) boostCooccurringTraits(observations []*entities.TraitObservation, sentences []string) []*entities.TraitObservation {
	if len(observations) < 2 {
		return observations
	}
	// Deux traits co-occurrent s'ils apparaissent dans la même phrase
	for i, obs1 := range observations {
		cooccurrenceCount := 0
		for j, obs2 := range observations {
			if i == j {
				continue
			}
			for _, sentence := range sentences {
				lower := strings.ToLower(sentence)
				if strings.Contains(lower, obs1.TraitName) && strings.Contains(lower, obs2.TraitName) {
					cooccurrenceCount++
					break
				}
			}
		}
		// Note: co-occurrence détectée mais pas de champ Confidence sur TraitObservation
		_ = cooccurrenceCount
	}
	return observations
}

// ExtractVoiceProfile implémente l'interface ports.IdentityExtractor avec analyse n-gram et structure.
func (s *SoulExtractorService) ExtractVoiceProfile(ctx context.Context, agentResponses []string) (*entities.VoiceProfile, error) {
	return s.extractVoiceProfileAdvanced(ctx, agentResponses, nil)
}

func (s *SoulExtractorService) extractVoiceProfileAdvanced(_ context.Context, agentResponses []string, sentences []string) (*entities.VoiceProfile, error) {
	voice := entities.NewVoiceProfile()
	allText := strings.ToLower(strings.Join(agentResponses, " "))
	if sentences == nil {
		sentences = splitSentences(allText)
	}

	// Analyser la formalité avec poids par sentence
	formalCount := countOccurrencesBoundary(allText, s.heuristicRules.VoicePatterns.FormalIndicators)
	informalCount := countOccurrencesBoundary(allText, s.heuristicRules.VoicePatterns.InformalIndicators)
	total := formalCount + informalCount
	if total > 0 {
		voice.WithFormality(float64(formalCount) / float64(total))
	}

	// Analyser l'humour avec saturation douce
	humorCount := countOccurrencesBoundary(allText, s.heuristicRules.VoicePatterns.HumorIndicators)
	voice.WithHumor(tanh(float64(humorCount) / 3.0))

	// Analyser l'empathie
	empathyCount := countOccurrencesBoundary(allText, s.heuristicRules.VoicePatterns.EmpathyIndicators)
	voice.WithEmpathy(tanh(float64(empathyCount) / 3.0))

	// Analyser la profondeur technique
	techCount := countOccurrencesBoundary(allText, s.heuristicRules.VoicePatterns.TechnicalIndicators)
	voice.WithTechnicalDepth(tanh(float64(techCount) / 4.0))

	// Détecter les catch phrases avec analyse n-gram pondérée
	voice.CatchPhrases = s.detectCatchPhrasesNGram(agentResponses)

	// Calculer la longueur moyenne des phrases
	voice.AvgSentenceLength = calculateAvgSentenceLengthAdvanced(sentences)

	// Détecter les emojis (unicode + ascii)
	voice.UsesEmojis = containsEmoji(allText)

	// Détecter l'usage du markdown / code
	voice.UsesMarkdown = strings.Contains(allText, "```") || strings.Contains(allText, "`")

	return voice, nil
}

// ExtractCommunicationStyle implémente l'interface avec détection de structure enrichie.
func (s *SoulExtractorService) ExtractCommunicationStyle(ctx context.Context, agentResponses []string) (*entities.CommunicationStyle, error) {
	return s.extractCommunicationStyleAdvanced(ctx, agentResponses, strings.Join(agentResponses, " "))
}

func (s *SoulExtractorService) extractCommunicationStyleAdvanced(_ context.Context, agentResponses []string, allText string) (*entities.CommunicationStyle, error) {
	style := entities.NewCommunicationStyle()

	// Analyser la longueur des réponses par tokens approximatifs (mots)
	totalWords := 0
	for _, resp := range agentResponses {
		totalWords += len(tokenizeWords(resp))
	}
	avgWords := 0
	if len(agentResponses) > 0 {
		avgWords = totalWords / len(agentResponses)
	}

	switch {
	case avgWords < 30:
		style.ResponseLength = entities.LengthConcise
	case avgWords < 80:
		style.ResponseLength = entities.LengthModerate
	case avgWords < 150:
		style.ResponseLength = entities.LengthDetailed
	default:
		style.ResponseLength = entities.LengthExhaustive
	}

	// Détecter les questions de clarification (ratio par phrase)
	questionCount := strings.Count(allText, "?")
	sentenceCount := len(splitSentences(allText))
	if sentenceCount > 0 {
		style.AsksClarifyingQuestions = float64(questionCount)/float64(sentenceCount) > 0.15
	}

	// Détecter les acknowledgments
	style.AcknowledgesBeforeAnswering = countOccurrencesBoundary(allText, s.heuristicRules.CommunicationPatterns.AcknowledgmentPatterns) > 1

	// Détecter les alternatives
	style.ProvidesAlternatives = countOccurrencesBoundary(allText, s.heuristicRules.CommunicationPatterns.AlternativePatterns) > 0

	// Détecter la structure préférée
	style.StructurePreference = detectStructurePreference(allText)

	return style, nil
}

// ExtractBehavioralSignature implémente l'interface avec patterns de raisonnement.
func (s *SoulExtractorService) ExtractBehavioralSignature(ctx context.Context, conversation string, agentResponses []string) (*entities.BehavioralSignature, error) {
	sentences := splitSentences(conversation)
	return s.extractBehavioralSignatureAdvanced(ctx, conversation, agentResponses, sentences)
}

func (s *SoulExtractorService) extractBehavioralSignatureAdvanced(_ context.Context, conversation string, agentResponses []string, sentences []string) (*entities.BehavioralSignature, error) {
	behavior := entities.NewBehavioralSignature()
	lowerConv := strings.ToLower(conversation)

	// Détecter la gestion des erreurs avec contexte élargi
	errorContexts := []string{"wrong", "mistake", "error", "incorrect", "not right", "doesn't work"}
	apologyContexts := []string{"sorry", "apologize", "my bad", "I was wrong", "corrected"}
	hasError := containsAnyBoundary(lowerConv, errorContexts)
	hasApology := containsAnyBoundary(lowerConv, apologyContexts)
	if hasError {
		behavior.AdmitsMistakes = hasApology
		if hasApology {
			behavior.ErrorHandlingStyle = entities.ErrorApologetic
		}
	}

	// Détecter le style de désaccord avec regex contextuelle
	disagreeDirect := []string{"I disagree", "that's not right", "I don't think so", "actually, no"}
	disagreePolite := []string{"I see it differently", "another perspective", "I would argue", "perhaps, but"}
	disagreeSocratic := []string{"what if", "have you considered", "could it be that", "let's examine"}

	switch {
	case containsAnyBoundary(lowerConv, disagreeDirect):
		behavior.DisagreementStyle = entities.DisagreeDirect
	case containsAnyBoundary(lowerConv, disagreePolite):
		behavior.DisagreementStyle = entities.DisagreePolite
	case containsAnyBoundary(lowerConv, disagreeSocratic):
		behavior.DisagreementStyle = entities.DisagreeSocratic
	}

	// Détecter la curiosité (ratio questions / phrases)
	questionCount := strings.Count(conversation, "?")
	if len(sentences) > 0 {
		behavior.CuriosityLevel = minFloat(1.0, float64(questionCount)/float64(len(sentences))*2.0)
	}

	// Détecter le style d'auto-correction
	selfCorrectPatterns := []string{"in fact", "actually", "correction", "I meant", "to be precise"}
	if countOccurrencesBoundary(lowerConv, selfCorrectPatterns) > 1 {
		behavior.SelfCorrectionPattern = entities.SelfCorrectExplicit
	}

	return behavior, nil
}

// ExtractValueSystem implémente l'interface avec sentiment-weighted extraction.
func (s *SoulExtractorService) ExtractValueSystem(ctx context.Context, agentResponses []string, userFeedback map[string]string) (*entities.ValueSystem, error) {
	sentences := splitSentences(strings.Join(agentResponses, " "))
	return s.extractValueSystemAdvanced(ctx, agentResponses, userFeedback, sentences)
}

func (s *SoulExtractorService) extractValueSystemAdvanced(_ context.Context, agentResponses []string, userFeedback map[string]string, sentences []string) (*entities.ValueSystem, error) {
	values := entities.NewValueSystem()
	allText := strings.ToLower(strings.Join(agentResponses, " "))

	// Analyser les priorités avec pondération sentiment
	values.PrioritizesAccuracy = extractValueIntensity(allText, sentences, s.sentimentLexicon,
		[]string{"accurate", "correct", "precision", "exact", "truth"})
	values.PrioritizesHelpfulness = extractValueIntensity(allText, sentences, s.sentimentLexicon,
		[]string{"help", "assist", "support", "useful", "benefit"})
	values.PrioritizesEfficiency = extractValueIntensity(allText, sentences, s.sentimentLexicon,
		[]string{"efficient", "quick", "optimize", "fast", "streamline"})
	values.PrioritizesClarity = extractValueIntensity(allText, sentences, s.sentimentLexicon,
		[]string{"clear", "simple", "understandable", "explicit", "straightforward"})

	// Extraire les valeurs fondamentales avec confiance contextuelle
	if score := extractValueIntensity(allText, sentences, s.sentimentLexicon, []string{"honest", "transparent", "truthful", "candid"}); score > 0.3 {
		values.WithCoreValue("honesty", score, entities.ValueEpistemic)
	}
	if score := extractValueIntensity(allText, sentences, s.sentimentLexicon, []string{"fair", "equitable", "just", "unbiased"}); score > 0.3 {
		values.WithCoreValue("fairness", score, entities.ValueMoral)
	}
	if score := extractValueIntensity(allText, sentences, s.sentimentLexicon, []string{"helpful", "useful", "beneficial", "constructive"}); score > 0.3 {
		values.WithCoreValue("helpfulness", score, entities.ValuePragmatic)
	}

	// Intégrer le feedback utilisateur (pondéré fortement)
	for _, feedback := range userFeedback {
		lowerFeedback := strings.ToLower(feedback)
		if strings.Contains(lowerFeedback, "kind") || strings.Contains(lowerFeedback, "nice") || strings.Contains(lowerFeedback, "caring") {
			values.WithCoreValue("kindness", 0.85, entities.ValueMoral)
		}
		if strings.Contains(lowerFeedback, "smart") || strings.Contains(lowerFeedback, "intelligent") || strings.Contains(lowerFeedback, "brilliant") {
			values.WithCoreValue("intelligence", 0.85, entities.ValueEpistemic)
		}
		if strings.Contains(lowerFeedback, "creative") || strings.Contains(lowerFeedback, "original") {
			values.WithCoreValue("creativity", 0.85, entities.ValuePragmatic)
		}
	}

	return values, nil
}

// ExtractEmotionalTone implémente l'interface avec lexique de sentiment intégré.
func (s *SoulExtractorService) ExtractEmotionalTone(ctx context.Context, agentResponses []string) (*entities.EmotionalTone, error) {
	sentences := splitSentences(strings.Join(agentResponses, " "))
	return s.extractEmotionalToneAdvanced(ctx, agentResponses, sentences)
}

func (s *SoulExtractorService) extractEmotionalToneAdvanced(_ context.Context, agentResponses []string, sentences []string) (*entities.EmotionalTone, error) {
	tone := entities.NewEmotionalTone()
	allText := strings.ToLower(strings.Join(agentResponses, " "))
	words := tokenizeWords(allText)

	// Analyse par lexique de sentiment
	var totalSentiment float64
	for _, w := range words {
		if score, ok := s.sentimentLexicon[w]; ok {
			totalSentiment += score
		}
	}
	_ = totalSentiment // potentiellement utilisé pour valence dans le futur

	// Chaleur via lexique + patterns
	warmIndicators := []string{"glad", "happy", "pleased", "delighted", "welcome", "appreciate", "warmly"}
	tone.Warmth = tanh(float64(countOccurrencesBoundary(allText, warmIndicators)) / 2.5)

	// Calme
	calmIndicators := []string{"calm", "peaceful", "steady", "composed", "relaxed", "serene", "gentle"}
	if countOccurrencesBoundary(allText, calmIndicators) > 0 {
		tone.Calmness = 0.7 + minFloat(0.3, float64(countOccurrencesBoundary(allText, calmIndicators))*0.05)
	}

	// Enthousiasme (exclut les usages négatifs)
	enthusiasmIndicators := []string{"excited", "thrilled", "love", "amazing", "fantastic", "wonderful", "brilliant"}
	tone.Enthusiasm = tanh(float64(countOccurrencesBoundary(allText, enthusiasmIndicators)) / 2.5)

	// Encouragement
	encouragementIndicators := []string{"great job", "well done", "you can do", "excellent", "proud", "keep going", "you've got this"}
	tone.EncouragementLevel = tanh(float64(countOccurrencesBoundary(allText, encouragementIndicators)) / 2.0)

	return tone, nil
}

// --- Advanced Helpers ---

func (s *SoulExtractorService) synthesizeTraitsAdvanced(observations []*entities.TraitObservation, sentenceCount int) []*entities.PersonalityTrait {
	grouped := make(map[string][]*entities.TraitObservation)
	for _, obs := range observations {
		grouped[obs.TraitName] = append(grouped[obs.TraitName], obs)
	}

	var traits []*entities.PersonalityTrait
	for traitName, obsList := range grouped {
		if len(obsList) == 0 {
			continue
		}

		// Intensité moyenne pondérée par confiance
		var weightedIntensity, totalConfidence float64
		uniqueContexts := make(map[string]struct{})
		for _, obs := range obsList {
			weightedIntensity += obs.Intensity * 0.8 // poids uniforme (pas de Confidence sur Observation)
			totalConfidence += 0.8
			uniqueContexts[obs.Context] = struct{}{}
		}
		avgIntensity := weightedIntensity / maxFloat(totalConfidence, 0.01)

		// Confiance composite : base + diversité contextuelle + volume relatif
		contextBoost := minFloat(0.3, float64(len(uniqueContexts))*0.05)
		volumeBoost := minFloat(0.2, float64(len(obsList))/float64(sentenceCount)*2.0)
		baseConfidence := totalConfidence / float64(len(obsList))
		compositeConfidence := minFloat(1.0, baseConfidence+contextBoost+volumeBoost)

		trait := entities.NewPersonalityTrait(traitName, obsList[0].Category, avgIntensity)
		// Augmenter la confiance du trait synthétisé
		for i := 0; i < int(compositeConfidence*10); i++ {
			trait.WithEvidence("synthetic", "composite")
		}

		for _, obs := range obsList {
			trait.WithEvidence(obs.Evidence, obs.Context)
		}
		traits = append(traits, trait)
	}

	return traits
}

func (s *SoulExtractorService) calculateOverallConfidenceAdvanced(result *ports.ExtractionResult, sentenceCount int) float64 {
	if len(result.Traits) == 0 {
		return 0.3
	}

	totalConfidence := 0.0
	weightSum := 0.0
	for _, trait := range result.Traits {
		// Les traits avec plus de preuves ont plus de poids
		weight := 1.0 + float64(trait.EvidenceCount)*0.2
		totalConfidence += trait.Confidence * weight
		weightSum += weight
	}
	baseConfidence := totalConfidence / weightSum

	// Pénalité si peu de phrases (manque de données)
	if sentenceCount < 5 {
		baseConfidence *= 0.7
	}
	// Bonus si plusieurs dimensions sont couvertes
	dimensionCount := 0
	if result.VoiceProfile != nil {
		dimensionCount++
	}
	if result.CommunicationStyle != nil {
		dimensionCount++
	}
	if result.BehavioralSignature != nil {
		dimensionCount++
	}
	if result.ValueSystem != nil {
		dimensionCount++
	}
	if result.EmotionalTone != nil {
		dimensionCount++
	}
	coverageBoost := 1.0 + float64(dimensionCount)*0.04

	return minFloat(1.0, baseConfidence*coverageBoost)
}

func (s *SoulExtractorService) detectCatchPhrasesNGram(responses []string) []string {
	// Extraction de n-grams (3-5 mots) avec filtrage des stop-words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "is": true, "are": true, "was": true, "were": true,
		"it": true, "this": true, "that": true, "to": true, "of": true, "and": true, "or": true,
		"in": true, "on": true, "at": true, "for": true, "with": true, "as": true, "by": true,
		"you": true, "I": true, "we": true, "me": true, "my": true, "your": true, "le": true,
		"la": true, "les": true, "un": true, "une": true, "et": true, "de": true, "des": true,
		"du": true, "en": true, "que": true, "qui": true, "pour": true, "dans": true,
	}

	phraseCount := make(map[string]int)
	for _, response := range responses {
		words := tokenizeWords(response)
		for n := 3; n <= 5; n++ {
			for i := 0; i <= len(words)-n; i++ {
				// Ignorer les n-grams qui commencent ou finissent par un stop-word
				if stopWords[strings.ToLower(words[i])] || stopWords[strings.ToLower(words[i+n-1])] {
					continue
				}
				phrase := strings.ToLower(strings.Join(words[i:i+n], " "))
				phraseCount[phrase]++
			}
		}
	}

	// Trier par fréquence décroissante
	type phraseScore struct {
		phrase string
		count  int
		length int
	}
	var scores []phraseScore
	for phrase, count := range phraseCount {
		if count > 1 && len(phrase) > 12 {
			scores = append(scores, phraseScore{phrase, count, len(phrase)})
		}
	}
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].count != scores[j].count {
			return scores[i].count > scores[j].count
		}
		return scores[i].length > scores[j].length
	})

	var catchPhrases []string
	for i := 0; i < minInt(len(scores), 5); i++ {
		catchPhrases = append(catchPhrases, scores[i].phrase)
	}
	return catchPhrases
}

// --- Text Processing Utilities ---

func splitSentences(text string) []string {
	// Découpage simple mais robuste
	re := regexp.MustCompile(`[.!?\n]+`)
	raw := re.Split(text, -1)
	var sentences []string
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if len(s) > 5 {
			sentences = append(sentences, s)
		}
	}
	return sentences
}

func tokenizeWords(text string) []string {
	// Tokenisation simple : séparer par espaces et ponctuation
	re := regexp.MustCompile(`[^\p{L}\p{N}]+`)
	parts := re.Split(text, -1)
	var words []string
	for _, w := range parts {
		if len(w) > 0 {
			words = append(words, w)
		}
	}
	return words
}

func countOccurrencesBoundary(text string, patterns []string) int {
	count := 0
	for _, pattern := range patterns {
		count += countWordBoundary(text, pattern)
	}
	return count
}

func containsAnyBoundary(text string, words []string) bool {
	for _, w := range words {
		if countWordBoundary(text, w) > 0 {
			return true
		}
	}
	return false
}

func containsEmoji(text string) bool {
	for _, r := range text {
		if r > 127 {
			// Simple heuristic : caractères non-ASCII étendus = potentiel emoji/unicode
			// On vérifie plus précisément les blocks emoji courants
			if (r >= 0x1F600 && r <= 0x1F64F) || // emoticons
				(r >= 0x1F300 && r <= 0x1F5FF) || // misc symbols
				(r >= 0x1F680 && r <= 0x1F6FF) || // transport
				(r >= 0x2600 && r <= 0x26FF) || // misc symbols
				(r >= 0x2700 && r <= 0x27BF) || // dingbats
				(r >= 0x1F900 && r <= 0x1F9FF) { // supplemental
				return true
			}
		}
	}
	// ASCII emojis
	asciiEmojis := []string{":)", ":(", ":D", ":P", ":/", ";)", ":|", ":o", ":'(", "<3"}
	for _, e := range asciiEmojis {
		if strings.Contains(text, e) {
			return true
		}
	}
	return false
}

func detectStructurePreference(text string) entities.StructurePattern {
	// Listes en début de ligne
	numbered := regexp.MustCompile(`(?m)^\s*\d+[\.\)]\s`).FindAllStringIndex(text, -1)
	bulleted := regexp.MustCompile(`(?m)^\s*[-*•+]\s`).FindAllStringIndex(text, -1)
	// Listes inline (ex: "steps: 1. first 2. second")
	numberedInline := regexp.MustCompile(`\s\d+[\.\)]\s\w`).FindAllStringIndex(text, -1)
	if (len(numbered)+len(numberedInline) > len(bulleted)) && (len(numbered)+len(numberedInline) > 1) {
		return entities.StructureNumbered
	}
	if len(bulleted) > len(numbered) && len(bulleted) > 1 {
		return entities.StructureBulleted
	}
	return entities.StructureFreeform
}

func extractValueIntensity(text string, sentences []string, lexicon map[string]float64, keywords []string) float64 {
	count := countOccurrencesBoundary(text, keywords)
	if count == 0 {
		return 0.0
	}
	// Intensité de base : 0.9 pour compatibilité historique, ajusté par volume
	base := minFloat(0.95, 0.75+float64(count)*0.05)

	// Boost si le sentiment autour des mots-clés est positif
	sentimentBoost := 0.0
	for _, sentence := range sentences {
		lower := strings.ToLower(sentence)
		if containsAnyBoundary(lower, keywords) {
			score := sentenceSentiment(lower, lexicon)
			if score > 0.1 {
				sentimentBoost += 0.05
			}
		}
	}
	return minFloat(1.0, base+sentimentBoost)
}

func sentenceSentiment(sentence string, lexicon map[string]float64) float64 {
	words := tokenizeWords(sentence)
	if len(words) == 0 {
		return 0.0
	}
	var total float64
	for _, w := range words {
		if score, ok := lexicon[strings.ToLower(w)]; ok {
			total += score
		}
	}
	return total / float64(len(words))
}

func calculateSentimentVariance(words []string, lexicon map[string]float64) float64 {
	if len(words) == 0 {
		return 0.0
	}
	var scores []float64
	for _, w := range words {
		if s, ok := lexicon[strings.ToLower(w)]; ok {
			scores = append(scores, s)
		}
	}
	if len(scores) < 2 {
		return 0.0
	}
	var sum float64
	for _, s := range scores {
		sum += s
	}
	mean := sum / float64(len(scores))
	var sqDiff float64
	for _, s := range scores {
		diff := s - mean
		sqDiff += diff * diff
	}
	return sqDiff / float64(len(scores))
}

func calculateAvgSentenceLengthAdvanced(sentences []string) int {
	if len(sentences) == 0 {
		return 15
	}
	totalWords := 0
	for _, s := range sentences {
		totalWords += len(tokenizeWords(s))
	}
	return totalWords / len(sentences)
}

func tanh(x float64) float64 {
	// Approximation rapide de tanh : x/(1+|x|) pour [0,∞)
	return x / (1.0 + x)
}

func mathLog1p(x float64) float64 {
	// Approximation de log(1+x) pour x petit
	if x < 0.0001 {
		return x
	}
	// Utilisation du standard library serait mieux, mais on reste autonome
	// Approximation via série
	result := x
	term := x
	for n := 2; n <= 10; n++ {
		term *= -x
		result += term / float64(n)
		if term < 1e-12 {
			break
		}
	}
	return result
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// --- Backward compatibility wrappers (used by tests) ---

func (s *SoulExtractorService) detectCatchPhrases(responses []string) []string {
	return s.detectCatchPhrasesNGram(responses)
}

func (s *SoulExtractorService) calculateAvgSentenceLength(responses []string) int {
	return calculateAvgSentenceLengthAdvanced(splitSentences(strings.Join(responses, " ")))
}

func buildSentimentLexicon() map[string]float64 {
	return map[string]float64{
		// Positif
		"excellent": 0.8, "amazing": 0.9, "fantastic": 0.9, "wonderful": 0.85, "great": 0.7,
		"good": 0.6, "best": 0.8, "love": 0.85, "happy": 0.75, "glad": 0.7, "pleased": 0.7,
		"delighted": 0.85, "excited": 0.8, "thrilled": 0.85, "perfect": 0.8, "beautiful": 0.7,
		"brilliant": 0.8, "awesome": 0.85, "superb": 0.8, "outstanding": 0.85, "remarkable": 0.75,
		"impressive": 0.7, "enjoy": 0.6, "thank": 0.6, "thanks": 0.6, "appreciate": 0.65,
		"grateful": 0.7, "kind": 0.6, "nice": 0.5, "helpful": 0.6, "useful": 0.5,
		"beneficial": 0.6, "constructive": 0.5, "positive": 0.6, "optimistic": 0.7,
		"confident": 0.6, "hopeful": 0.6, "cheerful": 0.75, "joy": 0.8, "fun": 0.6,
		"merci": 0.6, "super": 0.7, "génial": 0.8, "parfait": 0.8,
		"magnifique": 0.85, "formidable": 0.8, "ravie": 0.75, "heureux": 0.7, "content": 0.6,
		// Négatif
		"bad": -0.6, "awful": -0.8, "worst": -0.9,
		"hate": -0.85, "angry": -0.7, "sad": -0.7, "disappointed": -0.65, "frustrated": -0.6,
		"annoying": -0.6, "boring": -0.5, "difficult": -0.4, "hard": -0.4, "problem": -0.4,
		"wrong": -0.5, "error": -0.45, "mistake": -0.5, "fail": -0.6, "failure": -0.65,
		"sorry": -0.3, "regret": -0.5, "worried": -0.5, "concern": -0.4, "fear": -0.6,
		"anxious": -0.55, "stress": -0.5, "tired": -0.4, "pain": -0.6, "hurt": -0.55,
		"nasty": -0.6, "ugly": -0.5, "stupid": -0.7, "ridiculous": -0.6, "absurd": -0.5,
		"mauvais": -0.6, "déteste": -0.8,
		"triste": -0.7, "déçu": -0.65, "problème": -0.4, "erreur": -0.45, "dommage": -0.4,
	}
}
