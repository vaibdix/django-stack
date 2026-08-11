package tui

type Step int

const (
	StepSplashScreen Step = iota
	StepProjectName
	StepPythonSelection
	StepDjangoVersion
	StepDatabase
	StepDockerCheck
	StepProjectConfig
	StepAppName
	StepSetup
	StepDevServerPrompt
	StepComplete
)

type ProjectProgressMsg struct {
	Percent float64
	Status  string
}

type ProjectCreationDoneMsg struct {
	Err error
}

type TickMsg struct{}

type BlinkMsg struct{}
