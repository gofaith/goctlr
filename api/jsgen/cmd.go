package jsgen

import (
	"errors"
	"io/ioutil"

	"github.com/gofaith/goctl/api/parser"

	"github.com/urfave/cli"
)

func JsCommand(c *cli.Context) error {
	apiFile := c.String("api")
	if apiFile == "" {
		return errors.New("missing -api")
	}
	dir := c.String("dir")
	if dir == "" {
		return errors.New("missing -dir")
	}

	b, e := ioutil.ReadFile(apiFile)
	if e != nil {
		return e
	}
	return JsGen(string(b), dir)
}

func JsGen(apiStr, dir string) error {
	p, e := parser.NewParserFromStr(apiStr)
	if e != nil {
		return e
	}

	api, e := p.Parse()
	if e != nil {
		return e
	}

	e = genBase(dir, api)
	if e != nil {
		return e
	}
	e = genApi(dir, api)
	if e != nil {
		return e
	}
	return nil
}
