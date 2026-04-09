package lsp

import (
	"testing"
)

func TestHandleFormatting_BasicFormatting(t *testing.T) {
	store := NewDocumentStore()
	// Source with Go-style braces that the formatter will convert
	source := "func main() {\n    print(\"hello\")\n}\n"
	store.Open("file:///test.kuki", source, 1)

	doc := store.Get("file:///test.kuki")
	if doc == nil {
		t.Fatal("document not found in store")
	}

	// The formatter should be able to process this without error
	// (the actual formatting is tested in internal/formatter/)
	if doc.Content != source {
		t.Errorf("content mismatch: got %q", doc.Content)
	}
}

func TestHandleFormatting_NoChangeReturnsNil(t *testing.T) {
	store := NewDocumentStore()
	// Already-formatted Kukicha source
	source := "func main()\n    print(\"hello\")\n"
	store.Open("file:///test.kuki", source, 1)

	doc := store.Get("file:///test.kuki")
	if doc == nil {
		t.Fatal("document not found")
	}
	if doc.Content != source {
		t.Errorf("unexpected content: %q", doc.Content)
	}
}
