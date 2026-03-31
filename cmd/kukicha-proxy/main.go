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
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// versionRecord tracks when a module version was first observed.
type versionRecord struct {
	Module    string    `json:"module"`
	Version   string    `json:"version"`
	FirstSeen time.Time `json:"first_seen"`
}

// seenDB is a simple JSON-backed first-seen timestamp database.
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

// proxy is the cooldown-filtering Go module proxy.
type proxy struct {
	upstream   string
	cooldown   time.Duration
	db         *seenDB
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
		version := strings.TrimSuffix(rest, ".info")
		p.handleInfo(w, r, module, version)
	case strings.HasSuffix(rest, ".mod"):
		version := strings.TrimSuffix(rest, ".mod")
		p.handleMod(w, r, module, version)
	case strings.HasSuffix(rest, ".zip"):
		version := strings.TrimSuffix(rest, ".zip")
		p.handleZip(w, r, module, version)
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func (p *proxy) handleList(w http.ResponseWriter, r *http.Request, module string) {
	body, err := p.fetchUpstream(module + "/@v/list")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
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

func (p *proxy) handleLatest(w http.ResponseWriter, r *http.Request, module string) {
	body, err := p.fetchUpstream(module + "/@latest")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
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

func (p *proxy) handleInfo(w http.ResponseWriter, r *http.Request, module, version string) {
	// .info requests for a specific version — record first-seen but always serve.
	// The cooldown filtering happens at list/latest level.
	p.db.firstSeen(module, version)
	p.proxyPassthrough(w, module+"/@v/"+version+".info", "application/json")
}

func (p *proxy) handleMod(w http.ResponseWriter, r *http.Request, module, version string) {
	p.proxyPassthrough(w, module+"/@v/"+version+".mod", "text/plain; charset=utf-8")
}

func (p *proxy) handleZip(w http.ResponseWriter, r *http.Request, module, version string) {
	p.proxyPassthrough(w, module+"/@v/"+version+".zip", "application/zip")
}

func (p *proxy) proxyPassthrough(w http.ResponseWriter, path, contentType string) {
	body, err := p.fetchUpstream(path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer body.Close()
	w.Header().Set("Content-Type", contentType)
	io.Copy(w, body)
}

func (p *proxy) fetchUpstream(path string) (io.ReadCloser, error) {
	upstreamURL := p.upstream + "/" + path
	resp, err := p.httpClient.Get(upstreamURL)
	if err != nil {
		return nil, fmt.Errorf("upstream error: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
		resp.Body.Close()
		return nil, fmt.Errorf("not found upstream: %s", resp.Status)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("upstream returned %s", resp.Status)
	}
	return resp.Body, nil
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

	db, err := newSeenDB(filepath.Join(dir, "seen.json"))
	if err != nil {
		log.Fatalf("failed to open seen database: %v", err)
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
		db:       db,
		cacheDir: filepath.Join(dir, "cache"),
		trusted:  trustedPrefixes,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	log.Printf("kukicha-proxy starting (%s mode)", *mode)
	log.Printf("  listen:   %s", *addr)
	log.Printf("  upstream: %s", up)
	log.Printf("  cooldown: %s", cooldownDur)
	log.Printf("  trusted:  %v", trustedPrefixes)
	log.Printf("  data:     %s", dir)

	if err := http.ListenAndServe(*addr, p); err != nil {
		log.Fatal(err)
	}
}
