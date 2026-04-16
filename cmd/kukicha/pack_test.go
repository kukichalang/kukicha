package main

import (
	"strings"
	"testing"

	"github.com/kukichalang/kukicha/internal/ast"
	"gopkg.in/yaml.v3"
)

func TestGenerateSkillMD_ProducesParseableYAML(t *testing.T) {
	skill := &ast.SkillDecl{
		Name:        &ast.Identifier{Value: "HelloSkill"},
		Description: "desc: has colon and \"quotes\"",
		Version:     "1.2.3",
	}

	functions := []FunctionSchema{
		{
			Name: "DoThing",
			Parameters: []ParameterSchema{
				{Name: "message", Type: "string", Default: "line1\nline2: value", HasDefault: true},
				{Name: "count", Type: "integer", Default: int64(3), HasDefault: true},
			},
		},
	}

	skillName := toKebabCase(skill.Name.Value)
	out := generateSkillMD(skill, functions, skillName, "kukicha run scripts/"+skillName+".kuki [args]")
	if !strings.HasPrefix(out, "---\n") {
		t.Fatalf("expected frontmatter opener, got: %q", out[:min(40, len(out))])
	}

	// Split frontmatter from body. After the opening "---\n", find the
	// next "---\n" line which closes the frontmatter.
	rest := strings.TrimPrefix(out, "---\n")
	frontmatter, body, ok := strings.Cut(rest, "---\n")
	if !ok {
		t.Fatalf("expected frontmatter closer '---', got: %q", out)
	}

	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(frontmatter), &parsed); err != nil {
		t.Fatalf("generated YAML should be parseable: %v\n%s", err, frontmatter)
	}

	if parsed["name"] != "hello-skill" {
		t.Fatalf("unexpected kebab-case name: %#v", parsed["name"])
	}

	// version belongs under metadata, not at the top level.
	if _, topLevel := parsed["version"]; topLevel {
		t.Fatalf("version must live under metadata, not top-level frontmatter")
	}
	meta, ok := parsed["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected metadata map, got: %#v", parsed["metadata"])
	}
	if meta["version"] != "1.2.3" {
		t.Fatalf("expected metadata.version=1.2.3, got: %#v", meta["version"])
	}

	// Body should include invocation instructions pointing at kukicha run.
	if !strings.Contains(body, "kukicha run scripts/hello-skill.kuki") {
		t.Fatalf("body missing `kukicha run` invocation:\n%s", body)
	}
	if !strings.Contains(body, "# HelloSkill") {
		t.Fatalf("body missing H1 title:\n%s", body)
	}
	if !strings.Contains(body, "## Exposed functions") {
		t.Fatalf("body missing functions section:\n%s", body)
	}
	if !strings.Contains(body, "**DoThing**") {
		t.Fatalf("body missing DoThing listing:\n%s", body)
	}
}

func TestGenerateSkillMD_NoVersion_OmitsMetadata(t *testing.T) {
	skill := &ast.SkillDecl{
		Name:        &ast.Identifier{Value: "Plain"},
		Description: "A plain skill.",
	}
	out := generateSkillMD(skill, nil, "plain", "kukicha run scripts/plain.kuki [args]")

	// When version is empty, metadata should not be emitted at all.
	if strings.Contains(out, "metadata:") {
		t.Fatalf("expected no metadata block when version is empty, got:\n%s", out)
	}
}

func TestToKebabCase(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"HelloSkill", "hello-skill"},
		{"HTTPClient", "http-client"},
		{"Mizuya", "mizuya"},
		{"McpSandbox", "mcp-sandbox"},
		{"A", "a"},
		{"ABCDef", "abc-def"},
	}
	for _, tc := range cases {
		if got := toKebabCase(tc.in); got != tc.want {
			t.Errorf("toKebabCase(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestDefaultValueToYAML(t *testing.T) {
	cases := []struct {
		name string
		expr ast.Expression
		want any
		ok   bool
	}{
		{name: "int", expr: &ast.IntegerLiteral{Value: 7}, want: int64(7), ok: true},
		{name: "float", expr: &ast.FloatLiteral{Value: 3.14}, want: 3.14, ok: true},
		{name: "string", expr: &ast.StringLiteral{Value: "x:y"}, want: "x:y", ok: true},
		{name: "bool", expr: &ast.BooleanLiteral{Value: true}, want: true, ok: true},
		{name: "unsupported", expr: &ast.Identifier{Value: "x"}, want: nil, ok: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := defaultValueToYAML(tc.expr)
			if ok != tc.ok {
				t.Fatalf("ok mismatch: got %v want %v", ok, tc.ok)
			}
			if got != tc.want {
				t.Fatalf("value mismatch: got %#v want %#v", got, tc.want)
			}
		})
	}
}
