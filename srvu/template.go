package srvu

import (
	"html/template"
	"io"
	"io/fs"
	"log"
	"sync"
)

type TemplateProvider func() *template.Template

func NewTemplateProvider(files fs.FS, watch bool) TemplateProvider {
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

func (t TemplateProvider) Lookup(name string) *template.Template {
	return t().Lookup(name)
}

func (t TemplateProvider) ExecuteTemplate(w io.Writer, name string, data interface{}) error {
	return t().ExecuteTemplate(w, name, data)
}
