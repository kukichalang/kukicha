package main

import (
	"encoding/json"
	"fmt"
	"io"
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
			_, _ = fmt.Fprintln(w, "v0.1.0")
			_, _ = fmt.Fprintln(w, "v0.2.0")
			_, _ = fmt.Fprintln(w, "v0.3.0")

		case strings.HasSuffix(path, "/@latest"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Version": "v0.3.0",
				"Time":    time.Now().UTC().Format(time.RFC3339),
			})

		case strings.HasSuffix(path, ".info"):
			version := strings.TrimSuffix(filepath.Base(path), ".info")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"Version": version,
				"Time":    time.Now().UTC().Format(time.RFC3339),
			})

		case path == "gone/mod/@v/v1.0.0.mod":
			w.WriteHeader(http.StatusGone)

		case strings.HasSuffix(path, ".mod"):
				_, _ = fmt.Fprintln(w, "module example.com/mod\n\ngo 1.21")

		case strings.HasSuffix(path, ".zip"):
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write([]byte("fake-zip-content"))

		default:
			http.NotFound(w, r)
		}
	}))
}

func newTestProxy(t *testing.T, upstream string, cooldown time.Duration, trusted []string) (*proxy, *seenDB) {
	t.Helper()
	dir := t.TempDir()
	db, err := newSeenDB(filepath.Join(dir, "seen.json"))
	if err != nil {
		t.Fatal(err)
	}
	p := &proxy{
		upstream:   upstream,
		cooldown:   cooldown,
		db:         db,
		cacheDir:   filepath.Join(dir, "cache"),
		trusted:    trusted,
		httpClient: http.DefaultClient,
	}
	return p, db
}

func TestListFiltersCooldown(t *testing.T) {
	up := fakeUpstream(t)
	defer up.Close()

	p, _ := newTestProxy(t, up.URL, 7*24*time.Hour, nil)

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

	p, db := newTestProxy(t, up.URL, 7*24*time.Hour, nil)

	// Pre-seed versions with old first-seen timestamps.
	oldTime := time.Now().UTC().Add(-10 * 24 * time.Hour) // 10 days ago
	db.seedRecord("example.com/mod", "v0.1.0", oldTime)
	db.seedRecord("example.com/mod", "v0.2.0", oldTime)
	// v0.3.0 is new (not pre-seeded, will be first-seen NOW).

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

	p, _ := newTestProxy(t, up.URL, 7*24*time.Hour, []string{"github.com/kukichalang/"})

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

	p, _ := newTestProxy(t, up.URL, 7*24*time.Hour, nil)

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

	p, db := newTestProxy(t, up.URL, 7*24*time.Hour, nil)

	// Pre-seed v0.3.0 as seen 10 days ago so it passes cooldown.
	db.seedRecord("example.com/mod", "v0.3.0", time.Now().UTC().Add(-10*24*time.Hour))

	req := httptest.NewRequest("GET", "/example.com/mod/@latest", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var info struct {
		Version string `json:"Version"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &info); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if info.Version != "v0.3.0" {
		t.Fatalf("expected v0.3.0, got %s", info.Version)
	}
}

func TestInfoInCooldownReturns404(t *testing.T) {
	up := fakeUpstream(t)
	defer up.Close()

	p, db := newTestProxy(t, up.URL, 7*24*time.Hour, nil)

	req := httptest.NewRequest("GET", "/example.com/mod/@v/v0.1.0.info", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	// Version first-seen NOW → in cooldown → 404.
	if w.Code != 404 {
		t.Fatalf("expected 404 (in cooldown), got %d", w.Code)
	}
	// First-seen should still be recorded.
	if !db.hasRecord("example.com/mod", "v0.1.0") {
		t.Error("expected first-seen record for v0.1.0")
	}
}

func TestInfoAfterCooldown(t *testing.T) {
	up := fakeUpstream(t)
	defer up.Close()

	p, db := newTestProxy(t, up.URL, 7*24*time.Hour, nil)
	db.seedRecord("example.com/mod", "v0.1.0", time.Now().UTC().Add(-10*24*time.Hour))

	req := httptest.NewRequest("GET", "/example.com/mod/@v/v0.1.0.info", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestInfoTrustedBypassesCooldown(t *testing.T) {
	up := fakeUpstream(t)
	defer up.Close()

	p, _ := newTestProxy(t, up.URL, 7*24*time.Hour, []string{"github.com/kukichalang/"})

	req := httptest.NewRequest("GET", "/github.com/kukichalang/infer/@v/v0.1.0.info", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200 (trusted bypass), got %d", w.Code)
	}
}

func TestModInCooldownReturns404(t *testing.T) {
	up := fakeUpstream(t)
	defer up.Close()

	p, _ := newTestProxy(t, up.URL, 7*24*time.Hour, nil)

	req := httptest.NewRequest("GET", "/example.com/mod/@v/v0.1.0.mod", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404 (in cooldown), got %d", w.Code)
	}
}

func TestModAfterCooldown(t *testing.T) {
	up := fakeUpstream(t)
	defer up.Close()

	p, db := newTestProxy(t, up.URL, 7*24*time.Hour, nil)
	db.seedRecord("example.com/mod", "v0.1.0", time.Now().UTC().Add(-10*24*time.Hour))

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

func TestZipInCooldownReturns404(t *testing.T) {
	up := fakeUpstream(t)
	defer up.Close()

	p, _ := newTestProxy(t, up.URL, 7*24*time.Hour, nil)

	req := httptest.NewRequest("GET", "/example.com/mod/@v/v0.1.0.zip", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404 (in cooldown), got %d", w.Code)
	}
}

func TestDiskCacheServesMod(t *testing.T) {
	up := fakeUpstream(t)
	defer up.Close()

	p, db := newTestProxy(t, up.URL, 7*24*time.Hour, nil)
	db.seedRecord("example.com/mod", "v0.1.0", time.Now().UTC().Add(-10*24*time.Hour))

	// First request — fetches from upstream and writes cache.
	req1 := httptest.NewRequest("GET", "/example.com/mod/@v/v0.1.0.mod", nil)
	w1 := httptest.NewRecorder()
	p.ServeHTTP(w1, req1)
	if w1.Code != 200 {
		t.Fatalf("first request: expected 200, got %d", w1.Code)
	}

	// Verify cache file was written.
	cachePath := filepath.Join(p.cacheDir, "example.com", "mod", "@v", "v0.1.0.mod")
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("cache file not written: %v", err)
	}

	// Kill the upstream so any miss would fail.
	up.Close()

	// Second request — must be served from cache.
	req2 := httptest.NewRequest("GET", "/example.com/mod/@v/v0.1.0.mod", nil)
	w2 := httptest.NewRecorder()
	p.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("cached request: expected 200, got %d", w2.Code)
	}
	if !strings.Contains(w2.Body.String(), "module example.com/mod") {
		t.Error("expected cached go.mod content")
	}
}

func TestDiskCacheServesZip(t *testing.T) {
	up := fakeUpstream(t)
	defer up.Close()

	p, db := newTestProxy(t, up.URL, 7*24*time.Hour, nil)
	db.seedRecord("example.com/mod", "v0.1.0", time.Now().UTC().Add(-10*24*time.Hour))

	// Prime the cache.
	req1 := httptest.NewRequest("GET", "/example.com/mod/@v/v0.1.0.zip", nil)
	w1 := httptest.NewRecorder()
	p.ServeHTTP(w1, req1)
	if w1.Code != 200 {
		t.Fatalf("first request: expected 200, got %d", w1.Code)
	}

	up.Close()

	// Second request from cache.
	req2 := httptest.NewRequest("GET", "/example.com/mod/@v/v0.1.0.zip", nil)
	w2 := httptest.NewRecorder()
	p.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("cached request: expected 200, got %d", w2.Code)
	}
}

func TestSumDBProxy(t *testing.T) {
	// Start a fake sumdb server.
	fakeSumDB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, "fake-sumdb-response")
	}))
	defer fakeSumDB.Close()

	// Point the proxy at the fake sumdb by overriding the httpClient transport.
	// Instead, we test via the ServeHTTP path with a real sumdb request path,
	// using a custom httpClient that redirects sum.golang.org to our fake.
	p, _ := newTestProxy(t, "https://proxy.golang.org", 7*24*time.Hour, nil)

	// Confirm the sumdb/ path is routed (we can't easily override the target
	// URL in this unit test, so just verify routing doesn't 404).
	req := httptest.NewRequest("GET", "/sumdb/sum.golang.org/lookup/golang.org/x/text@v0.3.0", nil)
	w := httptest.NewRecorder()
	// Use a client that will fail fast (no real network in tests).
	p.httpClient = &http.Client{Timeout: 1 * time.Millisecond}
	p.ServeHTTP(w, req)

	// Either 200 (real network) or 502 (no network) — but NOT 404.
	// The important thing is that the sumdb/ prefix was routed correctly.
	if w.Code == http.StatusNotFound {
		t.Fatalf("sumdb path should be routed, not 404")
	}
}

func Test410PropagatesGone(t *testing.T) {
	up := fakeUpstream(t)
	defer up.Close()

	p, db := newTestProxy(t, up.URL, 7*24*time.Hour, nil)
	db.seedRecord("gone/mod", "v1.0.0", time.Now().UTC().Add(-10*24*time.Hour))

	req := httptest.NewRequest("GET", "/gone/mod/@v/v1.0.0.mod", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != http.StatusGone {
		t.Fatalf("expected 410 Gone, got %d", w.Code)
	}
}

func TestHealthEndpoint(t *testing.T) {
	up := fakeUpstream(t)
	defer up.Close()

	p, _ := newTestProxy(t, up.URL, 7*24*time.Hour, nil)

	req := httptest.NewRequest("GET", "/_health", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var health map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &health); err != nil {
		t.Fatalf("json decode: %v", err)
	}
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
	db1.FirstSeen("example.com/mod", "v1.0.0")

	// Verify file was written.
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatal("seen.json not persisted")
	}

	// Reload from disk.
	db2, err := newSeenDB(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if !db2.hasRecord("example.com/mod", "v1.0.0") {
		t.Error("expected record to survive reload")
	}
}

func TestSQLiteSeenStore(t *testing.T) {
	dir := t.TempDir()
	s, err := newSQLiteSeenStore(filepath.Join(dir, "seen.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Shutdown() }()

	// First call records now.
	t1 := s.FirstSeen("example.com/mod", "v1.0.0")
	if t1.IsZero() {
		t.Fatal("firstSeen returned zero time")
	}

	// Second call returns same timestamp.
	t2 := s.FirstSeen("example.com/mod", "v1.0.0")
	if !t1.Equal(t2) {
		t.Errorf("second call returned different time: %v vs %v", t1, t2)
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

	p, _ := newTestProxy(t, up.URL, 7*24*time.Hour, nil)

	req := httptest.NewRequest("POST", "/example.com/mod/@v/list", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

// fakeOSV creates a test server that mimics the OSV API (api.osv.dev/v1/query).
// It responds with a single vulnerability whose "fixed" version is fixedVersion.
func fakeOSV(t *testing.T, module, fixedVersion string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req osvQueryReq
		_ = json.Unmarshal(body, &req)

		if req.Package.Name != module {
			_ = json.NewEncoder(w).Encode(osvQueryResp{})
			return
		}

		_ = json.NewEncoder(w).Encode(osvQueryResp{
			Vulns: []osvVuln{{
				ID: "GO-2026-9999",
				Affected: []osvAffected{{
					Ranges: []osvRange{{
						Type: "SEMVER",
						Events: []osvEvent{
							{Introduced: "0"},
							{Fixed: fixedVersion},
						},
					}},
				}},
			}},
		})
	}))
}

func newTestProxyWithVuln(t *testing.T, upstream string, cooldown time.Duration, osvURL string) (*proxy, *seenDB) {
	t.Helper()
	dir := t.TempDir()
	db, err := newSeenDB(filepath.Join(dir, "seen.json"))
	if err != nil {
		t.Fatal(err)
	}
	client := http.DefaultClient
	vc := newVulnChecker(client)
	vc.dbURL = osvURL
	p := &proxy{
		upstream:   upstream,
		cooldown:   cooldown,
		db:         db,
		cacheDir:   filepath.Join(dir, "cache"),
		httpClient: client,
		vuln:       vc,
	}
	return p, db
}

func TestVulnFixBypassesCooldownOnList(t *testing.T) {
	up := fakeUpstream(t)
	defer up.Close()

	// OSV says v0.2.0 is the fix version (without "v" prefix, as in OSV format).
	osv := fakeOSV(t, "example.com/mod", "0.2.0")
	defer osv.Close()

	p, _ := newTestProxyWithVuln(t, up.URL, 7*24*time.Hour, osv.URL)

	// All versions are first-seen NOW → in cooldown.
	// But v0.2.0 is a security fix → should bypass cooldown.
	req := httptest.NewRequest("GET", "/example.com/mod/@v/list", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	body := strings.TrimSpace(w.Body.String())
	if body != "v0.2.0" {
		t.Fatalf("expected only v0.2.0 (security fix), got: %q", body)
	}
}

func TestVulnFixBypassesCooldownOnInfo(t *testing.T) {
	up := fakeUpstream(t)
	defer up.Close()

	osv := fakeOSV(t, "example.com/mod", "0.2.0")
	defer osv.Close()

	p, _ := newTestProxyWithVuln(t, up.URL, 7*24*time.Hour, osv.URL)

	// v0.2.0 is in cooldown but is a security fix → should return 200.
	req := httptest.NewRequest("GET", "/example.com/mod/@v/v0.2.0.info", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200 (security fix bypass), got %d: %s", w.Code, w.Body.String())
	}
}

func TestVulnFixBypassesCooldownOnLatest(t *testing.T) {
	up := fakeUpstream(t)
	defer up.Close()

	// upstream /@latest returns v0.3.0; mark that as the security fix.
	osv := fakeOSV(t, "example.com/mod", "0.3.0")
	defer osv.Close()

	p, _ := newTestProxyWithVuln(t, up.URL, 7*24*time.Hour, osv.URL)

	req := httptest.NewRequest("GET", "/example.com/mod/@latest", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200 (security fix bypass), got %d: %s", w.Code, w.Body.String())
	}
}

func TestNoVulnStillBlockedByCooldown(t *testing.T) {
	up := fakeUpstream(t)
	defer up.Close()

	// OSV says v0.9.0 is the fix — NOT any version in the upstream list.
	osv := fakeOSV(t, "example.com/mod", "0.9.0")
	defer osv.Close()

	p, _ := newTestProxyWithVuln(t, up.URL, 7*24*time.Hour, osv.URL)

	// v0.1.0 is in cooldown and is NOT a security fix → should be blocked.
	req := httptest.NewRequest("GET", "/example.com/mod/@v/v0.1.0.info", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404 (in cooldown, not a fix), got %d", w.Code)
	}
}

func TestVulnCheckerDisabled(t *testing.T) {
	up := fakeUpstream(t)
	defer up.Close()

	// Proxy with vuln=nil (disabled) — security fixes should NOT bypass.
	p, _ := newTestProxy(t, up.URL, 7*24*time.Hour, nil)

	req := httptest.NewRequest("GET", "/example.com/mod/@v/v0.2.0.info", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404 (vulncheck disabled), got %d", w.Code)
	}
}
