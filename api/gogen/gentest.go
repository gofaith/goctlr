package gogen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/gofaith/goctl/api/spec"
	"github.com/gofaith/goctl/api/util"
	"github.com/iancoleman/strcase"
)

const (
	testTemplate = `

package test

import (
	"{{.baseDir}}/client"
	"{{.baseDir}}/internal/types"
	"testing"
)

func Test_{{.funcName}}(t *testing.T) {
	cli := client.NewClient(){{if ne .requestType ""}}
	req := types.{{.requestType}}{}{{end}}
	{{if ne .responseType ""}}_, {{end}}e := cli.{{.apiFuncName}}({{if ne .requestType ""}}req{{end}})
	if e != nil {
		t.Error(e)
		return
	}
}
`
)

func genTest(dir string, api *spec.ApiSpec) error {
	dir, e := filepath.Abs(dir)
	if e != nil {
		return e
	}

	testDir := filepath.Join(dir, "internal", "test")
	e = os.MkdirAll(testDir, 0755)
	if e != nil {
		return e
	}

	baseDir := getImport(dir)
	// range routes
	for _, route := range api.Service.Routes {
		handler, ok := util.GetAnnotationValue(route.Annotations, "server", "handler")
		if !ok {
			return fmt.Errorf("missing handler annotation for %q", route.Path)
		}
		filename := strings.ToLower(getHandlerBaseName(handler)) + "_test.go"
		filePath := filepath.Join(testDir, filename)
		group, ok := util.GetAnnotationValue(route.Annotations, "server", "folder")
		if ok {
			e = os.MkdirAll(filepath.Join(testDir, group), 0755)
			if e != nil {
				return e
			}

			filePath = filepath.Join(testDir, group, filename)
		}

		_, e = os.Stat(filePath)
		if e == nil {
			continue
		}

		file, e := os.OpenFile(filePath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
		if e != nil {
			return e
		}
		defer file.Close()

		t, e := template.New(filename).Funcs(util.FuncsMap).Parse(testTemplate)
		if e != nil {
			return e
		}
		e = t.Execute(file, map[string]interface{}{
			"baseDir":      baseDir,
			"funcName":     getHandlerBaseName(handler),
			"requestType":  route.RequestType.Name,
			"apiFuncName":  strcase.ToCamel(util.RouteToFuncName(route.Method, route.Path)),
			"responseType": route.ResponseType.Name,
		})
		if e != nil {
			return e
		}

	}
	return nil
}
