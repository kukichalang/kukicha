package lsp

import (
	"testing"

	"github.com/sourcegraph/go-lsp"
)

func TestFindAllReferences_Function(t *testing.T) {
	s := NewServer(nil, nil)
	uri := lsp.DocumentURI("file:///tmp/test.kuki")
	s.documents.Open(uri, `func greet(name string) string
    return "hello " + name

func main()
    msg := greet("world")
    other := greet("bob")
`, 1)

	locs := s.findAllReferences("greet", true)

	// Should find: declaration (line 0), 2 call sites (lines 4, 5)
	if len(locs) != 3 {
		t.Errorf("expected 3 references, got %d: %v", len(locs), locs)
	}
}

func TestFindAllReferences_ExcludeDeclaration(t *testing.T) {
	s := NewServer(nil, nil)
	uri := lsp.DocumentURI("file:///tmp/test.kuki")
	s.documents.Open(uri, `func greet(name string) string
    return "hello " + name

func main()
    msg := greet("world")
    other := greet("bob")
`, 1)

	locs := s.findAllReferences("greet", false)

	// Should find only call sites, not declaration
	if len(locs) != 2 {
		t.Errorf("expected 2 references (no declaration), got %d: %v", len(locs), locs)
	}
	for _, loc := range locs {
		if loc.Range.Start.Line == 0 {
			t.Errorf("declaration line 0 should be excluded")
		}
	}
}

func TestFindAllReferences_Variable(t *testing.T) {
	s := NewServer(nil, nil)
	uri := lsp.DocumentURI("file:///tmp/test.kuki")
	s.documents.Open(uri, `func main()
    count := 0
    count = count + 1
    count = count + 2
`, 1)

	locs := s.findAllReferences("count", true)

	// Declaration (line 1) + 4 uses (2 assignments with lhs+rhs each)
	if len(locs) < 3 {
		t.Errorf("expected at least 3 references to 'count', got %d", len(locs))
	}
}

func TestFindAllReferences_MultiDoc(t *testing.T) {
	s := NewServer(nil, nil)
	uri1 := lsp.DocumentURI("file:///tmp/a.kuki")
	uri2 := lsp.DocumentURI("file:///tmp/b.kuki")

	s.documents.Open(uri1, `func add(x int, y int) int
    return x + y
`, 1)
	s.documents.Open(uri2, `func main()
    result := add(1, 2)
`, 1)

	locs := s.findAllReferences("add", true)

	if len(locs) != 2 {
		t.Errorf("expected 2 references across 2 docs, got %d", len(locs))
	}
}

func TestFindAllReferences_EmptyWord(t *testing.T) {
	s := NewServer(nil, nil)
	uri := lsp.DocumentURI("file:///tmp/test.kuki")
	s.documents.Open(uri, "func main()\n    x := 1\n", 1)

	locs := s.findAllReferences("", true)
	if len(locs) != 0 {
		t.Errorf("expected 0 results for empty word, got %d", len(locs))
	}
}

func TestFindAllReferences_NilProgram(t *testing.T) {
	s := NewServer(nil, nil)

	locs := s.findAllReferences("anything", true)
	if len(locs) != 0 {
		t.Errorf("expected 0 results with no open docs, got %d", len(locs))
	}
}
