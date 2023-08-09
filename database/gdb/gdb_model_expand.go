package gdb

import (
	"fmt"
	"github.com/gogf/gf/text/gstr"
)

func (m *Model) Expands(param ...string) *Model {
	model := m.getModel()
	var array []string
	if gstr.Contains(model.tables, "AS") {
		array = gstr.SplitAndTrim(model.tables, "AS")
	} else if gstr.Contains(model.tables, " ") {
		array = gstr.SplitAndTrim(model.tables, " ")
	}
	if len(array) < 2 {
		panic(fmt.Sprintf(`The extended attribute main table %s must have an alias set`, m.tables))
	}
	table := array[0]
	charLeft, charRight := model.db.GetChars()
	table = gstr.Trim(table, charLeft+charRight)
	alias := array[1]
	if len(param) == 1 {
		var array1 []string
		if gstr.Contains(param[0], "AS") {
			array1 = gstr.SplitAndTrim(param[0], "AS")
		} else if gstr.Contains(param[0], " ") {
			array1 = gstr.SplitAndTrim(param[0], " ")
		} else {
			array1 = append(array1, fmt.Sprintf("%s_extend ", table))
			array1 = append(array1, param[0])
		}
		model.expandsTable = array1[0]
		model.expands = array1[1]
	} else if len(param) > 1 {
		model.expandsTable = param[0]
		model.expands = param[1]
	} else {
		if len(model.expandsTable) == 0 {
			model.expandsTable = fmt.Sprintf("%s_extend ", table)
		}
		model.expands = "ext"
	}

	if model.fields == "*" {
		model.fields = fmt.Sprintf("%s.%s", alias, m.fields)
	}
	model = model.LeftJoin(model.expandsTable, model.expands, fmt.Sprintf("%s.id = %s.row_key", alias, model.expands))
	model = model.Group(fmt.Sprintf("%s.id", alias))
	return model
}
