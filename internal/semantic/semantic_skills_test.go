package semantic

import (
	"strings"
	"testing"
)

func TestSkillDeclValid(t *testing.T) {
	input := `petiole weather

skill WeatherService
    description: "Provides weather data."
    version: "1.0.0"

func GetForecast(city string) string
    return city
`

	analyzer, errors := analyzeSource(t, input)
	_ = analyzer

	if len(errors) > 0 {
		t.Fatalf("expected no errors, got: %v", errors)
	}
}

func TestSkillDeclWithoutPetiole(t *testing.T) {
	input := `skill WeatherService
    description: "Provides weather data."
    version: "1.0.0"
`

	analyzer, errors := analyzeSource(t, input)
	_ = analyzer

	found := false
	for _, e := range errors {
		if strings.Contains(e.Error(), "requires a petiole") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected 'requires a petiole' error, got: %v", errors)
	}
}

func TestSkillDeclLowercaseName(t *testing.T) {
	input := `petiole myskill

skill weatherService
    description: "Provides weather data."
    version: "1.0.0"
`

	analyzer, errors := analyzeSource(t, input)
	_ = analyzer

	found := false
	for _, e := range errors {
		if strings.Contains(e.Error(), "must be exported") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected 'must be exported' error, got: %v", errors)
	}
}

func TestSkillDeclEmptyDescription(t *testing.T) {
	input := `petiole myskill

skill MySkill
    version: "1.0.0"
`

	analyzer, errors := analyzeSource(t, input)
	_ = analyzer

	found := false
	for _, e := range errors {
		if strings.Contains(e.Error(), "should have a description") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected 'should have a description' error, got: %v", errors)
	}
}

func TestSkillDeclBadSemver(t *testing.T) {
	input := `petiole myskill

skill MySkill
    description: "A skill."
    version: "not-a-version"
`

	analyzer, errors := analyzeSource(t, input)
	_ = analyzer

	found := false
	for _, e := range errors {
		if strings.Contains(e.Error(), "should follow semver") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected 'should follow semver' error, got: %v", errors)
	}
}
