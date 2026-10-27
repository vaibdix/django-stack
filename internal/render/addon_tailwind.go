package render

import (
	"bytes"
	"encoding/json"
	"path/filepath"

	"github.com/vaibdix/django-stack/internal/config"
)

const TailwindBinary = "tailwindcss"

func TailwindVersion() string { return config.Tools().Tailwind }

func TailwindMinNode() int { return config.NodeMinimum() }

func TailwindPackages() []string {
	return []string{"tailwindcss@" + TailwindVersion(), "@tailwindcss/cli@" + TailwindVersion()}
}

const (
	TailwindSourceDir  = "assets"
	TailwindSourceFile = "tailwind.css"
	TailwindOutput     = "static/dist/styles.css"
)

func (p Project) tailwindFiles() FileSet {
	files := FileSet{}
	if p.Cfg.UsesTailwind() {
		files.add(filepath.Join(TailwindSourceDir, TailwindSourceFile), RenderStatic("tailwind-source.css.tmpl"), FileMode)
	}
	return files
}

func MarshalPackageJSON(manifest any) ([]byte, error) {
	var b bytes.Buffer
	encoder := json.NewEncoder(&b)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(manifest); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func tailwindFeature() feature {
	return feature{
		enabled: config.Spec.UsesTailwind,
		contribute: func(Project) contribution {
			return contribution{readme: func(d *readmeData) { d.Tailwind = true }}
		},
	}
}
