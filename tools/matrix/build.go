package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

const cliPackage = "./cmd/django-stack"

func runBuild(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	dir := fs.String("run", "", "run folder written by select")
	if err := fs.Parse(args); err != nil {
		return err
	}
	r, err := openRun(*dir)
	if err != nil {
		return err
	}
	head, err := gitOutput(ctx, ".", "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	if head != r.matrix.Commit {
		_, _ = fmt.Fprintf(stdout, "note: HEAD is %s, the selection was made at %s\n", head[:7], r.matrix.Commit[:7])
	}

	ldflags := "-X main.version=sweep-" + head[:7]
	if err := goCommand(ctx, stdout, nil, "build", "-race", "-ldflags", ldflags, "-o", r.binary(), cliPackage); err != nil {
		return fmt.Errorf("race build: %w", err)
	}
	for _, goos := range []string{"linux", "darwin", "windows"} {
		if err := goCommand(ctx, stdout, []string{"GOOS=" + goos}, "build", "-o", os.DevNull, "./..."); err != nil {
			return fmt.Errorf("GOOS=%s build: %w", goos, err)
		}
	}
	_, _ = fmt.Fprintf(stdout, "built %s (-race) and cross-compiled for linux, darwin and windows\n", r.binary())
	return nil
}

func goCommand(ctx context.Context, stdout io.Writer, env []string, args ...string) error {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout, cmd.Stderr = stdout, stdout
	_, _ = fmt.Fprintf(stdout, "go %s\n", strings.Join(args, " "))
	return cmd.Run()
}
