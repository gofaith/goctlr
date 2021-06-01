package gogen

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofaith/go-zero/core/logx"
	"github.com/gofaith/goctlr/api/parser"
	"github.com/gofaith/goctlr/api/spec"
	"github.com/gofaith/goctlr/util"
	"github.com/logrusorgru/aurora"
	"github.com/urfave/cli"
)

const tmpFile = "%s-%d"

var tmpDir = path.Join(os.TempDir(), "goctl")

func GoCommand(c *cli.Context) error {
	apiFile := c.String("api")
	dir := c.String("dir")
	proto := c.String("proto")
	if len(apiFile) == 0 {
		return errors.New("missing -api")
	}
	if len(dir) == 0 {
		return errors.New("missing -dir")
	}
	if len(proto) > 0 {
		logx.Must(genProto(dir, proto))
	}

	info, e := os.Stat(apiFile)
	if e != nil {
		log.Println(e)
		return e
	}

	if info.IsDir() {

		//check
		typeMap := make(map[string]string)
		routeMap := make(map[string]string)
		apiList := []*spec.ApiSpec{}
		e = filepath.Walk(apiFile, func(path string, info fs.FileInfo, err error) error {
			if info.IsDir() || !strings.HasSuffix(path, ".api") {
				return nil
			}

			p, e := parser.NewParser(path)
			if e != nil {
				log.Println(path + ":" + e.Error())
				return e
			}
			api, e := p.Parse()
			if e != nil {
				log.Println(path + ":" + e.Error())
				return e
			}
			//type check
			for _, typ := range api.Types {
				if before, ok := typeMap[typ.Name]; ok {
					return errors.New(path + ": type name duplicated \"" + typ.Name + "\", between file \"" + filepath.Base(path) + "\" and \"" + filepath.Base(before) + "\"")
				}
				typeMap[typ.Name] = path
			}
			//route check
			for _, route := range api.Service.Routes {
				if before, ok := routeMap[route.Path]; ok {
					return errors.New(path + ": route path duplicated \"" + route.Path + "\", between file \"" + filepath.Base(path) + "\" and \"" + filepath.Base(before) + "\"")
				}
				routeMap[route.Path] = path
			}

			apiList = append(apiList, api)
			return nil
		})
		if e != nil {
			log.Println(e)
			return e
		}

		//generate
		for _, api := range apiList {
			logx.Must(util.MkdirIfNotExist(dir))
			logx.Must(genEtc(dir, api))
			logx.Must(genConfig(dir))
			logx.Must(genServiceContext(dir, api))
			if len(proto) == 0 {
				logx.Must(genTypes(dir, api))
			}
			logx.Must(genHandlers(dir, proto, api))
			logx.Must(genRoutes(dir, api))
			logx.Must(genLogic(dir, proto, api))
			if c.Bool("clitest") {
				logx.Must(genClient(dir, api))
				logx.Must(genTest(dir, api))
			}
			api.Service.Name = "application"
			logx.Must(genMain(dir, api))
		}
	} else {
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

		logx.Must(util.MkdirIfNotExist(dir))
		logx.Must(genEtc(dir, api))
		logx.Must(genConfig(dir))
		logx.Must(genMain(dir, api))
		logx.Must(genServiceContext(dir, api))
		if len(proto) == 0 {
			logx.Must(genTypes(dir, api))
		}
		logx.Must(genHandlers(dir, proto, api))
		logx.Must(genRoutes(dir, api))
		logx.Must(genLogic(dir, proto, api))
		if c.Bool("clitest") {
			logx.Must(genClient(dir, api))
			logx.Must(genTest(dir, api))
		}
	}

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
