# 2. Use polling over fsnotify for filesystem watching

Date: 2026-05-19

## Status

Accepted

## Context

The server must detect file changes under the served directory and signal connected SSE clients to reload.

The dominant Go filesystem-watching library, [fsnotify](https://github.com/fsnotify/fsnotify), uses OS-level inotify (Linux) and FSEvents (macOS) to deliver change events. It has several reliability issues for directory-tree watching:

- Editors that use atomic saves (write to a temp file, then rename) remove the original inode; the watcher silently stops receiving events for that path unless re-registered on every `Rename` event.
- fsnotify v1 does not support recursive directory watching; every subdirectory must be added individually on startup, and newly created subdirectories must be dynamically added as they appear.
- Deep or large directory trees can exhaust OS inotify descriptor limits.

For a local development tool, a 500ms change-detection latency is acceptable.

## Decision

We will implement filesystem watching as a pure-Go poller. Every 500ms, the poller walks the served directory tree using `filepath.Walk`, records `(path → mtime, size)` for each file, and broadcasts a reload signal to the SSE broker when any entry differs from the previous snapshot.

## Consequences

- No third-party watcher dependency; no OS-level inotify limits or descriptor management.
- Atomic saves, renames, and new subdirectories are handled automatically — any change visible to `os.Stat` is detected.
- CPU overhead scales with directory size; negligible for typical local project trees.
- Change detection latency is fixed at ~500ms rather than near-instant.
- The tick channel is injected, making the polling loop fully testable without real timers.
