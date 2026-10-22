package report

type Reporter interface {
	Step(message string)
	Progress(status string)
}

type Nop struct{}

func (Nop) Step(string) {}

func (Nop) Progress(string) {}
