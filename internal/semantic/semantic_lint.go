package semantic

import "github.com/kukichalang/kukicha/internal/ast"

// LintKind categorizes lint warnings for filtering and configuration.
type LintKind int

const (
	LintDeprecation LintKind = iota
	LintPanic
	LintOnerr
	LintPipe
	LintEnum
	LintTypeMismatch
	LintSecurity
	LintTodo
)

// LintCandidate captures a potential warning during analysis, to be emitted
// in a separate pass after type checking completes.
type LintCandidate struct {
	Kind    LintKind
	Pos     ast.Position
	Message string
}

func (a *Analyzer) recordLint(kind LintKind, pos ast.Position, message string) {
	a.lintCandidates = append(a.lintCandidates, LintCandidate{
		Kind:    kind,
		Pos:     pos,
		Message: message,
	})
}

// emitLintWarnings converts collected lint candidates into warnings.
// Called once at the end of Analyze(), after all type checking is complete.
func (a *Analyzer) emitLintWarnings() {
	for _, lc := range a.lintCandidates {
		a.warn(lc.Pos, lc.Message)
	}
}
