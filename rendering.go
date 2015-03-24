package main

import (
	"errors"
	"html/template"
	"net/http"
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

func InitTemplates() error {
	allTemplates := template.New("").Funcs(tmplFuncMap)
	var err error
	allTemplates, err = allTemplates.ParseGlob("templates/*" + tmplExt)
	if err != nil {
		return err
	}

	layout := allTemplates.Lookup("layout" + tmplExt)
	if layout == nil {
		return errors.New("layout" + tmplExt + " not found")
	}
	for _, tmpl := range allTemplates.Templates() {
		if tmpl.Name() != ("layout" + tmplExt) {
			keyName := strings.TrimSuffix(tmpl.Name(), tmplExt)
			newTmpl := template.New(keyName)
			newTmpl.Funcs(tmplFuncMap)
			newTmpl, err = newTmpl.AddParseTree("root", layout.Tree)
			if err != nil {
				return err
			}
			newTmpl, err = newTmpl.AddParseTree("body", tmpl.Tree)
			if err != nil {
				return err
			}
			tmplMap[keyName] = newTmpl
		}
	}
	return nil
}

func RenderHtml(w http.ResponseWriter, keyName string, data interface{}) {
	// TODO: run this in debug mode only
	if err := InitTemplates(); err != nil {
		panic(err)
	}
	renderTmpl, ok := tmplMap[keyName]
	if !ok {
		panic("Template " + keyName + " not found")
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := renderTmpl.ExecuteTemplate(w, "root", data); err != nil {
		panic(err)
	}
}
