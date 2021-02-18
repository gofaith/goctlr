package gogen

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func genProto(dir, proto string) error {
	_, e := os.Stat(proto)
	if e != nil {
		log.Println(e)
		return e
	}

	dst := filepath.Join(dir, "internal", "pb")
	e = os.MkdirAll(dst, 0755)
	if e != nil {
		log.Println(e)
		return e
	}

	e = exec.Command("protoc", "--go_out="+dst, proto).Run()
	if e != nil {
		log.Println(e)
		return e
	}

	return nil
}
