package lsp

import "testing"

func TestFindCallContext_SimpleCall(t *testing.T) {
	line := "    print(x, y)"
	// Cursor after "x, " -> activeParam = 1
	name, param := findCallContext(line, 14)
	if name != "print" {
		t.Errorf("expected function name 'print', got %q", name)
	}
	if param != 1 {
		t.Errorf("expected active param 1, got %d", param)
	}
}

func TestFindCallContext_FirstArg(t *testing.T) {
	line := "    foo(x"
	name, param := findCallContext(line, 9)
	if name != "foo" {
		t.Errorf("expected 'foo', got %q", name)
	}
	if param != 0 {
		t.Errorf("expected active param 0, got %d", param)
	}
}

func TestFindCallContext_EmptyParens(t *testing.T) {
	line := "    bar()"
	name, param := findCallContext(line, 8) // cursor between parens
	if name != "bar" {
		t.Errorf("expected 'bar', got %q", name)
	}
	if param != 0 {
		t.Errorf("expected active param 0, got %d", param)
	}
}

func TestFindCallContext_ThirdArg(t *testing.T) {
	line := "    add(1, 2, 3)"
	name, param := findCallContext(line, 15)
	if name != "add" {
		t.Errorf("expected 'add', got %q", name)
	}
	if param != 2 {
		t.Errorf("expected active param 2, got %d", param)
	}
}

func TestFindCallContext_NestedCall(t *testing.T) {
	line := "    outer(inner(x), y)"
	// Cursor at y position (after "inner(x), ")
	name, param := findCallContext(line, 21)
	if name != "outer" {
		t.Errorf("expected 'outer', got %q", name)
	}
	if param != 1 {
		t.Errorf("expected active param 1, got %d", param)
	}
}

func TestFindCallContext_NotInCall(t *testing.T) {
	line := "    x := 5"
	name, _ := findCallContext(line, 10)
	if name != "" {
		t.Errorf("expected empty name, got %q", name)
	}
}

func TestFindSignature_Builtin(t *testing.T) {
	store := NewDocumentStore()
	source := "func main()\n    print(\"hello\")\n"
	store.Open("file:///test.kuki", source, 1)
	doc := store.Get("file:///test.kuki")

	sig := findSignature(doc, "print")
	if sig == nil {
		t.Fatal("expected signature for 'print'")
	}
	if sig.Label != "func print(args ...any)" {
		t.Errorf("unexpected label: %s", sig.Label)
	}
	if len(sig.Parameters) != 1 {
		t.Errorf("expected 1 parameter, got %d", len(sig.Parameters))
	}
}

func TestFindSignature_UserFunction(t *testing.T) {
	store := NewDocumentStore()
	source := "func Add(a int, b int) int\n    return a + b\n"
	store.Open("file:///test.kuki", source, 1)
	doc := store.Get("file:///test.kuki")

	sig := findSignature(doc, "Add")
	if sig == nil {
		t.Fatal("expected signature for 'Add'")
	}
	if len(sig.Parameters) != 2 {
		t.Errorf("expected 2 parameters, got %d", len(sig.Parameters))
	}
	if sig.Parameters[0].Label != "a int" {
		t.Errorf("unexpected first param label: %s", sig.Parameters[0].Label)
	}
	if sig.Parameters[1].Label != "b int" {
		t.Errorf("unexpected second param label: %s", sig.Parameters[1].Label)
	}
}

func TestFindSignature_Unknown(t *testing.T) {
	store := NewDocumentStore()
	source := "func main()\n    print(\"hello\")\n"
	store.Open("file:///test.kuki", source, 1)
	doc := store.Get("file:///test.kuki")

	sig := findSignature(doc, "nonexistent")
	if sig != nil {
		t.Errorf("expected nil for unknown function, got %+v", sig)
	}
}
