package semantic

import (
	"strings"
	"testing"
)

func TestOnerrBlockErrInterpolationIsError(t *testing.T) {
	input := `func readFile(path string) (string, error)
    return path, empty

func Process(path string) (string, error)
    data := readFile(path) onerr
        return "", error "{err}"
    return data, empty
`

	result := analyzeSourceResult(t, input)

	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Error(), "use {error} not {err} inside onerr") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected 'use {error} not {err} inside onerr' error, got: %v", result.Errors)
	}
}

func TestOnerrInlineErrInterpolationIsError(t *testing.T) {
	input := `func readFile(path string) (string, error)
    return path, empty

func Process(path string) (string, error)
    data := readFile(path) onerr return "", error "{err}"
    return data, empty
`

	result := analyzeSourceResult(t, input)

	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Error(), "use {error} not {err} inside onerr") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected 'use {error} not {err} inside onerr' error, got: %v", result.Errors)
	}
}

func TestOnerrErrorInterpolationIsValid(t *testing.T) {
	input := `func readFile(path string) (string, error)
    return path, empty

func Process(path string) (string, error)
    data := readFile(path) onerr return "", error "{error}"
    return data, empty
`

	result := analyzeSourceResult(t, input)

	for _, e := range result.Errors {
		if strings.Contains(e.Error(), "use {error} not {err} inside onerr") {
			t.Fatalf("unexpected onerr interpolation error: %v", e)
		}
	}
}

func TestOnerrAliasInterpolationIsValid(t *testing.T) {
	input := `func readFile(path string) (string, error)
    return path, empty

func Process(path string) (string, error)
    data := readFile(path) onerr as myErr
        return "", error "{myErr}"
    return data, empty
`

	result := analyzeSourceResult(t, input)

	for _, e := range result.Errors {
		if strings.Contains(e.Error(), "undefined identifier 'myErr'") {
			t.Fatalf("unexpected alias interpolation error: %v", e)
		}
	}
}

func TestStringInterpolationUndefinedIdentifierIsError(t *testing.T) {
	input := `func Process() string
    return "Hello {name}"
`

	result := analyzeSourceResult(t, input)

	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Error(), "undefined identifier 'name'") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected undefined identifier error for interpolation, got: %v", result.Errors)
	}
}

func TestStringInterpolationDefinedIdentifierIsValid(t *testing.T) {
	input := `func Process(name string) string
    return "Hello {name}"
`

	result := analyzeSourceResult(t, input)

	for _, e := range result.Errors {
		if strings.Contains(e.Error(), "undefined identifier 'name'") {
			t.Fatalf("unexpected interpolation identifier error: %v", e)
		}
	}
}

func TestComplexStringInterpolationAnalyzed(t *testing.T) {
	input := `type User
    name string

func main()
    u := User{name: "Alice"}
    msg := "Hello {u.name}!"
    print(msg)
`

	result := analyzeSourceResult(t, input)

	// Should not report errors — u.name is valid
	for _, e := range result.Errors {
		t.Errorf("unexpected error: %v", e)
	}
}

func TestComplexStringInterpolationUndefinedError(t *testing.T) {
	input := `func main()
    msg := "value is {unknown.field}"
    print(msg)
`

	result := analyzeSourceResult(t, input)

	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Error(), "undefined identifier 'unknown'") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected undefined identifier error for 'unknown' in interpolation, got: %v", result.Errors)
	}
}
