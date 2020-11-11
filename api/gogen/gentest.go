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
	testTemplate = `package test

import (
	"{{.baseDir}}/client"
	{{if ne .requestType ""}}"{{.baseDir}}/internal/types"
	{{end}}"testing"
)

func {{camelCase .funcName}}(t *testing.T) {
	cli := client.NewClient()
	if !cli.Ping() {
		return
	}{{if ne .requestType ""}}
	
	req := types.{{.requestType}}{}{{end}}
	{{if ne .responseType ""}}_, {{end}}e := cli.{{.apiFuncName}}({{if ne .requestType ""}}req{{end}})
	if e != nil {
		t.Error(e)
		return
	}
	// TODO test code
}
`
	test_testTemplate = `package test

import (
	"testing"
)
{{range .}}
func Test_{{.funcName}}(t *testing.T) {
	{{camelCase .funcName}}(t)
}{{end}}
`
)

func genTest(dir string, api *spec.ApiSpec) error {
	dir, e := filepath.Abs(dir)
	if e != nil {
		return e
	}

	testDir := filepath.Join(dir, "test")
	e = os.MkdirAll(testDir, 0755)
	if e != nil {
		return e
	}

	baseDir := getImport(dir)

	testFiles := make(map[string][]map[string]interface{})
	// range routes
	for _, route := range api.Service.Routes {
		handler, ok := util.GetAnnotationValue(route.Annotations, "server", "handler")
		if !ok {
			return fmt.Errorf("missing handler annotation for %q", route.Path)
		}
		group, ok := util.GetAnnotationValue(route.Annotations, "server", "folder")

		// basic file
		filename := strings.ToLower(getHandlerBaseName(handler)) + ".go"
		filePath := filepath.Join(testDir, filename)
		if ok {
			os.MkdirAll(filepath.Join(testDir, group), 0755)
			filePath = filepath.Join(testDir, group, filename)
		}

		_, e = os.Stat(filePath)
		if os.IsNotExist(e) {
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

		// test file
		testFiles[group] = append(testFiles[group], map[string]interface{}{
			"funcName": getHandlerBaseName(handler),
		})
	}

	for group, vs := range testFiles {
		file, e := os.OpenFile(filepath.Join(testDir, group, group+"_test.go"), os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
		if e != nil {
			return e
		}
		defer file.Close()
		t, e := template.New(group + "_test.go").Funcs(util.FuncsMap).Parse(test_testTemplate)
		if e != nil {
			return e
		}
		e = t.Execute(file, vs)
		if e != nil {
			return e
		}
	}
	return nil
}
