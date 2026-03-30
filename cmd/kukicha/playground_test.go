package main

import (
	"strings"
	"testing"

	"github.com/kukichalang/kukicha/internal/ast"
	"github.com/kukichalang/kukicha/internal/lexer"
)

func makeImport(path string) *ast.ImportDecl {
	return &ast.ImportDecl{
		Token: lexer.Token{Lexeme: "import", Line: 1, File: "test.kuki"},
		Path:  &ast.StringLiteral{Value: `"` + path + `"`},
	}
}

func TestValidatePlaygroundImports_Blocked(t *testing.T) {
	blocked := []string{"os/exec", "net", "net/http", "syscall", "unsafe", "plugin"}
	for _, pkg := range blocked {
		program := &ast.Program{
			Imports: []*ast.ImportDecl{makeImport(pkg)},
		}
		err := validatePlaygroundImports(program)
		if err == nil {
			t.Errorf("expected import %q to be blocked, but it was allowed", pkg)
		}
	}
}

func TestValidatePlaygroundImports_Allowed(t *testing.T) {
	allowed := []string{
		"fmt",
		"strings",
		"strconv",
		"math",
		"sort",
		"slices",
		"maps",
		"time",
		"io",
		"bytes",
		"bufio",
		"regexp",
		"unicode",
		"os",
		"stdlib/json",
		"stdlib/slice",
		"stdlib/http",
	}
	for _, pkg := range allowed {
		program := &ast.Program{
			Imports: []*ast.ImportDecl{makeImport(pkg)},
		}
		err := validatePlaygroundImports(program)
		if err != nil {
			t.Errorf("expected import %q to be allowed, got error: %v", pkg, err)
		}
	}
}

func TestValidatePlaygroundImports_MultipleImports(t *testing.T) {
	program := &ast.Program{
		Imports: []*ast.ImportDecl{
			makeImport("fmt"),
			makeImport("os/exec"),
			makeImport("strings"),
		},
	}
	err := validatePlaygroundImports(program)
	if err == nil {
		t.Fatal("expected error for os/exec in multi-import program")
	}
}

func TestValidatePlaygroundImports_Empty(t *testing.T) {
	program := &ast.Program{}
	err := validatePlaygroundImports(program)
	if err != nil {
		t.Errorf("expected no error for empty imports, got: %v", err)
	}
}

func TestValidatePlaygroundImports_ErrorMessage(t *testing.T) {
	program := &ast.Program{
		Imports: []*ast.ImportDecl{
			{
				Token: lexer.Token{Lexeme: "import", Line: 3, File: "playground.kuki"},
				Path:  &ast.StringLiteral{Value: `"os/exec"`},
			},
		},
	}
	err := validatePlaygroundImports(program)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	for _, want := range []string{"playground.kuki:3", `"os/exec"`, "not allowed", "command execution"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error message %q missing expected substring %q", msg, want)
		}
	}
}
