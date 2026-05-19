package server_test

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/derrick-wippler-anchor/host/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startServer finds a free port, starts Run in a goroutine, waits for the
// server to be ready, and returns (port, stdout, stderr).
func startServer(t *testing.T, dir string, extraArgs []string, env []string) (int, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()

	// Find a free port.
	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()

	var stdout, stderr bytes.Buffer

	// Build args: --port PORT dir [extraArgs...]
	args := append([]string{"--port", fmt.Sprintf("%d", port), dir}, extraArgs...)

	go func() {
		//nolint:errcheck
		server.Run(args, env, &stdout, &stderr)
	}()

	// Poll until the server is ready (up to 500ms).
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", port))
		if err == nil {
			_ = resp.Body.Close()
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	return port, &stdout, &stderr
}

func TestRunServesMarkdownFile(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "hello.md"), []byte("# Hello\n"), 0644)
	require.NoError(t, err)

	port, _, _ := startServer(t, dir, nil, nil)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/hello.md", port))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/html")

	var body bytes.Buffer
	_, _ = body.ReadFrom(resp.Body)
	assert.Contains(t, body.String(), "<h1>")
}

func TestRunServesDirectory(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# Readme\n"), 0644)
	require.NoError(t, err)

	port, _, _ := startServer(t, dir, nil, nil)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", port))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body bytes.Buffer
	_, _ = body.ReadFrom(resp.Body)
	assert.Contains(t, body.String(), "readme.md")
}

func TestRunServesDirectoryWithSubdirectory(t *testing.T) {
	dir := t.TempDir()
	err := os.Mkdir(filepath.Join(dir, "docs"), 0755)
	require.NoError(t, err)

	port, _, _ := startServer(t, dir, nil, nil)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", port))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body bytes.Buffer
	_, _ = body.ReadFrom(resp.Body)
	// Directory entries are linked with trailing slash.
	assert.Contains(t, body.String(), "docs")
}

func TestRunServesSubdirectory(t *testing.T) {
	dir := t.TempDir()
	err := os.Mkdir(filepath.Join(dir, "docs"), 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "docs", "guide.md"), []byte("# Guide\n"), 0644)
	require.NoError(t, err)

	port, _, _ := startServer(t, dir, nil, nil)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/docs/", port))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body bytes.Buffer
	_, _ = body.ReadFrom(resp.Body)
	assert.Contains(t, body.String(), "guide.md")
}

func TestRunServesDirectoryContainsReloadScript(t *testing.T) {
	dir := t.TempDir()

	port, _, _ := startServer(t, dir, nil, nil)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", port))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var body bytes.Buffer
	_, _ = body.ReadFrom(resp.Body)
	assert.Contains(t, body.String(), "/__reload")
}

func TestRunServesHTMLWithReloadScript(t *testing.T) {
	dir := t.TempDir()
	content := `<!DOCTYPE html><html><body><p>hello</p></body></html>`
	err := os.WriteFile(filepath.Join(dir, "page.html"), []byte(content), 0644)
	require.NoError(t, err)

	port, _, _ := startServer(t, dir, nil, nil)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/page.html", port))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	var body bytes.Buffer
	_, _ = body.ReadFrom(resp.Body)
	bodyStr := body.String()

	assert.Contains(t, bodyStr, "/__reload")
	// Script must be injected before </body>.
	scriptIdx := strings.Index(bodyStr, "<script>")
	bodyTagIdx := strings.Index(bodyStr, "</body>")
	assert.True(t, scriptIdx < bodyTagIdx, "script tag should appear before </body>")
}

func TestRunServesStaticFile(t *testing.T) {
	dir := t.TempDir()
	cssContent := "body { color: red; }"
	err := os.WriteFile(filepath.Join(dir, "style.css"), []byte(cssContent), 0644)
	require.NoError(t, err)

	port, _, _ := startServer(t, dir, nil, nil)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/style.css", port))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/css")

	var body bytes.Buffer
	_, _ = body.ReadFrom(resp.Body)
	assert.Equal(t, cssContent, body.String())
}

func TestRunPathTraversalReturns404(t *testing.T) {
	dir := t.TempDir()

	port, _, _ := startServer(t, dir, nil, nil)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/../etc/passwd", port))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestRunNotFound(t *testing.T) {
	dir := t.TempDir()

	port, _, _ := startServer(t, dir, nil, nil)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/nonexistent.txt", port))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestRunStartupURLLocal(t *testing.T) {
	dir := t.TempDir()

	port, stdout, _ := startServer(t, dir, nil, nil)

	// Give the goroutine a moment to write startup URL.
	time.Sleep(50 * time.Millisecond)

	assert.Contains(t, stdout.String(), fmt.Sprintf("http://localhost:%d", port))
}

func TestRunStartupURLGCW(t *testing.T) {
	dir := t.TempDir()
	env := []string{"WEB_HOST=myhost.cloudworkstations.dev"}

	port, stdout, _ := startServer(t, dir, nil, env)

	// Give the goroutine a moment to write startup URL.
	time.Sleep(50 * time.Millisecond)

	expected := fmt.Sprintf("https://%d-myhost.cloudworkstations.dev/", port)
	assert.Contains(t, stdout.String(), expected)
}

func TestRunDefaultPort(t *testing.T) {
	// Pre-occupy port 8080 to confirm the default port is 8080.
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		t.Skip("port 8080 is already in use; skipping default port test")
	}
	defer func() { _ = ln.Close() }()

	dir := t.TempDir()
	var stdout, stderr bytes.Buffer

	// Run with no --port flag; it should attempt port 8080 and fail.
	runErr := server.Run([]string{dir}, nil, &stdout, &stderr)
	require.Error(t, runErr)
	assert.Contains(t, stderr.String(), "8080")
	assert.Contains(t, stderr.String(), "is already in use")
}

func TestRunPortConflict(t *testing.T) {
	// Pre-occupy a port.
	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port
	defer func() { _ = ln.Close() }()

	dir := t.TempDir()
	var stdout, stderr bytes.Buffer

	runErr := server.Run([]string{"--port", fmt.Sprintf("%d", port), dir}, nil, &stdout, &stderr)
	require.Error(t, runErr)
	assert.Contains(t, stderr.String(), "is already in use")
}

func TestRunInvalidDirectory(t *testing.T) {
	var stdout, stderr bytes.Buffer

	runErr := server.Run([]string{"/nonexistent/path/that/does/not/exist"}, nil, &stdout, &stderr)
	require.Error(t, runErr)
	assert.Contains(t, stderr.String(), "does not exist")
}

func TestRunUnknownFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer

	runErr := server.Run([]string{"--unknownflag"}, nil, &stdout, &stderr)
	require.Error(t, runErr)
	assert.NotEmpty(t, stderr.String())
}

func TestRunPortFlagAlias(t *testing.T) {
	dir := t.TempDir()

	// Find a free port.
	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()

	var stdout, stderr bytes.Buffer

	// Use -p shorthand instead of --port.
	go func() {
		//nolint:errcheck
		server.Run([]string{"-p", fmt.Sprintf("%d", port), dir}, nil, &stdout, &stderr)
	}()

	// Wait for server to be ready.
	deadline := time.Now().Add(500 * time.Millisecond)
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", port))
		if err == nil {
			_ = resp.Body.Close()
			lastErr = nil
			break
		}
		lastErr = err
		time.Sleep(10 * time.Millisecond)
	}
	require.NoError(t, lastErr, "server did not become ready using -p flag")
}

func TestRunSSELiveReload(t *testing.T) {
	dir := t.TempDir()
	// Write an initial file so the directory is non-empty.
	initialFile := filepath.Join(dir, "initial.txt")
	err := os.WriteFile(initialFile, []byte("initial content"), 0644)
	require.NoError(t, err)

	port, _, _ := startServer(t, dir, nil, nil)

	// Connect to the SSE endpoint and keep the connection open.
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/__reload", port))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	events := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			if line := scanner.Text(); strings.HasPrefix(line, "data:") {
				events <- line
			}
		}
	}()

	// Modify a file to trigger a reload event.
	// Wait briefly to ensure the poller has taken its first snapshot.
	time.Sleep(600 * time.Millisecond)
	err = os.WriteFile(initialFile, []byte("modified content"), 0644)
	require.NoError(t, err)

	// Wait for SSE reload event; the 500ms ticker gives ~4 chances in 2s.
	select {
	case event := <-events:
		assert.Contains(t, event, "reload")
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: no SSE reload event received")
	}
}

func TestRunSSEHeaders(t *testing.T) {
	dir := t.TempDir()

	port, _, _ := startServer(t, dir, nil, nil)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/__reload", port))
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Contains(t, resp.Header.Get("Content-Type"), "text/event-stream")
	assert.Equal(t, "no-cache", resp.Header.Get("Cache-Control"))
}
