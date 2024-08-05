package template

import (
	"bytes"
	"embed"
	"path/filepath"
	"text/template"
)

const (
	templateFolder      = "templates"
	CephClusterTemplate = templateFolder + "/ceph-cluster"
	AddonTemplate       = templateFolder + "/addons"
)

//go:embed all:templates/*
var templates embed.FS

// Render renders a templates in the package `templates` folder. The templates
// files are embedded in build-time.
func Render(folder, template string, context interface{}) (*bytes.Buffer, error) {
	if folder == "" {
		folder = templateFolder
	}
	tplBytes, err := templates.ReadFile(filepath.Join(folder, template))
	if err != nil {
		return nil, err
	}
	return RenderTemplate(string(tplBytes), context)
}

func RenderTemplate(tpl string, context interface{}) (*bytes.Buffer, error) {
	result := bytes.NewBufferString("")
	tmpl, err := template.New("").Parse(tpl)
	if err != nil {
		return nil, err
	}

	err = tmpl.Execute(result, context)
	if err != nil {
		return nil, err
	}

	return result, nil
}
