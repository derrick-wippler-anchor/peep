package server

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

type fileState struct {
	mtime time.Time
	size  int64
}

func poll(ctx context.Context, root string, tick <-chan time.Time, b *broker) {
	prev := make(map[string]fileState)

	for {
		select {
		case <-ctx.Done():
			return

		case <-tick:
			curr := make(map[string]fileState)

			// Walk the root directory collecting current file states.
			_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					// Skip paths that produce errors; continue walking.
					return nil
				}
				rel, relErr := filepath.Rel(root, path)
				if relErr != nil {
					return nil
				}
				curr[rel] = fileState{mtime: info.ModTime(), size: info.Size()}
				return nil
			})

			// Detect additions and changes.
			changed := false
			for path, cs := range curr {
				ps, exists := prev[path]
				if !exists || cs.mtime != ps.mtime || cs.size != ps.size {
					changed = true
					break
				}
			}

			// Detect removals.
			if !changed {
				for path := range prev {
					if _, exists := curr[path]; !exists {
						changed = true
						break
					}
				}
			}

			if changed {
				// Non-blocking send to avoid blocking if broadcast channel is full.
				select {
				case b.broadcast <- struct{}{}:
				default:
				}
			}

			prev = curr
		}
	}
}
