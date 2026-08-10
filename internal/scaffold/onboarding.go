package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
)

const superuserGuideFile = "SUPERUSER_SETUP.md"

func writeSuperuserSetupGuide(projectPath string) error {
	content := renderStatic("superuser-setup.md.tmpl")

	return os.WriteFile(filepath.Join(projectPath, superuserGuideFile), []byte(content), 0644)
}

func (e *Env) writeSuperuserGuide(projectPath string) error {
	if err := writeSuperuserSetupGuide(projectPath); err != nil {
		return fmt.Errorf("failed to create %s: %w", superuserGuideFile, err)
	}
	e.Step("Created " + superuserGuideFile + " with venv-only superuser setup instructions.")
	return nil
}
