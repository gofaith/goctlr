package util

import (
	"log"
	"reflect"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
)

var FuncsMap = template.FuncMap{
	"tagGet":          tagGet,
	"lowCamelCase":    strcase.ToLowerCamel,
	"routeToFuncName": routeToFuncName,
	"parseType":       parseType,
	"add":             add,
	"upperCase":       upperCase,
	"isDirectType":    isDirectType,
	"isClassListType": isClassListType,
	"getCoreType":     getCoreType,
}

func isDirectType(s string) bool {
	return isAtomicType(s) || isListType(s) && isAtomicType(getCoreType(s))
}

func isAtomicType(s string) bool {
	switch s {
	case "String", "int", "double", "bool":
		return true
	default:
		return false
	}
}

func isListType(s string) bool {
	return strings.HasPrefix(s, "List<")
}

func isClassListType(s string) bool {
	return strings.HasPrefix(s, "List<") && !isAtomicType(getCoreType(s))
}

func getCoreType(s string) string {
	if isAtomicType(s) {
		return s
	}
	if isListType(s) {
		s = strings.Replace(s, "List<", "", -1)
		return strings.Replace(s, ">", "", -1)
	}
	return s
}

func tagGet(tag, k string) (reflect.Value, error) {
	v, _ := TagLookup(tag, k)
	out := strings.Split(v, ",")[0]
	return reflect.ValueOf(out), nil
}
func routeToFuncName(method, path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	path = strings.ReplaceAll(path, "/", "_")
	path = strings.ReplaceAll(path, "-", "_")
	path = strings.ReplaceAll(path, ":", "With_")

	return strings.ToLower(method) + strcase.ToCamel(path)
}

func parseType(t string) string {
	t = strings.Replace(t, "*", "", -1)
	if strings.HasPrefix(t, "[]") {
		return "List<" + parseType(t[2:]) + ">"
	}

	if strings.HasPrefix(t, "map") {
		tys, e := DecomposeType(t)
		if e != nil {
			log.Fatal(e)
		}
		if len(tys) != 2 {
			log.Fatal("Map type number !=2")
		}
		return "Map<String," + parseType(tys[1]) + ">"
	}

	switch t {
	case "string":
		return "String"
	case "int", "int32", "int64":
		return "Int"
	case "float", "float32", "float64":
		return "Double"
	case "bool":
		return "Boolean"
	default:
		return t
	}
}

func add(a, i int) int {
	return a + i
}

func upperCase(s string) string {
	return strings.ToUpper(s)
}
