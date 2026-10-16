package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"go/format"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/pypi"
)

const fetchTimeout = 30 * time.Second

func main() {
	out := flag.String("o", "internal/tui/django_fallback.go", "file to write")
	flag.Parse()

	if err := run(*out); err != nil {
		fmt.Fprintln(os.Stderr, "djangoreleases:", err)
		os.Exit(1)
	}
}

func run(out string) error {
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	releases, err := pypi.Fetch(ctx, &http.Client{Timeout: fetchTimeout}, "django")
	if err != nil {
		return err
	}

	newest := map[string]string{}
	for _, release := range releases {
		if !isFinal(release.Version) {
			continue
		}
		series := config.DjangoSeries(release.Version)
		if current, ok := newest[series]; !ok || newer(release.Version, current) {
			newest[series] = release.Version
		}
	}

	var versions []string
	for _, rule := range config.SupportedDjangoSeries(time.Now()) {
		if version, ok := newest[rule.Series]; ok {
			versions = append(versions, version)
		}
	}
	if len(versions) == 0 {
		return fmt.Errorf("PyPI listed no release for any supported Django series")
	}
	sort.Slice(versions, func(i, j int) bool { return newer(versions[i], versions[j]) })

	var src bytes.Buffer
	src.WriteString("package tui\n\nvar fallbackDjangoReleases = []string{\n")
	for _, version := range versions {
		fmt.Fprintf(&src, "\t%q,\n", version)
	}
	src.WriteString("}\n")

	formatted, err := format.Source(src.Bytes())
	if err != nil {
		return err
	}
	return os.WriteFile(out, formatted, 0o644)
}

func isFinal(version string) bool {
	for _, part := range strings.Split(version, ".") {
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}
	return true
}

func newer(left, right string) bool {
	l, r := strings.Split(left, "."), strings.Split(right, ".")
	for i := 0; i < len(l) || i < len(r); i++ {
		lv, rv := 0, 0
		if i < len(l) {
			lv, _ = strconv.Atoi(l[i])
		}
		if i < len(r) {
			rv, _ = strconv.Atoi(r[i])
		}
		if lv != rv {
			return lv > rv
		}
	}
	return false
}
