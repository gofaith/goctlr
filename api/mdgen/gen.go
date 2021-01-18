package mdgen

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofaith/goctlr/api/parser"
	"github.com/urfave/cli"
)

func MdCommand(c *cli.Context) error {
	dir := c.String("dir")
	if len(dir) == 0 {
		return errors.New("missing -dir")
	}

	files, err := filePathWalkDir(dir)
	if err != nil {
		return errors.New(fmt.Sprintf("dir %s not exist", dir))
	}

	for _, f := range files {
		p, err := parser.NewParser(f)
		if err != nil {
			return errors.New(fmt.Sprintf("parse file: %s, err: %s", f, err.Error()))
		}
		api, err := p.Parse()
		if err != nil {
			return err
		}
		genMd(api, f)
	}
	return nil
}

func filePathWalkDir(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(path, ".api") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}
