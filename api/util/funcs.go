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
	"toKtType":        toKtType,
	"toDartType":      toDartType,
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
	case "string", "bool", "uint8", "uint16", "uint32", "uint", "uint64", "int8", "int16", "int32", "int", "int64", "float32", "float64":
		return true
	default:
		return false
	}
}

func isListType(s string) bool {
	return strings.Contains(s, "[]")
}

func isClassListType(s string) bool {
	return isListType(s) && !isAtomicType(getCoreType(s))
}

func getCoreType(s string) string {
	if isAtomicType(s) {
		return s
	}
	if isListType(s) {
		return s[len("[]"):]
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

func toDartType(t string) string {
	t = strings.Replace(t, "*", "", -1)
	if strings.HasPrefix(t, "[]") {
		return "List<" + toDartType(t[2:]) + ">"
	}

	if strings.HasPrefix(t, "map") {
		tys, e := DecomposeType(t)
		if e != nil {
			log.Fatal(e)
		}
		if len(tys) != 2 {
			log.Fatal("Map type number !=2")
		}
		return "Map<" + toDartType(tys[0]) + "," + toDartType(tys[1]) + ">"
	}

	switch t {
	case "string":
		return "String"
	case "int", "int32", "int64":
		return "int"
	case "float32", "float64":
		return "double"
	case "bool":
		return "bool"
	default:
		return t
	}
}

func toKtType(t string) string {
	t = strings.Replace(t, "*", "", -1)
	if strings.HasPrefix(t, "[]") {
		return "MutableList<" + toKtType(t[2:]) + ">"
	}

	if strings.HasPrefix(t, "map") {
		tys, e := DecomposeType(t)
		if e != nil {
			log.Fatal(e)
		}
		if len(tys) != 2 {
			log.Fatal("Map type number !=2")
		}
		return "MutableMap<String," + toKtType(tys[1]) + ">"
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
