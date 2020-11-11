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
}
`
	test_testTemplate = `package test

import (
	"testing"
)

func Test_{{.funcName}}(t *testing.T) {
	{{camelCase .funcName}}(t)
}
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
	// range routes
	for _, route := range api.Service.Routes {
		handler, ok := util.GetAnnotationValue(route.Annotations, "server", "handler")
		if !ok {
			return fmt.Errorf("missing handler annotation for %q", route.Path)
		}
		group, _ := util.GetAnnotationValue(route.Annotations, "server", "folder")

		// basic file
		filename := strings.ToLower(group+getHandlerBaseName(handler)) + ".go"
		filePath := filepath.Join(testDir, filename)

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
				"funcName":     group + strcase.ToCamel(getHandlerBaseName(handler)),
				"requestType":  route.RequestType.Name,
				"apiFuncName":  strcase.ToCamel(util.RouteToFuncName(route.Method, route.Path)),
				"responseType": route.ResponseType.Name,
			})
			if e != nil {
				return e
			}
		}

		// test file
		testFilename := strings.ToLower(group+getHandlerBaseName(handler)) + "_test.go"
		testFilePath := filepath.Join(testDir, testFilename)
		_, e = os.Stat(testFilePath)
		if os.IsNotExist(e) {
			file, e := os.OpenFile(testFilePath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
			if e != nil {
				return e
			}
			defer file.Close()

			t, e := template.New(testFilename).Funcs(util.FuncsMap).Parse(test_testTemplate)
			if e != nil {
				return e
			}
			e = t.Execute(file, map[string]interface{}{
				"baseDir":      baseDir,
				"funcName":     group + strcase.ToCamel(getHandlerBaseName(handler)),
				"requestType":  route.RequestType.Name,
				"apiFuncName":  strcase.ToCamel(util.RouteToFuncName(route.Method, route.Path)),
				"responseType": route.ResponseType.Name,
			})
			if e != nil {
				return e
			}
		}
	}
	return nil
}
