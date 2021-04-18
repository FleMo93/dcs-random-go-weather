package randomdcsweather

import (
	"log"
	"strconv"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

func luaKeyString(key lua.LValue) string {
	_, err := strconv.Atoi(key.String())
	if err != nil {
		return "[\"" + key.String() + "\"]"
	} else {
		return "[" + key.String() + "]"
	}
}

func depthToString(depth int) string {
	return strings.Repeat("    ", depth)
}

func tableToString(table *lua.LTable, depth int) string {
	res := ""
	table.ForEach(func(a lua.LValue, b lua.LValue) {
		switch b.Type() {
		case lua.LTTable:
			tableStr := tableToString(b.(*lua.LTable), depth+1)
			res += depthToString(depth) + luaKeyString(a) + " = \n" +
				depthToString(depth) + "{\n" +
				tableStr +
				depthToString(depth) + "}, -- end of " + luaKeyString(a) + "\n"
			break
		case lua.LTString:
			res += depthToString(depth) + luaKeyString(a) + " = \"" + b.String() + "\",\n"
			break
		case lua.LTNumber:
			res += depthToString(depth) + luaKeyString(a) + " = " + b.String() + ",\n"
		case lua.LTBool:
			res += depthToString(depth) + luaKeyString(a) + " = " + b.String() + ",\n"
			break
		default:
			log.Panic("Unsupported type " + b.Type().String())
		}
	})

	return res
}

func luaTableToString(tableVarName string, table *lua.LTable) string {
	res := tableVarName + " = \n{\n"
	res += tableToString(table, 1)
	return res + "}"
}
