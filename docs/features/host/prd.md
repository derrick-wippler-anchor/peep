# host PRD

## Problem

Developers writing documentation or content in Markdown need a way to preview their work locally in a browser with live feedback as they edit. Existing tools either require a runtime dependency (Node.js for `live-server`, Python for `http.server`) or are full static site generators with build steps and configuration overhead.

There is no single self-contained binary that serves local files with live-reload and treats Markdown rendering as a first-class feature. `host` fills that gap.

## Users

Primary users are developers who write documentation, READMEs, or content in Markdown and want to preview it locally while they work. This is a single-developer tool — there are no multi-user, team collaboration, or access-control requirements.

## Core Design Principles

- **Zero configuration.** Running `host` with no arguments works from any project directory.
- **Markdown is first-class.** `.md` files are rendered to HTML, not served as raw text. This is the primary differentiator from generic file servers.
- **Single binary.** No runtime dependencies. Installable via `go install`.
- **Local-only.** No auth, no TLS, no remote paths — this is a local development tool.

## User Stories

- As a developer writing Markdown docs, I want to run `host` in my project directory and see `.md` files rendered as HTML in the browser, so I can preview formatting without a build step.
- As a developer, I want the browser to automatically reload when I save a file, so I can see changes immediately without a manual refresh.
- As a developer, I want to navigate my project's files and directories in the browser, so I can open any doc without knowing its exact path.
- As a developer on a Google Cloud Workstation, I want the tool to print a clickable URL that works from my local machine, so I don't have to manually construct the proxied hostname.
- As a developer, I want code blocks in Markdown to be syntax-highlighted, so rendered docs look polished and are easy to read.

## Scope

### In Scope

- Serve a local directory and all subdirectories over HTTP
- Render `.md` files to HTML using GitHub Flavored Markdown (tables, fenced code blocks, task lists, strikethrough)
- Syntax highlight fenced code blocks
- Strip YAML frontmatter silently — do not render it as content
- Auto-generated directory index pages with links to files and subdirectories
- Serve static assets (HTML, CSS, images, JS, fonts, etc.) from the directory tree unchanged
- Live reload via SSE: inject a `<script>` tag into every served HTML page and every Markdown-rendered page; the script triggers `window.location.reload()` when the server signals a file change
- Filesystem watcher that detects any file change under the served directory and signals connected SSE clients
- CLI: `host [directory] [--port/-p PORT]`
  - `directory`: path to serve; defaults to current working directory
  - `--port / -p`: port to listen on; defaults to `8080`
- On startup, print the correct URL: `https://PORT-WEB_HOST/` if the `WEB_HOST` environment variable is set (Google Cloud Workstations), otherwise `http://localhost:PORT`
- On port conflict, print a clear error message and exit with a non-zero status code

### Out of Scope / Non-Goals

- HTTPS / TLS termination
- Authentication or access control
- Custom Markdown themes or user-configurable CSS for rendered pages
- Serving from a remote or network-mounted path
- Watch pattern filtering (e.g., ignoring `node_modules` or `.git`)
- Parsing or using Markdown frontmatter for page metadata
- Auto-opening the browser on startup

## Dependencies and Constraints

- Written in Go; distributed as a single binary via `go install`
- End users have no runtime dependencies to install
- Must run correctly on Google Cloud Workstations (Linux), where `localhost` is not directly accessible from the developer's browser — GCW sets the `WEB_HOST` environment variable automatically; the correct externally-accessible URL is `https://PORT-WEB_HOST/`; when `WEB_HOST` is absent the tool falls back to `http://localhost:PORT`
- Expected key library dependencies (resolved at tech-spec stage): a GFM-capable Markdown parser, a syntax highlighting library, and a filesystem watcher

## Open Questions

None — scope is fully resolved for tech spec to begin.
