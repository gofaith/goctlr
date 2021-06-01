package feature

import (
	"log"

	"github.com/logrusorgru/aurora"
	"github.com/urfave/cli"
)

var feature = `
1、增加goctl model支持
`

func Feature(_ *cli.Context) error {
	log.Println(aurora.Blue("\nFEATURE:"))
	log.Println(aurora.Blue(feature))
	return nil
}
