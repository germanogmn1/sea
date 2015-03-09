package main

import (
	"html/template"
	"io"
	"log"
)

var (
	allTemplates *template.Template
	templateMap  = make(map[string]*template.Template)
)

func InitTemplates() {
	var err error
	allTemplates, err = template.ParseGlob("templates/*.gohtml")
	if err != nil {
		log.Fatal(err)
	}

	layout := allTemplates.Lookup("layout.gohtml")
	if layout == nil {
		log.Fatal("layout.gohtml not found")
	}
	for _, tmpl := range allTemplates.Templates() {
		if tmpl.Name() != "layout.gohtml" {
			var err error
			newTmpl := template.New(tmpl.Name())
			newTmpl, err = newTmpl.AddParseTree(tmpl.Name(), layout.Tree)
			if err != nil {
				log.Fatal(err)
			}
			newTmpl, err = newTmpl.AddParseTree("body", tmpl.Tree)
			if err != nil {
				log.Fatal(err)
			}
			templateMap[tmpl.Name()] = newTmpl
		}
	}
}

func RenderHtml(w io.Writer, tmpl string, data interface{}) {
	renderTmpl := templateMap[tmpl+".gohtml"]
	if renderTmpl == nil {
		log.Fatal("Template " + tmpl + ".gohtml not found")
	}
	err := renderTmpl.ExecuteTemplate(w, tmpl+".gohtml", data)
	if err != nil {
		log.Fatal(err)
	}
}
