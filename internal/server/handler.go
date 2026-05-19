package server

import "net/http"

type handler struct {
	root   string
	broker *broker
}

func newHandler(root string, b *broker) *handler {
	return &handler{root: root, broker: b}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}
