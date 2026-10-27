package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaibdix/django-stack/internal/config"
)

func TestTheISBNRuleIsOneRuleOnEveryWritePath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "fields.py"), []byte(generateBookFields(typingData{RuntimeGenerics: false})), 0o644); err != nil {
		t.Fatalf("writing fields.py: %v", err)
	}

	out, ok := uvRunPydantic(t, root, `
import sys
from datetime import timedelta
sys.path.insert(0, ".")
import django
from django.conf import settings
settings.configure(USE_TZ=True)
django.setup()
from django.core.exceptions import ValidationError
from django.utils import timezone
from fields import ISBN_VALIDATOR, ISBNField, validate_not_in_future

field = ISBNField(max_length=17, validators=[ISBN_VALIDATOR])
assert field.clean("978-0-7352-1129-2", None) == "9780735211292"
assert field.clean("978 0 7352 1129 2", None) == "9780735211292"
for bad in ("123", "abcdefghijklm", "978-0-7352-1129-22"):
    try:
        field.clean(bad, None)
    except ValidationError:
        pass
    else:
        raise AssertionError(f"{bad!r} was accepted")
assert field.get_prep_value("978-1") == "9781"

form_field = field.formfield()
assert form_field is not None
assert form_field.clean("978-0-7352-1129-2") == "978-0-7352-1129-2", "the model field, not the form, normalises"

validate_not_in_future(timezone.localdate())
try:
    validate_not_in_future(timezone.localdate() + timedelta(days=1))
except ValidationError:
    pass
else:
    raise AssertionError("a future date was accepted")
print("OK")
`)
	if !ok || !strings.Contains(out, "OK") {
		t.Fatalf("the shared ISBN and date rules do not behave as one rule:\n%s", out)
	}
}

func TestEveryWritePathDelegatesToTheModelRules(t *testing.T) {
	t.Parallel()

	model := generateBookModel("catalog", true)
	for _, want := range []string{
		"isbn = ISBNField(",
		"validate_not_in_future]",

		"MinValueValidator(EARLIEST_PUBLICATION_DATE, PUBLICATION_DATE_TOO_EARLY)",
		"MinValueValidator(0, NEGATIVE_PRICE)",
	} {
		if !strings.Contains(model, want) {
			t.Errorf("models.py is missing %q, so the model is not the source of the rule:\n%s", want, model)
		}
	}

	serializer := generateAPIPackage("catalog")["serializers.py"]
	if !strings.Contains(serializer, "ISBNField: ISBNSerializerField") {
		t.Errorf("the serializer does not normalise through the model's ISBN field:\n%s", serializer)
	}
	for _, copied := range []string{"def validate_isbn", "def validate_publication_date", "timezone.localdate"} {
		if strings.Contains(serializer, copied) {
			t.Errorf("the serializer re-states a model rule (%q), so the two can drift apart:\n%s", copied, serializer)
		}
	}

	ninja := generateNinjaPackage("catalog", "my-shop")
	if !strings.Contains(ninja["router.py"], "book.full_clean()") {
		t.Errorf("Ninja writes without running the model's validators:\n%s", ninja["router.py"])
	}
	if !strings.Contains(ninja["api.py"], "@api.exception_handler(ValidationError)") {
		t.Errorf("a model ValidationError escapes Ninja as a 500:\n%s", ninja["api.py"])
	}
	if strings.Contains(ninja["schemas.py"], "field_validator") {
		t.Errorf("the Ninja schema re-states a model rule:\n%s", ninja["schemas.py"])
	}

	for _, source := range []string{renderBookViews(bookViewsData{AppName: "catalog", HTMX: true, RuntimeGenerics: true}), generateBookAPI(), generateSampleData("catalog", config.SQLite)} {
		if strings.Contains(source, "BookIn") || strings.Contains(source, "model_validate") {
			t.Errorf("a pydantic re-validation of already-validated data is still generated:\n%s", source)
		}
	}
}
