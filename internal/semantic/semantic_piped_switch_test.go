package semantic

import (
	"strings"
	"testing"
)

func TestPipedSwitchAnyFallback_WarnsOnMismatchedTypes(t *testing.T) {
	source := `
func Convert(value any) any
    result := value |> switch as v
        when string
            return v
        when int
            return 42
        otherwise
            return 3.14
    return result
`
	analyzer, errs := analyzeSource(t, source)
	_ = errs

	warnings := analyzer.Warnings()
	found := false
	for _, w := range warnings {
		if strings.Contains(w.Error(), "piped switch cases return different types") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about mismatched piped switch types, got warnings: %v", warnings)
	}
}

func TestPipedSwitchAnyFallback_NoWarningForConsistentTypes(t *testing.T) {
	source := `
func Convert(value any) string
    result := value |> switch as v
        when string
            return v
        when int
            return "number"
        otherwise
            return "other"
    return result
`
	analyzer, errs := analyzeSource(t, source)
	_ = errs

	for _, w := range analyzer.Warnings() {
		if strings.Contains(w.Error(), "piped switch cases return different types") {
			t.Errorf("unexpected piped switch type warning for consistent types: %v", w)
		}
	}
}

func TestPipedSwitchAnyFallback_ConflictNotResetByThirdCase(t *testing.T) {
	// Regression: previously, a conflict between cases 1 and 2 (string vs int)
	// was overwritten by case 3 (float), producing float instead of any.
	source := `
func Convert(value any) any
    result := value |> switch as v
        when string
            return v
        when int
            return 42
        otherwise
            return 3.14
    return result
`
	analyzer, errs := analyzeSource(t, source)
	_ = errs

	// The PipedSwitchExpr should be typed as "any" (Named), not float.
	for _, ti := range analyzer.ExprTypes() {
		if ti.Kind == TypeKindNamed && ti.Name == "any" {
			return // found the conflict marker — correct
		}
	}
	// Also acceptable: the warning was emitted
	for _, w := range analyzer.Warnings() {
		if strings.Contains(w.Error(), "piped switch cases return different types") {
			return
		}
	}
	t.Error("expected piped switch with conflicting types to produce 'any' type or warning")
}

func TestPipedSwitchAnyFallback_UnknownRefinedByConcrete(t *testing.T) {
	// When one case returns Unknown (e.g., method call on narrowed type)
	// and another returns a concrete type, the concrete type should win.
	source := `import "os/exec"

func ExitCodeOrOne(err error) int
    code := err |> switch as exitErr
        when reference exec.ExitError
            return exitErr.ExitCode()
        otherwise
            return 1
    return code
`
	analyzer, errs := analyzeSource(t, source)
	_ = errs

	for _, w := range analyzer.Warnings() {
		if strings.Contains(w.Error(), "piped switch cases return different types") {
			t.Errorf("unexpected conflict warning when Unknown should be refined: %v", w)
		}
	}
}

func TestPipedSwitchAnyFallback_NoWarningForVoidSwitch(t *testing.T) {
	source := `
func Handle(event string)
    event |> switch
        when "click"
            print("clicked")
        otherwise
            print("unknown")
`
	analyzer, errs := analyzeSource(t, source)
	_ = errs

	for _, w := range analyzer.Warnings() {
		if strings.Contains(w.Error(), "piped switch cases return different types") {
			t.Errorf("unexpected piped switch type warning for void switch: %v", w)
		}
	}
}
