package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
)

type command struct {
	name    string
	summary string
	run     func(ctx context.Context, args []string, stdout io.Writer) error
}

func commands() []command {
	return []command{
		{"select", "write the Tier 2 project set (PICT 3-wise plus fixed edge sets) to a run folder", runSelect},
		{"build", "build the -race binary into the run folder and cross-compile for every OS", runBuild},
		{"generate", "generate every selected project with the run's binary", runGenerate},
		{"check", "run the static, runtime and web UI checks on every generated project", runCheck},
		{"diff", "diff every project's tree against the same project in a baseline run", runDiff},
		{"clean", "stop Docker databases and remove .venv and node_modules from every project", runClean},
		{"report", "write results.json and summary.md for the run", runReport},
		{"render-all", "render every configuration from the blueprint, deduplicate files and run the G9 checks", runRenderAll},
	}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := dispatch(ctx, os.Args[1:], os.Stdout, os.Stderr)
	stop()
	if err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(os.Stderr, "matrix:", err)
		}
		os.Exit(1)
	}
}

func dispatch(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		usage(stderr)
		return flag.ErrHelp
	}
	for _, c := range commands() {
		if c.name == args[0] {
			return c.run(ctx, args[1:], stdout)
		}
	}
	usage(stderr)
	return fmt.Errorf("unknown command %q", args[0])
}

func usage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "usage: go run ./tools/matrix <command> [flags]")
	_, _ = fmt.Fprintln(w)
	for _, c := range commands() {
		_, _ = fmt.Fprintf(w, "  %-11s %s\n", c.name, c.summary)
	}
}
