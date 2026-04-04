package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func (m *Model) InitializeGitRepository(projectPath string) error {
	if !m.InitializeGit {
		return nil
	}

	gitCmd := exec.Command("git", "init")
	gitCmd.Dir = projectPath
	if output, err := gitCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to initialize Git repository: %w\nOutput: %s", err, string(output))
	}
	m.AppendStepMessage("Git repository initialized.")

	gitignoreContent := `# Django
*.log
*.pot
*.pyc
__pycache__/
local_settings.py
db.sqlite3
db.sqlite3-journal
media
node_modules/

# Environment variables (NEVER commit these)
.env
.env.local
.env.*.local

# Virtual environment
venv/
.venv/
env/
ENV/

# IDE
.vscode/
.idea/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db
`
	if err := os.WriteFile(filepath.Join(projectPath, ".gitignore"), []byte(gitignoreContent), 0644); err != nil {
		return fmt.Errorf("failed to create .gitignore: %w", err)
	}
	m.AppendStepMessage(".gitignore file created.")

	return nil
}

func (m *Model) RunDjangoMigrations(projectPath string) error {
	m.UpdateProgress("Running database migrations...")

	pythonPath := getPythonPath(projectPath)
	runCmd := func(args ...string) *exec.Cmd {
		return exec.Command(pythonPath, args...)
	}

	cmd := runCmd(ManageFile, "makemigrations")
	cmd.Dir = projectPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create migrations: %w\nOutput: %s", err, string(output))
	}
	m.AppendStepMessage("Created database migrations.")

	cmd = runCmd(ManageFile, "migrate")
	cmd.Dir = projectPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to apply migrations: %w\nOutput: %s", err, string(output))
	}
	m.AppendStepMessage("Applied database migrations.")
	m.AppendStepMessage("Database configured via .env file.")

	if m.SetupRestFramework && m.AppName != "" {
		m.UpdateProgress("Creating sample data...")
		cmd = runCmd(ManageFile, "create_sample_data")
		cmd.Dir = projectPath
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to create sample data: %w\nOutput: %s", err, string(output))
		}
		m.AppendStepMessage("Created sample data for the API.")
	}

	return nil
}
