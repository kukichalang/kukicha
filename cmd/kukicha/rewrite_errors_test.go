package main

import (
	"strings"
	"testing"
)

func TestRewriteGoErrors_Basic(t *testing.T) {
	stderr := []byte("/tmp/kukicha-run-123.go:10:5: undefined: foo")
	result := rewriteGoErrors(stderr, "/tmp/kukicha-run-123.go", "/home/user/app.kuki")
	expected := "/home/user/app.kuki: undefined: foo"
	if string(result) != expected {
		t.Errorf("got %q, want %q", string(result), expected)
	}
}

func TestRewriteGoErrors_MultipleOccurrences(t *testing.T) {
	stderr := []byte("/tmp/run.go:1: error1\n/tmp/run.go:5: error2\n")
	result := rewriteGoErrors(stderr, "/tmp/run.go", "app.kuki")
	expected := "app.kuki: error1\napp.kuki: error2\n"
	if string(result) != expected {
		t.Errorf("got %q, want %q", string(result), expected)
	}
}

func TestRewriteGoErrors_EmptyStderr(t *testing.T) {
	result := rewriteGoErrors([]byte{}, "/tmp/run.go", "app.kuki")
	if len(result) != 0 {
		t.Errorf("expected empty result for empty input, got %q", string(result))
	}
}

func TestRewriteGoErrors_NoMatch(t *testing.T) {
	stderr := []byte("some other error message\n")
	result := rewriteGoErrors(stderr, "/tmp/run.go", "app.kuki")
	if string(result) != string(stderr) {
		t.Errorf("expected unchanged output when no match, got %q", string(result))
	}
}

func TestRewriteGoErrors_NilStderr(t *testing.T) {
	result := rewriteGoErrors(nil, "/tmp/run.go", "app.kuki")
	if result != nil {
		t.Errorf("expected nil for nil input, got %q", string(result))
	}
}

func TestRewriteGoErrors_ImportFailure(t *testing.T) {
	stderr := []byte(`/tmp/run.go:3:8: cannot find package "foo"`)
	result := rewriteGoErrors(stderr, "/tmp/run.go", "app.kuki")
	expected := `app.kuki: cannot find package "foo"`
	if string(result) != expected {
		t.Errorf("got %q, want %q", string(result), expected)
	}
}

func TestRewriteGoErrors_LinkerError(t *testing.T) {
	stderr := []byte("/tmp/run.go: undefined: Foo")
	result := rewriteGoErrors(stderr, "/tmp/run.go", "app.kuki")
	expected := "app.kuki: undefined: Foo"
	if string(result) != expected {
		t.Errorf("got %q, want %q", string(result), expected)
	}
}

func TestRewriteGoErrors_PackageError(t *testing.T) {
	stderr := []byte("/tmp/run.go:1:1: expected 'package', found 'EOF'")
	result := rewriteGoErrors(stderr, "/tmp/run.go", "app.kuki")
	expected := "app.kuki: expected 'package', found 'EOF'"
	if string(result) != expected {
		t.Errorf("got %q, want %q", string(result), expected)
	}
}

func TestRewriteGoErrors_LineOnlyNoCol(t *testing.T) {
	stderr := []byte("/tmp/run.go:42: some error")
	result := rewriteGoErrors(stderr, "/tmp/run.go", "app.kuki")
	expected := "app.kuki: some error"
	if string(result) != expected {
		t.Errorf("got %q, want %q", string(result), expected)
	}
}

func TestRewriteGoErrors_PreservesKukiPositions(t *testing.T) {
	input := "/home/user/app.kuki:15:3: real semantic error\n"
	stderr := []byte(input)
	result := rewriteGoErrors(stderr, "/tmp/run.go", "/home/user/app.kuki")
	if string(result) != input {
		t.Errorf("got %q, want %q", string(result), input)
	}
}

func TestRewriteVarNames_MatchingVars(t *testing.T) {
	stderr := []byte("panic: runtime error: invalid memory address\npipe_1 = nil\n")
	varMap := map[string]string{
		"pipe_1": "line 10: fetchData(...)",
		"err_2":  "line 10: onerr",
	}
	result := string(rewriteVarNames(stderr, varMap))
	if !strings.Contains(result, "kukicha: variable hints:") {
		t.Errorf("expected variable hints section, got: %s", result)
	}
	if !strings.Contains(result, "pipe_1 = line 10: fetchData(...)") {
		t.Errorf("expected pipe_1 hint, got: %s", result)
	}
	// err_2 is not in stderr, so should not appear
	if strings.Contains(result, "err_2") {
		t.Errorf("err_2 should not appear (not in stderr), got: %s", result)
	}
}

func TestRewriteVarNames_NoMatch(t *testing.T) {
	stderr := []byte("some normal error\n")
	varMap := map[string]string{"pipe_1": "line 5: parse(...)"}
	result := rewriteVarNames(stderr, varMap)
	if string(result) != string(stderr) {
		t.Errorf("expected unchanged output, got: %s", string(result))
	}
}

func TestRewriteVarNames_EmptyVarMap(t *testing.T) {
	stderr := []byte("error\n")
	result := rewriteVarNames(stderr, nil)
	if string(result) != string(stderr) {
		t.Errorf("expected unchanged output, got: %s", string(result))
	}
}

func TestRewriteVarNames_EmptyStderr(t *testing.T) {
	result := rewriteVarNames([]byte{}, map[string]string{"pipe_1": "desc"})
	if len(result) != 0 {
		t.Errorf("expected empty result, got: %s", string(result))
	}
}
