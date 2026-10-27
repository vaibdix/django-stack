package event

import "time"

type Event interface{ isEvent() }

type Started struct {
	Effect string
	Label  string
	At     time.Time
}

type Output struct {
	Effect string
	Line   string
}

type Finished struct {
	Effect string
	Label  string
	Took   time.Duration
	Err    error
}

type Note struct {
	Text string
}

func (Started) isEvent() {}

func (Output) isEvent() {}

func (Finished) isEvent() {}

func (Note) isEvent() {}

type Notes interface {
	Step(message string)
}

type Discard struct{}

func (Discard) Step(string) {}
