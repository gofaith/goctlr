package gingen

import (
	"bytes"
	"fmt"
	goformat "go/format"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gofaith/go-zero/core/collection"
	"github.com/gofaith/goctlr/api/spec"
	"github.com/gofaith/goctlr/api/util"
	goctlutil "github.com/gofaith/goctlr/util"
)

func ReplaceLine(file string, prefix string, replacement string) error {
	b, e := ioutil.ReadFile(file)
	if e != nil {
		log.Println(e)
		return e
	}
	out := new(bytes.Buffer)

	for _, s := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(s, prefix) {
			if s == replacement {
				return nil
			}
			s = replacement
		}
		out.WriteString(s + "\n")
	}

	fo, e := os.OpenFile(file, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if e != nil {
		log.Println(e)
		return e
	}
	defer fo.Close()
	_, e = fo.Write(out.Bytes())
	if e != nil {
		log.Println(e)
		return e
	}
	return nil
}

func getParentPackage(dir string) (string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	absDir = strings.ReplaceAll(absDir, `\`, `/`)
	rootPath, _ := goctlutil.FindGoModPath(dir)

	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		home, e := os.UserHomeDir()
		if e == nil {
			gopath = filepath.Join(home, "go")
		}
	}
	parent := path.Join(gopath, "src")
	pos := strings.Index(absDir, parent)
	if pos < 0 {
		fmt.Printf("%s not in go.mod project path, or not in GOPATH of %s directory\n", absDir, gopath)
		tempPath := filepath.Dir(absDir)
		rootPath = absDir[len(tempPath)+1:]
	} else {
		rootPath = absDir[len(parent)+1:]
	}

	return rootPath, nil
}

func writeIndent(writer io.Writer, indent int) {
	for i := 0; i < indent; i++ {
		fmt.Fprint(writer, "\t")
	}
}

func writeProperty(writer io.Writer, name, tp, tag, comment string, indent int) error {
	writeIndent(writer, indent)
	var err error
	if len(comment) > 0 {
		comment = strings.TrimPrefix(comment, "//")
		comment = "//" + comment
		_, err = fmt.Fprintf(writer, "%s %s %s %s\n", strings.Title(name), tp, tag, comment)
	} else {
		_, err = fmt.Fprintf(writer, "%s %s %s\n", strings.Title(name), tp, tag)
	}
	return err
}

func getAuths(api *spec.ApiSpec) []string {
	authNames := collection.NewSet()
	for _, g := range api.Service.Groups {
		if value, ok := util.GetAnnotationValue(g.Annotations, "server", "jwt"); ok {
			authNames.Add(value)
		}
		if value, ok := util.GetAnnotationValue(g.Annotations, "server", "signature"); ok {
			authNames.Add(value)
		}
	}
	return authNames.KeysStr()
}

func formatCode(code string) string {
	ret, err := goformat.Source([]byte(code))
	if err != nil {
		return code
	}

	return string(ret)
}
