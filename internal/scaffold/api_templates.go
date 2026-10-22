package scaffold

type AppTemplateConfig struct {
	AppName  string
	Tailwind bool
	HTMX     bool
}

func (c AppTemplateConfig) style() string {
	if c.Tailwind {
		return "tailwind"
	}
	return "plain"
}

func (c AppTemplateConfig) templateFor(page string) string {
	return page + "_" + c.style() + ".html.tmpl"
}

func (c AppTemplateConfig) render(page string) string {
	return render(c.templateFor(page), c)
}
