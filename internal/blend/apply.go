package blend

import (
	"fmt"
	"sort"
	"strings"
)

// Apply takes suggestions and the original source, applies all replacements,
// and returns the transformed source. Suggestions are applied back-to-front
// (by descending offset) so earlier offsets remain valid.
//
// Overlapping suggestions are skipped — the first (outermost) one wins.
func Apply(src []byte, suggestions []Suggestion) []byte {
	if len(suggestions) == 0 {
		return src
	}

	// Sort descending by Start offset; for ties, larger spans first.
	sorted := make([]Suggestion, len(suggestions))
	copy(sorted, suggestions)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Start != sorted[j].Start {
			return sorted[i].Start > sorted[j].Start
		}
		return sorted[i].End > sorted[j].End
	})

	result := make([]byte, len(src))
	copy(result, src)

	lastStart := len(result) // track the lowest start offset we've touched
	for _, s := range sorted {
		if s.Start < 0 || s.End > len(result) || s.Start >= s.End {
			continue
		}
		// Skip if this overlaps with a previously applied replacement
		if s.End > lastStart {
			continue
		}

		// Replace in the result buffer
		before := result[:s.Start]
		after := result[s.End:]
		replacement := []byte(s.Replacement)

		newResult := make([]byte, 0, len(before)+len(replacement)+len(after))
		newResult = append(newResult, before...)
		newResult = append(newResult, replacement...)
		newResult = append(newResult, after...)
		result = newResult

		lastStart = s.Start
	}

	return result
}

// Diff produces a unified diff between the original source and the blended
// output. The diff uses the given filenames for the header.
func Diff(originalName, blendedName string, original, blended []byte) string {
	origLines := splitLines(string(original))
	newLines := splitLines(string(blended))

	var b strings.Builder
	b.WriteString(fmt.Sprintf("--- %s\n", originalName))
	b.WriteString(fmt.Sprintf("+++ %s\n", blendedName))

	hunks := computeHunks(origLines, newLines, 3)
	for _, h := range hunks {
		b.WriteString(h)
	}

	return b.String()
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.SplitAfter(s, "\n")
	// Remove trailing empty string if the file ends with newline
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// computeHunks generates unified diff hunks with the given context lines.
func computeHunks(a, b []string, context int) []string {
	edits := diffLines(a, b)
	if len(edits) == 0 {
		return nil
	}

	// Check if there are any actual changes
	hasChanges := false
	for _, e := range edits {
		if e.kind != editKeep {
			hasChanges = true
			break
		}
	}
	if !hasChanges {
		return nil
	}

	var ann []annotated
	oi, ni := 0, 0
	for _, e := range edits {
		switch e.kind {
		case editKeep:
			ann = append(ann, annotated{editKeep, oi, ni})
			oi++
			ni++
		case editDelete:
			ann = append(ann, annotated{editDelete, oi, -1})
			oi++
		case editInsert:
			ann = append(ann, annotated{editInsert, -1, ni})
			ni++
		}
	}

	// Find change regions (runs of non-keep edits)
	type region struct{ start, end int }
	var regions []region
	i := 0
	for i < len(ann) {
		if ann[i].kind != editKeep {
			start := i
			for i < len(ann) && ann[i].kind != editKeep {
				i++
			}
			regions = append(regions, region{start, i})
		} else {
			i++
		}
	}

	// Merge regions that are within 2*context of each other
	var merged []region
	for _, r := range regions {
		if len(merged) > 0 && r.start-merged[len(merged)-1].end <= 2*context {
			merged[len(merged)-1].end = r.end
		} else {
			merged = append(merged, r)
		}
	}

	// Generate hunks
	var hunks []string
	for _, r := range merged {
		// Expand with context
		hunkStart := r.start - context
		if hunkStart < 0 {
			hunkStart = 0
		}
		hunkEnd := r.end + context
		if hunkEnd > len(ann) {
			hunkEnd = len(ann)
		}

		// Count orig/new lines and determine start line numbers
		origStart, newStart := -1, -1
		origCount, newCount := 0, 0
		var body strings.Builder

		for j := hunkStart; j < hunkEnd; j++ {
			e := ann[j]
			switch e.kind {
			case editKeep:
				if origStart == -1 {
					origStart = e.origIdx
					newStart = e.newIdx
				}
				body.WriteString(" " + ensureNewline(a[e.origIdx]))
				origCount++
				newCount++
			case editDelete:
				if origStart == -1 {
					origStart = e.origIdx
					// Find the new line index from nearest context
					newStart = findNewStart(ann, j)
				}
				body.WriteString("-" + ensureNewline(a[e.origIdx]))
				origCount++
			case editInsert:
				if origStart == -1 {
					origStart = findOrigStart(ann, j)
					newStart = e.newIdx
				}
				body.WriteString("+" + ensureNewline(b[e.newIdx]))
				newCount++
			}
		}

		header := fmt.Sprintf("@@ -%d,%d +%d,%d @@\n",
			origStart+1, origCount, newStart+1, newCount)
		hunks = append(hunks, header+body.String())
	}

	return hunks
}

func findNewStart(ann []annotated, from int) int {
	// Look backwards for a keep to find the new line index
	for i := from - 1; i >= 0; i-- {
		if ann[i].kind == editKeep {
			return ann[i].newIdx + 1
		}
	}
	return 0
}

func findOrigStart(ann []annotated, from int) int {
	for i := from - 1; i >= 0; i-- {
		if ann[i].kind == editKeep {
			return ann[i].origIdx + 1
		}
	}
	return 0
}

type annotated struct {
	kind    editKind
	origIdx int
	newIdx  int
}

func ensureNewline(s string) string {
	if len(s) > 0 && s[len(s)-1] != '\n' {
		return s + "\n"
	}
	return s
}

type editKind int

const (
	editKeep editKind = iota
	editDelete
	editInsert
)

type edit struct {
	kind editKind
}

// diffLines computes the edit script to transform a into b using a simple
// Myers-style diff algorithm.
func diffLines(a, b []string) []edit {
	n, m := len(a), len(b)
	if n == 0 && m == 0 {
		return nil
	}

	// Build LCS table
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else {
				dp[i][j] = max(dp[i+1][j], dp[i][j+1])
			}
		}
	}

	// Trace back to produce edits
	var edits []edit
	i, j := 0, 0
	for i < n && j < m {
		if a[i] == b[j] {
			edits = append(edits, edit{editKeep})
			i++
			j++
		} else if dp[i+1][j] >= dp[i][j+1] {
			edits = append(edits, edit{editDelete})
			i++
		} else {
			edits = append(edits, edit{editInsert})
			j++
		}
	}
	for i < n {
		edits = append(edits, edit{editDelete})
		i++
	}
	for j < m {
		edits = append(edits, edit{editInsert})
		j++
	}

	return edits
}
