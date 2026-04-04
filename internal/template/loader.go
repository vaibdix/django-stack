package template

import (
	"embed"
)

//go:embed templ/*
var embedded embed.FS

var (
	baseCSS string
	homeCSS string
)

func init() {
	loadEmbeddedCSS()
}

func loadEmbeddedCSS() {
	if data, err := embedded.ReadFile("templ/css/base.css"); err == nil {
		baseCSS = string(data)
	}
	if data, err := embedded.ReadFile("templ/css/home.css"); err == nil {
		homeCSS = string(data)
	}
}

func GetBaseCSS() string {
	return baseCSS
}

func GetHomeCSS() string {
	return homeCSS
}
