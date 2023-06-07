package gingen

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/gofaith/goctlr/api/spec"
	"github.com/gofaith/goctlr/api/util"
)

const (
	contextFilename = "servicecontext.go"
	contextTemplate = `package svc

type ServiceContext struct {
}

var Current = newInstance()

func newInstance() *ServiceContext {
	return &ServiceContext{}
}
`
)

func genServiceContext(dir string, api *spec.ApiSpec) error {
	fp, created, err := util.MaybeCreateFile(dir, contextDir, contextFilename)
	if err != nil {
		return err
	}
	if !created {
		return nil
	}
	defer fp.Close()

	var authNames = getAuths(api)
	var auths []string
	for _, item := range authNames {
		auths = append(auths, fmt.Sprintf("%s config.AuthConfig", item))
	}

	t := template.Must(template.New("contextTemplate").Parse(contextTemplate))
	buffer := new(bytes.Buffer)
	err = t.Execute(buffer, map[string]string{})
	if err != nil {
		return nil
	}
	formatCode := formatCode(buffer.String())
	_, err = fp.WriteString(formatCode)
	return err
}
