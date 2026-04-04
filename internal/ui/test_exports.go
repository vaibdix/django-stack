package ui

import (
	"net"
	"net/http"
)

// TestDjangoReleaseFile is a lightweight exported adapter used by tests outside this package.
type TestDjangoReleaseFile struct {
	UploadTimeISO8601 string
	Yanked            bool
}

func ParsePythonVersionOutputForTest(output string) (int, int, bool) {
	return parsePythonVersionOutput(output)
}

func BuildDjangoOptionsFromReleaseIndexForTest(pythonOption PythonOption, releases map[string][]TestDjangoReleaseFile) []DjangoOption {
	internalReleases := make(map[string][]djangoReleaseFile, len(releases))
	for version, files := range releases {
		converted := make([]djangoReleaseFile, 0, len(files))
		for _, file := range files {
			converted = append(converted, djangoReleaseFile{
				UploadTimeISO8601: file.UploadTimeISO8601,
				Yanked:            file.Yanked,
			})
		}
		internalReleases[version] = converted
	}
	return buildDjangoOptionsFromReleaseIndex(pythonOption, internalReleases)
}

func FallbackDjangoOptionsForPythonForTest(pythonOption PythonOption) []DjangoOption {
	return fallbackDjangoOptionsForPython(pythonOption)
}

func SetDjangoReleaseHTTPClientForTest(client *http.Client) func() {
	original := djangoReleaseHTTPClient
	djangoReleaseHTTPClient = client
	return func() {
		djangoReleaseHTTPClient = original
	}
}

func FetchDjangoReleaseOptionsFromPyPIForTest(pythonOption PythonOption) ([]DjangoOption, string) {
	return fetchDjangoReleaseOptionsFromPyPI(pythonOption)
}

func RenderEnvFileContentForTest(model *Model) string {
	return model.renderEnvFileContent()
}

func AdminPyContentForTest(model *Model) string {
	return model.adminPyContent()
}

func RenderEnvExampleContentForTest(model *Model) string {
	return model.renderEnvExampleContent()
}

func PackageInstallCommandForTest(projectPath string) (string, []string) {
	return packageInstallCommand(projectPath)
}

func InstallPackagesCommandForTest(projectPath string, packages ...string) (string, []string) {
	cmd := installPackages(projectPath, packages...)
	return cmd.Path, cmd.Args
}

func DevServerCommandForTest() string {
	return devServerCommand()
}

func GenerateStartScriptForTest(projectPath, vscodePath string, useTailwind bool) string {
	return generateStartScript(projectPath, vscodePath, useTailwind)
}

func AddProjectContextProcessorForTest(settingsContent, projectModuleName string) string {
	se := &SettingsEditor{content: settingsContent}
	se.AddProjectContextProcessor(projectModuleName)
	return se.Content()
}

func UpdateSettingsForTemplatesForTest(settingsContent string) string {
	se := &SettingsEditor{content: settingsContent}
	se.ConfigureTemplatesDirs()
	return se.Content()
}

func BuildProjectURLsContentForTest(model *Model) string {
	return model.buildProjectURLsContent()
}

func CreateExampleAPIForTest(model *Model, projectPath string) error {
	return model.createExampleAPI(projectPath)
}

func SetPickerLoadersForTest(
	pythonLoader func() []PythonOption,
	djangoLoader func(PythonOption) ([]DjangoOption, string),
) func() {
	originalPythonLoader := loadPythonOptions
	originalDjangoLoader := fetchDjangoReleaseOptions
	loadPythonOptions = pythonLoader
	fetchDjangoReleaseOptions = djangoLoader
	return func() {
		loadPythonOptions = originalPythonLoader
		fetchDjangoReleaseOptions = originalDjangoLoader
	}
}

func DiscoverPythonOptionsFromCandidatesForTest(candidates []string, defaultPath string) []PythonOption {
	return discoverPythonOptionsFromCandidates(candidates, defaultPath)
}

func NormalizeExecutablePathForTest(path string) (string, bool) {
	return normalizeExecutablePath(path)
}

func InjectEnvConfigForTest(settingsStr string, model *Model) string {
	se := &SettingsEditor{content: settingsStr}
	se.InjectEnvConfig(model.DatabaseType)
	return se.Content()
}

func WriteSuperuserSetupGuideForTest(projectPath string) error {
	return writeSuperuserSetupGuide(projectPath)
}

func FindAvailablePortForTest(preferred int) (string, error) {
	return getAvailablePort(preferred)
}

func SetPortListenerForTest(listener func(network, address string) (net.Listener, error)) func() {
	original := listenTCP
	listenTCP = listener
	return func() {
		listenTCP = original
	}
}

func GenerateBookViewsForTest(appName string, htmx bool) string {
	return generateBookViews(appName, htmx)
}

func PythonToolingMetadataForTest(model *Model) (string, string, string) {
	metadata := model.pythonToolingMetadata()
	return metadata.RequiresPython, metadata.RuffTargetVersion, metadata.MypyPythonVersion
}

type TestFormattingCommand struct {
	Stage  string
	Module string
	Args   []string
}

func BuildScaffoldFormattingPlanForTest(projectPath string) ([]string, []TestFormattingCommand, []string, error) {
	plan, err := buildScaffoldFormattingPlan(projectPath)
	if err != nil {
		return nil, nil, nil, err
	}

	commands := make([]TestFormattingCommand, 0, len(plan.Commands))
	for _, command := range plan.Commands {
		commands = append(commands, TestFormattingCommand{
			Stage:  command.Stage,
			Module: command.Module,
			Args:   append([]string{}, command.Args...),
		})
	}

	return append([]string{}, plan.Packages...), commands, append([]string{}, plan.TemplateFiles...), nil
}

func SetFormattingInstallHookForTest(hook func(m *Model, projectPath string, packages ...string) error) func() {
	original := installFormattingPackages
	installFormattingPackages = hook
	return func() {
		installFormattingPackages = original
	}
}

func SetPythonModuleCommandRunnerForTest(hook func(projectPath, module string, args ...string) ([]byte, error)) func() {
	original := runPythonModuleCommand
	runPythonModuleCommand = hook
	return func() {
		runPythonModuleCommand = original
	}
}

func RunMandatoryScaffoldFormattingForTest(model *Model, projectPath string) error {
	return model.RunMandatoryScaffoldFormatting(projectPath)
}
