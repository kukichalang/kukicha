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

	result := analyzeSourceResult(t, input)

	if len(result.Errors) > 0 {
		t.Fatalf("expected no errors, got: %v", result.Errors)
	}
}

func TestSkillDeclWithoutPetiole(t *testing.T) {
	input := `skill WeatherService
    description: "Provides weather data."
    version: "1.0.0"
`

	result := analyzeSourceResult(t, input)

	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Error(), "requires a petiole") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected 'requires a petiole' error, got: %v", result.Errors)
	}
}

func TestSkillDeclLowercaseName(t *testing.T) {
	input := `petiole myskill

skill weatherService
    description: "Provides weather data."
    version: "1.0.0"
`

	result := analyzeSourceResult(t, input)

	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Error(), "must be exported") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected 'must be exported' error, got: %v", result.Errors)
	}
}

func TestSkillDeclEmptyDescription(t *testing.T) {
	input := `petiole myskill

skill MySkill
    version: "1.0.0"
`

	result := analyzeSourceResult(t, input)

	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Error(), "should have a description") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected 'should have a description' error, got: %v", result.Errors)
	}
}

func TestSkillDeclBadSemver(t *testing.T) {
	input := `petiole myskill

skill MySkill
    description: "A skill."
    version: "not-a-version"
`

	result := analyzeSourceResult(t, input)

	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Error(), "should follow semver") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected 'should follow semver' error, got: %v", result.Errors)
	}
}
