package blend

import "go/token"

// Suggestion represents a single transformation that kukicha-blend can apply
// to convert Go code into Kukicha idioms.
type Suggestion struct {
	// Pattern category (e.g., "operators", "comparisons", "types", "onerr").
	Pattern string

	// Human-readable description of the suggestion.
	Message string

	// Source location.
	File    string
	Line    int
	Col     int
	EndLine int
	EndCol  int

	// Byte offsets into the original source for text replacement.
	Start int
	End   int

	// The original Go text and its Kukicha replacement.
	Original    string
	Replacement string
}

// PatternSet controls which transformation patterns are active.
type PatternSet struct {
	Operators   bool // &&, ||, !
	Comparisons bool // ==, !=, nil
	Types       bool // []T, map[K]V, *T, &x
	Onerr       bool // if err != nil { return ... }
	Package     bool // package → petiole
}

// AllPatterns returns a PatternSet with every pattern enabled.
func AllPatterns() PatternSet {
	return PatternSet{
		Operators:   true,
		Comparisons: true,
		Types:       true,
		Onerr:       true,
		Package:     true,
	}
}

// ParsePatterns parses a comma-separated pattern string (e.g., "operators,onerr")
// into a PatternSet. An empty string enables all patterns.
func ParsePatterns(s string) PatternSet {
	if s == "" {
		return AllPatterns()
	}
	ps := PatternSet{}
	for _, p := range splitCSV(s) {
		switch p {
		case "operators":
			ps.Operators = true
		case "comparisons":
			ps.Comparisons = true
		case "types":
			ps.Types = true
		case "onerr":
			ps.Onerr = true
		case "package":
			ps.Package = true
		}
	}
	return ps
}

func splitCSV(s string) []string {
	var parts []string
	start := 0
	for i := range len(s) {
		if s[i] == ',' {
			p := trim(s[start:i])
			if p != "" {
				parts = append(parts, p)
			}
			start = i + 1
		}
	}
	if p := trim(s[start:]); p != "" {
		parts = append(parts, p)
	}
	return parts
}

func trim(s string) string {
	i, j := 0, len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t') {
		j--
	}
	return s[i:j]
}

// posToOffset converts a token.Pos to a byte offset in the source.
func posToOffset(fset *token.FileSet, pos token.Pos) int {
	return fset.Position(pos).Offset
}
