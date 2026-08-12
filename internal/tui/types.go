package tui

type Step int

const (
	StepSplashScreen Step = iota
	StepProjectName
	StepPythonSelection
	StepDjangoVersion
	StepDatabase
	StepDockerCheck
	StepStack
	StepProjectConfig
	StepAppName
	StepSetup
	StepDevServerPrompt
	StepComplete
)

type TickMsg struct{}

type BlinkMsg struct{}
