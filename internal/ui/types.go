package ui

type Step int

const (
	StepSplashScreen Step = iota
	StepProjectName
	StepPythonSelection
	StepDjangoVersion
	StepDatabase
	StepProjectConfig
	StepAppName
	StepFeatures
	StepTemplates
	StepSetup
	StepCreateApp
	StepAppTemplates
	StepServerOption
	StepGitInit
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
