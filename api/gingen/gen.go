package gingen

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofaith/go-zero/core/logx"
	"github.com/gofaith/goctlr/api/parser"
	"github.com/gofaith/goctlr/util"
	"github.com/logrusorgru/aurora"
	"github.com/urfave/cli"
)

const tmpFile = "%s-%d"

var tmpDir = path.Join(os.TempDir(), "goctl")

func GoCommand(c *cli.Context) error {
	apiFile := c.String("api")
	dir := c.String("dir")
	onlyTypes := c.Bool("onlyTypes")
	if len(apiFile) == 0 {
		return errors.New("missing -api")
	}
	if len(dir) == 0 {
		return errors.New("missing -dir")
	}

	p, e := parser.NewParser(apiFile)
	if e != nil {
		log.Println(apiFile + ":" + e.Error())
		return e
	}
	api, e := p.Parse()
	if e != nil {
		log.Println(apiFile + ":" + e.Error())
		return e
	}

	if onlyTypes {
		logx.Must(genTypes(dir, api))
		return nil
	}
	logx.Must(util.MkdirIfNotExist(dir))
	logx.Must(genServiceContext(dir, api))
	logx.Must(genTypes(dir, api))
	logx.Must(genHandlers(dir, api))

	fmt.Println(aurora.Green("Done."))
	return nil
}

func sweep() error {
	keepTime := time.Now().AddDate(0, 0, -7)
	return filepath.Walk(tmpDir, func(fpath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		pos := strings.LastIndexByte(info.Name(), '-')
		if pos > 0 {
			timestamp := info.Name()[pos+1:]
			seconds, err := strconv.ParseInt(timestamp, 10, 64)
			if err != nil {
				// print error and ignore
				log.Println(aurora.Red(fmt.Sprintf("sweep ignored file: %s", fpath)))
				return nil
			}

			tm := time.Unix(seconds, 0)
			if tm.Before(keepTime) {
				if err := os.Remove(fpath); err != nil {
					log.Println(aurora.Red(fmt.Sprintf("failed to remove file: %s", fpath)))
					return err
				}
			}
		}

		return nil
	})
}
