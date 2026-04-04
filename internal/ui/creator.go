package ui

import (
	"fmt"
	"os"
	"path/filepath"
)

func CreateProject(m *Model) (currentErr error) {

	defer func() {
		if m.Program != nil {
			m.Program.Send(ProjectCreationDoneMsg{Err: currentErr})
		}
		if currentErr == nil {
			m.ShowPerformanceTip()
		}
	}()

	if m.ProjectName == "" {
		currentErr = fmt.Errorf("project name cannot be empty")
		return
	}

	if err := m.ValidateProjectName(); err != nil {
		currentErr = fmt.Errorf("invalid project name '%s': %w", m.ProjectName, err)
		return
	}

	if err := m.ValidateAppName(); err != nil {
		currentErr = fmt.Errorf("invalid app name '%s': %w", m.AppName, err)
		return
	}

	projectPath := m.ProjectName
	if !filepath.IsAbs(projectPath) {
		wd, err := os.Getwd()
		if err != nil {
			currentErr = fmt.Errorf("failed to get working directory: %w", err)
			return
		}
		projectPath = filepath.Join(wd, m.ProjectName)
	}

	m.TotalSteps = m.CalculateTotalSteps()
	m.CompletedSteps = 0

	info, statErr := os.Stat(projectPath)
	if statErr == nil {
		if !info.IsDir() {
			currentErr = fmt.Errorf("project path exists and is not a directory: %s", projectPath)
			return
		}
		entries, err := os.ReadDir(projectPath)
		if err != nil {
			currentErr = fmt.Errorf("failed to inspect project directory: %w", err)
			return
		}
		if len(entries) > 0 {
			currentErr = fmt.Errorf("project directory already exists and is not empty: %s", projectPath)
			return
		}
	} else if !os.IsNotExist(statErr) {
		currentErr = fmt.Errorf("failed to inspect project directory: %w", statErr)
		return
	}

	if err := os.MkdirAll(projectPath, 0755); err != nil {
		currentErr = fmt.Errorf("failed to create project directory: %w", err)
		return
	}
	m.AppendStepMessage(fmt.Sprintf("Project directory created: %s", projectPath))

	if currentErr = m.CreateVirtualEnvironment(projectPath); currentErr != nil {
		return
	}
	if currentErr = m.InstallDjango(projectPath); currentErr != nil {
		return
	}
	if currentErr = m.CreateDjangoProject(projectPath); currentErr != nil {
		return
	}

	if m.DatabaseType == "postgres-docker" {
		m.AppendStepMessage("Setting up PostgreSQL with Docker...")
		connStr, err := m.SetupDockerPostgres(projectPath)
		if err != nil {
			currentErr = fmt.Errorf("failed to setup PostgreSQL Docker: %w", err)
			return
		}
		m.AppendStepMessage("PostgreSQL Docker container started!")
		m.AppendStepMessage("Connection: " + connStr)
	}

	if m.DatabaseType == "mysql-docker" {
		m.AppendStepMessage("Setting up MySQL with Docker...")
		connStr, err := m.SetupDockerMySQL(projectPath)
		if err != nil {
			currentErr = fmt.Errorf("failed to setup MySQL Docker: %w", err)
			return
		}
		m.AppendStepMessage("MySQL Docker container started!")
		m.AppendStepMessage("Connection: " + connStr)

		if currentErr = m.installAndReport(projectPath, "mysqlclient", PkgMySQLClient); currentErr != nil {
			return
		}
	}

	m.AppendStepMessage("Creating environment configuration...")
	if currentErr = m.CreateEnvFile(projectPath); currentErr != nil {
		return
	}
	if currentErr = m.CreateEnvExampleFile(projectPath); currentErr != nil {
		return
	}

	if m.DatabaseType == "postgres-docker" {
		m.AppendStepMessage("Creating docker-compose.yml for PostgreSQL...")
		if err := generateDockerComposeFile(projectPath); err != nil {
			currentErr = fmt.Errorf("failed to create docker-compose.yml: %w", err)
			return
		}
		m.AppendStepMessage("Created: docker-compose.yml")
	}

	if m.DatabaseType == "mysql-docker" {
		m.AppendStepMessage("Creating docker-compose.yml for MySQL...")
		if err := generateDockerComposeFileMySQL(projectPath); err != nil {
			currentErr = fmt.Errorf("failed to create docker-compose.yml: %w", err)
			return
		}
		m.AppendStepMessage("Created: docker-compose.yml")
	}

	if m.DatabaseType == "postgres-string" {
		m.AppendStepMessage("Creating PostgreSQL setup instructions...")
		if currentErr = m.CreatePostgresReadme(projectPath); currentErr != nil {
			return
		}
	}

	settingsPath := filepath.Join(projectPath, m.ProjectModuleName(), SettingsFile)
	if currentErr = m.ConfigureDjangoSettings(settingsPath); currentErr != nil {
		return
	}
	if currentErr = m.SetupProjectUrls(projectPath); currentErr != nil {
		return
	}
	if m.InitializeGit {
		if currentErr = m.InitializeGitRepository(projectPath); currentErr != nil {
			return
		}
	}

	if m.CreateTemplates {
		if currentErr = m.SetupGlobalTemplates(projectPath); currentErr != nil {
			return
		}
		se, err := NewSettingsEditor(settingsPath)
		if err != nil {
			currentErr = err
			return
		}
		se.ConfigureTemplatesDirs()
		if err := se.Save(); err != nil {
			currentErr = err
			return
		}
		m.AppendStepMessage("Configured settings for global templates and static files.")
	}

	if m.SetupTailwind {
		m.UpdateProgress("Setting up Tailwind CSS v4...")
		if currentErr = m.SetupTailwindCSS(projectPath); currentErr != nil {
			return
		}
	}

	if m.SetupHTMX {
		if currentErr = m.SetupHTMXIntegration(projectPath); currentErr != nil {
			return
		}
	}

	if m.SetupAllAuth {
		if currentErr = m.SetupAllAuthIntegration(projectPath); currentErr != nil {
			return
		}
	}

	if m.SetupRuff {
		if currentErr = m.SetupRuffLinting(projectPath); currentErr != nil {
			return
		}
	}

	if m.SetupMypy {
		if currentErr = m.SetupMypyTypeChecking(projectPath); currentErr != nil {
			return
		}
	}

	// Pre-commit must run after ruff config is in place
	if m.SetupPreCommit {
		if currentErr = m.SetupPreCommitHooks(projectPath); currentErr != nil {
			return
		}
	}

	if currentErr != nil {
		return
	}

	m.UpdateProgress("Finalizing settings configuration...")

	if m.AppName != "" {
		if currentErr = m.CreateDjangoApp(projectPath, settingsPath); currentErr != nil {
			return
		}
		m.AppendStepMessage(fmt.Sprintf("Created Django app '%s' with templates and URLs.", m.AppName))
	}

	if m.SetupRestFramework {
		if currentErr = m.SetupDjangoRestFramework(projectPath); currentErr != nil {
			return
		}
	}

	// Multi-env runs LAST — it splits settings.py into a directory.
	// All other features must have finished writing to settings.py first.
	if m.SetupMultiEnv {
		if currentErr = m.SetupMultiEnvSettings(projectPath); currentErr != nil {
			return
		}
	}

	if currentErr = m.RunDjangoMigrations(projectPath); currentErr != nil {
		return
	}
	if currentErr = m.CreateSuperuserSetupGuide(projectPath); currentErr != nil {
		return
	}

	if currentErr = m.RunMandatoryScaffoldFormatting(projectPath); currentErr != nil {
		return
	}

	if m.RunServer {
		m.SetupServerInstructions(projectPath)
	}

	createVSCodeTasks(projectPath, m.SetupTailwind)
	m.AppendStepMessage("VS Code configuration created (.vscode/)")

	vscodePath := findVSCode()
	scriptPath := filepath.Join(projectPath, devStartScriptName())
	script := generateStartScript(projectPath, vscodePath, m.SetupTailwind)
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		m.AppendStepMessage("Warning: Could not create development start script: " + err.Error())
	} else {
		m.AppendStepMessage(fmt.Sprintf("Created: %s", filepath.Base(scriptPath)))
	}

	m.AppendStepMessage("Django project setup complete!")
	return nil
}
