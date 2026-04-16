package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsModulePath(t *testing.T) {
	tests := []struct {
		arg  string
		want bool
	}{
		{"github.com/foo/cmd@latest", true},
		{"github.com/foo/cmd@v1.0.0", true},
		{"github.com/user/repo@v1.2.3", true},
		{"main.kuki", false},
		{"./app", false},
		{"../app", false},
		{"/abs/path", false},
		{"myapp/", false},
		{"file.kuki", false},
		{"cmd/", false},
		{"no-at-sign", false},
		{"@empty-module", false},
		{"module@", false},
	}

	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			got := isModulePath(tt.arg)
			if got != tt.want {
				t.Errorf("isModulePath(%q) = %v, want %v", tt.arg, got, tt.want)
			}
		})
	}
}

func TestIsModulePathLocalDirWithAt(t *testing.T) {
	// A directory literally named "foo@v1" must be treated as local, not a module.
	parent := t.TempDir()
	dirName := "foo@v1"
	if err := os.MkdirAll(filepath.Join(parent, dirName), 0755); err != nil {
		t.Fatal(err)
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
	if err := os.Chdir(parent); err != nil {
		t.Fatal(err)
	}

	if isModulePath(dirName) {
		t.Errorf("isModulePath(%q) = true, want false (directory exists on disk)", dirName)
	}
}

func TestParseModulePath(t *testing.T) {
	tests := []struct {
		arg     string
		wantMod string
		wantVer string
		wantErr bool
	}{
		{"github.com/foo/cmd@latest", "github.com/foo/cmd", "latest", false},
		{"github.com/foo/cmd@v1.2.3", "github.com/foo/cmd", "v1.2.3", false},
		{"golang.org/x/tools@v0.21.0", "golang.org/x/tools", "v0.21.0", false},
		{"no-at-sign", "", "", true},
		{"@empty-module", "", "", true},
		{"module@", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			mod, ver, err := parseModulePath(tt.arg)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseModulePath(%q) error = %v, wantErr %v", tt.arg, err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if mod != tt.wantMod {
				t.Errorf("parseModulePath(%q) module = %q, want %q", tt.arg, mod, tt.wantMod)
			}
			if ver != tt.wantVer {
				t.Errorf("parseModulePath(%q) version = %q, want %q", tt.arg, ver, tt.wantVer)
			}
		})
	}
}

func TestDetectMultipleMains(t *testing.T) {
	root := t.TempDir()

	writeFile := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	writeFile("main.kuki", "petiole main\nfunc main() {\n}\n")
	writeFile("cmd.kuki", "petiole main\nfunc main() {\n}\n")
	writeFile("helper.kuki", "petiole main\nfunc helper() {}\n")

	files := []string{
		filepath.Join(root, "main.kuki"),
		filepath.Join(root, "cmd.kuki"),
		filepath.Join(root, "helper.kuki"),
	}

	mains := detectMultipleMains(files)
	if len(mains) != 2 {
		t.Errorf("detectMultipleMains found %d mains, want 2: %v", len(mains), mains)
	}
}

func TestDetectMultipleMainsSingle(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.kuki"), []byte("petiole main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mains := detectMultipleMains([]string{filepath.Join(root, "main.kuki")})
	if len(mains) != 1 {
		t.Errorf("detectMultipleMains found %d mains, want 1", len(mains))
	}
}

func TestCopyModuleFiles(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create source structure
	os.MkdirAll(filepath.Join(srcDir, "cmd"), 0755)
	os.WriteFile(filepath.Join(srcDir, "main.kuki"), []byte("petiole main\nfunc main() {}"), 0644)
	os.WriteFile(filepath.Join(srcDir, "go.mod"), []byte("module example.com/cmd\n\ngo 1.21\n"), 0644)
	os.WriteFile(filepath.Join(srcDir, "cmd", "handler.kuki"), []byte("petiole main"), 0644)

	err := copyModuleFiles(srcDir, dstDir)
	if err != nil {
		t.Fatalf("copyModuleFiles error: %v", err)
	}

	// Verify files were copied
	data, err := os.ReadFile(filepath.Join(dstDir, "main.kuki"))
	if err != nil {
		t.Errorf("main.kuki not copied: %v", err)
	}
	if string(data) != "petiole main\nfunc main() {}" {
		t.Errorf("main.kuki content mismatch: %s", string(data))
	}

	_, err = os.ReadFile(filepath.Join(dstDir, "cmd", "handler.kuki"))
	if err != nil {
		t.Errorf("cmd/handler.kuki not copied: %v", err)
	}
}

func TestCopyModuleFilesSkipsHiddenDirs(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	os.MkdirAll(filepath.Join(srcDir, ".git", "objects"), 0755)
	os.WriteFile(filepath.Join(srcDir, ".git", "skip.kuki"), []byte("skip"), 0644)
	os.MkdirAll(filepath.Join(srcDir, "cmd"), 0755)
	os.WriteFile(filepath.Join(srcDir, "cmd", "main.kuki"), []byte("petiole main"), 0644)

	err := copyModuleFiles(srcDir, dstDir)
	if err != nil {
		t.Fatalf("copyModuleFiles error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dstDir, "cmd", "main.kuki"))
	if err != nil {
		t.Errorf("cmd/main.kuki not copied: %v", err)
	}
	if string(data) != "petiole main" {
		t.Errorf("cmd/main.kuki content mismatch")
	}

	if _, err := os.Stat(filepath.Join(dstDir, ".git")); !os.IsNotExist(err) {
		t.Error(".git directory should not be copied")
	}
}

func TestSetupModuleWorkspace(t *testing.T) {
	srcDir := t.TempDir()
	os.MkdirAll(filepath.Join(srcDir, "cmd"), 0755)
	os.WriteFile(filepath.Join(srcDir, "main.kuki"), []byte("petiole main\nfunc main() {}"), 0644)

	workspaceDir, cleanup, err := setupModuleWorkspace(srcDir)
	if err != nil {
		t.Fatalf("setupModuleWorkspace error: %v", err)
	}
	defer cleanup()

	if workspaceDir == "" {
		t.Error("workspaceDir is empty")
	}

	// Verify workspace contains copied files
	data, err := os.ReadFile(filepath.Join(workspaceDir, "main.kuki"))
	if err != nil {
		t.Errorf("main.kuki not in workspace: %v", err)
	}
	if string(data) != "petiole main\nfunc main() {}" {
		t.Errorf("main.kuki content mismatch in workspace")
	}

	// Verify cleanup works
	cleanup()
	if _, err := os.Stat(workspaceDir); !os.IsNotExist(err) {
		t.Error("workspace directory should be removed after cleanup")
	}
}
