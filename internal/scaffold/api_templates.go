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

func generateBookListTemplate(cfg AppTemplateConfig) string {
	return render(cfg.templateFor("book_list"), cfg)
}

func generateBookFormTemplate(cfg AppTemplateConfig) string {
	return render(cfg.templateFor("book_form"), cfg)
}

func generateBookDeleteTemplate(cfg AppTemplateConfig) string {
	return render(cfg.templateFor("book_delete"), cfg)
}

func generateBookListItemsPartial(cfg AppTemplateConfig) string {
	return render(cfg.templateFor("book_list_items"), cfg)
}

func generateBookFormPartial(cfg AppTemplateConfig) string {
	return render(cfg.templateFor("book_form_partial"), cfg)
}

func generateBookFieldsPartial(cfg AppTemplateConfig) string {
	return render(cfg.templateFor("book_fields"), cfg)
}
