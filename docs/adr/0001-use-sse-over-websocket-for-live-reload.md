# 1. Use SSE over WebSocket for live-reload

Date: 2026-05-18

## Status

Accepted

## Context

The `host` tool injects a script into served pages that connects to the server and triggers a browser reload when a file changes. This requires a persistent server→browser channel.

Two options exist:

- [Server-Sent Events (SSE)](https://html.spec.whatwg.org/multipage/server-sent-events.html) — a unidirectional HTTP-based push protocol with built-in browser reconnection via `EventSource`
- [WebSocket](https://websockets.spec.whatwg.org/) — a bidirectional protocol requiring an HTTP upgrade handshake and custom framing

Live-reload requires only one-way communication: the server signals the browser to reload. There is no message the browser needs to send back to the server to support this feature.

## Decision

We will use SSE for the live-reload channel. The server exposes a plain HTTP endpoint; the injected script connects via `EventSource` and calls `location.reload()` on each received event.

## Consequences

- Server implementation is minimal: write `data: reload\n\n` to a kept-alive HTTP response
- `EventSource` reconnects automatically if the server restarts — no client-side retry logic needed
- Bidirectional communication is not possible over this channel; any future feature requiring browser→server messaging would require adding WebSocket alongside SSE
