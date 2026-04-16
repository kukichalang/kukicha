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

func TestSkillDeclDescriptionTooLong(t *testing.T) {
	longDesc := strings.Repeat("x", 1025)
	input := "petiole myskill\n\nskill MySkill\n    description: \"" + longDesc + "\"\n"

	result := analyzeSourceResult(t, input)

	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Error(), "description is too long") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected 'description is too long' error, got: %v", result.Errors)
	}
}

func TestSkillDeclDescriptionAtLimit(t *testing.T) {
	desc := strings.Repeat("x", 1024)
	input := "petiole myskill\n\nskill MySkill\n    description: \"" + desc + "\"\n"

	result := analyzeSourceResult(t, input)

	for _, e := range result.Errors {
		if strings.Contains(e.Error(), "description is too long") {
			t.Fatalf("1024-char description should be allowed, got: %v", e)
		}
	}
}

func TestSkillDeclNameTooLong(t *testing.T) {
	// 65 chars after kebab-casing: lowercase so each letter maps 1:1.
	longName := strings.ToUpper(string(rune('A'))) + strings.Repeat("x", 64)
	input := "petiole myskill\n\nskill " + longName + "\n    description: \"ok\"\n"

	result := analyzeSourceResult(t, input)

	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Error(), "name") && strings.Contains(e.Error(), "too long") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected 'name ... too long' error, got: %v", result.Errors)
	}
}
