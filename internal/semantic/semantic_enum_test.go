package semantic

import (
	"strings"
	"testing"
)

func TestEnumDecl_Valid(t *testing.T) {
	input := `enum Status
    OK = 200
    NotFound = 404

func main()
    s := Status.OK
`
	_, errs := analyzeSource(t, input)
	if len(errs) > 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestEnumDecl_StringEnum(t *testing.T) {
	input := `enum LogLevel
    Debug = "debug"
    Info = "info"
`
	_, errs := analyzeSource(t, input)
	if len(errs) > 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestEnumDecl_MixedTypes(t *testing.T) {
	input := `enum Bad
    A = 1
    B = "two"
`
	_, errs := analyzeSource(t, input)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "mixes value types") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'mixes value types' error, got: %v", errs)
	}
}

func TestEnumDecl_ZeroValueWarning(t *testing.T) {
	input := `enum Status
    OK = 200
    NotFound = 404
`
	result := analyzeSourceResult(t, input)
	warnings := result.Warnings
	found := false
	for _, w := range warnings {
		if strings.Contains(w.Error(), "no case with value 0") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected zero-value warning, got warnings: %v", warnings)
	}
}

func TestEnumDecl_NoZeroValueWarningWhenPresent(t *testing.T) {
	input := `enum Color
    Unknown = 0
    Red = 1
    Blue = 2
`
	result := analyzeSourceResult(t, input)
	for _, w := range result.Warnings {
		if strings.Contains(w.Error(), "no case with value 0") {
			t.Errorf("unexpected zero-value warning: %v", w)
		}
	}
}

func TestEnumDecl_NoZeroValueWarningForStringEnum(t *testing.T) {
	input := `enum LogLevel
    Debug = "debug"
    Info = "info"
`
	result := analyzeSourceResult(t, input)
	for _, w := range result.Warnings {
		if strings.Contains(w.Error(), "no case with value 0") {
			t.Errorf("unexpected zero-value warning for string enum: %v", w)
		}
	}
}

func TestEnumDecl_ExhaustivenessWarning(t *testing.T) {
	input := `enum Status
    OK = 0
    NotFound = 1
    Error = 2

func handle(s Status)
    switch s
        when Status.OK
            return
        when Status.NotFound
            return
`
	result := analyzeSourceResult(t, input)
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w.Error(), "missing cases") && strings.Contains(w.Error(), "Status.Error") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected exhaustiveness warning mentioning Status.Error, got warnings: %v", result.Warnings)
	}
}

func TestEnumDecl_ExhaustivenessNoWarningWithOtherwise(t *testing.T) {
	input := `enum Status
    OK = 0
    NotFound = 1
    Error = 2

func handle(s Status)
    switch s
        when Status.OK
            return
        otherwise
            return
`
	result := analyzeSourceResult(t, input)
	for _, w := range result.Warnings {
		if strings.Contains(w.Error(), "missing cases") {
			t.Errorf("unexpected exhaustiveness warning with otherwise clause: %v", w)
		}
	}
}

func TestEnumDecl_ExhaustivenessAllCoveered(t *testing.T) {
	input := `enum Dir
    Up = 0
    Down = 1

func handle(d Dir)
    switch d
        when Dir.Up
            return
        when Dir.Down
            return
`
	result := analyzeSourceResult(t, input)
	for _, w := range result.Warnings {
		if strings.Contains(w.Error(), "missing cases") {
			t.Errorf("unexpected exhaustiveness warning when all cases covered: %v", w)
		}
	}
}

func TestEnumDecl_InvalidCaseAccess(t *testing.T) {
	input := `enum Status
    OK = 200

func main()
    s := Status.Missing
`
	_, errs := analyzeSource(t, input)
	found := false
	for _, e := range errs {
		if strings.Contains(e.Error(), "is not a case of enum Status") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'is not a case of enum' error, got: %v", errs)
	}
}
