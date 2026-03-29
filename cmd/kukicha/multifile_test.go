package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kukichalang/kukicha/internal/ast"
)

func TestResolveKukiFiles_SingleFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "main.kuki")
	os.WriteFile(f, []byte("func main()\n    print(\"hello\")\n"), 0644)

	files, isDir, err := resolveKukiFiles(f)
	if err != nil {
		t.Fatal(err)
	}
	if isDir {
		t.Error("expected isDir=false for single file")
	}
	if len(files) != 1 || files[0] != f {
		t.Errorf("expected [%s], got %v", f, files)
	}
}

func TestResolveKukiFiles_Directory(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.kuki"), []byte("func main()\n    print(\"hi\")\n"), 0644)
	os.WriteFile(filepath.Join(dir, "helper.kuki"), []byte("func helper() string\n    return \"ok\"\n"), 0644)
	os.WriteFile(filepath.Join(dir, "helper_test.kuki"), []byte("# test file\n"), 0644)
	os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# not kuki\n"), 0644)

	files, isDir, err := resolveKukiFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !isDir {
		t.Error("expected isDir=true for directory")
	}
	if len(files) != 2 {
		t.Errorf("expected 2 files (excluding _test.kuki and .md), got %d: %v", len(files), files)
	}
}

func TestResolveKukiFiles_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	_, _, err := resolveKukiFiles(dir)
	if err == nil {
		t.Error("expected error for empty directory")
	}
}

func TestResolveKukiFilesRecursive(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.kuki"), []byte("func main()\n    print(\"hi\")\n"), 0644)
	sub := filepath.Join(dir, "components")
	os.MkdirAll(sub, 0755)
	os.WriteFile(filepath.Join(sub, "hero.kuki"), []byte("func Hero() string\n    return \"hero\"\n"), 0644)
	os.WriteFile(filepath.Join(sub, "hero_test.kuki"), []byte("# test\n"), 0644)

	files, err := resolveKukiFilesRecursive(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Errorf("expected 2 files (main.kuki + hero.kuki), got %d: %v", len(files), files)
	}
}

func TestMergePrograms_PetioleMismatch(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.kuki")
	f2 := filepath.Join(dir, "b.kuki")
	os.WriteFile(f1, []byte("petiole alpha\nfunc A() string\n    return \"a\"\n"), 0644)
	os.WriteFile(f2, []byte("petiole beta\nfunc B() string\n    return \"b\"\n"), 0644)

	prog1, err := parseFile(f1)
	if err != nil {
		t.Fatal(err)
	}
	prog2, err := parseFile(f2)
	if err != nil {
		t.Fatal(err)
	}

	_, mergeErr := mergePrograms([]*ast.Program{prog1, prog2}, []string{f1, f2})
	if mergeErr == nil {
		t.Error("expected petiole mismatch error")
	}
}

func TestMergePrograms_SamePetiole(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.kuki")
	f2 := filepath.Join(dir, "b.kuki")
	os.WriteFile(f1, []byte("import \"fmt\"\nfunc A() string\n    return \"a\"\n"), 0644)
	os.WriteFile(f2, []byte("import \"fmt\"\nimport \"os\"\nfunc B() string\n    return \"b\"\n"), 0644)

	prog1, err := parseFile(f1)
	if err != nil {
		t.Fatal(err)
	}
	prog2, err := parseFile(f2)
	if err != nil {
		t.Fatal(err)
	}

	merged, mergeErr := mergePrograms([]*ast.Program{prog1, prog2}, []string{f1, f2})
	if mergeErr != nil {
		t.Fatal(mergeErr)
	}

	if len(merged.Declarations) != 2 {
		t.Errorf("expected 2 declarations, got %d", len(merged.Declarations))
	}
	if len(merged.Imports) != 2 {
		t.Errorf("expected 2 imports (fmt deduplicated), got %d", len(merged.Imports))
	}
}

func TestLoadAndAnalyzeMulti_CrossFileReferences(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "main.kuki")
	f2 := filepath.Join(dir, "helper.kuki")
	os.WriteFile(f1, []byte("func main()\n    msg := Helper()\n    print(msg)\n"), 0644)
	os.WriteFile(f2, []byte("func Helper() string\n    return \"hello\"\n"), 0644)

	_, _, _, err := loadAndAnalyzeMulti([]string{f1, f2})
	if err != nil {
		t.Errorf("cross-file reference should work, got: %v", err)
	}
}

func TestMergePrograms_MultipleInitsAllowed(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.kuki")
	f2 := filepath.Join(dir, "b.kuki")
	os.WriteFile(f1, []byte("func init()\n    print(\"init a\")\n"), 0644)
	os.WriteFile(f2, []byte("func init()\n    print(\"init b\")\n"), 0644)

	prog1, _ := parseFile(f1)
	prog2, _ := parseFile(f2)

	merged, err := mergePrograms([]*ast.Program{prog1, prog2}, []string{f1, f2})
	if err != nil {
		t.Fatalf("multiple init functions should be allowed, got: %v", err)
	}

	initCount := 0
	for _, decl := range merged.Declarations {
		if f, ok := decl.(*ast.FunctionDecl); ok && f.Name.Value == "init" {
			initCount++
		}
	}
	if initCount != 2 {
		t.Errorf("expected 2 init functions, got %d", initCount)
	}
}

func TestMergePrograms_DuplicateMainRejected(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.kuki")
	f2 := filepath.Join(dir, "b.kuki")
	os.WriteFile(f1, []byte("func main()\n    print(\"a\")\n"), 0644)
	os.WriteFile(f2, []byte("func main()\n    print(\"b\")\n"), 0644)

	prog1, _ := parseFile(f1)
	prog2, _ := parseFile(f2)

	_, err := mergePrograms([]*ast.Program{prog1, prog2}, []string{f1, f2})
	if err == nil {
		t.Fatal("expected error for duplicate main functions, got nil")
	}
	if !strings.Contains(err.Error(), "function 'main' already declared") {
		t.Errorf("expected duplicate main error, got: %v", err)
	}
}
