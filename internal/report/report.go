package report

import "time"

type Steps interface {
	Step(message string)
}

type Reporter interface {
	Steps
	Started(step, label string)
	Finished(step, label string, took time.Duration, err error)
}

type Nop struct{}

func (Nop) Step(string) {}

func (Nop) Started(string, string) {}

func (Nop) Finished(string, string, time.Duration, error) {}
