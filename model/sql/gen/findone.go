package gen

import (
	"github.com/gofaith/goctlr/model/sql/template"
	"github.com/gofaith/goctlr/util"
	"github.com/gofaith/goctlr/util/stringx"
)

func genFindOne(table Table, withCache bool) (string, error) {
	camel := table.Name.ToCamel()
	output, err := util.With("findOne").
		Parse(template.FindOne).
		Execute(map[string]interface{}{
			"withCache":                 withCache,
			"upperStartCamelObject":     camel,
			"lowerStartCamelObject":     stringx.From(camel).UnTitle(),
			"originalPrimaryKey":        table.PrimaryKey.Name.Source(),
			"lowerStartCamelPrimaryKey": stringx.From(table.PrimaryKey.Name.ToCamel()).UnTitle(),
			"dataType":                  table.PrimaryKey.DataType,
			"cacheKey":                  table.CacheKey[table.PrimaryKey.Name.Source()].KeyExpression,
			"cacheKeyVariable":          table.CacheKey[table.PrimaryKey.Name.Source()].Variable,
		})
	if err != nil {
		return "", err
	}
	return output.String(), nil
}
