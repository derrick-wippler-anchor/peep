package server

import (
	"embed"
	"io"
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
	return nil
}
