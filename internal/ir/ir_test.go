package ir

import "testing"

func TestBlockAdd(t *testing.T) {
	b := &Block{}
	b.Add(&RawStmt{Code: "x := 1"})
	b.Add(&RawStmt{Code: "y := 2"})

	if len(b.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(b.Nodes))
	}
}

func TestBlockAddAll(t *testing.T) {
	b1 := &Block{}
	b1.Add(&RawStmt{Code: "a"})
	b1.Add(&RawStmt{Code: "b"})

	b2 := &Block{}
	b2.Add(&RawStmt{Code: "c"})
	b2.AddAll(b1)

	if len(b2.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(b2.Nodes))
	}
}

func TestBlockAddAllNil(t *testing.T) {
	b := &Block{}
	b.Add(&RawStmt{Code: "x"})
	b.AddAll(nil)

	if len(b.Nodes) != 1 {
		t.Fatalf("expected 1 node after AddAll(nil), got %d", len(b.Nodes))
	}
}

func TestNodeInterface(t *testing.T) {
	// Verify all node types implement Node
	var _ Node = &Block{}
	var _ Node = &Assign{}
	var _ Node = &VarDecl{}
	var _ Node = &IfErrCheck{}
	var _ Node = &Goto{}
	var _ Node = &Label{}
	var _ Node = &ScopedBlock{}
	var _ Node = &RawStmt{}
	var _ Node = &ReturnStmt{}
	var _ Node = &ExprStmt{}
	var _ Node = &Comment{}
}

func TestReturnStmtFields(t *testing.T) {
	r := &ReturnStmt{Values: []string{"0", "err"}}
	if len(r.Values) != 2 {
		t.Errorf("expected 2 values, got %d", len(r.Values))
	}

	bare := &ReturnStmt{}
	if len(bare.Values) != 0 {
		t.Errorf("expected 0 values for bare return, got %d", len(bare.Values))
	}
}

func TestExprStmtFields(t *testing.T) {
	e := &ExprStmt{Expr: "continue"}
	if e.Expr != "continue" {
		t.Errorf("expected 'continue', got %q", e.Expr)
	}
}

func TestCommentFields(t *testing.T) {
	c := &Comment{Text: "kukicha: inferred"}
	if c.Text != "kukicha: inferred" {
		t.Errorf("expected 'kukicha: inferred', got %q", c.Text)
	}
}

func TestAssignFields(t *testing.T) {
	a := &Assign{
		Names:  []string{"x", "err"},
		Expr:   "foo()",
		Walrus: true,
	}
	if len(a.Names) != 2 {
		t.Errorf("expected 2 names, got %d", len(a.Names))
	}
	if !a.Walrus {
		t.Error("expected Walrus=true")
	}
}

func TestIfErrCheckStructure(t *testing.T) {
	body := &Block{}
	body.Add(&RawStmt{Code: "return err"})

	check := &IfErrCheck{
		ErrVar: "err_1",
		Body:   body,
	}

	if check.ErrVar != "err_1" {
		t.Errorf("expected err_1, got %s", check.ErrVar)
	}
	if len(check.Body.Nodes) != 1 {
		t.Errorf("expected 1 body node, got %d", len(check.Body.Nodes))
	}
}
