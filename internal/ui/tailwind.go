package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SetupTailwindCSS configures Tailwind CSS v4. Errors are non-fatal:
// the project is fully functional without Tailwind, so failures
// are logged as warnings and the function returns nil.
func (m *Model) SetupTailwindCSS(projectPath string) error {
	if !m.SetupTailwind {
		return nil
	}

	m.UpdateProgress("Setting up Tailwind CSS v4...")

	if !isCommandAvailable("npm") {
		m.AppendStepMessage("Warning: npm not found. Please install Node.js to use Tailwind CSS.")
		return nil
	}

	// npm install (not pip) — installAndReport is for Python packages only
	cmd := exec.Command("npm", "init", "-y")
	cmd.Dir = projectPath
	if output, err := cmd.CombinedOutput(); err != nil {
		m.AppendStepMessage(fmt.Sprintf("Warning: Failed to initialize npm: %v\nOutput: %s", err, string(output)))
		return nil
	}
	m.AppendStepMessage("npm initialized.")

	cmd = exec.Command("npm", "install", "tailwindcss", "@tailwindcss/cli")
	cmd.Dir = projectPath
	if output, err := cmd.CombinedOutput(); err != nil {
		m.AppendStepMessage(fmt.Sprintf("Warning: Failed to install Tailwind CSS: %v\nOutput: %s", err, string(output)))
		return nil
	}
	m.AppendStepMessage("Tailwind CSS v4 installed.")

	staticSrcPath := filepath.Join(projectPath, "static", "src")
	staticDistPath := filepath.Join(projectPath, "static", "dist")
	if err := os.MkdirAll(staticSrcPath, 0755); err != nil {
		m.AppendStepMessage(fmt.Sprintf("Warning: Failed to create static/src directory: %v", err))
		return nil
	}
	if err := os.MkdirAll(staticDistPath, 0755); err != nil {
		m.AppendStepMessage(fmt.Sprintf("Warning: Failed to create static/dist directory: %v", err))
		return nil
	}
	m.AppendStepMessage("Tailwind directory structure created.")
	tailwindCSS := `@import "tailwindcss";

@theme {
  /* Primary - Blue 600 */
  --color-primary: hsl(215 100% 50%);
  --color-primary-foreground: hsl(0 0% 100%);

  /* Background - White */
  --color-background: hsl(0 0% 100%);
  --color-foreground: hsl(222 47% 11%);

  /* Card - White with shadow */
  --color-card: hsl(0 0% 100%);
  --color-card-foreground: hsl(222 47% 11%);

  /* Popover */
  --color-popover: hsl(0 0% 100%);
  --color-popover-foreground: hsl(222 47% 11%);

  /* Muted */
  --color-muted: hsl(210 40% 96%);
  --color-muted-foreground: hsl(215 16% 47%);

  /* Secondary */
  --color-secondary: hsl(210 40% 96%);
  --color-secondary-foreground: hsl(222 47% 11%);

  /* Border */
  --color-border: hsl(214 32% 91%);
  --color-input: hsl(214 32% 91%);
  --color-ring: hsl(215 100% 50%);

  /* Destructive */
  --color-destructive: hsl(0 84% 60%);
  --color-destructive-foreground: hsl(0 0% 100%);

  /* Accent */
  --color-accent: hsl(210 40% 96%);
  --color-accent-foreground: hsl(222 47% 11%);

  /* Sidebar */
  --color-sidebar-background: hsl(0 0% 100%);
  --color-sidebar-foreground: hsl(222 47% 11%);
  --color-sidebar-primary: hsl(215 100% 50%);
  --color-sidebar-primary-foreground: hsl(0 0% 100%);
  --color-sidebar-accent: hsl(210 40% 96%);
  --color-sidebar-accent-foreground: hsl(222 47% 11%);
  --color-sidebar-border: hsl(214 32% 91%);
  --color-sidebar-ring: hsl(215 100% 50%);

  /* Chart colors */
  --color-chart-1: hsl(215 100% 50%);
  --color-chart-2: hsl(142 71% 45%);
  --color-chart-3: hsl(38 92% 50%);
  --color-chart-4: hsl(280 65% 60%);
  --color-chart-5: hsl(340 75% 55%);

  /* Border radius */
  --radius-lg: 0.5rem;
  --radius-md: 0.375rem;
  --radius-sm: 0.25rem;
}

/* Base styles */
body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
  background-color: hsl(0 0% 0%);
  color: hsl(0 0% 100%);
  line-height: 1.5;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

/* Card hover effect */
.card-hover {
  transition: all 0.2s ease-in-out;
}
.card-hover:hover {
  box-shadow: 0 10px 15px -3px rgb(0 0 0 / 0.1), 0 4px 6px -4px rgb(0 0 0 / 0.1);
  transform: translateY(-2px);
}

/* Button base */
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  white-space: nowrap;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  font-weight: 500;
  transition: all 0.2s;
  cursor: pointer;
}

.btn-primary {
  background-color: hsl(215 100% 50%);
  color: white;
}
.btn-primary:hover {
  background-color: hsl(215 100% 45%);
}

.btn-secondary {
  background-color: hsl(210 40% 96%);
  color: hsl(222 47% 11%);
  border: 1px solid hsl(214 32% 91%);
}
.btn-secondary:hover {
  background-color: hsl(210 40% 93%);
}

/* Form inputs */
input, select, textarea {
  width: 100%;
  padding: 0.5rem 0.75rem;
  border: 1px solid rgba(255,255,255,0.1);
  border-radius: 0.375rem;
  background-color: rgba(255,255,255,0.03);
  color: white;
  font-size: 0.875rem;
  transition: border-color 0.2s, box-shadow 0.2s;
}
input:focus, select:focus, textarea:focus {
  outline: none;
  border-color: rgba(255,255,255,0.3);
  box-shadow: 0 0 0 3px rgba(255,255,255,0.05);
}

/* Badge */
.badge {
  display: inline-flex;
  align-items: center;
  border-radius: 9999px;
  padding: 0.125rem 0.625rem;
  font-size: 0.75rem;
  font-weight: 500;
}

/* Empty state animation */
@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
.animate-in {
  animation: fadeIn 0.3s ease-out forwards;
}`
	if err := os.WriteFile(filepath.Join(staticSrcPath, "styles.css"), []byte(tailwindCSS), 0644); err != nil {
		m.AppendStepMessage(fmt.Sprintf("Warning: Failed to create styles.css: %v", err))
		return nil
	}
	m.AppendStepMessage("Tailwind source CSS created.")
	if err := m.updatePackageJSONForTailwind(projectPath); err != nil {
		m.AppendStepMessage(fmt.Sprintf("Warning: Failed to update package.json: %v", err))
		return nil
	}
	if m.CreateTemplates {
		if err := m.updateBaseTemplateForTailwind(projectPath); err != nil {
			m.AppendStepMessage(fmt.Sprintf("Warning: Failed to update base.html: %v", err))
			return nil
		}
	}
	cmd = exec.Command("npm", "run", "build:css")
	cmd.Dir = projectPath
	if output, err := cmd.CombinedOutput(); err != nil {
		m.AppendStepMessage(fmt.Sprintf("Warning: Failed to build Tailwind CSS: %v\nOutput: %s", err, string(output)))
	} else {
		m.AppendStepMessage("Tailwind CSS compiled successfully.")
		m.AppendStepMessage("Run 'npm run watch:css' for development or 'npm run build:css' for production.")
	}
	return nil
}

func (m *Model) updatePackageJSONForTailwind(projectPath string) error {
	packageJSONPath := filepath.Join(projectPath, "package.json")
	packageData, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return fmt.Errorf("failed to read package.json: %w", err)
	}
	var packageJSON map[string]interface{}
	if err := json.Unmarshal(packageData, &packageJSON); err != nil {
		return fmt.Errorf("failed to parse package.json: %w", err)
	}
	scripts := map[string]string{
		"build:css": "npx tailwindcss -i ./static/src/styles.css -o ./static/dist/styles.css",
		"watch:css": "npx tailwindcss -i ./static/src/styles.css -o ./static/dist/styles.css --watch",
	}
	if existingScripts, ok := packageJSON["scripts"].(map[string]interface{}); ok {
		for key, value := range scripts {
			existingScripts[key] = value
		}
		packageJSON["scripts"] = existingScripts
	} else {
		packageJSON["scripts"] = scripts
	}
	updatedData, err := json.MarshalIndent(packageJSON, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal package.json: %w", err)
	}
	if err := os.WriteFile(packageJSONPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write package.json: %w", err)
	}
	m.AppendStepMessage("package.json updated with Tailwind scripts.")
	return nil
}

func (m *Model) updateBaseTemplateForTailwind(projectPath string) error {
	baseTemplatePath := filepath.Join(projectPath, "templates", "base.html")
	baseContent, err := os.ReadFile(baseTemplatePath)
	if err != nil {
		return fmt.Errorf("failed to read base.html: %w", err)
	}
	updatedBaseContent := strings.Replace(string(baseContent),
		`<link rel="stylesheet" href="{% static 'css/style.css' %}">`,
		`<link rel="stylesheet" href="{% static 'dist/styles.css' %}">`, 1)
	if err := os.WriteFile(baseTemplatePath, []byte(updatedBaseContent), 0644); err != nil {
		return fmt.Errorf("failed to update base.html: %w", err)
	}

	m.AppendStepMessage("base.html updated to use Tailwind CSS.")
	return nil
}
