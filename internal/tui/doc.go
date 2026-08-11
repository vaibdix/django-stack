// Package tui is the Bubble Tea front end: model, update loop, and rendering.
//
// It is the only package in the repository that imports Bubble Tea, huh or
// lipgloss. Everything below it — config, scaffold and the scaffold subpackages
// — is terminal-free and testable on its own. `go list -deps` is the mechanical
// check for that, and CI runs it.
//
// # Single writer
//
// Every field of Model is written only by Update, which Bubble Tea runs on one
// goroutine. The project generator runs on its own goroutine but never touches
// the model: it reports through a Reporter that only writes to a channel, and
// Update drains that channel as ordinary messages. There are no mutexes here,
// and adding a field cannot introduce a race, because no other goroutine has a
// reference to write through.
//
// # No I/O at construction
//
// NewModel is pure. Interpreter discovery and the PyPI release lookup are
// tea.Cmds issued by Init, so the splash screen renders immediately and the
// model can be built in a test with no Python installed and no network.
package tui
