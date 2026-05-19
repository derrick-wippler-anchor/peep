package server

import (
	"fmt"
	"html/template"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type handler struct {
	root   string
	broker *broker
}

func newHandler(root string, b *broker) *handler {
	return &handler{root: root, broker: b}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Path sanitization: clean the URL path.
	cleaned := path.Clean(r.URL.Path)

	// Route /__reload to SSE handler before filesystem resolution.
	if cleaned == "/__reload" {
		h.serveSSE(w, r)
		return
	}

	// Resolve to absolute filesystem path.
	resolved := filepath.Join(h.root, filepath.FromSlash(cleaned))

	// Verify the resolved path is within root (prevents traversal attacks).
	if resolved != h.root && !strings.HasPrefix(resolved, h.root+string(filepath.Separator)) {
		http.NotFound(w, r)
		return
	}

	// Stat the resolved path.
	info, err := os.Stat(resolved)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Dispatch based on type.
	if info.IsDir() {
		h.serveDirectory(w, r, resolved)
		return
	}

	switch {
	case strings.HasSuffix(resolved, ".md"):
		h.serveMarkdown(w, r, resolved)
	case strings.HasSuffix(resolved, ".html"):
		h.serveHTML(w, r, resolved)
	default:
		h.serveStatic(w, r, resolved)
	}
}

// serveSSE registers the client with the broker and streams reload events.
func (h *handler) serveSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")

	ch := make(chan struct{}, 1)
	h.broker.subscribe <- ch
	defer func() {
		h.broker.unsubscribe <- ch
	}()

	flusher, ok := w.(http.Flusher)
	if ok {
		flusher.Flush()
	}

	for {
		select {
		case <-ch:
			_, _ = fmt.Fprint(w, "data: reload\n\n")
			if ok {
				flusher.Flush()
			}
		case <-r.Context().Done():
			return
		}
	}
}

// dirEntry holds one directory listing entry for the template.
type dirEntry struct {
	Name string
	Href string
}

// serveDirectory renders the directory listing using the directory.html template.
func (h *handler) serveDirectory(w http.ResponseWriter, r *http.Request, resolved string) {
	entries, err := os.ReadDir(resolved)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var items []dirEntry
	for _, e := range entries {
		name := e.Name()
		href := name
		if e.IsDir() {
			href = name + "/"
		}
		items = append(items, dirEntry{Name: name, Href: href})
	}

	tmplBytes, err := templateFS.ReadFile("templates/directory.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("directory").Parse(string(tmplBytes))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, items); err != nil {
		// Headers already sent; can't change status code.
		return
	}
}

// serveMarkdown renders a .md file as HTML wrapped in the markdown.html template.
func (h *handler) serveMarkdown(w http.ResponseWriter, r *http.Request, resolved string) {
	src, err := os.ReadFile(resolved)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rendered, err := RenderMarkdown(src)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmplBytes, err := templateFS.ReadFile("templates/markdown.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.New("markdown").Parse(string(tmplBytes))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, template.HTML(rendered)); err != nil {
		return
	}
}

// serveHTML reads an .html file and injects the reload script before </body>.
func (h *handler) serveHTML(w http.ResponseWriter, r *http.Request, resolved string) {
	data, err := os.ReadFile(resolved)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	const reloadScript = `<script>var es = new EventSource('/__reload'); es.onmessage = function() { location.reload(); };</script>`
	body := strings.Replace(string(data), "</body>", reloadScript+"</body>", 1)

	w.Header().Set("Content-Type", "text/html")
	_, _ = fmt.Fprint(w, body)
}

// serveStatic serves raw file bytes with a Content-Type detected from the extension.
func (h *handler) serveStatic(w http.ResponseWriter, r *http.Request, resolved string) {
	data, err := os.ReadFile(resolved)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ext := filepath.Ext(resolved)
	ct := mime.TypeByExtension(ext)
	if ct != "" {
		w.Header().Set("Content-Type", ct)
	}

	_, _ = w.Write(data)
}
