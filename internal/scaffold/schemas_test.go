package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSchemasRejectPayloadsTheModelWouldReject(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "schemas.py"), []byte(generateBookSchemas()), 0o644); err != nil {
		t.Fatalf("writing schemas.py: %v", err)
	}

	for _, tc := range []struct {
		name    string
		payload string
		field   string
	}{
		{name: "isbn too short", payload: `{**base, 'isbn': '123'}`, field: "isbn"},
		{name: "isbn not digits", payload: `{**base, 'isbn': 'abcdefghijklm'}`, field: "isbn"},
		{name: "negative price", payload: `{**base, 'price': '-1'}`, field: "price"},
		{name: "empty title", payload: `{**base, 'title': ''}`, field: "title"},
		{name: "over-long title", payload: `{**base, 'title': 'x' * 201}`, field: "title"},
		{name: "unknown field", payload: `{**base, 'publisher': 'nope'}`, field: "publisher"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			script := `
import sys
sys.path.insert(0, '.')
from schemas import BookIn

base = {
    'title': 'A Book',
    'author': 'An Author',
    'isbn': '9781735467200',
    'publication_date': '2024-01-01',
    'price': '9.99',
}
BookIn.model_validate(` + tc.payload + `)
`
			out, ok := uvRunPydantic(t, root, script)
			if ok {
				t.Fatalf("%s was accepted:\n%s", tc.name, out)
			}
			if !strings.Contains(out, tc.field) {
				t.Errorf("the rejection does not name %s, so the API's 400 says nothing useful:\n%s", tc.field, out)
			}
		})
	}
}

func TestSchemasAcceptAValidBook(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "schemas.py"), []byte(generateBookSchemas()), 0o644); err != nil {
		t.Fatalf("writing schemas.py: %v", err)
	}

	out, ok := uvRunPydantic(t, root, `
import sys
sys.path.insert(0, '.')
from decimal import Decimal
from schemas import BookIn

book = BookIn.model_validate({
    'title': '  A Book  ',
    'author': 'An Author',
    'isbn': '9781735467200',
    'publication_date': '2024-01-01',
    'price': '9.99',
})
assert book.title == 'A Book', book.title
assert book.price == Decimal('9.99'), book.price
print('OK')
`)

	if !ok || !strings.Contains(out, "OK") {
		t.Fatalf("a valid book was rejected by its own schema:\n%s", out)
	}
}

func TestSeedDataIsValidatedAtImport(t *testing.T) {
	t.Parallel()

	seed := generateSampleData("catalog", true)
	if !strings.Contains(seed, "BookIn(") {
		t.Fatalf("the seed fixture is not built from the schema:\n%s", seed)
	}
	if strings.Contains(seed, "'title':") {
		t.Errorf("the seed fixture still holds bare dicts:\n%s", seed)
	}
}

func TestModelAndSchemaAgreeOnTheISBNRule(t *testing.T) {
	t.Parallel()

	const pattern = `^\d{13}$`

	if model := generateBookModel("catalog", true, "6.1"); !strings.Contains(model, pattern) {
		t.Errorf("models.py does not enforce the ISBN rule the schema does:\n%s", model)
	}
	if schema := generateBookSchemas(); !strings.Contains(schema, pattern) {
		t.Errorf("schemas.py does not enforce the ISBN rule the model does:\n%s", schema)
	}
}

func TestAPITranslatesSchemaFailuresIntoBadRequests(t *testing.T) {
	t.Parallel()

	api := generateBookAPI(true)

	for _, want := range []string{
		"except ValidationError as error:",
		"raise DRFValidationError(field_errors(error)) from error",
	} {
		if !strings.Contains(api, want) {
			t.Errorf("api.py lets a schema failure escape as a 500; missing %q:\n%s", want, api)
		}
	}

	if !strings.Contains(generateBookSchemas(), "def field_errors(") {
		t.Error("schemas.py does not render pydantic errors in the shape DRF returns")
	}
}
