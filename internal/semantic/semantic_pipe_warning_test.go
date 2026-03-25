package semantic

import (
	"strings"
	"testing"
)

func TestPipeDiscardedError_WarnsWithoutOnerr(t *testing.T) {
	source := `
import "strconv"

func Process(input string) int
    result := strconv.Atoi(input) |> doSomething()
    return result

func doSomething(n int) int
    return n + 1
`
	analyzer, errs := analyzeSource(t, source)
	// Ignore semantic errors about unknown functions — we care about warnings
	_ = errs

	warnings := analyzer.Warnings()
	found := false
	for _, w := range warnings {
		if strings.Contains(w.Error(), "pipe discards error") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about discarded pipe error, got warnings: %v", warnings)
	}
}

func TestPipeDiscardedError_NoWarningWithOnerr(t *testing.T) {
	source := `
import "strconv"

func Process(input string) int
    result := strconv.Atoi(input) |> doSomething() onerr panic "failed"
    return result

func doSomething(n int) int
    return n + 1
`
	analyzer, errs := analyzeSource(t, source)
	_ = errs

	for _, w := range analyzer.Warnings() {
		if strings.Contains(w.Error(), "pipe discards error") {
			t.Errorf("unexpected pipe discard warning when onerr is present: %v", w)
		}
	}
}

func TestPipeDiscardedError_WarnsOnIntermediateStep(t *testing.T) {
	// In a chain a |> b() |> c(), warn if b() returns (T, error)
	source := `
import "strconv"

func Process(input string) string
    result := strconv.Atoi(input) |> addOne() |> toString()
    return result

func addOne(n int) int
    return n + 1

func toString(n int) string
    return "{n}"
`
	analyzer, errs := analyzeSource(t, source)
	_ = errs

	warnings := analyzer.Warnings()
	found := false
	for _, w := range warnings {
		if strings.Contains(w.Error(), "pipe discards error") && strings.Contains(w.Error(), "Atoi") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning mentioning Atoi, got warnings: %v", warnings)
	}
}

func TestPipeDiscardedError_NoWarningForSingleReturnStep(t *testing.T) {
	// No warning when intermediate steps return single values
	source := `
func Process(input int) int
    result := addOne(input) |> addOne()
    return result

func addOne(n int) int
    return n + 1
`
	analyzer, errs := analyzeSource(t, source)
	_ = errs

	for _, w := range analyzer.Warnings() {
		if strings.Contains(w.Error(), "pipe discards error") {
			t.Errorf("unexpected pipe discard warning for single-return step: %v", w)
		}
	}
}

func TestPipeDiscardedError_WarnsOnAssign(t *testing.T) {
	source := `
import "strconv"

func Process(input string) int
    result := 0
    result = strconv.Atoi(input) |> addOne()
    return result

func addOne(n int) int
    return n + 1
`
	analyzer, errs := analyzeSource(t, source)
	_ = errs

	warnings := analyzer.Warnings()
	found := false
	for _, w := range warnings {
		if strings.Contains(w.Error(), "pipe discards error") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about discarded pipe error on assignment, got warnings: %v", warnings)
	}
}

func TestPipeDiscardedError_NoWarningOnAssignWithOnerr(t *testing.T) {
	source := `
import "strconv"

func Process(input string) int
    result := 0
    result = strconv.Atoi(input) |> addOne() onerr panic "fail"
    return result

func addOne(n int) int
    return n + 1
`
	analyzer, errs := analyzeSource(t, source)
	_ = errs

	for _, w := range analyzer.Warnings() {
		if strings.Contains(w.Error(), "pipe discards error") {
			t.Errorf("unexpected pipe discard warning when onerr is present: %v", w)
		}
	}
}
