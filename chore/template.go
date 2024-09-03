package chore

import (
	"html/template"
	"io/fs"
	"log"
	"sync"
)

func newTemplateProvider(files fs.FS, watch bool) TemplateProvider {
	getFn := func() *template.Template {
		tmpl, err := template.ParseFS(files, "**/*.gohtml")
		if err != nil {
			log.Fatal(err)
		}
		return tmpl
	}
	if watch {
		return getFn
	}
	return sync.OnceValue(getFn)
}

type TemplateProvider func() *template.Template

func (t TemplateProvider) Lookup(name string) *template.Template {
	return t().Lookup(name)
}
