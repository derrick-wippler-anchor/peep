package main

import (
	"os"

	"github.com/derrick-wippler-anchor/host/internal/server"
)

func main() {
	if err := server.Run(os.Args[1:], os.Environ(), os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}
