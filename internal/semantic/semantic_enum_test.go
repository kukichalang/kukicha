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
