package scaffold

import "github.com/vaibdix/django-stack/internal/assets"

func render(name string, data any) string { return assets.Render(name, data) }

func renderStatic(name string) string { return assets.Render(name, nil) }
