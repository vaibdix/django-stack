// Package assets holds the static content written into generated projects:
// stylesheets, the shared page layouts, and the django-allauth templates.
//
// It merges two packages that used to sit one character apart — internal/template
// (embedded CSS) and internal/templates (HTML strings) — which were imported
// together and re-exported through each other for no benefit.
//
// Stylesheets are embedded straight into string variables with //go:embed, so a
// renamed or moved asset is a compile error. The previous version read them out
// of an embed.FS and discarded the error, which meant a rename shipped a binary
// that silently wrote an empty <style> block into every generated page.
package assets
