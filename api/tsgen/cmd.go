package tsgen

import (
	"log"

	"github.com/gofaith/goctlr/api/parser"
	"github.com/urfave/cli"
)

func TsCommand(c *cli.Context) error {
	apiFile := c.String("api")
	dir := c.String("dir")

	p, e := parser.NewParser(apiFile)
	if e != nil {
		log.Println(e)
		return e
	}

	api, e := p.Parse()
	if e != nil {
		log.Println(e)
		return e
	}

	e = genBase(dir, api)
	if e != nil {
		log.Println(e)
		return e
	}

	e = genApi(dir, api)
	if e != nil {
		log.Println(e)
		return e
	}

	return nil
}
