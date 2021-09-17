package gocligen

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/gofaith/goctlr/api/parser"
	"github.com/gofaith/goctlr/api/spec"
	"github.com/gofaith/goctlr/api/util"
	"github.com/urfave/cli"
)

const (
	apiTemplate = `package api

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ErrorCode struct {
	Code int    ` + "`" + `json:"code"` + "`" + `
	Desc string ` + "`" + `json:"desc"` + "`" + `
}

const (
	server = "http://localhost:8888"
)

var (
	client = http.Client{
		Timeout: time.Second * 5,
	}
)

func apiRequest(method, uri string, req interface{}, onOk func(string), onFail func(ErrorCode), ev func()) {
	if ev != nil {
		defer ev()
	}

	var bodyReader io.Reader
	var isGzipped bool
	if req != nil {
		b, e := json.Marshal(req)
		if e != nil {
			log.Println(e)
			if onFail != nil {
				onFail(ErrorCode{Desc: e.Error()})
				return
			}
		}

		if len(b) > 1024 {
			isGzipped = true
			buf := new(bytes.Buffer)
			zw := gzip.NewWriter(buf)
			_, e = zw.Write(b)
			if e != nil {
				log.Println(e)
				if onFail != nil {
					onFail(ErrorCode{Desc: e.Error()})
					return
				}
			}
			zw.Close()
			bodyReader = buf
		} else {
			bodyReader = bytes.NewReader(b)
		}
	}
	r, e := http.NewRequest(method, server+uri, bodyReader)
	if e != nil {
		log.Println(e)
		if onFail != nil {
			onFail(ErrorCode{Desc: e.Error()})
			return
		}
	}
	r.Header.Set("Content-Type", "application/json")
	if isGzipped {
		r.Header.Set("Content-Encoding", "gzip")
	}

	//response
	res, e := client.Do(r)
	if e != nil {
		log.Println(e)
		if onFail != nil {
			onFail(ErrorCode{Desc: e.Error()})
			return
		}
	}
	defer res.Body.Close()

	var rp string
	if res.Header.Get("Content-Encoding") == "gzip" {
		zr, e := gzip.NewReader(res.Body)
		if e != nil {
			log.Println(e)
			if onFail != nil {
				onFail(ErrorCode{Desc: e.Error()})
				return
			}
		}
		defer zr.Close()
		buf := new(bytes.Buffer)
		_, e = io.Copy(buf, zr)
		if e != nil {
			log.Println(e)
			if onFail != nil {
				onFail(ErrorCode{Desc: e.Error()})
				return
			}
		}
		rp = buf.String()
	} else {
		b, e := io.ReadAll(res.Body)
		if e != nil {
			log.Println(e)
			if onFail != nil {
				onFail(ErrorCode{Desc: e.Error()})
				return
			}
		}
		rp = string(b)
	}

	switch res.StatusCode {
	case 200:
		if onOk != nil {
			onOk(rp)
		}
	case 400:
		var err ErrorCode
		if strings.HasPrefix(rp, "{") {
			e = json.Unmarshal([]byte(rp), &err)
			if e != nil {
				log.Println(e)
				if onFail != nil {
					onFail(ErrorCode{Desc: e.Error()})
					return
				}
			}
		} else {
			err.Desc = strconv.Itoa(res.StatusCode) + ":" + rp
		}
		if onFail != nil {
			onFail(err)
		}
	default:
		if onFail != nil {
			onFail(ErrorCode{Desc: strconv.Itoa(res.StatusCode) + ":" + rp})
		}
	}
}
`
	apiFilesTemplate = `package {{.Info.Desc}}
	
import (
	"encoding/json"
	"log"
)

type (
	{{range .Types}}
	{{if eq 0 (len .Members)}}{{.Name}} struct{} {{else}}{{.Name}} struct{ {{range .Members}}
		{{.Name}}	{{.Type}}	`+"`"+`json:"{{tagGet .Tag "json"}}"`+"`"+` {{end}}
	}{{end}}{{end}}
)
{{with .Service}}
{{range .Routes}}func {{camelCase (routeToFuncName .Method .Path)}}({{if ne .RequestType.Name ""}}req {{.RequestType.Name}},{{end}}onOk func({{if ne .ResponseType.Name ""}}res {{.ResponseType.Name}}{{end}}), onFail func(err ErrorCode), eventually func()) {
	apiRequest("{{upperCase .Method}}", "{{.Path}}", {{if ne .RequestType.Name ""}}req{{else}}nil{{end}}, func(res string){
		if onOk != nil{ {{if ne .ResponseType.Name ""}}
			rp := {{.ResponseType.Name}}{}
			e := json.Unmarshal([]byte(res), &rp)
			if e != nil {
				log.Println(e)
				if onFail != nil {
					onFail(ErrorCode{Desc: e.Error()})
					return
				}
			}
			onOk(rp){{else}}
			onOk(){{end}}
		}
	}, onFail, eventually)
}
{{end}}{{end}}
`
)

func GocliCommand(c *cli.Context) error {
	apiFile := c.String("api")
	if apiFile == "" {
		return errors.New("missing -api")
	}
	dir := c.String("dir")
	if dir == "" {
		return errors.New("missing -dir")
	}
	pkg := filepath.Base(dir)

	p, e := parser.NewParser(apiFile)
	if e != nil {
		return e
	}
	api, e := p.Parse()
	if e != nil {
		return e
	}
	e = genApi(dir, pkg, api)
	if e != nil {
		return e
	}
	e = genApiFiles(dir, pkg, api)
	if e != nil {
		return e
	}
	return nil
}

func genApi(dir, pkg string, api *spec.ApiSpec) error {
	e := os.MkdirAll(dir, 0755)
	if e != nil {
		return e
	}
	path := filepath.Join(dir, "api.go")
	if _, e := os.Stat(path); e == nil {
		return nil
	}

	file, e := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if e != nil {
		return e
	}
	defer file.Close()
	api.Info.Desc = pkg
	t, e := template.New("api.go").Parse(apiTemplate)
	if e != nil {
		return e
	}
	return t.Execute(file, pkg)
}

func genApiFiles(dir, pkg string, api *spec.ApiSpec) error {
	name := strings.ToLower(api.Info.Title + "api")
	path := filepath.Join(dir, name+".go")
	api.Info.Title = name
	api.Info.Desc = pkg
	e := os.MkdirAll(dir, 0755)
	if e != nil {
		return e
	}

	file, e := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if e != nil {
		return e
	}
	defer file.Close()

	t, e := template.New(name).Funcs(util.FuncsMap).Parse(apiFilesTemplate)
	if e != nil {
		return e
	}
	return t.Execute(file, api)
}
