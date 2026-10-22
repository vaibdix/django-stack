package scaffold

type RecordingReporter struct {
	Steps    []string
	Statuses []string
}

func (r *RecordingReporter) Step(message string) { r.Steps = append(r.Steps, message) }

func (r *RecordingReporter) Progress(status string) { r.Statuses = append(r.Statuses, status) }
