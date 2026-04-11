package lsp

import (
	"strings"
	"testing"

	"github.com/sourcegraph/go-lsp"
)

// --- Diagnostics: no false positives for valid untyped composite literals ---

func TestAnalyze_UntypedCompositeLiteral_ReturnContext(t *testing.T) {
	src := "type Config\n    host string\n    port int\n\nfunc NewConfig() Config\n    return {host: \"localhost\", port: 8080}\n"
	doc := newDocument("file:///test.kuki", src, 1)
	if doc.Program == nil {
		t.Fatal("expected non-nil program")
	}
	if len(doc.Errors) > 0 {
		t.Errorf("expected no errors for valid untyped composite literal, got: %v", doc.Errors)
	}
}

func TestAnalyze_UntypedCompositeLiteral_FuncArg(t *testing.T) {
	src := "type User\n    name string\n    age int\n\nfunc Greet(u User)\n    print(u.name)\n\nfunc main()\n    Greet({name: \"Alice\", age: 30})\n"
	doc := newDocument("file:///test.kuki", src, 1)
	if doc.Program == nil {
		t.Fatal("expected non-nil program")
	}
	if len(doc.Errors) > 0 {
		t.Errorf("expected no errors for valid untyped composite literal in func arg, got: %v", doc.Errors)
	}
}

func TestAnalyze_UntypedCompositeLiteral_Assignment(t *testing.T) {
	src := "type Config\n    host string\n    port int\n\nfunc main()\n    c := Config{host: \"\", port: 0}\n    c = {host: \"prod\", port: 443}\n"
	doc := newDocument("file:///test.kuki", src, 1)
	if doc.Program == nil {
		t.Fatal("expected non-nil program")
	}
	if len(doc.Errors) > 0 {
		t.Errorf("expected no errors for untyped literal in assignment, got: %v", doc.Errors)
	}
}

// --- Diagnostics: errors for invalid untyped composite literals ---

func TestAnalyze_UntypedCompositeLiteral_BadField(t *testing.T) {
	src := "type Config\n    host string\n    port int\n\nfunc NewConfig() Config\n    return {hosst: \"localhost\", port: 8080}\n"
	doc := newDocument("file:///test.kuki", src, 1)
	found := false
	for _, err := range doc.Errors {
		if strings.Contains(err.Error(), "unknown field") && strings.Contains(err.Error(), "hosst") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected error about unknown field 'hosst', got: %v", doc.Errors)
	}
}

// --- Hover: field type info inside untyped composite literals ---

func TestHover_UntypedCompositeLiteral_FieldKey(t *testing.T) {
	s := NewServer(nil, nil)
	store := s.documents
	uri := lsp.DocumentURI("file:///test.kuki")

	src := "type Config\n    host string\n    port int\n\nfunc NewConfig() Config\n    return {host: \"localhost\", port: 8080}\n"
	store.Open(uri, src, 1)
	doc := store.Get(uri)

	// Hover over "host" field key inside untyped literal
	content := s.getHoverContent(doc, "host", lsp.Position{Line: 5, Character: 12})
	if content == "" {
		t.Fatal("expected hover content for field key 'host'")
	}
	if !strings.Contains(content, "string") {
		t.Errorf("expected field type 'string' in hover, got: %s", content)
	}
	if !strings.Contains(content, "Config") {
		t.Errorf("expected struct name 'Config' in hover, got: %s", content)
	}
	if !strings.Contains(content, "field") {
		t.Errorf("expected 'field' label in hover, got: %s", content)
	}
}

func TestHover_UntypedCompositeLiteral_FuncArg(t *testing.T) {
	s := NewServer(nil, nil)
	store := s.documents
	uri := lsp.DocumentURI("file:///test.kuki")

	src := "type User\n    name string\n    age int\n\nfunc Greet(u User)\n    print(u.name)\n\nfunc main()\n    Greet({name: \"Alice\", age: 30})\n"
	store.Open(uri, src, 1)
	doc := store.Get(uri)

	// Hover over "name" field key inside untyped literal in func arg
	content := s.getHoverContent(doc, "name", lsp.Position{Line: 8, Character: 11})
	if content == "" {
		t.Fatal("expected hover content for field key 'name' in func arg")
	}
	if !strings.Contains(content, "string") {
		t.Errorf("expected field type 'string', got: %s", content)
	}
	if !strings.Contains(content, "User") {
		t.Errorf("expected struct name 'User', got: %s", content)
	}
}

func TestHover_UntypedCompositeLiteral_TypeName(t *testing.T) {
	s := NewServer(nil, nil)
	store := s.documents
	uri := lsp.DocumentURI("file:///test.kuki")

	src := "type Config\n    host string\n    port int\n\nfunc NewConfig() Config\n    return {host: \"localhost\", port: 8080}\n"
	store.Open(uri, src, 1)
	doc := store.Get(uri)

	// Hover over the type name should still work
	content := s.getHoverContent(doc, "Config", lsp.Position{Line: 0, Character: 5})
	if content == "" {
		t.Fatal("expected hover content for Config type")
	}
	if !strings.Contains(content, "Config") {
		t.Errorf("expected 'Config' in hover, got: %s", content)
	}
}

// --- Completion: struct field suggestions inside untyped composite literals ---

func TestCompletion_UntypedCompositeLiteral_FieldSuggestions(t *testing.T) {
	s := NewServer(nil, nil)
	store := s.documents
	uri := lsp.DocumentURI("file:///test.kuki")

	// Partially written literal with only one field — completion should suggest remaining fields
	src := "type Config\n    host string\n    port int\n\nfunc NewConfig() Config\n    return {host: \"localhost\", port: 8080}\n"
	store.Open(uri, src, 1)
	doc := store.Get(uri)

	items := s.getCompletions(doc, lsp.Position{Line: 5, Character: 12})

	// Should include struct field items
	foundHost := false
	foundPort := false
	for _, item := range items {
		if item.Kind == lsp.CIKField {
			switch item.Label {
			case "host":
				foundHost = true
				if item.Detail != "string" {
					t.Errorf("expected host detail 'string', got %q", item.Detail)
				}
			case "port":
				foundPort = true
				if item.Detail != "int" {
					t.Errorf("expected port detail 'int', got %q", item.Detail)
				}
			}
		}
	}
	// At cursor inside {host: ..., port: ...}, both fields are already used
	// so they should be filtered out.
	if foundHost || foundPort {
		t.Error("expected both fields to be filtered out since they are already used in the literal")
	}
}

func TestCompletion_UntypedCompositeLiteral_FiltersUsedFields(t *testing.T) {
	s := NewServer(nil, nil)
	store := s.documents
	uri := lsp.DocumentURI("file:///test.kuki")

	// Literal with only "host" filled in — "port" should be suggested, "host" should not
	src := "type Config\n    host string\n    port int\n    debug bool\n\nfunc NewConfig() Config\n    return {host: \"localhost\"}\n"
	store.Open(uri, src, 1)
	doc := store.Get(uri)

	items := s.getCompletions(doc, lsp.Position{Line: 6, Character: 26})

	fieldItems := make(map[string]lsp.CompletionItem)
	for _, item := range items {
		if item.Kind == lsp.CIKField {
			fieldItems[item.Label] = item
		}
	}

	if _, ok := fieldItems["host"]; ok {
		t.Error("'host' should be filtered out since it's already used in the literal")
	}
	if _, ok := fieldItems["port"]; !ok {
		t.Error("'port' should be suggested since it's not yet used")
	}
	if _, ok := fieldItems["debug"]; !ok {
		t.Error("'debug' should be suggested since it's not yet used")
	}
}

// --- Go-to-definition: field keys resolve to struct field definitions ---

func TestDefinition_UntypedCompositeLiteral_FieldKey(t *testing.T) {
	s := NewServer(nil, nil)
	store := s.documents
	uri := lsp.DocumentURI("file:///test.kuki")

	src := "type Config\n    host string\n    port int\n\nfunc NewConfig() Config\n    return {host: \"localhost\", port: 8080}\n"
	store.Open(uri, src, 1)
	doc := store.Get(uri)

	// Go-to-definition on "host" should find the struct field definition
	loc := s.findDefinition(doc, "host")
	if loc == nil {
		t.Fatal("expected definition location for field 'host'")
	}
	// The field "host" is defined on line 2 (0-indexed: line 1)
	if loc.Range.Start.Line != 1 {
		t.Errorf("expected definition at line 1, got line %d", loc.Range.Start.Line)
	}
}

// --- Edge case: untyped literal outside function ---

func TestHover_UntypedCompositeLiteral_OutsideFunction(t *testing.T) {
	s := NewServer(nil, nil)
	store := s.documents
	uri := lsp.DocumentURI("file:///test.kuki")

	// No function body — should not crash
	src := "type Config\n    host string\n    port int\n"
	store.Open(uri, src, 1)
	doc := store.Get(uri)

	content := s.getHoverContent(doc, "host", lsp.Position{Line: 1, Character: 4})
	// "host" is not inside a composite literal here, so no field hover.
	// But it shouldn't crash. It may match as a field definition via hover.
	_ = content
}

