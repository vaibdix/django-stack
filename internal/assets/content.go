package assets

import "strings"

type pageData struct {
	PageOptions
	AppTitle  string
	AppRoute  string
	DocsRoute string
	Boost     string
	NavClass  string
}

func (o PageOptions) data() pageData {
	d := pageData{PageOptions: o, AppRoute: o.appRoute(), DocsRoute: o.docsRoute(), NavClass: navLinkClass}
	if o.AppName != "" {
		d.AppTitle = strings.ToUpper(o.AppName[:1]) + o.AppName[1:]
	}
	if o.HTMX {
		d.Boost = ` hx-boost="false"`
	}
	return d
}

func RenderPage(page Page, opts PageOptions) string {
	style := "plain"
	if opts.Tailwind {
		style = "tailwind"
	}
	switch page {
	case PageBase:
		return Render("page_base_"+style+".html.tmpl", opts.data())
	case PageIndex:
		return Render("page_index_"+style+".html.tmpl", opts.data())
	default:
		panic("assets: no page named " + string(page))
	}
}

const navLinkClass = "text-body transition-colors duration-300 hover:text-heading"
