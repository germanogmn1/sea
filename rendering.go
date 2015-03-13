package main

import (
	"html/template"
	"io"
	"strings"
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
			keyName := strings.TrimSuffix(tmpl.Name(), tmplExt)
			newTmpl := template.New(keyName)
			newTmpl.Funcs(tmplFuncMap)
			newTmpl, err = newTmpl.AddParseTree("root", layout.Tree)
			if err != nil {
				panic(err)
			}
			newTmpl, err = newTmpl.AddParseTree("body", tmpl.Tree)
			if err != nil {
				panic(err)
			}
			tmplMap[keyName] = newTmpl
		}
	}
}

func RenderHtml(w io.Writer, keyName string, data interface{}) {
	InitTemplates() // Debug mode
	renderTmpl := tmplMap[keyName]
	if renderTmpl == nil {
		panic("Template " + keyName + " not found")
	}
	err := renderTmpl.ExecuteTemplate(w, "root", data)
	if err != nil {
		panic(err)
	}
}
