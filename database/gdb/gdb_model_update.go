// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

package gdb

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gogf/gf/errors/gcode"
	"reflect"
	"strings"

	"github.com/gogf/gf/errors/gerror"
	"github.com/gogf/gf/os/gtime"
	"github.com/gogf/gf/text/gstr"
	"github.com/gogf/gf/util/gconv"
)

// Update does "UPDATE ... " statement for the model.
//
// If the optional parameter `dataAndWhere` is given, the dataAndWhere[0] is the updated data field,
// and dataAndWhere[1:] is treated as where condition fields.
// Also see Model.Data and Model.Where functions.
func (m *Model) Update(dataAndWhere ...interface{}) (result sql.Result, err error) {
	if len(dataAndWhere) > 0 {
		if len(dataAndWhere) > 2 {
			return m.Data(dataAndWhere[0]).Where(dataAndWhere[1], dataAndWhere[2:]...).Update()
		} else if len(dataAndWhere) == 2 {
			return m.Data(dataAndWhere[0]).Where(dataAndWhere[1]).Update()
		} else {
			return m.Data(dataAndWhere[0]).Update()
		}
	}
	defer func() {
		if err == nil {
			m.checkAndRemoveCache()
		}
	}()
	if m.data == nil {
		return nil, gerror.NewCode(gcode.CodeMissingParameter, "updating table with empty data")
	}
	var (
		updateData                                    = m.data
		fieldNameUpdate                               = m.getSoftFieldNameUpdated()
		conditionWhere, conditionExtra, conditionArgs = m.formatCondition(false, false)
	)
	// Automatically update the record updating time.
	if !m.unscoped && fieldNameUpdate != "" {
		var (
			refValue = reflect.ValueOf(m.data)
			refKind  = refValue.Kind()
		)
		if refKind == reflect.Ptr {
			refValue = refValue.Elem()
			refKind = refValue.Kind()
		}
		switch refKind {
		case reflect.Map, reflect.Struct:
			dataMap := ConvertDataForTableRecord(m.data)
			if fieldNameUpdate != "" {
				dataMap[fieldNameUpdate] = gtime.Now().String()
			}
			updateData = dataMap
		default:
			updates := gconv.String(m.data)
			if fieldNameUpdate != "" && !gstr.Contains(updates, fieldNameUpdate) {
				updates += fmt.Sprintf(`,%s='%s'`, fieldNameUpdate, gtime.Now().String())
			}
			updateData = updates
		}
	}
	newData, err := m.filterDataForInsertOrUpdate(updateData)
	if err != nil {
		return nil, err
	}
	conditionStr := conditionWhere + conditionExtra
	if !gstr.ContainsI(conditionStr, " WHERE ") {
		return nil, gerror.NewCode(gcode.CodeMissingParameter, "there should be WHERE condition statement for UPDATE operation")
	}
	return m.db.DoUpdate(
		m.GetCtx(),
		m.getLink(true),
		m.tables,
		newData,
		conditionStr,
		m.mergeArguments(conditionArgs)...,
	)
}

func (m *Model) UpdateExtend(dataAndWhere ...interface{}) (result sql.Result, err error) {
	//判断是否存在扩展数据表
	exMap := map[string]interface{}{}
	if len(m.expandsTable) > 0 {
		tdata := m.data.(map[string]interface{})
		extData := gconv.Bytes(tdata["ExtData"])
		if len(extData) == 0 {
			extData = gconv.Bytes(tdata["extData"])
		}
		if len(extData) == 0 {
			extData = gconv.Bytes(tdata["ext_data"])
		}
		json.Unmarshal(extData, &exMap)
	}
	m.tables = strings.Split(m.tables, " ")[0]
	m.expandsTable = strings.Split(m.expandsTable, " ")[0]

	if len(exMap) > 0 {
		tdata := m.data.(map[string]interface{})
		var conditionWhere, conditionExtra, conditionArgs = m.formatCondition(false, false)
		conditionStr := conditionWhere + conditionExtra
		querySql := fmt.Sprintf("select id from %s %s", m.tables, conditionStr)
		rows, _ := m.db.DoQuery(m.GetCtx(), m.getLink(true), querySql, conditionArgs)
		defer rows.Close()
		for rows.Next() {
			var id int64
			rows.Scan(&id)
			var list List
			for key, value := range exMap {
				dataMap := map[string]interface{}{
					"row_key":      id,
					"filed_code":   key,
					"filed_value":  value,
					"updated_by":   tdata["updated_by"],
					"updated_name": tdata["updated_name"],
					"updated_time": gtime.Now().String(),
				}
				var whereArgs = []interface{}{id, key}
				updateSql := fmt.Sprintf(" WHERE row_key = ? and filed_code=?")
				data, _ := m.db.DoUpdate(m.GetCtx(), m.getLink(true), m.expandsTable, dataMap, updateSql, m.mergeArguments(whereArgs)...)
				affected, _ := data.RowsAffected()
				if affected == 0 {
					list = append(list, dataMap)
				}
			}
			if len(list) > 0 {
				columnNames := make([]string, 0, len(list[0])+3)
				for k, _ := range list[0] {
					columnNames = append(columnNames, k)
				}
				columnNames = append(columnNames, []string{"created_by", "created_name", "created_time"}...)
				doInsertOption, _ := m.formatDoInsertOption(insertOptionDefault, columnNames)
				for i, _ := range list {
					list[i]["created_by"] = tdata["updated_by"]
					list[i]["created_name"] = tdata["updated_name"]
					list[i]["created_time"] = gtime.Now().String()
				}
				expandsTable := m.expandsTable
				if strings.Contains(expandsTable, " ") {
					expandsTable = strings.Split(expandsTable, " ")[0]
				}
				m.db.DoInsert(m.GetCtx(), m.getLink(true), expandsTable, list, doInsertOption)
			}
		}

	}
	return m.Update(dataAndWhere...)
}

// Increment increments a column's value by a given amount.
// The parameter `amount` can be type of float or integer.
func (m *Model) Increment(column string, amount interface{}) (sql.Result, error) {
	return m.getModel().Data(column, &Counter{
		Field: column,
		Value: gconv.Float64(amount),
	}).Update()
}

// Decrement decrements a column's value by a given amount.
// The parameter `amount` can be type of float or integer.
func (m *Model) Decrement(column string, amount interface{}) (sql.Result, error) {
	return m.getModel().Data(column, &Counter{
		Field: column,
		Value: -gconv.Float64(amount),
	}).Update()
}
