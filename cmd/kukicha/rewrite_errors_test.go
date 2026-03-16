package main

import (
	"testing"
)

func TestRewriteGoErrors_Basic(t *testing.T) {
	stderr := []byte("/tmp/kukicha-run-123.go:10:5: undefined: foo")
	result := rewriteGoErrors(stderr, "/tmp/kukicha-run-123.go", "/home/user/app.kuki")
	expected := "/home/user/app.kuki:10:5: undefined: foo"
	if string(result) != expected {
		t.Errorf("got %q, want %q", string(result), expected)
	}
}

func TestRewriteGoErrors_MultipleOccurrences(t *testing.T) {
	stderr := []byte("/tmp/run.go:1: error1\n/tmp/run.go:5: error2\n")
	result := rewriteGoErrors(stderr, "/tmp/run.go", "app.kuki")
	expected := "app.kuki:1: error1\napp.kuki:5: error2\n"
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
