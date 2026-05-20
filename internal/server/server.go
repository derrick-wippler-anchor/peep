package server

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

//go:embed templates/*
var templateFS embed.FS

// Config holds parsed server configuration.
type Config struct {
	Dir    string
	Port   int
	Stdout io.Writer
	Stderr io.Writer
}

// Run parses flags from args, reads WEB_HOST from env, prints the startup URL
// to stdout, and blocks until the server exits. On startup failure it writes a
// message to stderr and returns a non-nil error.
func Run(args []string, env []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var port int
	fs.IntVar(&port, "port", 8080, "port to listen on")
	fs.IntVar(&port, "p", 8080, "port to listen on (shorthand)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	// Resolve directory from positional argument; default to ".".
	dir := "."
	if fs.NArg() > 0 {
		dir = fs.Arg(0)
	}

	// Validate the directory exists.
	if _, err := os.Stat(dir); err != nil {
		_, _ = fmt.Fprintf(stderr, "directory %s does not exist\n", dir)
		return err
	}

	// Bind the listener before starting goroutines so port-conflict errors are
	// synchronous and returned to the caller.
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		if strings.Contains(err.Error(), "address already in use") {
			_, _ = fmt.Fprintf(stderr, "port %d is already in use\n", port)
		}
		return err
	}

	// Read WEB_HOST from the injected env slice.
	var webHost string
	for _, e := range env {
		if strings.HasPrefix(e, "WEB_HOST=") {
			webHost = e[len("WEB_HOST="):]
			break
		}
	}

	// Wire up broker and poller.
	ctx, cancel := context.WithCancel(context.Background())
	_ = cancel // goroutine is acceptable; test binary exits after tests complete

	b := newBroker()
	go b.run(ctx)
	go poll(ctx, dir, time.NewTicker(500*time.Millisecond).C, b)

	// Print startup URL.
	if webHost != "" {
		_, _ = fmt.Fprintf(stdout, "https://%d-%s/\n", port, webHost)
	} else {
		_, _ = fmt.Fprintf(stdout, "http://localhost:%d\n", port)
	}

	return http.Serve(listener, newHandler(dir, b))
}
