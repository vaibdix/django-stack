package text

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/vaibdix/django-stack/internal/event"
)

func feed(events ...event.Event) <-chan event.Event {
	ch := make(chan event.Event, len(events))
	for _, ev := range events {
		ch <- ev
	}
	close(ch)
	return ch
}

func run() []event.Event {
	return []event.Event{
		event.Started{Effect: "install django", Label: "Installing Django..."},
		event.Output{Effect: "install django", Line: "Resolved 12 packages"},
		event.Note{Text: "Installed <ok>."},
		event.Started{Effect: "start database", Label: "Starting the database..."},
		event.Finished{Effect: "install django", Label: "Installing Django...", Took: 1500 * time.Millisecond},
		event.Finished{Effect: "start database", Label: "Starting the database...", Took: 9530 * time.Millisecond, Err: errors.New("port taken")},
	}
}

func TestLinesReportHowLongEachEffectTook(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	Lines(&out, feed(run()...), false)
	want := "==> Installing Django...\n  Installed <ok>.\n==> Starting the database...\n" +
		"<== Installing Django... done in 1.50s\n<== Starting the database... failed in 9.53s\n"
	if out.String() != want {
		t.Errorf("output:\n%s\nwant:\n%s", out.String(), want)
	}
}

func TestVerboseLinesAlsoShowCommandOutput(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	Lines(&out, feed(run()...), true)
	if !strings.Contains(out.String(), "==> Installing Django...\n    | Resolved 12 packages\n") {
		t.Errorf("verbose output does not show the command's line under its effect:\n%s", out.String())
	}
}

func TestJSONIsOneEventPerLine(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	JSON(&out, feed(run()...))
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	want := []string{
		`{"event":"started","step":"install django","text":"Installing Django..."}`,
		`{"event":"output","step":"install django","text":"Resolved 12 packages"}`,
		`{"event":"note","text":"Installed <ok>."}`,
		`{"event":"started","step":"start database","text":"Starting the database..."}`,
		`{"event":"finished","step":"install django","text":"Installing Django...","seconds":1.5}`,
		`{"event":"finished","step":"start database","text":"Starting the database...","seconds":9.53,"error":"port taken"}`,
	}
	if !slices.Equal(lines, want) {
		t.Errorf("events:\n%s\nwant:\n%s", strings.Join(lines, "\n"), strings.Join(want, "\n"))
	}
}
