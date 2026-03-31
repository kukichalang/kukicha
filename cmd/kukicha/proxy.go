package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultProxyPort = 8250
	defaultProxyHost = "127.0.0.1"
	hostedProxyURL   = "https://proxy.kukicha.dev"
	goProxyURL       = "https://proxy.golang.org"
)

// proxyCommand handles "kukicha proxy" subcommands.
func proxyCommand(args []string) {
	if len(args) < 1 {
		printProxyUsage()
		os.Exit(1)
	}

	sub := args[0]
	subArgs := args[1:]

	switch sub {
	case "start":
		proxyStartCommand(subArgs)
	case "stop":
		proxyStopCommand()
	case "status":
		proxyStatusCommand()
	case "help", "-h", "--help":
		printProxyUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown proxy command: %s\n", sub)
		printProxyUsage()
		os.Exit(1)
	}
}

func printProxyUsage() {
	fmt.Fprintln(os.Stderr, "Usage: kukicha proxy <command>")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  start [--cooldown 7d] [--port 8250]  Start local caching proxy")
	fmt.Fprintln(os.Stderr, "  stop                                  Stop local proxy daemon")
	fmt.Fprintln(os.Stderr, "  status                                Show proxy status")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "The local proxy caches Go modules and layers additional cooldown")
	fmt.Fprintln(os.Stderr, "filtering on top of proxy.kukicha.dev.")
}

func proxyStartCommand(args []string) {
	flags := flag.NewFlagSet("proxy start", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	port := flags.Int("port", defaultProxyPort, "listen port")
	cooldown := flags.String("cooldown", "7d", "cooldown duration (e.g., 7d, 168h)")
	daemon := flags.Bool("daemon", false, "run in background as daemon")
	if err := flags.Parse(args); err != nil {
		os.Exit(1)
	}

	proxyBin, err := findProxyBinary()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: kukicha-proxy binary not found.\n")
		fmt.Fprintln(os.Stderr, "Build it with: go build -o kukicha-proxy ./cmd/kukicha-proxy")
		os.Exit(1)
	}

	addr := fmt.Sprintf(":%d", *port)
	cmdArgs := []string{
		"--mode=local",
		"--addr=" + addr,
		"--cooldown=" + *cooldown,
	}

	if *daemon {
		// Run in background.
		cmd := exec.Command(proxyBin, cmdArgs...)
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting proxy daemon: %v\n", err)
			os.Exit(1)
		}

		// Write PID file.
		pidPath := proxyPIDPath()
		os.MkdirAll(filepath.Dir(pidPath), 0o755)
		os.WriteFile(pidPath, []byte(strconv.Itoa(cmd.Process.Pid)), 0o644)

		fmt.Printf("kukicha proxy started (pid %d) on %s:%d\n", cmd.Process.Pid, defaultProxyHost, *port)
		fmt.Printf("  cooldown: %s\n", *cooldown)
		fmt.Printf("  GOPROXY chain: http://%s:%d,%s,%s,direct\n", defaultProxyHost, *port, hostedProxyURL, goProxyURL)
	} else {
		// Run in foreground.
		fmt.Printf("Starting kukicha proxy on %s:%d (foreground)\n", defaultProxyHost, *port)
		cmd := exec.Command(proxyBin, cmdArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Proxy exited: %v\n", err)
			os.Exit(1)
		}
	}
}

func proxyStopCommand() {
	pidPath := proxyPIDPath()
	data, err := os.ReadFile(pidPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "No running proxy found (no PID file).")
		os.Exit(1)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid PID file: %v\n", err)
		os.Remove(pidPath)
		os.Exit(1)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Process %d not found: %v\n", pid, err)
		os.Remove(pidPath)
		os.Exit(1)
	}

	if err := proc.Signal(os.Interrupt); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to stop proxy (pid %d): %v\n", pid, err)
		os.Exit(1)
	}

	os.Remove(pidPath)
	fmt.Printf("Stopped kukicha proxy (pid %d)\n", pid)
}

func proxyStatusCommand() {
	// Check if local proxy is running.
	localURL := fmt.Sprintf("http://%s:%d/_health", defaultProxyHost, defaultProxyPort)
	client := &http.Client{Timeout: 2 * time.Second}

	fmt.Println("Kukicha proxy status:")
	fmt.Println()

	// Check local proxy.
	resp, err := client.Get(localURL)
	if err != nil {
		fmt.Printf("  Local proxy (%s:%d):  not running\n", defaultProxyHost, defaultProxyPort)
	} else {
		defer resp.Body.Close()
		var health map[string]any
		json.NewDecoder(resp.Body).Decode(&health)
		fmt.Printf("  Local proxy (%s:%d):  running\n", defaultProxyHost, defaultProxyPort)
		if cd, ok := health["cooldown"]; ok {
			fmt.Printf("    cooldown: %v\n", cd)
		}
		if up, ok := health["upstream"]; ok {
			fmt.Printf("    upstream: %v\n", up)
		}
	}

	fmt.Println()
	fmt.Printf("  Hosted proxy:              %s\n", hostedProxyURL)
	fmt.Printf("  GOPROXY chain:             %s\n", buildProxyChain())
}

// buildProxyChain constructs the GOPROXY value for Kukicha builds.
func buildProxyChain() string {
	// Allow full override.
	if override := os.Getenv("KUKICHA_PROXY"); override != "" {
		return override
	}

	chain := []string{}

	// Local proxy if running.
	if localProxy := detectLocalProxy(); localProxy != "" {
		chain = append(chain, localProxy)
	}

	// Hosted proxy always in chain.
	chain = append(chain, hostedProxyURL)

	// Go's default as fallback.
	chain = append(chain, goProxyURL)
	chain = append(chain, "direct")

	return strings.Join(chain, ",")
}

// detectLocalProxy checks if a local proxy is listening.
func detectLocalProxy() string {
	addr := fmt.Sprintf("%s:%d", defaultProxyHost, defaultProxyPort)
	conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
	if err != nil {
		return ""
	}
	conn.Close()
	return fmt.Sprintf("http://%s", addr)
}

// findProxyBinary locates the kukicha-proxy binary.
func findProxyBinary() (string, error) {
	// Check next to the kukicha binary.
	exe, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "kukicha-proxy")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	// Check PATH.
	path, err := exec.LookPath("kukicha-proxy")
	if err == nil {
		return path, nil
	}

	return "", fmt.Errorf("kukicha-proxy not found")
}

func proxyPIDPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/kukicha-proxy.pid"
	}
	return filepath.Join(home, ".kukicha", "proxy", "proxy.pid")
}
