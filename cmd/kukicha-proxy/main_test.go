package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeUpstream creates a test server that mimics proxy.golang.org.
func fakeUpstream(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")

		switch {
		case strings.HasSuffix(path, "/@v/list"):
			fmt.Fprintln(w, "v0.1.0")
			fmt.Fprintln(w, "v0.2.0")
			fmt.Fprintln(w, "v0.3.0")

		case strings.HasSuffix(path, "/@latest"):
			json.NewEncoder(w).Encode(map[string]any{
				"Version": "v0.3.0",
				"Time":    time.Now().UTC().Format(time.RFC3339),
			})

		case strings.HasSuffix(path, ".info"):
			version := strings.TrimSuffix(filepath.Base(path), ".info")
			json.NewEncoder(w).Encode(map[string]any{
				"Version": version,
				"Time":    time.Now().UTC().Format(time.RFC3339),
			})

		case strings.HasSuffix(path, ".mod"):
			fmt.Fprintln(w, "module example.com/mod\n\ngo 1.21")

		case strings.HasSuffix(path, ".zip"):
			w.Header().Set("Content-Type", "application/zip")
			w.Write([]byte("fake-zip-content"))

		default:
			http.NotFound(w, r)
		}
	}))
}

func newTestProxy(t *testing.T, upstream string, cooldown time.Duration, trusted []string) *proxy {
	t.Helper()
	dir := t.TempDir()
	db, err := newSeenDB(filepath.Join(dir, "seen.json"))
	if err != nil {
		t.Fatal(err)
	}
	return &proxy{
		upstream:   upstream,
		cooldown:   cooldown,
		db:         db,
		cacheDir:   filepath.Join(dir, "cache"),
		trusted:    trusted,
		httpClient: http.DefaultClient,
	}
}

func TestListFiltersCooldown(t *testing.T) {
	up := fakeUpstream(t)
	defer up.Close()

	p := newTestProxy(t, up.URL, 7*24*time.Hour, nil)

	// First request — all versions are first-seen NOW, so all are in cooldown.
	req := httptest.NewRequest("GET", "/example.com/mod/@v/list", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := strings.TrimSpace(w.Body.String())
	if body != "" {
		t.Fatalf("expected empty list (all in cooldown), got: %q", body)
	}
}

func TestListServesAfterCooldown(t *testing.T) {
	up := fakeUpstream(t)
	defer up.Close()

	p := newTestProxy(t, up.URL, 7*24*time.Hour, nil)

	// Pre-seed versions with old first-seen timestamps.
	oldTime := time.Now().UTC().Add(-10 * 24 * time.Hour) // 10 days ago
	p.db.mu.Lock()
	p.db.records["example.com/mod@v0.1.0"] = versionRecord{
		Module: "example.com/mod", Version: "v0.1.0", FirstSeen: oldTime,
	}
	p.db.records["example.com/mod@v0.2.0"] = versionRecord{
		Module: "example.com/mod", Version: "v0.2.0", FirstSeen: oldTime,
	}
	// v0.3.0 is new (not pre-seeded, will be first-seen NOW).
	p.db.mu.Unlock()

	req := httptest.NewRequest("GET", "/example.com/mod/@v/list", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := strings.TrimSpace(w.Body.String())
	versions := strings.Split(body, "\n")

	// v0.1.0 and v0.2.0 should pass cooldown; v0.3.0 should be filtered.
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d: %v", len(versions), versions)
	}
	for _, v := range versions {
		if v == "v0.3.0" {
			t.Error("v0.3.0 should be filtered by cooldown")
		}
	}
}

func TestTrustedBypassesCooldown(t *testing.T) {
	up := fakeUpstream(t)
	defer up.Close()

	p := newTestProxy(t, up.URL, 7*24*time.Hour, []string{"github.com/kukichalang/"})

	// Request a trusted module — all versions should pass even without cooldown.
	req := httptest.NewRequest("GET", "/github.com/kukichalang/infer/@v/list", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := strings.TrimSpace(w.Body.String())
	versions := strings.Split(body, "\n")
	if len(versions) != 3 {
		t.Fatalf("expected 3 versions (trusted bypass), got %d: %v", len(versions), versions)
	}
}

func TestLatestInCooldownReturns404(t *testing.T) {
	up := fakeUpstream(t)
	defer up.Close()

	p := newTestProxy(t, up.URL, 7*24*time.Hour, nil)

	req := httptest.NewRequest("GET", "/example.com/mod/@latest", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	// Latest version is first-seen NOW → in cooldown → 404.
	if w.Code != 404 {
		t.Fatalf("expected 404 (in cooldown), got %d", w.Code)
	}
}

func TestLatestAfterCooldown(t *testing.T) {
	up := fakeUpstream(t)
	defer up.Close()

	p := newTestProxy(t, up.URL, 7*24*time.Hour, nil)

	// Pre-seed v0.3.0 as seen 10 days ago so it passes cooldown.
	oldTime := time.Now().UTC().Add(-10 * 24 * time.Hour)
	p.db.mu.Lock()
	p.db.records["example.com/mod@v0.3.0"] = versionRecord{
		Module: "example.com/mod", Version: "v0.3.0", FirstSeen: oldTime,
	}
	p.db.mu.Unlock()

	req := httptest.NewRequest("GET", "/example.com/mod/@latest", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var info struct {
		Version string `json:"Version"`
	}
	json.Unmarshal(w.Body.Bytes(), &info)
	if info.Version != "v0.3.0" {
		t.Fatalf("expected v0.3.0, got %s", info.Version)
	}
}

func TestInfoPassthrough(t *testing.T) {
	up := fakeUpstream(t)
	defer up.Close()

	p := newTestProxy(t, up.URL, 7*24*time.Hour, nil)

	req := httptest.NewRequest("GET", "/example.com/mod/@v/v0.1.0.info", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Should record first-seen.
	p.db.mu.RLock()
	_, exists := p.db.records["example.com/mod@v0.1.0"]
	p.db.mu.RUnlock()
	if !exists {
		t.Error("expected first-seen record for v0.1.0")
	}
}

func TestModPassthrough(t *testing.T) {
	up := fakeUpstream(t)
	defer up.Close()

	p := newTestProxy(t, up.URL, 7*24*time.Hour, nil)

	req := httptest.NewRequest("GET", "/example.com/mod/@v/v0.1.0.mod", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "module example.com/mod") {
		t.Error("expected go.mod content")
	}
}

func TestHealthEndpoint(t *testing.T) {
	up := fakeUpstream(t)
	defer up.Close()

	p := newTestProxy(t, up.URL, 7*24*time.Hour, nil)

	req := httptest.NewRequest("GET", "/_health", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var health map[string]any
	json.Unmarshal(w.Body.Bytes(), &health)
	if health["status"] != "ok" {
		t.Errorf("expected status ok, got %v", health["status"])
	}
}

func TestSeenDBPersistence(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "seen.json")

	// Create DB and record a version.
	db1, err := newSeenDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	db1.firstSeen("example.com/mod", "v1.0.0")

	// Verify file was written.
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatal("seen.json not persisted")
	}

	// Reload from disk.
	db2, err := newSeenDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	db2.mu.RLock()
	_, exists := db2.records["example.com/mod@v1.0.0"]
	db2.mu.RUnlock()
	if !exists {
		t.Error("expected record to survive reload")
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"7d", 7 * 24 * time.Hour},
		{"1d", 24 * time.Hour},
		{"30d", 30 * 24 * time.Hour},
		{"168h", 168 * time.Hour},
		{"1h30m", 90 * time.Minute},
	}
	for _, tt := range tests {
		got, err := parseDuration(tt.input)
		if err != nil {
			t.Errorf("parseDuration(%q) error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseDuration(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestMethodNotAllowed(t *testing.T) {
	up := fakeUpstream(t)
	defer up.Close()

	p := newTestProxy(t, up.URL, 7*24*time.Hour, nil)

	req := httptest.NewRequest("POST", "/example.com/mod/@v/list", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}
