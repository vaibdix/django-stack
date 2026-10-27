package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const gigabyte = 1 << 30

func runSweep(ctx context.Context, args []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("sweep", flag.ContinueOnError)
	dir := flags.String("run", "", "run folder (default ~/django-stack-runs/<label>-<sha>); an existing one is resumed")
	label := flags.String("label", "sweep", "run folder label when select writes a new one")
	batch := flags.Int("batch", 20, "projects generated, checked and cleaned before the next batch starts")
	workers := flags.Int("workers", 4, "projects generated and checked at once within a batch")
	budget := flags.Int("max-gb", 20, "stop when the run folder grows past this many GiB after a batch is cleaned")
	uiBaseline := flags.String("ui-baseline", "", "run folder whose screenshots are compared with (default: record this run's own)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *batch < 1 {
		return errors.New("-batch must be at least 1")
	}

	if *dir == "" {
		d, err := defaultRunDir(ctx, *label)
		if err != nil {
			return err
		}
		*dir = d
	}
	if _, err := os.Stat(filepath.Join(*dir, matrixFile)); errors.Is(err, fs.ErrNotExist) {
		if err := runSelect(ctx, []string{"-run", *dir, "-label", *label}, stdout); err != nil {
			return err
		}
	}
	r, err := openRun(*dir)
	if err != nil {
		return err
	}
	if _, err := os.Stat(r.binary()); errors.Is(err, fs.ErrNotExist) {
		if err := runBuild(ctx, []string{"-run", r.dir}, stdout); err != nil {
			return err
		}
	}

	projects := r.matrix.Projects
	for start := 0; start < len(projects); start += *batch {
		if err := ctx.Err(); err != nil {
			return err
		}
		names := make([]string, 0, *batch)
		for _, p := range projects[start:min(start+*batch, len(projects))] {
			names = append(names, p.Name)
		}
		only := strings.Join(names, ",")
		_, _ = fmt.Fprintf(stdout, "batch %d of %d: %s\n", start / *batch + 1, (len(projects)+*batch-1) / *batch, only)
		workers := strconv.Itoa(*workers)
		checkArgs := []string{"-run", r.dir, "-only", only, "-workers", workers, "-resume"}
		if *uiBaseline != "" {
			checkArgs = append(checkArgs, "-ui-baseline", *uiBaseline)
		}
		for _, step := range []func() error{
			func() error {
				return runGenerate(ctx, []string{"-run", r.dir, "-only", only, "-workers", workers, "-resume"}, stdout)
			},
			func() error { return runCheck(ctx, checkArgs, stdout) },
			func() error { return runClean(ctx, []string{"-run", r.dir, "-only", only}, stdout) },
		} {
			if err := step(); err != nil {
				return err
			}
		}
		size, err := folderSize(r.dir)
		if err != nil {
			return err
		}
		if size > int64(*budget)*gigabyte {
			return fmt.Errorf("%s holds %.1f GiB after cleaning, over the %d GiB budget; free space or raise -max-gb, then run again to resume",
				r.dir, float64(size)/gigabyte, *budget)
		}
	}
	return runReport(ctx, []string{"-run", r.dir}, stdout)
}

func folderSize(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	return total, err
}
