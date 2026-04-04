package ui

import (
	"fmt"
	"os"
	"path/filepath"
)

const superuserGuideFile = "SUPERUSER_SETUP.md"

func writeSuperuserSetupGuide(projectPath string) error {
	content := `# Superuser Setup

This project does not create a default Django superuser automatically.

Use the project virtual environment when creating your first admin account.

## macOS / Linux

` + "```" + `
./.venv/bin/python manage.py createsuperuser
` + "```" + `

## Windows

` + "```" + `
.venv\Scripts\python.exe manage.py createsuperuser
` + "```" + `

After the account is created, open ` + "`/admin/`" + ` and sign in with the credentials you chose.

Important:
- Prefer the commands above for manual Django management tasks.
- Do not use plain ` + "`python manage.py ...`" + ` unless you have activated the project venv first.
`

	return os.WriteFile(filepath.Join(projectPath, superuserGuideFile), []byte(content), 0644)
}

func (m *Model) CreateSuperuserSetupGuide(projectPath string) error {
	if err := writeSuperuserSetupGuide(projectPath); err != nil {
		return fmt.Errorf("failed to create %s: %w", superuserGuideFile, err)
	}
	m.AppendStepMessage("Created " + superuserGuideFile + " with venv-only superuser setup instructions.")
	return nil
}
