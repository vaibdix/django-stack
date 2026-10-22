package releases

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/vaibdix/django-stack/internal/command"
	"github.com/vaibdix/django-stack/internal/config"
	"github.com/vaibdix/django-stack/internal/pyenv"
	"github.com/vaibdix/django-stack/internal/pypi"
	"github.com/vaibdix/django-stack/internal/version"
)

const FetchTimeout = 5 * time.Second

type Option struct {
	Version       string
	UploadedAt    time.Time
	Compatibility string
	Recommended   bool
	LTS           bool
	Label         string
}

type UnsupportedPythonError struct {
	Major, Minor int
	Newest       int
}

func (e *UnsupportedPythonError) Error() string {
	return fmt.Sprintf("no supported Django release runs on Python %d.%d yet; the newest it supports is 3.%d",
		e.Major, e.Minor, e.Newest)
}

type FetchError struct {
	Index string
	Err   error
}

func (e *FetchError) Error() string {
	return fmt.Sprintf("the Django releases could not be read from %s: %v", e.Index, e.Err)
}

func (e *FetchError) Unwrap() error { return e.Err }

type Catalog struct {
	client *http.Client
	index  func(context.Context) string

	mu    sync.Mutex
	cache map[string][]pypi.Release
}

func New(client *http.Client, index func(context.Context) string) *Catalog {
	return &Catalog{client: client, index: index, cache: map[string][]pypi.Release{}}
}

func ForMachine(run func() command.Runner, getenv func(string) string) *Catalog {
	return New(&http.Client{Timeout: FetchTimeout}, func(ctx context.Context) string {
		return IndexFor(DetectToolchain(ctx, run()), getenv)
	})
}

func ForToolchain(toolchain pyenv.Toolchain, getenv func(string) string) *Catalog {
	index := IndexFor(toolchain, getenv)
	return New(&http.Client{Timeout: FetchTimeout}, func(context.Context) string { return index })
}

func (c *Catalog) Index(ctx context.Context) string {
	return c.index(ctx)
}

func (c *Catalog) Options(ctx context.Context, python pyenv.Option, today time.Time) ([]Option, error) {
	if err := checkPython(python, today); err != nil {
		return nil, err
	}
	releases, err := c.fetch(ctx)
	if err != nil {
		return nil, err
	}
	options, err := optionsFrom(python, releases, today)
	if err != nil {
		return nil, &FetchError{Index: c.index(ctx), Err: err}
	}
	return options, nil
}

func (c *Catalog) Resolve(ctx context.Context, python pyenv.Option, djangoVersion string, today time.Time) error {
	if err := checkPython(python, today); err != nil {
		return err
	}
	if !config.DjangoRunsOn(djangoVersion, python.Major, python.Minor, today) {
		return fmt.Errorf("no supported Django series covers %s on Python %d.%d", djangoVersion, python.Major, python.Minor)
	}
	releases, err := c.fetch(ctx)
	if err != nil {
		return err
	}
	for _, release := range releases {
		if release.Version != djangoVersion {
			continue
		}
		supported, err := pypi.SupportsPython(release.RequiresPython, python.Major, python.Minor, python.Patch)
		if err != nil {
			return fmt.Errorf("django %s: %w", release.Version, err)
		}
		if !supported {
			return fmt.Errorf("django %s requires Python %s, not %s", release.Version, release.RequiresPython, python.Version)
		}
		return nil
	}
	return fmt.Errorf("django %s is not a release on %s", djangoVersion, c.Index(ctx))
}

func (c *Catalog) fetch(ctx context.Context) ([]pypi.Release, error) {
	index := c.index(ctx)

	c.mu.Lock()
	cached, ok := c.cache[index]
	c.mu.Unlock()
	if ok {
		return cached, nil
	}

	ctx, cancel := context.WithTimeout(ctx, FetchTimeout)
	defer cancel()
	releases, err := pypi.Fetch(ctx, c.client, index, "django")
	if err != nil {
		return nil, &FetchError{Index: index, Err: err}
	}

	c.mu.Lock()
	c.cache[index] = releases
	c.mu.Unlock()
	return releases, nil
}

func checkPython(python pyenv.Option, today time.Time) error {
	newest := config.NewestSupportedPythonMinor(today)
	if python.Major == 3 && python.Minor > newest {
		return &UnsupportedPythonError{Major: python.Major, Minor: python.Minor, Newest: newest}
	}
	return nil
}

func optionsFrom(python pyenv.Option, releases []pypi.Release, today time.Time) ([]Option, error) {
	versions := make([]string, 0, len(releases))
	uploads := make(map[string]time.Time, len(releases))
	for _, release := range releases {
		supported, err := pypi.SupportsPython(release.RequiresPython, python.Major, python.Minor, python.Patch)
		if err != nil {
			return nil, fmt.Errorf("django %s: %w", release.Version, err)
		}
		if !supported {
			continue
		}
		versions = append(versions, release.Version)
		uploads[release.Version] = release.UploadedAt
	}

	options := newestPerSeries(python, versions, today)
	for i := range options {
		options[i].UploadedAt = uploads[options[i].Version]
	}
	return options, nil
}

func newestPerSeries(python pyenv.Option, versions []string, today time.Time) []Option {
	newest := map[string]string{}
	parsed := map[string][]int{}
	for _, release := range versions {
		parts, err := version.Parse(release)
		if err != nil || !config.DjangoRunsOn(release, python.Major, python.Minor, today) {
			continue
		}
		parsed[release] = parts
		series := config.DjangoSeries(release)
		if current, seen := newest[series]; !seen || version.Compare(parts, parsed[current]) > 0 {
			newest[series] = release
		}
	}

	options := make([]Option, 0, len(newest))
	for _, rule := range config.SupportedDjangoSeries(today) {
		release, ok := newest[rule.Series]
		if !ok {
			continue
		}
		options = append(options, Option{
			Version:       release,
			LTS:           rule.LTS,
			Compatibility: fmt.Sprintf("compatible with Python %d.%d", python.Major, python.Minor),
		})
	}

	sort.SliceStable(options, func(i, j int) bool {
		return version.Compare(parsed[options[i].Version], parsed[options[j].Version]) > 0
	})
	for i := range options {
		options[i].Recommended = i == 0
		options[i].Label = label(options[i])
	}
	return options
}

func label(option Option) string {
	text := option.Version
	if option.LTS {
		text += " LTS"
	}
	if option.Recommended {
		text += " [recommended]"
	}
	if option.Compatibility != "" {
		text += " - " + option.Compatibility
	}
	return text
}
