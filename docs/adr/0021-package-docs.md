# 21. Each package has a doc.go, the one comment the no-comments rule allows

Status: proposed, 2026-10-27. This commit can be dropped on its own.

## Context

The repo allows no comments in Go, except machine directives, and `make comments` enforces that with the `nocomments` vet tool. An earlier decision kept `doc.go` out as well. new.md makes one exception: a few lines per package, because `go doc` and a newcomer read the package doc first, and the layout now has 27 packages.

## Decision

- Each package has a `doc.go` that holds only its package comment, in two to four lines: what the package is for and, for `render` and `plan`, that it performs no I/O.
- `nocomments` allows the package doc comment in a file named `doc.go` and nothing else. Any other comment, in `doc.go` or elsewhere, and a package comment in any other file are still reported. Its test data covers all three cases.

## Consequences

- `go doc ./internal/render` and the other packages describe themselves.
- The rule stays mechanical: one file name, one comment group per package.
- Dropping the commit removes the `doc.go` files, this ADR and the `nocomments` exception together.
