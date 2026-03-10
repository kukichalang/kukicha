package codegen

import (
	"strings"

	"github.com/duber000/kukicha/internal/semantic"
)

func (g *Generator) typeInfoToGoString(ti *semantic.TypeInfo) string {
	if ti == nil {
		return "any"
	}

	switch ti.Kind {
	case semantic.TypeKindInt:
		return "int"
	case semantic.TypeKindFloat:
		return "float64"
	case semantic.TypeKindString:
		return "string"
	case semantic.TypeKindBool:
		return "bool"
	case semantic.TypeKindList:
		return "[]" + g.typeInfoToGoString(ti.ElementType)
	case semantic.TypeKindMap:
		return "map[" + g.typeInfoToGoString(ti.KeyType) + "]" + g.typeInfoToGoString(ti.ValueType)
	case semantic.TypeKindChannel:
		return "chan " + g.typeInfoToGoString(ti.ElementType)
	case semantic.TypeKindReference:
		return "*" + g.typeInfoToGoString(ti.ElementType)
	case semantic.TypeKindNamed:
		// Rewrite package-qualified type names if the package was auto-aliased
		name := ti.Name
		if dotIdx := strings.Index(name, "."); dotIdx > 0 {
			pkgPart := name[:dotIdx]
			typePart := name[dotIdx:]
			if alias, ok := g.pkgAliases[pkgPart]; ok {
				return alias + typePart
			}
		}
		return name
	case semantic.TypeKindFunction:
		params := make([]string, len(ti.Params))
		for i, p := range ti.Params {
			params[i] = g.typeInfoToGoString(p)
		}
		returns := make([]string, len(ti.Returns))
		for i, r := range ti.Returns {
			returns[i] = g.typeInfoToGoString(r)
		}
		retStr := ""
		if len(returns) == 1 {
			retStr = " " + returns[0]
		} else if len(returns) > 1 {
			retStr = " (" + strings.Join(returns, ", ") + ")"
		}
		return "func(" + strings.Join(params, ", ") + ")" + retStr
	default:
		return "any"
	}
}
