package main

import (
	"html/template"
	"io"
	"log"
)

const tmplExt = ".gohtml"

var (
	tmplMap     = make(map[string]*template.Template)
	tmplFuncMap = map[string]interface{}{
		"slice": func(str string, start, end int) string {
			return str[start:end]
		},
		"stateName": func(buildState int) string {
			return stateNames[buildState]
		},
	}
)

func InitTemplates() {
	allTemplates := template.New("")
	allTemplates = allTemplates.Funcs(tmplFuncMap)
	var err error
	allTemplates, err = allTemplates.ParseGlob("templates/*" + tmplExt)
	if err != nil {
		log.Fatal(err)
	}

	layout := allTemplates.Lookup("layout" + tmplExt)
	if layout == nil {
		log.Fatal("layout" + tmplExt + " not found")
	}
	for _, tmpl := range allTemplates.Templates() {
		if tmpl.Name() != ("layout" + tmplExt) {
			var err error
			newTmpl := template.New(tmpl.Name())
			newTmpl.Funcs(tmplFuncMap)
			newTmpl, err = newTmpl.AddParseTree(tmpl.Name(), layout.Tree)
			if err != nil {
				log.Fatal(err)
			}
			newTmpl, err = newTmpl.AddParseTree("body", tmpl.Tree)
			if err != nil {
				log.Fatal(err)
			}
			tmplMap[tmpl.Name()] = newTmpl
		}
	}
}

func RenderHtml(w io.Writer, tmpl string, data interface{}) {
	InitTemplates() // Debug mode
	renderTmpl := tmplMap[tmpl+tmplExt]
	if renderTmpl == nil {
		log.Fatal("Template " + tmpl + tmplExt + " not found")
	}
	err := renderTmpl.ExecuteTemplate(w, tmpl+tmplExt, data)
	if err != nil {
		log.Fatal(err)
	}
}
