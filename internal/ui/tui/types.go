package tui

type wizardStep int

const (
	stepSplashScreen wizardStep = iota
	stepProjectName
	stepPythonSelection
	stepDjangoVersion
	stepDatabase
	stepDockerCheck
	stepStack
	stepProjectConfig
	stepAppName
	stepSetup
	stepDevServerPrompt
	stepComplete
)

type tickMsg struct{}
