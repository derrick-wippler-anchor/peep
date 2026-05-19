package server

import (
	"context"
	"time"
)

type fileState struct {
	mtime time.Time
	size  int64
}

func poll(ctx context.Context, root string, tick <-chan time.Time, b *broker) {
}
