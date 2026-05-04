// behavior_test.go - Tests for BehavioralSignature drift calculation
package entities

import "testing"

func TestCalculateBehavioralSignatureDrift_NoChange(t *testing.T) {
	old := BehavioralSignature{
		AdaptationSpeed:  0.6,
		CuriosityLevel:   0.7,
		PersistenceLevel: 0.5,
	}
	newSig := BehavioralSignature{
		AdaptationSpeed:  0.6,
		CuriosityLevel:   0.7,
		PersistenceLevel: 0.5,
	}
	drift := CalculateBehavioralSignatureDrift(old, newSig)
	if drift.Change != 0 {
		t.Errorf("expected 0 change, got %f", drift.Change)
	}
}

func TestCalculateBehavioralSignatureDrift_SignificantChange(t *testing.T) {
	old := BehavioralSignature{
		AdaptationSpeed:       0.1,
		PatternRecognition:    0.1,
		CuriosityLevel:        0.1,
		ExplorationTendency:   0.1,
		PersistenceLevel:      0.1,
		ErrorHandlingStyle:    ErrorImmediate,
		AdmitsMistakes:        false,
		SelfCorrectionPattern: SelfCorrectImmediate,
		DisagreementStyle:     DisagreeDirect,
	}
	newSig := BehavioralSignature{
		AdaptationSpeed:       0.9,
		PatternRecognition:    0.9,
		CuriosityLevel:        0.9,
		ExplorationTendency:   0.9,
		PersistenceLevel:      0.9,
		ErrorHandlingStyle:    ErrorApologetic,
		AdmitsMistakes:        true,
		SelfCorrectionPattern: SelfCorrectGradual,
		DisagreementStyle:     DisagreePolite,
	}
	drift := CalculateBehavioralSignatureDrift(old, newSig)
	if drift.Change < 0.5 {
		t.Errorf("expected significant change >= 0.5, got %f", drift.Change)
	}
	if !drift.IsSignificant {
		t.Error("expected significant drift")
	}
}
