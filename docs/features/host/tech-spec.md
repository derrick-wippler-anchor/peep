# host Tech Spec

_PRD: docs/features/host/prd.md_

## Overview

`host` is a self-contained Go binary that serves a local directory over HTTP with Markdown rendering and SSE-based live reload. It targets developers previewing documentation locally and on Google Cloud Workstations.

## Component Design

### Package layout

```
cmd/host/main.go              // entry point: parse args, call server.Run()
internal/server/
  server.go                   // Run(), Config, flag parsing, startup
  handler.go                  // http.Handler implementation
  broker.go                   // SSE client broker goroutine
  poller.go                   // filesystem polling goroutine
  markdown.go                 // goldmark rendering, frontmatter stripping
  templates/
    markdown.html             // Markdown page wrapper (embedded)
    directory.html            // directory index page (embedded)
```

Templates are embedded at compile time via `//go:embed templates/*`.

### Entry point

`cmd/host/main.go`:

```go
func main() {
    if err := server.Run(os.Args[1:], os.Environ(), os.Stdout, os.Stderr); err != nil {
        os.Exit(1)
    }
}
```

### `Run()` signature

```go
// internal/server/server.go
func Run(args []string, env []string, stdout, stderr io.Writer) error
```

`Run` parses flags from `args`, reads `WEB_HOST` from `env`, prints the startup URL to `stdout`, and blocks until the server exits. On startup failure (port conflict, bad args) it writes a message to `stderr` and returns a non-nil error.

### Config

```go
type Config struct {
    Dir    string
    Port   int
    Stdout io.Writer
    Stderr io.Writer
}
```

Parsed from flags; `Dir` defaults to `.`, `Port` defaults to `8080`.

### HTTP handler

A single `handler` struct implements `http.Handler`. Its `ServeHTTP` dispatches on path:

| Path | Action |
|------|--------|
| `/__reload` | SSE handler — registers client with broker, streams reload events |
| directory path | render directory index from `directory.html` template |
| `*.md` path | strip frontmatter, render Markdown, wrap in `markdown.html` template |
| `*.html` path | read file, inject reload script before `</body>`, write response |
| everything else | serve file bytes with correct `Content-Type` |

The handler holds a reference to the broker so the SSE path can subscribe/unsubscribe.

### Broker

```go
// internal/server/broker.go
type broker struct {
    subscribe   chan chan struct{}
    unsubscribe chan chan struct{}
    broadcast   chan struct{}
}

func newBroker() *broker
func (b *broker) run(ctx context.Context)
```

`run` owns a `map[chan struct{}]struct{}` of connected client channels. It selects on `subscribe`, `unsubscribe`, and `broadcast`. On `broadcast`, it sends (non-blocking) to every client channel. On `ctx` cancellation, it drains and returns.

The SSE handler creates `ch := make(chan struct{}, 1)`, sends it to `b.subscribe` on connect, defers sending it to `b.unsubscribe` on disconnect, and selects on `ch` to write `data: reload\n\n` and flush.

### Poller

```go
// internal/server/poller.go
type fileState struct {
    mtime time.Time
    size  int64
}

func poll(ctx context.Context, root string, tick <-chan time.Time, b *broker)
```

On each tick, `poll` walks `root` collecting `map[string]fileState`. It compares to the previous snapshot; any added, removed, or changed entry triggers `b.broadcast <- struct{}{}`. The `tick` channel is injected — production passes `time.NewTicker(500*time.Millisecond).C`, tests pass a controlled channel.

### Markdown renderer

```go
// internal/server/markdown.go
func stripFrontmatter(src []byte) []byte
func renderMarkdown(src []byte) ([]byte, error)
```

`stripFrontmatter` checks whether `src` starts with `---\n`; if so, it finds the closing `---\n` and returns the bytes after it. `renderMarkdown` runs the stripped bytes through a goldmark instance configured with GFM extensions and chroma highlighting.

## API Design

### HTTP endpoints

**`GET /__reload`** — SSE stream

```
Content-Type: text/event-stream
Cache-Control: no-cache

data: reload

data: reload
```

One `data: reload\n\n` frame per detected file change.

**`GET /<path>`** — file or directory

- Directory: `200 OK`, `Content-Type: text/html`, directory index page
- `.md` file: `200 OK`, `Content-Type: text/html`, rendered Markdown page
- Other file: `200 OK`, appropriate `Content-Type`, raw bytes
- Not found: `404 Not Found`

### Reload script

Injected into every HTML response (baked into `markdown.html` template; appended before `</body>` in static `.html` files):

```html
<script>
  var es = new EventSource('/__reload');
  es.onmessage = function() { location.reload(); };
</script>
```

### Startup output

Written to `stdout` on successful bind:

- GCW (`WEB_HOST` set): `https://PORT-WEB_HOST/`
- Local: `http://localhost:PORT`

## Dependencies

| Library | Purpose |
|---------|---------|
| `github.com/yuin/goldmark` | Markdown parsing |
| `github.com/yuin/goldmark/extension` | GFM extensions (tables, strikethrough, task lists, autolinks) |
| `github.com/alecthomas/chroma/v2` | Syntax highlighting |
| `github.com/yuin/goldmark-highlighting/v2` | goldmark ↔ chroma bridge |

No filesystem watcher library — polling is implemented directly using `path/filepath.Walk` and `os.Stat`.

## Error Handling

| Condition | Behavior |
|-----------|----------|
| Port already in use | Write `"port PORT is already in use\n"` to `stderr`; return error |
| Unknown flag | Write usage to `stderr`; return error |
| Directory arg does not exist | Write `"directory PATH does not exist\n"` to `stderr`; return error |
| `.md` file render error | Serve `500` with plain-text error body |
| Static file read error | Serve `500` with plain-text error body |
| Path traversal attempt (`../`) | `http.FileServer` handles via `path.Clean`; results in `404` |

## Observability

The poller and broker are internal goroutines with no user-visible status beyond the SSE stream itself. For testing async behavior, `poll` accepts an injectable tick channel (see Poller above) — this also serves as the primary observable path in tests.

No metrics, logging, or health endpoint is in scope for this tool.

## Testing

Testing follows the `surface-testing` skill.

Key surfaces:

- **Integration (`internal/server`)**: call `Run()` with a temp directory, random port, and controlled env. Make real HTTP requests. Assert response bodies and status codes.
- **Unit (`markdown.go`)**: `stripFrontmatter` and `renderMarkdown` take and return `[]byte` — table-driven tests against known inputs.
- **SSE live-reload path**: start server, connect to `/__reload` via a plain HTTP client reading the response body, write a file into the temp dir, send one tick on the injected channel, assert the client reads `data: reload\n\n`.
- **Fakes needed**: none — real OS filesystem via `os.MkdirTemp`; injectable tick channel replaces the real ticker.

The `tick <-chan time.Time` parameter on `poll` is the only substitution point needed. No mocks, no test-only interfaces beyond that.

## Deployment

Distributed as a single binary via:

```
go install github.com/anchorlabsinc/host/cmd/host@latest
```

No runtime dependencies. No configuration files. No migration steps.

## Open Questions

1. **Module path**: spec uses `github.com/anchorlabsinc/host` as a placeholder — confirm the target repository before `go.mod` is initialized.
