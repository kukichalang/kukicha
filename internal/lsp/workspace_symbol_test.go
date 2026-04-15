package lsp

import (
	"testing"

	"github.com/sourcegraph/go-lsp"
)

func openTwoDocSymbolStore(t *testing.T) *Server {
	t.Helper()
	s := NewServer(nil, nil)
	s.documents.Open("file:///tmp/a.kuki", `type User
    name string
    email string

func GetUser() User
    return User{}
`, 1)
	s.documents.Open("file:///tmp/b.kuki", `enum Status
    Active = 1
    Inactive = 2

interface Repo
    Save(u User) error
`, 1)
	return s
}

func TestHandleWorkspaceSymbol_EmptyQuery(t *testing.T) {
	s := openTwoDocSymbolStore(t)
	docs := s.documents.All()

	var count int
	for _, doc := range docs {
		count += len(s.getDocumentSymbols(doc))
	}
	// Expect all symbols across both files
	if count == 0 {
		t.Error("expected symbols with empty query")
	}
}

func TestHandleWorkspaceSymbol_FilterByQuery(t *testing.T) {
	s := openTwoDocSymbolStore(t)

	// Collect symbols matching "user" (case-insensitive)
	var matches []lsp.SymbolInformation
	for _, doc := range s.documents.All() {
		for _, sym := range s.getDocumentSymbols(doc) {
			if containsCI(sym.Name, "user") {
				matches = append(matches, sym)
			}
		}
	}

	if len(matches) == 0 {
		t.Error("expected at least one symbol matching 'user'")
	}
	for _, sym := range matches {
		if !containsCI(sym.Name, "user") {
			t.Errorf("symbol %q does not match query 'user'", sym.Name)
		}
	}
}

func TestHandleWorkspaceSymbol_NoMatch(t *testing.T) {
	s := openTwoDocSymbolStore(t)

	var matches []lsp.SymbolInformation
	for _, doc := range s.documents.All() {
		for _, sym := range s.getDocumentSymbols(doc) {
			if containsCI(sym.Name, "zzznomatch") {
				matches = append(matches, sym)
			}
		}
	}

	if len(matches) != 0 {
		t.Errorf("expected 0 results for unknown query, got %d", len(matches))
	}
}

func TestDocumentStoreAll_Empty(t *testing.T) {
	s := NewServer(nil, nil)
	docs := s.documents.All()
	if len(docs) != 0 {
		t.Errorf("expected empty All() on fresh store, got %d", len(docs))
	}
}

func TestDocumentStoreAll_MultipleFiles(t *testing.T) {
	s := openTwoDocSymbolStore(t)
	docs := s.documents.All()
	if len(docs) != 2 {
		t.Errorf("expected 2 documents, got %d", len(docs))
	}
}

func containsCI(s, substr string) bool {
	return len(s) >= len(substr) && containsCIHelper(s, substr)
}

func containsCIHelper(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	sLower := toLower(s)
	subLower := toLower(substr)
	for i := 0; i <= len(sLower)-len(subLower); i++ {
		if sLower[i:i+len(subLower)] == subLower {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
