package dbdocker

import (
	"context"
	"flag"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/vaibdix/django-cli/internal/config"
	"github.com/vaibdix/django-cli/internal/scaffold/command"
)

type recorder struct{ steps []string }

func (r *recorder) Step(m string) { r.steps = append(r.steps, m) }

func TestDatabaseName(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"myproject":      "myproject",
		"my-shop":        "my_shop",
		"My-Shop":        "my_shop",
		"/tmp/my-shop":   "my_shop",
		"nested/My-Proj": "my_proj",
		"already_snake":  "already_snake",
	}

	for in, want := range tests {
		if got := DatabaseName(in); got != want {
			t.Errorf("DatabaseName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPasswordLengthAndAlphabet(t *testing.T) {
	t.Parallel()

	for _, length := range []int{1, 8, 16, 31, 32} {
		got, err := Password(length)
		if err != nil {
			t.Fatalf("Password(%d): %v", length, err)
		}
		if len(got) != length {
			t.Errorf("Password(%d) returned %d characters (%q)", length, len(got), got)
		}
		if strings.Trim(got, "0123456789abcdef") != "" {
			t.Errorf("Password(%d) = %q, want hex characters only", length, got)
		}
	}
}

func TestPasswordsAreNotRepeated(t *testing.T) {
	t.Parallel()

	seen := make(map[string]bool)
	for i := range 50 {
		p, err := Password(16)
		if err != nil {
			t.Fatalf("Password: %v", err)
		}
		if seen[p] {
			t.Fatalf("Password returned a duplicate value %q on iteration %d", p, i)
		}
		seen[p] = true
	}
}

func TestAvailablePortPrefersTheRequestedPort(t *testing.T) {
	t.Parallel()

	free, err := AvailablePort(0)
	if err != nil {
		t.Fatalf("AvailablePort(0): %v", err)
	}
	n, err := strconv.Atoi(free)
	if err != nil {
		t.Fatalf("AvailablePort returned %q, not a number", free)
	}

	got, err := AvailablePort(n)
	if err != nil {
		t.Fatalf("AvailablePort(%d): %v", n, err)
	}
	if got != free {
		t.Errorf("AvailablePort(%d) = %q, want the preferred port %q", n, got, free)
	}
}

func TestSetupPostgresRunsTheExpectedDockerCommands(t *testing.T) {
	defer SkipReadyDelayForTest()()

	run := command.NewFake()
	rep := &recorder{}

	creds, err := SetupPostgres(context.Background(), run, rep, "my-shop")
	if err != nil {
		t.Fatalf("SetupPostgres: %v", err)
	}

	if !run.Ran("docker --version") {
		t.Error("did not check that docker is installed")
	}
	if !run.Ran("docker info") {
		t.Error("did not check that the docker daemon is running")
	}
	if !run.Ran("docker pull postgres:18") {
		t.Error("did not pull the postgres image")
	}

	var runLine string
	for _, line := range run.CommandLines() {
		if strings.HasPrefix(line, "docker run ") {
			runLine = line
		}
	}
	if runLine == "" {
		t.Fatalf("no `docker run` was issued; commands were %q", run.CommandLines())
	}
	for _, want := range []string{
		"--name django-stack-my-shop-" + creds.Port,
		"POSTGRES_PASSWORD=" + creds.Password,
		"POSTGRES_DB=my_shop",
		"127.0.0.1:" + creds.Port + ":5432",
		"-d postgres:18",
	} {
		if !strings.Contains(runLine, want) {
			t.Errorf("docker run is missing %q\ngot: %s", want, runLine)
		}
	}

	if creds.DatabaseName != "my_shop" {
		t.Errorf("DatabaseName = %q, want %q", creds.DatabaseName, "my_shop")
	}
	if !strings.HasPrefix(creds.ConnectionURL, "postgresql://postgres:") {
		t.Errorf("ConnectionURL = %q, want a postgresql:// URL", creds.ConnectionURL)
	}
	if creds.RootPassword != "" {
		t.Error("postgres does not use a root password but one was generated")
	}
}

func TestSetupMySQLGeneratesARootPassword(t *testing.T) {
	defer SkipReadyDelayForTest()()

	run := command.NewFake()

	creds, err := SetupMySQL(context.Background(), run, &recorder{}, "my-shop")
	if err != nil {
		t.Fatalf("SetupMySQL: %v", err)
	}

	if creds.RootPassword == "" {
		t.Error("MySQL requires MYSQL_ROOT_PASSWORD but none was generated")
	}
	if creds.RootPassword == creds.Password {
		t.Error("the root password and the user password are identical")
	}
	if !run.Ran("docker pull mysql:8.4") {
		t.Error("did not pull the mysql image")
	}
}

func TestSetupFailsWhenDockerIsUnavailable(t *testing.T) {
	t.Parallel()

	run := command.NewFake()
	run.FailWith("docker --version", "command not found")

	_, err := SetupPostgres(context.Background(), run, &recorder{}, "my-shop")
	if err == nil {
		t.Fatal("SetupPostgres succeeded with no docker installed")
	}
	if !strings.Contains(err.Error(), "not installed") {
		t.Errorf("error = %q, want it to mention docker not being installed", err)
	}
	if run.Ran("docker pull postgres:18") {
		t.Error("pulled an image despite docker being unavailable")
	}
}

func TestSetupContinuesWhenTheImagePullFails(t *testing.T) {
	defer SkipReadyDelayForTest()()

	run := command.NewFake()
	run.FailWith("docker pull postgres:18", "no network")
	rep := &recorder{}

	if _, err := SetupPostgres(context.Background(), run, rep, "my-shop"); err != nil {
		t.Fatalf("SetupPostgres should tolerate a failed pull, got: %v", err)
	}

	var warned bool
	for _, s := range rep.steps {
		if strings.Contains(s, "Warning") {
			warned = true
		}
	}
	if !warned {
		t.Errorf("no warning was reported for the failed pull; steps were %q", rep.steps)
	}
}

func TestStopReportsFailures(t *testing.T) {
	t.Parallel()

	run := command.NewFake()
	run.Responses["docker ps -q --filter name=django-stack-my-shop-*"] = []byte("abc123\n")
	run.FailWith("docker stop abc123", "permission denied")

	err := Stop(context.Background(), run, "my-shop")
	if err == nil {
		t.Fatal("Stop returned nil when `docker stop` failed")
	}
	if !strings.Contains(err.Error(), "failed to stop") {
		t.Errorf("error = %q, want it to describe the stop failure", err)
	}
}

func TestStopIsANoOpWhenNothingIsRunning(t *testing.T) {
	t.Parallel()

	run := command.NewFake()

	if err := Stop(context.Background(), run, "my-shop"); err != nil {
		t.Fatalf("Stop with no containers: %v", err)
	}
	if run.Ran("docker stop ") {
		t.Error("issued a stop with no containers to stop")
	}
}

func TestWriteComposeFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		db       config.Database
		contains []string
	}{
		{
			db: config.PostgresDocker,
			contains: []string{
				"image: postgres:18", "POSTGRES_DB: ${DB_NAME:?set DB_NAME in .env}",
				`"127.0.0.1:${DB_PORT}:5432"`, "pg_isready", "postgres_data:",
			},
		},
		{
			db: config.MySQLDocker,
			contains: []string{
				"image: mysql:8.4", "MYSQL_ROOT_PASSWORD: ${DB_ROOT_PASSWORD:?set DB_ROOT_PASSWORD in .env}",
				`"127.0.0.1:${DB_PORT}:3306"`, "mysqladmin", "mysql_data:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.db), func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			if err := WriteComposeFile(dir, tt.db); err != nil {
				t.Fatalf("WriteComposeFile: %v", err)
			}

			raw, err := os.ReadFile(filepath.Join(dir, "docker-compose.yml"))
			if err != nil {
				t.Fatalf("reading compose file: %v", err)
			}
			got := string(raw)
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("compose file is missing %q\n---\n%s", want, got)
				}
			}
		})
	}
}

func TestWriteComposeFileRejectsNonDockerDatabases(t *testing.T) {
	t.Parallel()

	for _, db := range []config.Database{config.SQLite, config.PostgresString} {
		if err := WriteComposeFile(t.TempDir(), db); err == nil {
			t.Errorf("WriteComposeFile(%q) returned nil; that database runs no container", db)
		}
	}
}

var updateGolden = flag.Bool("update", false, "rewrite golden files instead of comparing against them")

func TestComposeFileGoldens(t *testing.T) {
	t.Parallel()

	cases := map[config.Database]string{
		config.PostgresDocker: "docker-compose-postgres.yml",
		config.MySQLDocker:    "docker-compose-mysql.yml",
	}

	for db, name := range cases {
		t.Run(string(db), func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			if err := WriteComposeFile(dir, db); err != nil {
				t.Fatalf("WriteComposeFile: %v", err)
			}
			got, err := os.ReadFile(filepath.Join(dir, "docker-compose.yml"))
			if err != nil {
				t.Fatalf("reading generated file: %v", err)
			}

			path := filepath.Join("testdata", "golden", name)

			if *updateGolden {
				if err := os.WriteFile(path, got, 0o644); err != nil {
					t.Fatalf("writing golden %s: %v", name, err)
				}
				return
			}

			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading golden %s: %v\n\nRun: make goldens", name, err)
			}

			if string(got) != string(want) {
				t.Errorf("compose file for %s changed.\n--- want ---\n%s\n--- got ---\n%s", db, want, got)
			}
		})
	}
}

func TestAvailablePortProbesEveryInterfaceNotJustLoopback(t *testing.T) {
	var probed []string

	original := listenTCP
	listenTCP = func(network, address string) (net.Listener, error) {
		probed = append(probed, address)
		return original(network, address)
	}
	t.Cleanup(func() { listenTCP = original })

	if _, err := AvailablePort(45999); err != nil {
		t.Fatalf("AvailablePort: %v", err)
	}

	if len(probed) == 0 {
		t.Fatal("AvailablePort probed nothing")
	}
	for _, address := range probed {
		if strings.HasPrefix(address, "127.0.0.1") {
			t.Errorf("probed %q; docker publishes on every interface, so a container holding "+
				"0.0.0.0:PORT leaves loopback bindable and the port looks free when it is not", address)
		}
	}
}

func TestAFailedContainerStartIsCleanedUpSoARetryIsNotBlocked(t *testing.T) {
	defer SkipReadyDelayForTest()()

	failing := command.NewFake()
	failing.FailWithPrefix("docker run ", "Bind for 0.0.0.0:5432 failed: port is already allocated")

	if _, err := SetupPostgres(context.Background(), failing, &recorder{}, "my-shop"); err == nil {
		t.Fatal("a failed container start must surface as an error")
	}

	var removed bool
	for _, line := range failing.CommandLines() {
		if strings.HasPrefix(line, "docker rm -f ") {
			removed = true
		}
	}
	if !removed {
		t.Errorf("the container left behind by the failed start was not removed, so retrying "+
			"fails with a name clash; commands were %q", failing.CommandLines())
	}
}

func TestARunningContainerForTheSameProjectIsRefusedWithInstructions(t *testing.T) {
	defer SkipReadyDelayForTest()()

	run := command.NewFake()
	name := "django-stack-my-shop-" + mustPort(t, 5432)
	run.Responses["docker inspect -f {{.State.Running}} "+name] = []byte("true\n")

	_, err := SetupPostgres(context.Background(), run, &recorder{}, "my-shop")
	if err == nil {
		t.Fatal("reusing a running container silently would hand back the wrong password")
	}
	if !strings.Contains(err.Error(), "docker rm -f "+name) {
		t.Errorf("error = %q, want it to name the exact cleanup command", err)
	}
	for _, line := range run.CommandLines() {
		if strings.HasPrefix(line, "docker run ") {
			t.Error("a second container was started while one was already running")
		}
	}
}

func mustPort(t *testing.T, preferred int) string {
	t.Helper()

	port, err := AvailablePort(preferred)
	if err != nil {
		t.Fatalf("AvailablePort: %v", err)
	}
	return port
}

func TestComposeKeepsTheDatabaseOffTheNetworkAndOutOfTheAppSecrets(t *testing.T) {
	t.Parallel()

	for _, db := range []config.Database{config.PostgresDocker, config.MySQLDocker} {
		t.Run(string(db), func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			if err := WriteComposeFile(dir, db); err != nil {
				t.Fatalf("WriteComposeFile: %v", err)
			}
			raw, err := os.ReadFile(filepath.Join(dir, "docker-compose.yml"))
			if err != nil {
				t.Fatalf("reading compose file: %v", err)
			}
			got := string(raw)

			if strings.Contains(got, "env_file:") {
				t.Errorf("the whole application secret set is handed to the database container:\n%s", got)
			}
			if !strings.Contains(got, `"127.0.0.1:${DB_PORT}:`) {
				t.Errorf("the database port is published on every interface:\n%s", got)
			}
			if !strings.Contains(got, ":?set DB_NAME in .env}") {
				t.Errorf("a missing DB_NAME would silently start a differently-named database:\n%s", got)
			}
		})
	}
}

func TestTheComposeFilePassesNoServerOptionThePinnedImageRejects(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := WriteComposeFile(dir, config.MySQLDocker); err != nil {
		t.Fatalf("WriteComposeFile: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "docker-compose.yml"))
	if err != nil {
		t.Fatalf("reading compose file: %v", err)
	}

	if strings.Contains(string(raw), "default-authentication-plugin") {
		t.Errorf("mysql %s removed default-authentication-plugin and aborts on it, "+
			"so `docker compose up` would never start the database:\n%s", mysql.Image, raw)
	}
}

func TestTheStartedContainerIsPublishedOnLoopbackJustLikeTheComposeFile(t *testing.T) {
	defer SkipReadyDelayForTest()()

	for _, tc := range []struct {
		db           config.Database
		setup        func(context.Context, command.Runner, Reporter, string) (Credentials, error)
		internalPort string
	}{
		{config.PostgresDocker, SetupPostgres, "5432"},
		{config.MySQLDocker, SetupMySQL, "3306"},
	} {
		t.Run(string(tc.db), func(t *testing.T) {
			run := command.NewFake()

			creds, err := tc.setup(context.Background(), run, &recorder{}, "my-shop")
			if err != nil {
				t.Fatalf("setup: %v", err)
			}

			var runLine string
			for _, line := range run.CommandLines() {
				if strings.HasPrefix(line, "docker run ") {
					runLine = line
				}
			}
			if runLine == "" {
				t.Fatalf("no `docker run` was issued; commands were %q", run.CommandLines())
			}

			want := "-p 127.0.0.1:" + creds.Port + ":" + tc.internalPort
			if !strings.Contains(runLine, want) {
				t.Errorf("the database is published on every interface, while docker-compose.yml "+
					"promises loopback only; want %q\ngot: %s", want, runLine)
			}
		})
	}
}
