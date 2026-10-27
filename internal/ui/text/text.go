package text

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/vaibdix/django-stack/internal/event"
)

func Lines(w io.Writer, events <-chan event.Event, verbose bool) {
	for ev := range events {
		switch ev := ev.(type) {
		case event.Note:
			_, _ = fmt.Fprintln(w, "  "+ev.Text)
		case event.Started:
			_, _ = fmt.Fprintln(w, "==> "+ev.Label)
		case event.Output:
			if verbose {
				_, _ = fmt.Fprintln(w, "    | "+ev.Line)
			}
		case event.Finished:
			outcome := "done"
			if ev.Err != nil {
				outcome = "failed"
			}
			_, _ = fmt.Fprintf(w, "<== %s %s in %.2fs\n", ev.Label, outcome, ev.Took.Seconds())
		}
	}
}

type record struct {
	Event   string   `json:"event"`
	Step    string   `json:"step,omitempty"`
	Text    string   `json:"text,omitempty"`
	Seconds *float64 `json:"seconds,omitempty"`
	Error   string   `json:"error,omitempty"`
}

func JSON(w io.Writer, events <-chan event.Event) {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	for ev := range events {
		var r record
		switch ev := ev.(type) {
		case event.Note:
			r = record{Event: "note", Text: ev.Text}
		case event.Started:
			r = record{Event: "started", Step: ev.Effect, Text: ev.Label}
		case event.Output:
			r = record{Event: "output", Step: ev.Effect, Text: ev.Line}
		case event.Finished:
			seconds := ev.Took.Seconds()
			r = record{Event: "finished", Step: ev.Effect, Text: ev.Label, Seconds: &seconds}
			if ev.Err != nil {
				r.Error = ev.Err.Error()
			}
		default:
			continue
		}
		_ = encoder.Encode(r)
	}
}
