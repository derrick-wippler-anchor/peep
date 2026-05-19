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
}
