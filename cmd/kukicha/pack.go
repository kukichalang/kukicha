package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/kukichalang/kukicha/internal/ast"
	"gopkg.in/yaml.v3"
)

func packMain(args []string) {
	packFlags := flag.NewFlagSet("pack", flag.ContinueOnError)
	packFlags.SetOutput(os.Stderr)
	outputDir := packFlags.String("output", "", "Output directory (default: skills/<skill-name>/ next to source)")
	if err := packFlags.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, "Usage: kukicha pack [--output <dir>] <skill.kuki>")
		os.Exit(1)
	}
	packArgs := packFlags.Args()
	if len(packArgs) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: kukicha pack [--output <dir>] <skill.kuki>")
		os.Exit(1)
	}
	code := packCommand(packArgs[0], *outputDir)
	if code != 0 {
		os.Exit(code)
	}
}

// packCommand produces an agentskills.io-compliant skill directory.
//
// Single-file input produces:
//
//	<output>/
//	├── SKILL.md            # YAML frontmatter + markdown body
//	└── scripts/
//	    └── <name>.kuki     # verbatim source copy
//
// Directory input copies every .kuki file under the input (excluding tests)
// into scripts/<name>/, preserving relative paths, so multi-file projects
// remain runnable via `kukicha run scripts/<name>/`.
//
// Agents invoke via the command shown in SKILL.md — matching the
// "ship source, not binaries" pattern recommended by the spec.
func packCommand(filename string, outputDir string) int {
	cr, err := compile(filename, "", "mcp", false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if cr.program.SkillDecl == nil {
		fmt.Fprintln(os.Stderr, "Error: no skill declaration found in source")
		return 1
	}
	skill := cr.program.SkillDecl
	skillName := toKebabCase(skill.Name.Value)

	// Detect whether the source is a file or a directory.
	info, err := os.Stat(cr.absFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error stat-ing source: %v\n", err)
		return 1
	}
	srcIsDir := info.IsDir()

	// Default output: skills/<skill-name>/ next to the source.
	if outputDir == "" {
		outputDir = filepath.Join(filepath.Dir(cr.absFile), "skills", skillName)
	}

	// Build invocation string for the SKILL.md body.
	invocation := fmt.Sprintf("kukicha run scripts/%s.kuki [args]", skillName)
	if srcIsDir {
		invocation = fmt.Sprintf("kukicha run scripts/%s/ [args]", skillName)
	}

	functions := extractFunctionSchemas(cr.program)
	skillMD := generateSkillMD(skill, functions, skillName, invocation)

	scriptsDir := filepath.Join(outputDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		return 1
	}

	// Write SKILL.md
	skillMDPath := filepath.Join(outputDir, "SKILL.md")
	if err := os.WriteFile(skillMDPath, []byte(skillMD), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing SKILL.md: %v\n", err)
		return 1
	}
	fmt.Printf("Generated %s\n", skillMDPath)

	// Copy source(s) into scripts/.
	if srcIsDir {
		destRoot := filepath.Join(scriptsDir, skillName)
		if err := copyKukiTree(cr.absFile, destRoot); err != nil {
			fmt.Fprintf(os.Stderr, "Error copying source tree: %v\n", err)
			return 1
		}
		fmt.Printf("Copied source tree: %s\n", destRoot)
	} else {
		sourceBytes, err := os.ReadFile(cr.absFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading source: %v\n", err)
			return 1
		}
		scriptPath := filepath.Join(scriptsDir, skillName+".kuki")
		if err := os.WriteFile(scriptPath, sourceBytes, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing script: %v\n", err)
			return 1
		}
		fmt.Printf("Copied source: %s\n", scriptPath)
	}

	fmt.Printf("Skill packed successfully in %s\n", outputDir)
	return 0
}

// copyKukiTree walks src and copies every non-test .kuki file into dst,
// preserving relative paths. Other files are ignored — only Kukicha source
// matters for the runnable skill.
func copyKukiTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".kuki" {
			return nil
		}
		if strings.HasSuffix(info.Name(), "_test.kuki") {
			return nil
		}
		rel, relErr := filepath.Rel(src, path)
		if relErr != nil {
			return relErr
		}
		target := filepath.Join(dst, rel)
		if mkErr := os.MkdirAll(filepath.Dir(target), 0755); mkErr != nil {
			return mkErr
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		return os.WriteFile(target, data, 0644)
	})
}

// FunctionSchema holds extracted metadata for a function declaration
type FunctionSchema struct {
	Name        string
	Description string
	Parameters  []ParameterSchema
	Returns     []string
}

// ParameterSchema holds extracted metadata for a function parameter
type ParameterSchema struct {
	Name       string
	Type       string
	Default    any
	HasDefault bool
}

func extractFunctionSchemas(program *ast.Program) []FunctionSchema {
	var schemas []FunctionSchema

	for _, decl := range program.Declarations {
		fn, ok := decl.(*ast.FunctionDecl)
		if !ok {
			continue
		}

		// Only include exported functions (starting with uppercase)
		if len(fn.Name.Value) == 0 || !unicode.IsUpper(rune(fn.Name.Value[0])) {
			continue
		}

		// Skip methods (they have receivers)
		if fn.Receiver != nil {
			continue
		}

		schema := FunctionSchema{
			Name: fn.Name.Value,
		}

		// Extract parameters
		for _, param := range fn.Parameters {
			ps := ParameterSchema{
				Name: param.Name.Value,
				Type: typeToJSONSchemaType(param.Type),
			}
			if param.DefaultValue != nil {
				if def, ok := defaultValueToYAML(param.DefaultValue); ok {
					ps.Default = def
					ps.HasDefault = true
				}
			}
			schema.Parameters = append(schema.Parameters, ps)
		}

		// Extract return types
		for _, ret := range fn.Returns {
			schema.Returns = append(schema.Returns, typeAnnotationName(ret))
		}

		schemas = append(schemas, schema)
	}

	return schemas
}

// generateSkillMD emits an agentskills.io-compliant SKILL.md: YAML
// frontmatter with only the spec-recognized fields, plus a markdown body
// that tells the agent how to invoke the skill.
func generateSkillMD(skill *ast.SkillDecl, functions []FunctionSchema, skillName, invocation string) string {
	type yamlSkill struct {
		Name        string            `yaml:"name"`
		Description string            `yaml:"description"`
		Metadata    map[string]string `yaml:"metadata,omitempty"`
	}

	doc := yamlSkill{
		Name:        skillName,
		Description: skill.Description,
	}
	if skill.Version != "" {
		doc.Metadata = map[string]string{"version": skill.Version}
	}

	out, err := yaml.Marshal(doc)
	if err != nil {
		return "---\nname: " + skillName + "\n---\n"
	}

	var body strings.Builder
	body.WriteString("---\n")
	body.Write(out)
	body.WriteString("---\n\n")

	// Markdown body: title, description, invocation instructions.
	body.WriteString("# ")
	body.WriteString(skill.Name.Value)
	body.WriteString("\n\n")
	if skill.Description != "" {
		body.WriteString(skill.Description)
		body.WriteString("\n\n")
	}
	body.WriteString("## Usage\n\n")
	body.WriteString("Run this skill with:\n\n")
	body.WriteString("```bash\n")
	body.WriteString(invocation)
	body.WriteString("\n```\n")

	if len(functions) > 0 {
		body.WriteString("\n## Exposed functions\n\n")
		for _, fn := range functions {
			paramParts := make([]string, 0, len(fn.Parameters))
			for _, p := range fn.Parameters {
				paramParts = append(paramParts, fmt.Sprintf("%s: %s", p.Name, p.Type))
			}
			returnPart := ""
			if len(fn.Returns) > 0 {
				returnPart = " → " + strings.Join(fn.Returns, ", ")
			}
			fmt.Fprintf(&body, "- **%s**(%s)%s\n", fn.Name, strings.Join(paramParts, ", "), returnPart)
		}
	}

	return body.String()
}

// typeToJSONSchemaType maps Kukicha/Go type annotations to JSON Schema types
func typeToJSONSchemaType(typeAnn ast.TypeAnnotation) string {
	if typeAnn == nil {
		return "object"
	}
	switch t := typeAnn.(type) {
	case *ast.PrimitiveType:
		switch t.Name {
		case "string":
			return "string"
		case "int", "int8", "int16", "int32", "int64",
			"uint", "uint8", "uint16", "uint32", "uint64":
			return "integer"
		case "float32", "float64":
			return "number"
		case "bool":
			return "boolean"
		case "byte", "rune":
			return "integer"
		}
	case *ast.ListType:
		return "array"
	case *ast.MapType:
		return "object"
	case *ast.NamedType:
		if t.Name == "error" {
			return "string"
		}
		return "object"
	}
	return "object"
}

// typeAnnotationName returns a human-readable name for a type annotation
func typeAnnotationName(typeAnn ast.TypeAnnotation) string {
	if typeAnn == nil {
		return "any"
	}
	switch t := typeAnn.(type) {
	case *ast.PrimitiveType:
		return t.Name
	case *ast.NamedType:
		return t.Name
	case *ast.ListType:
		return "list"
	case *ast.MapType:
		return "map"
	case *ast.ReferenceType:
		return "reference"
	case *ast.ChannelType:
		return "channel"
	}
	return "any"
}

// defaultValueToYAML converts a literal default value expression into a YAML value.
func defaultValueToYAML(expr ast.Expression) (any, bool) {
	switch e := expr.(type) {
	case *ast.IntegerLiteral:
		return e.Value, true
	case *ast.FloatLiteral:
		return e.Value, true
	case *ast.StringLiteral:
		return e.Value, true
	case *ast.BooleanLiteral:
		return e.Value, true
	}
	return nil, false
}

// toKebabCase converts PascalCase to kebab-case per the agentskills.io
// `name` field rules: lowercase alphanumerics and hyphens only, no leading
// or trailing hyphens, no consecutive hyphens. Handles acronyms the same
// way snake_case does ("HTTPClient" → "http-client").
func toKebabCase(s string) string {
	runes := []rune(s)
	var result strings.Builder
	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := runes[i-1]
				if unicode.IsLower(prev) || (unicode.IsUpper(prev) && i+1 < len(runes) && unicode.IsLower(runes[i+1])) {
					result.WriteByte('-')
				}
			}
			result.WriteRune(unicode.ToLower(r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}
