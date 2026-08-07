// Package pyenv finds Python interpreters and manages a project's virtualenv.
//
// It covers three jobs that were previously spread across the TUI package:
// discovering which interpreters are installed and usable, creating the
// project's .venv, and installing packages into it with uv when available and
// pip otherwise.
//
// The package is a leaf: it imports command and the standard library, and
// declares its own narrow Reporter interface rather than importing one from a
// parent. That is what keeps the dependency graph one-way and lets the whole
// package be exercised without a terminal.
//
// Discovery is deliberately slow — it execs every candidate interpreter to read
// its version — so callers should treat Discover as I/O and keep it off any
// goroutine that must stay responsive.
package pyenv
