// communication_test.go - Tests for CommunicationStyle drift calculation
package entities

import "testing"

func TestCalculateCommunicationStyleDrift_NoChange(t *testing.T) {
	old := CommunicationStyle{
		ResponseLength:          LengthModerate,
		InformationDensity:      DensityBalanced,
		AsksClarifyingQuestions: true,
	}
	newStyle := CommunicationStyle{
		ResponseLength:          LengthModerate,
		InformationDensity:      DensityBalanced,
		AsksClarifyingQuestions: true,
	}
	drift := CalculateCommunicationStyleDrift(old, newStyle)
	if drift.Change != 0 {
		t.Errorf("expected 0 change, got %f", drift.Change)
	}
	if drift.IsSignificant {
		t.Error("expected not significant")
	}
}

func TestCalculateCommunicationStyleDrift_SignificantChange(t *testing.T) {
	old := CommunicationStyle{
		ResponseLength:          LengthTerse,
		InformationDensity:      DensitySparse,
		StructurePreference:     StructureFreeform,
		AsksClarifyingQuestions: false,
		ProvidesAlternatives:    false,
		ShowsUncertainty:        false,
	}
	newStyle := CommunicationStyle{
		ResponseLength:          LengthExhaustive,
		InformationDensity:      DensityDense,
		StructurePreference:     StructureSectioned,
		AsksClarifyingQuestions: true,
		ProvidesAlternatives:    true,
		ShowsUncertainty:        true,
	}
	drift := CalculateCommunicationStyleDrift(old, newStyle)
	if drift.Change <= 0.3 {
		t.Errorf("expected significant change > 0.3, got %f", drift.Change)
	}
	if !drift.IsSignificant {
		t.Error("expected significant drift")
	}
}
