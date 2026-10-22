package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed probe.py
var probeScript []byte

const (
	djlintSpec    = "djlint==1.46.6"
	checkTimeout  = 15 * time.Minute
	serverTimeout = 60 * time.Second
	superuser     = "matrix"
	superPassword = "matrix-sweep-password-1"
)

const superuserScript = `from django.contrib.auth import get_user_model
User = get_user_model()
user, _ = User._default_manager.get_or_create(**{User.USERNAME_FIELD: "` + superuser + `"})
user.is_staff = user.is_superuser = True
user.set_password("` + superPassword + `")
user.save()`

var settingsModulePattern = regexp.MustCompile(`"DJANGO_SETTINGS_MODULE",\s*"([^"]+)"`)

type checkResult struct {
	Name    string  `json:"name"`
	OK      bool    `json:"ok"`
	Skipped bool    `json:"skipped,omitempty"`
	Info    bool    `json:"info,omitempty"`
	Seconds float64 `json:"seconds"`
	Note    string  `json:"note,omitempty"`
}

type checker struct {
	r          run
	p          project
	dir        string
	logDir     string
	excludeEnv []string
	python     string
	settings   string
	production string
	secret     string
	uiDir      string
	uiBaseline string
	results    []checkResult
}

func runCheck(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	dir := fs.String("run", "", "run folder written by select")
	workers := fs.Int("workers", 4, "projects checked at once")
	only := fs.String("only", "", "comma-separated project names (default: all)")
	resume := fs.Bool("resume", false, "skip projects an earlier run of this command already checked")
	uiBaseline := fs.String("ui-baseline", "", "run folder whose screenshots this run is compared with (default: record new ones)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	r, err := openRun(*dir)
	if err != nil {
		return err
	}
	projects, err := r.only(*only)
	if err != nil {
		return err
	}
	uiDir, err := filepath.Abs(filepath.Join("tools", "matrix", "ui"))
	if err != nil {
		return err
	}

	var mu sync.Mutex
	portLane := func(p project) string { return databasePort(r.projectDir(p.Name)) }
	inLanes(projects, *workers, portLane, func(p project) {
		var previous []checkResult
		if *resume && readJSON(filepath.Join(r.logDir(p.Name), "checks.json"), &previous) == nil {
			return
		}
		results := checkOne(ctx, r, p, uiDir, *uiBaseline)
		failed := []string{}
		for _, c := range results {
			if !c.OK && !c.Skipped && !c.Info {
				failed = append(failed, c.Name)
			}
		}
		mu.Lock()
		defer mu.Unlock()
		status := "ok"
		if len(failed) > 0 {
			status = "FAIL"
		}
		_, _ = fmt.Fprintf(stdout, "%-4s %-28s %s\n", status, p.Name, strings.Join(failed, " "))
	})
	return nil
}

func checkOne(ctx context.Context, r run, p project, uiDir, uiBaseline string) []checkResult {
	c := &checker{r: r, p: p, dir: r.projectDir(p.Name), logDir: filepath.Join(r.logDir(p.Name), "checks"), uiDir: uiDir, uiBaseline: uiBaseline}
	c.excludeEnv = []string{"UV_EXCLUDE_NEWER=" + r.matrix.ExcludeNewer}
	_ = os.RemoveAll(c.logDir)
	_ = os.MkdirAll(c.logDir, 0o755)

	var gen generateResult
	if err := readJSON(filepath.Join(r.logDir(p.Name), "generate.json"), &gen); err != nil || p.Expect != expectMade || !gen.DirExists {
		c.results = append(c.results, checkResult{Name: "generated", Skipped: true, Note: "no project to check"})
		c.save()
		return c.results
	}
	if err := c.inspect(); err != nil {
		c.results = append(c.results, checkResult{Name: "inspect", Note: err.Error()})
		c.save()
		return c.results
	}

	compose := c.exists("docker-compose.yml")
	if compose {
		c.cmd(ctx, "docker-up", nil, "docker", "compose", "up", "-d", "--wait")
	}
	c.static(ctx)
	c.runtime(ctx, compose)
	c.ui(ctx)
	if compose {
		c.cmd(ctx, "docker-down", nil, "docker", "compose", "down")
	}
	c.save()
	return c.results
}

func (c *checker) save() {
	_ = writeJSON(filepath.Join(c.logDir, "..", "checks.json"), c.results)
}

func (c *checker) inspect() error {
	c.python = filepath.Join(c.dir, ".venv", "bin", "python")
	if !c.exists(".venv/bin/python") {
		return errors.New("the project has no .venv")
	}
	manage, err := os.ReadFile(filepath.Join(c.dir, "manage.py"))
	if err != nil {
		return err
	}
	match := settingsModulePattern.FindSubmatch(manage)
	if match == nil {
		return errors.New("manage.py names no DJANGO_SETTINGS_MODULE")
	}
	c.settings = string(match[1])
	module, _, _ := strings.Cut(c.settings, ".")
	c.production = module + ".settings"
	if c.exists(module + "/settings") {
		c.production = module + ".settings.production"
	}
	env, err := os.ReadFile(filepath.Join(c.dir, ".env"))
	if err != nil {
		return err
	}
	for line := range strings.SplitSeq(string(env), "\n") {
		if value, ok := strings.CutPrefix(line, "SECRET_KEY="); ok {
			c.secret = value
		}
	}
	return nil
}

func (c *checker) productionEnv() []string {
	return []string{
		"ENVIRONMENT=production",
		"DJANGO_SETTINGS_MODULE=" + c.production,
		"SECRET_KEY=" + c.secret,
		"ALLOWED_HOSTS=testserver,localhost",
		"DEFAULT_FROM_EMAIL=matrix@example.com",
	}
}

func (c *checker) static(ctx context.Context) {
	py := c.python
	c.cmd(ctx, "check-deploy", c.productionEnv(), py, "manage.py", "check", "--deploy")
	c.cmd(ctx, "makemigrations", nil, py, "manage.py", "makemigrations", "--check", "--dry-run")

	ruff := c.tool("ruff", "ruff")
	ruffCheck := append(slices.Clone(ruff), "check")
	if !c.exists(".venv/bin/ruff") {
		ruffCheck = append(ruffCheck, "--isolated", "--select", "E4,E7,E9,F", "--ignore", "F403,F405")
		if target := c.pythonTarget(); target != "" {
			ruffCheck = append(ruffCheck, "--target-version", target)
		}
	}
	c.cmd(ctx, "ruff-check", nil, append(ruffCheck, ".")...)
	c.cmd(ctx, "ruff-format", nil, append(ruff, "format", "--check", ".")...)
	c.cmd(ctx, "complexity", nil, "uvx", "--quiet", "ruff", "check", "--isolated", "--select", "C901",
		"--config", "lint.mccabe.max-complexity = 10", "--exclude", ".venv,node_modules", ".")
	c.info(ctx, "radon", "uvx", "--quiet", "radon", "cc", "-n", "C", "-s", "-e", ".venv/*,node_modules/*", ".")
	if c.exists(".venv/bin/mypy") {
		c.cmd(ctx, "mypy", nil, filepath.Join(c.dir, ".venv", "bin", "mypy"), ".")
	}
	if templates := c.templates(); len(templates) > 0 {
		djlint := c.tool("djlint", djlintSpec)
		c.cmd(ctx, "djlint", nil, append(append(djlint, "--lint", "--check", "--profile=django"), templates...)...)
	}
	if c.exists("uv.lock") {
		c.cmd(ctx, "deps", nil, "uv", "pip", "check", "--python", py)
		c.cmd(ctx, "lock", nil, "uv", "lock", "--check")
	} else {
		c.cmd(ctx, "deps", nil, py, "-m", "pip", "check")
	}
	c.audit(ctx)
	if c.exists(".pre-commit-config.yaml") {
		precommit := c.tool("pre-commit", "pre-commit")
		c.cmd(ctx, "pre-commit-config", nil, append(precommit, "validate-config", ".pre-commit-config.yaml")...)
	}
	if c.exists(".git") || c.exists(".pre-commit-config.yaml") {
		c.commitInClone(ctx)
	}
	if c.exists("docker-compose.yml") {
		c.cmd(ctx, "compose-config", nil, "docker", "compose", "config", "-q")
	}
	if _, err := exec.LookPath("scc"); err == nil {
		c.info(ctx, "scc", "scc", "--no-cocomo", "--exclude-dir", ".venv,node_modules,staticfiles,.git", "--format", "json", ".")
	}
}

func (c *checker) pythonTarget() string {
	data, err := os.ReadFile(filepath.Join(c.dir, ".python-version"))
	if err != nil {
		return ""
	}
	major, minor, ok := strings.Cut(strings.TrimSpace(string(data)), ".")
	if !ok {
		return ""
	}
	minor, _, _ = strings.Cut(minor, ".")
	return "py" + major + minor
}

func (c *checker) audit(ctx context.Context) {
	audit := []string{"uvx", "--quiet", "pip-audit", "--progress-spinner", "off", "--disable-pip"}
	if c.exists("uv.lock") {
		exported := filepath.Join(c.logDir, "audit-requirements.txt")
		c.cmd(ctx, "pip-audit", nil, "sh", "-c", "uv export --frozen --all-groups --no-hashes --format requirements-txt -o "+
			shellJoin([]string{exported})+" >/dev/null && "+shellJoin(append(audit, "--no-deps", "-r", exported)))
		return
	}
	args := append(slices.Clone(audit), "--require-hashes")
	for _, file := range []string{"requirements.txt", "requirements-dev.txt"} {
		if c.exists(file) {
			args = append(args, "-r", file)
		}
	}
	c.cmd(ctx, "pip-audit", nil, args...)
}

func (c *checker) runtime(ctx context.Context, compose bool) {
	py := c.python
	c.cmd(ctx, "test", nil, py, "manage.py", "test")
	prod := c.productionEnv()
	c.cmd(ctx, "deploy-migrate", prod, py, "manage.py", "migrate", "--noinput")
	c.cmd(ctx, "deploy-cachetable", prod, py, "manage.py", "createcachetable")
	c.cmd(ctx, "deploy-collectstatic", prod, py, "manage.py", "collectstatic", "--noinput")
	c.probe(ctx, prod)
	if compose {
		c.cmd(ctx, "docker-restart", nil, "sh", "-c", "docker compose down && docker compose up -d --wait")
		c.cmd(ctx, "test-after-restart", nil, py, "manage.py", "test")
	}
}

func (c *checker) probe(ctx context.Context, env []string) {
	script := filepath.Join(c.logDir, "probe.py")
	if err := os.WriteFile(script, probeScript, 0o644); err != nil {
		c.results = append(c.results, checkResult{Name: "probe", Note: err.Error()})
		return
	}
	start := time.Now()
	out, err := c.outputEnv(ctx, env, c.python, script)
	res := checkResult{Name: "probe", Seconds: time.Since(start).Seconds(), Note: strings.TrimSpace(out)}
	var probe struct {
		Debug    bool `json:"debug"`
		Healthz  int  `json:"healthz"`
		CSP      bool `json:"csp"`
		NotFound int  `json:"notFound"`
	}
	if err == nil && json.Unmarshal([]byte(lastLine(out)), &probe) == nil {
		res.OK = !probe.Debug && probe.Healthz == 200 && probe.CSP && probe.NotFound == 404
	}
	c.results = append(c.results, res)
}

func (c *checker) commitInClone(ctx context.Context) {
	clone := filepath.Join(c.r.dir, "scratch", c.p.Name)
	_ = os.RemoveAll(clone)
	_ = os.MkdirAll(filepath.Dir(clone), 0o755)
	if !c.cmd(ctx, "clone", nil, "cp", "-c", "-R", c.dir, clone) {
		return
	}
	defer func() { _ = os.RemoveAll(clone) }()
	identity := []string{"GIT_AUTHOR_NAME=matrix", "GIT_AUTHOR_EMAIL=matrix@example.com", "GIT_COMMITTER_NAME=matrix", "GIT_COMMITTER_EMAIL=matrix@example.com"}
	if c.exists(".git") {
		c.cmdIn(ctx, clone, "first-commit", identity, "sh", "-c", "git add -A && git commit -q -m 'Initial commit'")
	}
	if c.exists(".pre-commit-config.yaml") {
		precommit := c.tool("pre-commit", "pre-commit")
		c.cmdIn(ctx, clone, "pre-commit-run", identity, "sh", "-c",
			"{ [ -d .git ] || git init -q; } && git add -A && "+shellJoin(precommit)+" run --all-files")
	}
}

func (c *checker) ui(ctx context.Context) {
	if c.p.Answers.Stack == "api-only" {
		return
	}
	if !c.existsAt(filepath.Join(c.uiDir, "node_modules", "@playwright", "test")) {
		c.results = append(c.results, checkResult{Name: "ui", Note: "Playwright is not installed: npm ci && npx playwright install chromium in tools/matrix/ui"})
		return
	}
	if !c.cmd(ctx, "ui-superuser", nil, c.python, "manage.py", "shell", "-c", superuserScript) {
		return
	}
	port, err := freePort(ctx)
	if err != nil {
		c.results = append(c.results, checkResult{Name: "ui", Note: err.Error()})
		return
	}
	base := "http://127.0.0.1:" + strconv.Itoa(port)
	serverLog, err := os.Create(filepath.Join(c.logDir, "ui-server.log"))
	if err != nil {
		return
	}
	defer func() { _ = serverLog.Close() }()
	server := exec.CommandContext(ctx, c.python, "manage.py", "runserver", "--noreload", "127.0.0.1:"+strconv.Itoa(port))
	server.Dir = c.dir
	server.Env = os.Environ()
	server.Stdout, server.Stderr = serverLog, serverLog
	if err := server.Start(); err != nil {
		c.results = append(c.results, checkResult{Name: "ui", Note: err.Error()})
		return
	}
	defer func() {
		_ = server.Process.Kill()
		_ = server.Wait()
	}()
	if !waitForPort(ctx, port, serverTimeout) {
		c.results = append(c.results, checkResult{Name: "ui", Note: "the development server did not start; see ui-server.log"})
		return
	}

	snapshots, mode := filepath.Join(c.r.dir, "ui-snapshots"), "all"
	if c.uiBaseline != "" {
		snapshots, mode = filepath.Join(c.uiBaseline, "ui-snapshots"), "none"
	}
	env := []string{
		"BASE_URL=" + base,
		"APP=" + c.p.Answers.App,
		"HTMX=" + strconv.FormatBool(strings.Contains(string(c.p.Answers.Stack), "htmx")),
		"ALLAUTH=" + strconv.FormatBool(c.p.hasAddOn("allauth")),
		"ADMIN_USER=" + superuser,
		"ADMIN_PASSWORD=" + superPassword,
		"SNAPSHOT_DIR=" + filepath.Join(snapshots, c.p.Name),
		"RESULTS_DIR=" + filepath.Join(c.logDir, "ui"),
	}
	args := []string{"playwright", "test", "--reporter=line", "--update-snapshots=" + mode}
	c.cmdIn(ctx, c.uiDir, "ui", env, append([]string{"npx"}, args...)...)
}

func (c *checker) tool(binary, spec string) []string {
	if c.exists(".venv/bin/" + binary) {
		return []string{filepath.Join(c.dir, ".venv", "bin", binary)}
	}
	return []string{"uvx", "--quiet", "--from", spec, binary}
}

func (c *checker) templates() []string {
	var dirs []string
	_ = filepath.WalkDir(c.dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && slices.Contains([]string{".venv", "node_modules", ".git", "staticfiles"}, d.Name()) {
			return filepath.SkipDir
		}
		if d.IsDir() && d.Name() == "templates" {
			rel, _ := filepath.Rel(c.dir, path)
			dirs = append(dirs, rel)
			return filepath.SkipDir
		}
		return nil
	})
	return dirs
}

func (c *checker) cmd(ctx context.Context, name string, env []string, args ...string) bool {
	return c.cmdIn(ctx, c.dir, name, env, args...)
}

func (c *checker) info(ctx context.Context, name string, args ...string) {
	c.cmdIn(ctx, c.dir, name, nil, args...)
	c.results[len(c.results)-1].Info = true
}

func (c *checker) cmdIn(ctx context.Context, dir, name string, env []string, args ...string) bool {
	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Env = append(append(os.Environ(), c.excludeEnv...), env...)
	var log bytes.Buffer
	cmd.Stdout, cmd.Stderr = &log, &log
	start := time.Now()
	err := cmd.Run()
	res := checkResult{Name: name, OK: err == nil, Seconds: time.Since(start).Seconds()}
	if err != nil {
		res.Note = lastLine(log.String())
	}
	header := fmt.Sprintf("$ %s\n(in %s)\n\n", shellJoin(args), dir)
	_ = os.WriteFile(filepath.Join(c.logDir, name+".log"), append([]byte(header), log.Bytes()...), 0o644)
	c.results = append(c.results, res)
	return res.OK
}

func (c *checker) outputEnv(ctx context.Context, env []string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = c.dir
	cmd.Env = append(append(os.Environ(), c.excludeEnv...), env...)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func (c *checker) exists(rel string) bool { return c.existsAt(filepath.Join(c.dir, rel)) }

func (c *checker) existsAt(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func databasePort(projectDir string) string {
	if _, err := os.Stat(filepath.Join(projectDir, "docker-compose.yml")); err != nil {
		return ""
	}
	env, err := os.ReadFile(filepath.Join(projectDir, ".env"))
	if err != nil {
		return ""
	}
	for line := range strings.SplitSeq(string(env), "\n") {
		if port, ok := strings.CutPrefix(line, "DB_PORT="); ok {
			return strings.TrimSpace(port)
		}
	}
	return ""
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

func shellJoin(args []string) string {
	quoted := make([]string, len(args))
	for i, a := range args {
		if strings.ContainsAny(a, " '\"$*&;|<>()") {
			a = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
		}
		quoted[i] = a
	}
	return strings.Join(quoted, " ")
}

func freePort(ctx context.Context) (int, error) {
	var lc net.ListenConfig
	l, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer func() { _ = l.Close() }()
	addr, ok := l.Addr().(*net.TCPAddr)
	if !ok {
		return 0, errors.New("listener has no TCP address")
	}
	return addr.Port, nil
}

func waitForPort(ctx context.Context, port int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	dialer := net.Dialer{Timeout: time.Second}
	for time.Now().Before(deadline) && ctx.Err() == nil {
		conn, err := dialer.DialContext(ctx, "tcp", "127.0.0.1:"+strconv.Itoa(port))
		if err == nil {
			_ = conn.Close()
			return true
		}
		time.Sleep(250 * time.Millisecond)
	}
	return false
}
