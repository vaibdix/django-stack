package render

import (
	"github.com/vaibdix/django-stack/internal/assets"
)

func Render(name string, data any) string { return assets.Render(name, data) }

func RenderStatic(name string) string { return assets.Render(name, nil) }
