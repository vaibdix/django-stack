package tui

// Step identifies which screen of the setup flow is active.
//
// Every value here is reachable. Six further constants (StepFeatures,
// StepTemplates, StepCreateApp, StepAppTemplates, StepServerOption,
// StepGitInit) were removed: they were never assigned and never matched in
// Update's transition switch or View's render switch, so they described a state
// machine that did not exist.
type Step int

// The screens of the setup flow, in the order the user sees them.
const (
	StepSplashScreen Step = iota
	StepProjectName
	StepPythonSelection
	StepDjangoVersion
	StepDatabase
	StepProjectConfig
	StepAppName
	StepSetup
	StepDevServerPrompt
	StepComplete
)

// ProjectProgressMsg reports scaffolding progress to the update loop.
type ProjectProgressMsg struct {
	Percent float64
	Status  string
}

// ProjectCreationDoneMsg signals that scaffolding finished, successfully or not.
type ProjectCreationDoneMsg struct {
	Err error
}

// TickMsg drives the splash-screen countdown.
type TickMsg struct{}

// BlinkMsg drives the mascot's blink animation.
type BlinkMsg struct{}
