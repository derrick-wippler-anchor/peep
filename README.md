# peep

A zero-configuration local file server with Markdown rendering and live reload.

Run `peep` in any directory to browse files in the browser. Markdown files are rendered as HTML with GitHub Flavored Markdown and syntax-highlighted code blocks. The browser reloads automatically whenever a file changes.

Works on Google Cloud Workstations — when the `WEB_HOST` environment variable is set, peep prints the correct externally-accessible URL instead of `localhost`.

## Install

```
go install github.com/derrick-wippler-anchor/peep/cmd/peep@latest
```

## Usage

```
peep [directory] [-p port]
```

`directory` defaults to `.` and `port` defaults to `8080`.
