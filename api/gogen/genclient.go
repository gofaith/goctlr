package gogen

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/StevenZack/tools/strToolkit"
	"github.com/gofaith/goctl/api/spec"
	"github.com/gofaith/goctl/api/util"
)

const (
	clientTemplate = `package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gofaith/go-zero/core/logx"
)

type Client struct {
	Server string
	Header map[string]string
}

var (
	server = "http://localhost:8080"
)

func NewClient() *Client {
	return &Client{
		Server: server,
		Header: make(map[string]string),
	}
}

func (c *Client) request(method, path string, body interface{}) ([]byte, error) {
	cli := http.Client{
		Timeout: time.Second,
	}

	var reader io.Reader
	if body != nil {
		b, e := json.Marshal(body)
		if e != nil {
			logx.Error(e)
			return nil, e
		}
		reader = bytes.NewReader(b)
	}
	req, e := http.NewRequest(method, c.Server+path, reader)
	if e != nil {
		logx.Error(e)
		return nil, e
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range c.Header {
		req.Header.Set(k, v)
	}

	res, e := cli.Do(req)
	if e != nil {
		logx.Error(e)
		return nil, e
	}
	defer res.Body.Close()
	b, e := ioutil.ReadAll(res.Body)
	if e != nil {
		logx.Error(e)
		return nil, e
	}

	if res.StatusCode == 200 {
		return b, nil
	}

	return nil, errors.New(string(b))
}
`
	apiTemplate = `package client

import (
	"encoding/json"

	"{{.Info.Desc}}/internal/types"

	"github.com/gofaith/go-zero/core/logx"
)
{{with .Service}}{{range .Routes}}
func (c *Client) {{camelCase (routeToFuncName .Method .Path)}}({{with .RequestType}}{{if ne .Name ""}}request types.{{.Name}}{{end}}{{end}}) {{with .ResponseType}}{{if ne .Name ""}}(*types.{{.Name}}, error){{else}}error{{end}}{{end}} {
	{{with .ResponseType}}{{if ne .Name ""}}res{{else}}_{{end}}{{end}}, e := c.request("{{upperCase .Method}}", "{{.Path}}", {{with .RequestType}}{{if ne .Name ""}}request{{else}}nil{{end}}{{end}})
	if e != nil {
		logx.Error(e)
		return {{with .ResponseType}}{{if ne .Name ""}}nil, {{end}}{{end}}e
	}{{with .ResponseType}}{{if ne .Name ""}}
	v := types.{{.Name}}{}
	e = json.Unmarshal(res, &v)
	if e != nil {
		logx.Error(e)
		return nil, e
	}
	return &v, nil{{else}}
	return nil{{end}}{{end}}
}{{end}}{{end}}
`
)

func genClient(dir string, api *spec.ApiSpec) error {
	dir, e := filepath.Abs(dir)
	if e != nil {
		return e
	}

	clientDir := filepath.Join(dir, "client")
	e = os.MkdirAll(clientDir, 0755)
	if e != nil {
		return e
	}

	clientFile := filepath.Join(clientDir, "client.go")
	_, e = os.Stat(clientFile)
	if os.IsNotExist(e) {
		// gen client.go
		file, e := os.OpenFile(clientFile, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
		if e != nil {
			return e
		}
		defer file.Close()
		_, e = file.WriteString(clientTemplate)
		if e != nil {
			return e
		}
	} else if e == nil {
		fmt.Println("client.go exists. skipped it.")
	}

	// gen api.go
	apiFile, e := os.OpenFile(filepath.Join(clientDir, "api.go"), os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if e != nil {
		return e
	}
	defer apiFile.Close()

	api.Info.Desc = getImport(dir)
	t, e := template.New("api.go").Funcs(util.FuncsMap).Parse(apiTemplate)
	if e != nil {
		return e
	}
	e = t.Execute(apiFile, api)
	if e != nil {
		return e
	}

	return nil
}

func getImport(dir string) string {
	return strToolkit.TrimStart(strToolkit.SubAfter(dir, filepath.Join(os.Getenv("GOPATH"), "src"), dir), "/")
}
