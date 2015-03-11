package main

import (
	"html/template"
	"io"
)

const tmplExt = ".gohtml"

var (
	tmplMap     = make(map[string]*template.Template)
	tmplFuncMap = map[string]interface{}{
		"slice": func(str string, start, end int) string {
			return str[start:end]
		},
	}
)

func InitTemplates() {
	allTemplates := template.New("")
	allTemplates = allTemplates.Funcs(tmplFuncMap)
	var err error
	allTemplates, err = allTemplates.ParseGlob("templates/*" + tmplExt)
	if err != nil {
		panic(err)
	}

	layout := allTemplates.Lookup("layout" + tmplExt)
	if layout == nil {
		panic("layout" + tmplExt + " not found")
	}
	for _, tmpl := range allTemplates.Templates() {
		if tmpl.Name() != ("layout" + tmplExt) {
			var err error
			newTmpl := template.New(tmpl.Name())
			newTmpl.Funcs(tmplFuncMap)
			newTmpl, err = newTmpl.AddParseTree(tmpl.Name(), layout.Tree)
			if err != nil {
				panic(err)
			}
			newTmpl, err = newTmpl.AddParseTree("body", tmpl.Tree)
			if err != nil {
				panic(err)
			}
			tmplMap[tmpl.Name()] = newTmpl
		}
	}
}

func RenderHtml(w io.Writer, tmpl string, data interface{}) {
	InitTemplates() // Debug mode
	renderTmpl := tmplMap[tmpl+tmplExt]
	if renderTmpl == nil {
		panic("Template " + tmpl + tmplExt + " not found")
	}
	err := renderTmpl.Execute(w, data)
	if err != nil {
		panic(err)
	}
}
