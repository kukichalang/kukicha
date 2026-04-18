package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// isModulePath returns true if arg looks like a Go module path with a version
// suffix (e.g. "github.com/foo/cmd@latest"). It returns false for local paths
// like "./app", "/abs/path", "file.kuki", "myapp/", or any path that exists
// on disk.
func isModulePath(arg string) bool {
	if !strings.Contains(arg, "@") {
		return false
	}
	// Local paths start with ./, /, or have a .kuki extension
	if strings.HasPrefix(arg, "./") || strings.HasPrefix(arg, "/") || strings.HasPrefix(arg, "../") {
		return false
	}
	if strings.HasSuffix(arg, ".kuki") {
		return false
	}
	// If it contains @ and doesn't look like a local path, treat it as a module path
	atIdx := strings.LastIndex(arg, "@")
	if atIdx == 0 || atIdx == len(arg)-1 {
		return false
	}
	// If the arg exists on disk, treat it as a local path (e.g. a dir named "foo@v1").
	if _, err := os.Stat(arg); err == nil {
		return false
	}
	return true
}

// parseModulePath splits a module@version string into its module path and
// version components. It uses the last @ as the separator since module paths
// can contain @ in host parts (though this is rare).
func parseModulePath(arg string) (modulePath, version string, err error) {
	atIdx := strings.LastIndex(arg, "@")
	if atIdx < 0 {
		return "", "", fmt.Errorf("invalid module path: missing @version suffix in %q", arg)
	}
	modulePath = arg[:atIdx]
	version = arg[atIdx+1:]
	if modulePath == "" {
		return "", "", fmt.Errorf("invalid module path: empty module before @ in %q", arg)
	}
	if version == "" {
		return "", "", fmt.Errorf("invalid module path: empty version after @ in %q", arg)
	}
	return modulePath, version, nil
}

// modDownloadResult holds the parsed output of `go mod download -json`.
type modDownloadResult struct {
	Dir   string `json:"Dir"`
	Error string `json:"Error"`
}

// downloadModule downloads a Go module at the specified version using
// `go mod download -json` and returns the directory path in the module cache.
func downloadModule(modulePath, version string) (string, error) {
	arg := modulePath + "@" + version
	cmd := exec.Command("go", "mod", "download", "-json", arg)
	cmd.Env = os.Environ()
	// Run outside the CWD to avoid interference from any local go.mod.
	cmd.Dir = os.TempDir()
	out, _ := cmd.CombinedOutput()
	var result modDownloadResult
	if err := json.Unmarshal(out, &result); err != nil {
		return "", fmt.Errorf("downloading module %s: %s", arg, strings.TrimSpace(string(out)))
	}
	if result.Error != "" {
		msg := fmt.Sprintf("downloading module %s: %s", arg, result.Error)
		if strings.Contains(modulePath, "/") && strings.Count(modulePath, "/") >= 2 {
			msg += "\n(if this is a subpackage path, kukicha run only supports module roots; try the parent module)"
		}
		return "", fmt.Errorf("%s", msg)
	}
	if result.Dir == "" {
		return "", fmt.Errorf("go mod download returned empty Dir for %s", arg)
	}
	return result.Dir, nil
}

// copyModuleFiles copies all files from srcDir to dstDir, preserving directory
// structure. It makes written files writable since the module cache is read-only.
func copyModuleFiles(srcDir, dstDir string) error {
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return fmt.Errorf("computing relative path: %w", err)
		}
		dstPath := filepath.Join(dstDir, relPath)
		if d.IsDir() {
			// Skip hidden directories (except root)
			if strings.HasPrefix(d.Name(), ".") && path != srcDir {
				return filepath.SkipDir
			}
			return os.MkdirAll(dstPath, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}
		if err := os.WriteFile(dstPath, data, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", dstPath, err)
		}
		return nil
	})
}

// setupModuleWorkspace creates a temporary workspace directory, copies the
// module's files from cacheDir into it, and returns the workspace path along
// with a cleanup function. The workspace is suitable for use as a project
// directory by the existing compile/run pipeline.
func setupModuleWorkspace(cacheDir string) (string, func(), error) {
	workspaceDir, err := os.MkdirTemp("", "kukicha-run-*")
	if err != nil {
		return "", nil, fmt.Errorf("creating temp workspace: %w", err)
	}
	cleanup := func() {
		os.RemoveAll(workspaceDir)
	}
	if err := copyModuleFiles(cacheDir, workspaceDir); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("copying module files: %w", err)
	}
	return workspaceDir, cleanup, nil
}

// detectMultipleMains returns paths of all .kuki files that contain a
// "func main(" declaration. Used as a pre-flight check before compilation.
func detectMultipleMains(files []string) []string {
	var mains []string
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if strings.Contains(string(data), "func main(") {
			mains = append(mains, f)
		}
	}
	return mains
}

// runModuleCommand handles `kukicha run github.com/foo/cmd@latest`.
// It downloads the module, sets up a temp workspace, and delegates to
// runCommand which uses the existing compile+run pipeline.
func runModuleCommand(moduleArg, targetFlag string, scriptArgs []string) int {
	modulePath, version, err := parseModulePath(moduleArg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	cacheDir, err := downloadModule(modulePath, version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	workspaceDir, cleanup, err := setupModuleWorkspace(cacheDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	defer cleanup()

	kukiFiles, err := resolveKukiFilesRecursive(workspaceDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: module %s: %v\n", moduleArg, err)
		return 1
	}

	if mains := detectMultipleMains(kukiFiles); len(mains) > 1 {
		fmt.Fprintf(os.Stderr, "Error: module %s contains multiple commands (%s).\nkukicha run only supports single-command modules.\n",
			moduleArg, strings.Join(mains, ", "))
		return 1
	}

	return runCommand(workspaceDir, targetFlag, scriptArgs)
}
