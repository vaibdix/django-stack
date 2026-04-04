package main

import (
	"fmt"
	"os"
	"strings"

	"django-stack/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
)

func main() {
	if len(os.Args) > 1 {
		arg := os.Args[1]
		switch arg {
		case "--install", "-install":
			if err := install(); err != nil {
				fmt.Fprintf(os.Stderr, "Installation failed: %v\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		case "--help", "-h", "help":
			printHelp()
			os.Exit(0)
		case "--version", "-v", "version":
			printVersion()
			os.Exit(0)
		}
	}

	model := ui.NewModel()
	program := tea.NewProgram(model, tea.WithAltScreen())
	model.SetProgram(program)
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to run django-stack: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	helpText := []string{
		"django-stack",
		"",
		"TUI-first Django project generator.",
		"",
		"Usage:",
		"  django-stack            Start the interactive Bubble Tea setup",
		"  django-stack --help     Show this help",
		"  django-stack --version  Show build version information",
		"  django-stack --install  Install the binary into your local PATH",
		"",
		"The setup flow walks through project name, Python interpreter, Django version,",
		"database choice, and optional project features.",
	}
	fmt.Println(strings.Join(helpText, "\n"))
}

func printVersion() {
	fmt.Printf("django-stack %s\n", version)
	if commit != "" {
		fmt.Printf("commit: %s\n", commit)
	}
	if date != "" {
		fmt.Printf("date: %s\n", date)
	}
}
