package parser

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/gofaith/goctlr/api/spec"
)

type Parser struct {
	r  *bufio.Reader
	st string
}

func NewParser(filename string) (*Parser, error) {
	api, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return NewParserFromStr(string(api))
}

func NewParserFromStr(str string) (*Parser, error) {
	info, body, service, err := MatchStruct(str)
	if err != nil {
		return nil, err
	}
	var buffer = new(bytes.Buffer)
	buffer.WriteString(info)
	buffer.WriteString(service)
	return &Parser{
		r:  bufio.NewReader(buffer),
		st: body,
	}, nil
}

func (p *Parser) Parse() (*spec.ApiSpec, error) {
	api := new(spec.ApiSpec)
	types, err := parseStructAst(p.st)
	if err != nil {
		return nil, err
	}
	api.Types = types
	var lineNumber = 1
	st := newRootState(p.r, &lineNumber)
	for {
		st, err = st.process(api)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("near line: %d, %s", lineNumber, err.Error())
		}
		if st == nil {
			break
		}
	}

	err = p.validate(api)
	if err != nil {
		return api, err
	}

	for i, r := range api.Service.Routes {
		for _, a := range r.Annotations {
			if a.Name == "doc" {
				api.Service.Routes[i].Summary = a.Properties["summary"]
				break
			}
		}
	}

	for i, g := range api.Service.Groups {
		for _, a := range g.Annotations {
			if a.Name == "server" {
				api.Service.Groups[i].Desc = a.Properties["desc"]
				_, api.Service.Groups[i].Jwt = a.Properties["jwt"]
				break
			}
		}
		for j, r := range g.Routes {
			for _, a := range r.Annotations {
				if a.Name == "doc" {
					api.Service.Groups[i].Routes[j].Summary = a.Properties["summary"]
				}
			}
		}
	}
	return api, nil
}
