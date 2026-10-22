package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

func TestCreateExampleAPIWritesRefinedBookTemplates(t *testing.T) {
	t.Parallel()

	projectPath := t.TempDir()
	appPath := filepath.Join(projectPath, "library")
	if err := os.MkdirAll(appPath, 0755); err != nil {
		t.Fatalf("failed to create app dir: %v", err)
	}

	apiEnv := &Env{cfg: config.Project{AppName: "library", SetupTailwind: true, CreateTemplates: true}}
	if err := apiEnv.writeExampleApp(projectPath, generateAPIPackage(apiEnv.cfg.AppName)); err != nil {
		t.Fatalf("writeExampleApp failed: %v", err)
	}

	bookListBytes, err := os.ReadFile(filepath.Join(appPath, "templates", "library", "book_list.html"))
	if err != nil {
		t.Fatalf("failed to read generated book_list.html: %v", err)
	}
	bookList := string(bookListBytes)
	if !strings.Contains(bookList, "Book library") {
		t.Fatalf("expected the book list heading, got:\n%s", bookList)
	}
	if !strings.Contains(bookList, "Open admin") {
		t.Fatalf("expected book list to include admin CTA, got:\n%s", bookList)
	}

	bookFormBytes, err := os.ReadFile(filepath.Join(appPath, "templates", "library", "book_form.html"))
	if err != nil {
		t.Fatalf("failed to read generated book_form.html: %v", err)
	}
	if !strings.Contains(string(bookFormBytes), "{{ action }} book") {
		t.Fatalf("expected the form heading to carry the action, got:\n%s", string(bookFormBytes))
	}

	bookDeleteBytes, err := os.ReadFile(filepath.Join(appPath, "templates", "library", "book_delete.html"))
	if err != nil {
		t.Fatalf("failed to read generated book_delete.html: %v", err)
	}
	if !strings.Contains(string(bookDeleteBytes), "Delete permanently") {
		t.Fatalf("expected refined delete CTA, got:\n%s", string(bookDeleteBytes))
	}

	formsBytes, err := os.ReadFile(filepath.Join(appPath, "forms.py"))
	if err != nil {
		t.Fatalf("failed to read generated forms.py: %v", err)
	}
	if !strings.Contains(string(formsBytes), "border border-rule bg-ground") {
		t.Fatalf("expected token-based input classes in forms.py, got:\n%s", string(formsBytes))
	}
}
