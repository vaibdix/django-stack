package scaffold

type appTemplateConfig struct {
	AppName  string
	Tailwind bool
	HTMX     bool
}

func (c appTemplateConfig) style() string {
	if c.Tailwind {
		return "tailwind"
	}
	return "plain"
}

func (c appTemplateConfig) templateFor(page string) string {
	return page + "_" + c.style() + ".html.tmpl"
}

func (c appTemplateConfig) render(page string) string {
	return render(c.templateFor(page), c)
}
