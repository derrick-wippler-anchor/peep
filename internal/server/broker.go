package server

import "context"

type broker struct {
	subscribe   chan chan struct{}
	unsubscribe chan chan struct{}
	broadcast   chan struct{}
}

func newBroker() *broker {
	return &broker{
		subscribe:   make(chan chan struct{}, 1),
		unsubscribe: make(chan chan struct{}, 1),
		broadcast:   make(chan struct{}, 1),
	}
}

func (b *broker) run(ctx context.Context) {
	clients := make(map[chan struct{}]struct{})

	for {
		select {
		case ch := <-b.subscribe:
			clients[ch] = struct{}{}

		case ch := <-b.unsubscribe:
			delete(clients, ch)

		case <-b.broadcast:
			for ch := range clients {
				// Non-blocking send: drop the signal if the client channel is full.
				select {
				case ch <- struct{}{}:
				default:
				}
			}

		case <-ctx.Done():
			for ch := range clients {
				close(ch)
			}
			return
		}
	}
}
