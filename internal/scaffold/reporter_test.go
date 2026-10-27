package scaffold

import (
	"time"
)

type RecordingReporter struct {
	Steps    []string
	Statuses []string
}

func (r *RecordingReporter) Step(message string) { r.Steps = append(r.Steps, message) }

func (r *RecordingReporter) Started(_, label string) { r.Statuses = append(r.Statuses, label) }

func (r *RecordingReporter) Finished(string, string, time.Duration, error) {}
