// Package scaffold executes the plan for a Spec through the ports: it writes each
// effect's file groups, runs the tools, stages and publishes the project, undoes
// what it created when a run fails, and sends events.
package scaffold
