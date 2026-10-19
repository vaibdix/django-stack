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
	"github.com/vaibdix/django-cli/internal/scaffold/command/commandtest"
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
				"- postgres_data:/var/lib/postgresql\n",
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
			if err := WriteComposeFile(dir, "my-shop", tt.db); err != nil {
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

	for _, db := range []config.Database{config.SQLite} {
		if err := WriteComposeFile(t.TempDir(), "my-shop", db); err == nil {
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
			if err := WriteComposeFile(dir, "my-shop", db); err != nil {
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

func TestAvailablePortProbesTheWildcardAndTheLoopbackInterface(t *testing.T) {
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

	for _, host := range []string{"0.0.0.0:", "127.0.0.1:"} {
		found := false
		for _, address := range probed {
			found = found || strings.HasPrefix(address, host)
		}
		if !found {
			t.Errorf("AvailablePort never probed %s*; a container can hold either address "+
				"while the other still binds, probed: %v", host, probed)
		}
	}
}

func TestAvailablePortSkipsAPortHeldOnLoopback(t *testing.T) {
	var lc net.ListenConfig
	held, err := lc.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("holding a loopback port: %v", err)
	}
	t.Cleanup(func() { _ = held.Close() })

	port := held.Addr().(*net.TCPAddr).Port
	got, err := AvailablePort(port)
	if err != nil {
		t.Fatalf("AvailablePort(%d): %v", port, err)
	}
	if got == strconv.Itoa(port) {
		t.Errorf("AvailablePort(%d) returned the port another process holds on 127.0.0.1, "+
			"so docker would fail with \"port is already allocated\"", port)
	}
}

func TestComposeKeepsTheDatabaseOffTheNetworkAndOutOfTheAppSecrets(t *testing.T) {
	t.Parallel()

	for _, db := range []config.Database{config.PostgresDocker, config.MySQLDocker} {
		t.Run(string(db), func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			if err := WriteComposeFile(dir, "my-shop", db); err != nil {
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
	if err := WriteComposeFile(dir, "my-shop", config.MySQLDocker); err != nil {
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

func composeCalls(run *commandtest.Fake) []string {
	var calls []string
	for _, line := range run.CommandLines() {
		if strings.HasPrefix(line, "docker compose ") && line != "docker compose version" {
			calls = append(calls, line)
		}
	}
	return calls
}

// healthyDocker is a fake docker that is installed, running, new enough and
// holds no volume from an earlier project.
func healthyDocker() *commandtest.Fake {
	run := commandtest.NewFake()
	run.Responses[apiVersionCall] = []byte("1.47\n")
	return run
}

const apiVersionCall = "docker version --format {{.Server.APIVersion}}"

func volumeListing(volume string) string {
	return "docker volume ls --filter name=^" + volume + "$ --format {{.Name}}"
}

func TestStartBringsTheComposeServiceUpAndWaitsForItsHealthcheck(t *testing.T) {
	t.Parallel()

	run := healthyDocker()

	if err := Start(context.Background(), run, &recorder{}, "/proj", "my-shop", config.PostgresDocker); err != nil {
		t.Fatalf("Start: %v", err)
	}

	for _, want := range []string{"docker --version", "docker compose version", "docker info", "docker compose pull db"} {
		if !run.Ran(want) {
			t.Errorf("Start did not run %q; commands were %q", want, run.CommandLines())
		}
	}
	var up string
	for _, call := range composeCalls(run) {
		if strings.HasPrefix(call, "docker compose up ") {
			up = call
		}
	}
	if !strings.Contains(up, "-d --wait") || !strings.HasSuffix(up, " db") {
		t.Errorf("the database is not started with `docker compose up -d --wait db`: %q", up)
	}
	for _, call := range run.Calls() {
		if len(call.Args) > 1 && call.Args[0] == "compose" && call.Args[1] != "version" && call.Dir != "/proj" {
			t.Errorf("%s ran in %q, not the project directory whose .env compose reads", call, call.Dir)
		}
		if len(call.Args) > 0 && call.Args[0] == "run" {
			t.Errorf("a container was started outside compose: %s", call)
		}
	}
}

func TestStartRefusesAStaleVolumeFromAnEarlierProject(t *testing.T) {
	t.Parallel()

	run := healthyDocker()
	run.Responses[volumeListing("my_shop_mysql_data")] = []byte("my_shop_mysql_data\n")

	err := Start(context.Background(), run, &recorder{}, "/proj", "my-shop", config.MySQLDocker)
	if err == nil {
		t.Fatal("an existing volume holds another password; reusing it must be refused")
	}
	if !strings.Contains(err.Error(), "docker volume rm my_shop_mysql_data") {
		t.Errorf("error = %q, want the exact cleanup command", err)
	}
	if len(composeCalls(run)) > 0 {
		t.Errorf("compose ran despite the stale volume: %q", composeCalls(run))
	}
}

func TestStartFailsClearlyWithoutCompose(t *testing.T) {
	t.Parallel()

	run := commandtest.NewFake()
	run.FailWith("docker compose version", "unknown command")

	err := Start(context.Background(), run, &recorder{}, "/proj", "my-shop", config.PostgresDocker)
	if err == nil || !strings.Contains(err.Error(), "compose") {
		t.Fatalf("Start without compose = %v, want an error naming docker compose", err)
	}
}

func TestStartFailsWhenDockerIsUnavailable(t *testing.T) {
	t.Parallel()

	run := commandtest.NewFake()
	run.FailWith("docker --version", "command not found")

	err := Start(context.Background(), run, &recorder{}, "/proj", "my-shop", config.PostgresDocker)
	if err == nil || !strings.Contains(err.Error(), "not installed") {
		t.Fatalf("Start without docker = %v, want an error saying docker is not installed", err)
	}
	if len(composeCalls(run)) > 0 {
		t.Errorf("compose ran without docker: %q", composeCalls(run))
	}
}

func TestAFailedPullFailsTheRun(t *testing.T) {
	t.Parallel()

	run := healthyDocker()
	run.FailWith("docker compose pull db", "no network")

	err := Start(context.Background(), run, &recorder{}, "/proj", "my-shop", config.PostgresDocker)
	if err == nil || !strings.Contains(err.Error(), "docker pull postgres:18") {
		t.Fatalf("Start after a failed pull = %v, want an error with the pull command to retry", err)
	}
	for _, call := range composeCalls(run) {
		if strings.HasPrefix(call, "docker compose up") {
			t.Errorf("compose started a database from whatever image was lying around: %s", call)
		}
	}
}

func TestAVolumeCheckThatFailsIsNotTakenAsNoVolume(t *testing.T) {
	t.Parallel()

	run := healthyDocker()
	run.FailWith(volumeListing("my_shop_postgres_data"), "Cannot connect to the Docker daemon")

	err := Start(context.Background(), run, &recorder{}, "/proj", "my-shop", config.PostgresDocker)
	if err == nil || !strings.Contains(err.Error(), "existing my_shop_postgres_data volume") {
		t.Fatalf("Start = %v, want the failed volume check reported", err)
	}
}

func TestAnEngineWhoseAPIVersionCannotBeReadIsRefused(t *testing.T) {
	t.Parallel()

	for name, run := range map[string]*commandtest.Fake{
		"command fails": func() *commandtest.Fake {
			r := commandtest.NewFake()
			r.FailWith(apiVersionCall, "permission denied")
			return r
		}(),
		"prints nothing": commandtest.NewFake(),
	} {
		if err := Start(context.Background(), run, &recorder{}, "/proj", "my-shop", config.PostgresDocker); err == nil {
			t.Errorf("%s: an engine of unknown API version was accepted", name)
		}
		if len(composeCalls(run)) > 0 {
			t.Errorf("%s: compose ran on an unchecked engine: %q", name, composeCalls(run))
		}
	}
}

func TestAnUnhealthyDatabaseIsTornDownSoARetryStartsClean(t *testing.T) {
	t.Parallel()

	run := healthyDocker()
	run.FailWithPrefix("docker compose up ", "container db is unhealthy")
	run.Responses["docker compose logs --tail 30 db"] = []byte("FATAL: something broke")

	err := Start(context.Background(), run, &recorder{}, "/proj", "my-shop", config.PostgresDocker)
	if err == nil {
		t.Fatal("an unhealthy database must surface as an error")
	}
	if !strings.Contains(err.Error(), "FATAL: something broke") {
		t.Errorf("error = %q, want the container logs so the cause is visible", err)
	}
	if !run.Ran("docker compose down -v") {
		t.Errorf("the failed database was left behind; commands were %q", run.CommandLines())
	}
}

func TestNewCredentialsGenerateWhatEachEngineNeeds(t *testing.T) {
	t.Parallel()

	pg, err := NewCredentials(config.PostgresDocker, "my-shop")
	if err != nil {
		t.Fatalf("NewCredentials(postgres): %v", err)
	}
	if pg.Password == "" || pg.Port == "" || pg.DatabaseName != "my_shop" {
		t.Errorf("postgres credentials are incomplete: %+v", pg)
	}
	if pg.RootPassword == "" || pg.RootPassword == pg.Password {
		t.Errorf("postgres needs a superuser password distinct from the app role's: %+v", pg)
	}

	my, err := NewCredentials(config.MySQLDocker, "my-shop")
	if err != nil {
		t.Fatalf("NewCredentials(mysql): %v", err)
	}
	if my.RootPassword == "" || my.RootPassword == my.Password {
		t.Errorf("mysql needs a distinct root password: %+v", my)
	}
}

func TestTheComposeProjectNameIsPinnedSoTheReadmeCommandFindsTheSameContainer(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := WriteComposeFile(dir, "My-Shop", config.PostgresDocker); err != nil {
		t.Fatalf("WriteComposeFile: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, ComposeFile))
	if err != nil {
		t.Fatalf("reading compose file: %v", err)
	}
	if !strings.HasPrefix(string(raw), "name: my_shop\n") {
		t.Errorf("the compose project name is not pinned:\n%s", raw)
	}
}

func TestStartRefusesADockerEngineTooOldForTheStartupHealthcheck(t *testing.T) {
	t.Parallel()

	run := commandtest.NewFake()
	run.Responses["docker version --format {{.Server.APIVersion}}"] = []byte("1.43\n")

	err := Start(context.Background(), run, &recorder{}, "/proj", "my-shop", config.PostgresDocker)
	if err == nil || !strings.Contains(err.Error(), "Docker Engine 25") {
		t.Fatalf("Start on engine API 1.43 = %v, want an error naming the required engine", err)
	}
	if len(composeCalls(run)) > 0 {
		t.Errorf("compose ran on an engine that rejects the compose file: %q", composeCalls(run))
	}
}

func TestTheHealthchecksProbeOverTCPSoTheInitServerDoesNotCount(t *testing.T) {
	t.Parallel()

	for _, db := range []config.Database{config.PostgresDocker, config.MySQLDocker} {
		dir := t.TempDir()
		if err := WriteComposeFile(dir, "my-shop", db); err != nil {
			t.Fatalf("WriteComposeFile: %v", err)
		}
		raw, err := os.ReadFile(filepath.Join(dir, ComposeFile))
		if err != nil {
			t.Fatalf("reading compose file: %v", err)
		}
		if !strings.Contains(string(raw), "-h 127.0.0.1") {
			t.Errorf("%s: the healthcheck uses the socket, which the init-time server answers before the database is ready:\n%s", db, raw)
		}
		if !strings.Contains(string(raw), "start_interval: 1s") {
			t.Errorf("%s: the first health probe waits a full interval:\n%s", db, raw)
		}
	}
}

func TestComposeKeepsTheAppOffTheSuperuserAndPasswordsOffArgv(t *testing.T) {
	t.Parallel()

	pg := composeFile(postgres, "my_shop")
	for _, want := range []string{"POSTGRES_USER: postgres", "${DB_ROOT_PASSWORD:?", `CREATE ROLE "${DB_USER}" LOGIN CREATEDB`, `OWNER TO "${DB_USER}"`} {
		if !strings.Contains(pg, want) {
			t.Errorf("postgres compose is missing %q, so the app connects as the superuser:\n%s", want, pg)
		}
	}

	my := composeFile(mysql, "my_shop")
	if strings.Contains(my, "-p${DB_PASSWORD}") {
		t.Errorf("the MySQL healthcheck passes the password on the command line:\n%s", my)
	}
	if !strings.Contains(my, `MYSQL_PWD=\"$$MYSQL_PASSWORD\"`) {
		t.Errorf("the MySQL healthcheck does not authenticate through MYSQL_PWD:\n%s", my)
	}
}
