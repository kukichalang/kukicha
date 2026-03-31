// kukicha-proxy is a Go module proxy with dependency cooldown filtering.
//
// It implements the GOPROXY protocol and adds first-seen timestamp tracking
// to filter out recently published module versions — mitigating supply chain
// attacks by ensuring only versions that have been publicly available for a
// configurable cooldown period are served.
//
// Two modes:
//   - local:  caches to disk, upstream is proxy.kukicha.dev (or configurable)
//   - hosted: SQLite for first-seen tracking, upstream is proxy.golang.org
//
// See: https://github.com/golang/go/issues/76485
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	_ "github.com/glebarez/go-sqlite"
)

// versionRecord tracks when a module version was first observed.
type versionRecord struct {
	Module    string    `json:"module"`
	Version   string    `json:"version"`
	FirstSeen time.Time `json:"first_seen"`
}

// seenStore is the interface for first-seen timestamp storage.
type seenStore interface {
	firstSeen(module, version string) time.Time
	close() error
}

// seenDB is a simple JSON-backed first-seen timestamp database (local mode).
type seenDB struct {
	mu      sync.RWMutex
	path    string
	records map[string]versionRecord // key: "module@version"
}

func newSeenDB(path string) (*seenDB, error) {
	db := &seenDB{
		path:    path,
		records: make(map[string]versionRecord),
	}
	data, err := os.ReadFile(path)
	if err == nil {
		var records []versionRecord
		if err := json.Unmarshal(data, &records); err == nil {
			for _, r := range records {
				db.records[r.Module+"@"+r.Version] = r
			}
		}
	}
	return db, nil
}

func (db *seenDB) firstSeen(module, version string) time.Time {
	db.mu.Lock()
	defer db.mu.Unlock()

	key := module + "@" + version
	if r, ok := db.records[key]; ok {
		return r.FirstSeen
	}

	// First time seeing this version — record now.
	now := time.Now().UTC()
	db.records[key] = versionRecord{
		Module:    module,
		Version:   version,
		FirstSeen: now,
	}
	db.persist()
	return now
}

func (db *seenDB) persist() {
	records := make([]versionRecord, 0, len(db.records))
	for _, r := range db.records {
		records = append(records, r)
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return
	}
	os.MkdirAll(filepath.Dir(db.path), 0o755)
	os.WriteFile(db.path, data, 0o644)
}

func (db *seenDB) close() error { return nil }

// seedRecord sets a record directly; used only in tests.
func (db *seenDB) seedRecord(module, version string, t time.Time) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.records[module+"@"+version] = versionRecord{Module: module, Version: version, FirstSeen: t}
}

// hasRecord reports whether a first-seen record exists; used only in tests.
func (db *seenDB) hasRecord(module, version string) bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	_, ok := db.records[module+"@"+version]
	return ok
}

// sqliteSeenStore is a SQLite-backed first-seen database (hosted mode).
type sqliteSeenStore struct {
	mu sync.Mutex
	db *sql.DB
}

func newSQLiteSeenStore(path string) (*sqliteSeenStore, error) {
	os.MkdirAll(filepath.Dir(path), 0o755)
	db, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS seen (
		key        TEXT PRIMARY KEY,
		module     TEXT NOT NULL,
		version    TEXT NOT NULL,
		first_seen TEXT NOT NULL
	)`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create table: %w", err)
	}
	return &sqliteSeenStore{db: db}, nil
}

func (s *sqliteSeenStore) firstSeen(module, version string) time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := module + "@" + version

	// Try existing record first.
	var raw string
	err := s.db.QueryRow(`SELECT first_seen FROM seen WHERE key = ?`, key).Scan(&raw)
	if err == nil {
		if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
			return t
		}
	}

	// First time seeing this version — insert now.
	now := time.Now().UTC()
	s.db.Exec(
		`INSERT OR IGNORE INTO seen(key, module, version, first_seen) VALUES(?,?,?,?)`,
		key, module, version, now.Format(time.RFC3339Nano),
	)
	return now
}

func (s *sqliteSeenStore) close() error { return s.db.Close() }

// errGone is returned by fetchUpstream when the upstream responds with 410 Gone.
var errGone = errors.New("410 Gone")

// proxy is the cooldown-filtering Go module proxy.
type proxy struct {
	upstream   string
	cooldown   time.Duration
	db         seenStore
	cacheDir   string
	trusted    []string // module path prefixes that bypass cooldown
	httpClient *http.Client
}

func (p *proxy) isTrusted(module string) bool {
	for _, prefix := range p.trusted {
		if strings.HasPrefix(module, prefix) {
			return true
		}
	}
	return false
}

// ServeHTTP handles GOPROXY protocol requests.
func (p *proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		fmt.Fprintf(w, "kukicha-proxy — cooldown: %s\n", p.cooldown)
		return
	}

	// Health check endpoint.
	if path == "_health" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":   "ok",
			"cooldown": p.cooldown.String(),
			"upstream": p.upstream,
		})
		return
	}

	// Sumdb proxy: forward sumdb/ requests to sum.golang.org.
	if strings.HasPrefix(path, "sumdb/") {
		p.handleSumDB(w, r, strings.TrimPrefix(path, "sumdb/"))
		return
	}

	// Parse GOPROXY path: $module/@v/$request
	// Find the @v/ or @latest segment.
	atIdx := strings.Index(path, "/@v/")
	latestIdx := strings.Index(path, "/@latest")

	if atIdx < 0 && latestIdx < 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	if latestIdx >= 0 && (atIdx < 0 || latestIdx < atIdx) {
		// /@latest request
		module := path[:latestIdx]
		p.handleLatest(w, r, module)
		return
	}

	module := path[:atIdx]
	rest := path[atIdx+len("/@v/"):]

	switch {
	case rest == "list":
		p.handleList(w, r, module)
	case strings.HasSuffix(rest, ".info"):
		version, _ := strings.CutSuffix(rest, ".info")
		p.handleInfo(w, r, module, version)
	case strings.HasSuffix(rest, ".mod"):
		version, _ := strings.CutSuffix(rest, ".mod")
		p.handleMod(w, r, module, version)
	case strings.HasSuffix(rest, ".zip"):
		version, _ := strings.CutSuffix(rest, ".zip")
		p.handleZip(w, r, module, version)
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func (p *proxy) handleList(w http.ResponseWriter, _ *http.Request, module string) {
	body, err := p.fetchUpstream(module + "/@v/list")
	if err != nil {
		if errors.Is(err, errGone) {
			http.Error(w, "gone", http.StatusGone)
		} else {
			http.Error(w, err.Error(), http.StatusBadGateway)
		}
		return
	}
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		http.Error(w, "upstream read error", http.StatusBadGateway)
		return
	}

	// If trusted publisher, return unfiltered.
	if p.isTrusted(module) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write(data)
		return
	}

	// Filter versions by cooldown.
	versions := strings.Split(strings.TrimSpace(string(data)), "\n")
	now := time.Now().UTC()
	var filtered []string
	for _, v := range versions {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		seen := p.db.firstSeen(module, v)
		age := now.Sub(seen)
		if age >= p.cooldown {
			filtered = append(filtered, v)
		}
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if len(filtered) > 0 {
		fmt.Fprintln(w, strings.Join(filtered, "\n"))
	}
}

func (p *proxy) handleLatest(w http.ResponseWriter, _ *http.Request, module string) {
	body, err := p.fetchUpstream(module + "/@latest")
	if err != nil {
		if errors.Is(err, errGone) {
			http.Error(w, "gone", http.StatusGone)
		} else {
			http.Error(w, err.Error(), http.StatusBadGateway)
		}
		return
	}
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		http.Error(w, "upstream read error", http.StatusBadGateway)
		return
	}

	// If trusted, pass through.
	if p.isTrusted(module) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
		return
	}

	// Parse and check cooldown.
	var info struct {
		Version string    `json:"Version"`
		Time    time.Time `json:"Time"`
	}
	if err := json.Unmarshal(data, &info); err != nil {
		http.Error(w, "upstream parse error", http.StatusBadGateway)
		return
	}

	seen := p.db.firstSeen(module, info.Version)
	age := time.Now().UTC().Sub(seen)
	if age < p.cooldown {
		// Latest version is too new — return 404 so Go falls back to next proxy.
		http.Error(w, "latest version in cooldown", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func (p *proxy) handleInfo(w http.ResponseWriter, _ *http.Request, module, version string) {
	// Always record first-seen to start the cooldown clock.
	seen := p.db.firstSeen(module, version)
	if !p.isTrusted(module) && time.Since(seen) < p.cooldown {
		http.Error(w, "version in cooldown", http.StatusNotFound)
		return
	}
	p.proxyPassthrough(w, module+"/@v/"+version+".info", "application/json")
}

func (p *proxy) handleMod(w http.ResponseWriter, _ *http.Request, module, version string) {
	seen := p.db.firstSeen(module, version)
	if !p.isTrusted(module) && time.Since(seen) < p.cooldown {
		http.Error(w, "version in cooldown", http.StatusNotFound)
		return
	}
	p.cachedProxyPassthrough(w, module+"/@v/"+version+".mod", "text/plain; charset=utf-8")
}

func (p *proxy) handleZip(w http.ResponseWriter, _ *http.Request, module, version string) {
	seen := p.db.firstSeen(module, version)
	if !p.isTrusted(module) && time.Since(seen) < p.cooldown {
		http.Error(w, "version in cooldown", http.StatusNotFound)
		return
	}
	p.cachedProxyPassthrough(w, module+"/@v/"+version+".zip", "application/zip")
}

// handleSumDB proxies requests to sum.golang.org, preventing module fetch
// patterns from leaking to Google's infrastructure.
func (p *proxy) handleSumDB(w http.ResponseWriter, _ *http.Request, subpath string) {
	target := "https://sum.golang.org/" + subpath
	resp, err := p.httpClient.Get(target)
	if err != nil {
		http.Error(w, "sumdb upstream error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (p *proxy) proxyPassthrough(w http.ResponseWriter, path, contentType string) {
	body, err := p.fetchUpstream(path)
	if err != nil {
		if errors.Is(err, errGone) {
			http.Error(w, "gone", http.StatusGone)
		} else {
			http.Error(w, err.Error(), http.StatusBadGateway)
		}
		return
	}
	defer body.Close()
	w.Header().Set("Content-Type", contentType)
	io.Copy(w, body)
}

// cachedProxyPassthrough serves from disk cache when available, falling back
// to upstream. Suitable for immutable responses (.mod and .zip).
func (p *proxy) cachedProxyPassthrough(w http.ResponseWriter, path, contentType string) {
	if p.cacheDir != "" {
		cachePath := filepath.Join(p.cacheDir, filepath.FromSlash(path))
		if data, err := os.ReadFile(cachePath); err == nil {
			w.Header().Set("Content-Type", contentType)
			w.Write(data)
			return
		}
	}

	body, err := p.fetchUpstream(path)
	if err != nil {
		if errors.Is(err, errGone) {
			http.Error(w, "gone", http.StatusGone)
		} else {
			http.Error(w, err.Error(), http.StatusBadGateway)
		}
		return
	}
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		http.Error(w, "upstream read error", http.StatusBadGateway)
		return
	}

	// Persist to cache (best-effort).
	if p.cacheDir != "" {
		cachePath := filepath.Join(p.cacheDir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err == nil {
			os.WriteFile(cachePath, data, 0o644)
		}
	}

	w.Header().Set("Content-Type", contentType)
	w.Write(data)
}

func (p *proxy) fetchUpstream(path string) (io.ReadCloser, error) {
	upstreamURL := p.upstream + "/" + path
	resp, err := p.httpClient.Get(upstreamURL)
	if err != nil {
		return nil, fmt.Errorf("upstream error: %w", err)
	}
	switch resp.StatusCode {
	case http.StatusOK:
		return resp.Body, nil
	case http.StatusGone:
		resp.Body.Close()
		return nil, errGone
	case http.StatusNotFound:
		resp.Body.Close()
		return nil, fmt.Errorf("not found upstream: %s", resp.Status)
	default:
		resp.Body.Close()
		return nil, fmt.Errorf("upstream returned %s", resp.Status)
	}
}

func parseDuration(s string) (time.Duration, error) {
	// Support day suffix: "7d" → 7 * 24h
	if strings.HasSuffix(s, "d") {
		s = strings.TrimSuffix(s, "d")
		var days int
		if _, err := fmt.Sscanf(s, "%d", &days); err != nil {
			return 0, fmt.Errorf("invalid day duration: %s", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

func main() {
	mode := flag.String("mode", "local", "proxy mode: local or hosted")
	addr := flag.String("addr", ":8250", "listen address")
	cooldown := flag.String("cooldown", "7d", "cooldown duration (e.g., 7d, 168h)")
	upstream := flag.String("upstream", "", "upstream proxy URL (auto-detected from mode if empty)")
	dataDir := flag.String("data", "", "data directory (default: ~/.kukicha/proxy)")
	trusted := flag.String("trusted", "github.com/kukichalang/,golang.org/x/,gopkg.in/", "comma-separated trusted module prefixes (bypass cooldown)")
	flag.Parse()

	cooldownDur, err := parseDuration(*cooldown)
	if err != nil {
		log.Fatalf("invalid cooldown: %v", err)
	}

	// Resolve data directory.
	dir := *dataDir
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Fatal(err)
		}
		dir = filepath.Join(home, ".kukicha", "proxy")
	}
	os.MkdirAll(dir, 0o755)

	// Resolve upstream.
	up := *upstream
	if up == "" {
		switch *mode {
		case "local":
			up = "https://proxy.kukicha.dev"
		case "hosted":
			up = "https://proxy.golang.org"
		default:
			log.Fatalf("unknown mode: %s", *mode)
		}
	}
	// Validate upstream URL.
	if _, err := url.Parse(up); err != nil {
		log.Fatalf("invalid upstream URL: %v", err)
	}

	// Choose seenStore implementation based on mode.
	var store seenStore
	switch *mode {
	case "hosted":
		s, err := newSQLiteSeenStore(filepath.Join(dir, "seen.db"))
		if err != nil {
			log.Fatalf("failed to open seen database: %v", err)
		}
		store = s
	default:
		db, err := newSeenDB(filepath.Join(dir, "seen.json"))
		if err != nil {
			log.Fatalf("failed to open seen database: %v", err)
		}
		store = db
	}

	var trustedPrefixes []string
	for _, t := range strings.Split(*trusted, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			trustedPrefixes = append(trustedPrefixes, t)
		}
	}

	p := &proxy{
		upstream: up,
		cooldown: cooldownDur,
		db:       store,
		cacheDir: filepath.Join(dir, "cache"),
		trusted:  trustedPrefixes,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	srv := &http.Server{
		Addr:    *addr,
		Handler: p,
	}

	log.Printf("kukicha-proxy starting (%s mode)", *mode)
	log.Printf("  listen:   %s", *addr)
	log.Printf("  upstream: %s", up)
	log.Printf("  cooldown: %s", cooldownDur)
	log.Printf("  trusted:  %v", trustedPrefixes)
	log.Printf("  data:     %s", dir)

	// Graceful shutdown on SIGINT/SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-quit
		log.Println("shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		store.close()
		srv.Shutdown(ctx)
	}()

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
