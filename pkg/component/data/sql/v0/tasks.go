package sql

import (
	"fmt"
	"strings"

	"github.com/xwb1989/sqlparser"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
)

type InsertInput struct {
	Data      map[string]any `json:"data"`
	TableName string         `json:"table-name"`
}

type InsertOutput struct {
	Status string `json:"status"`
}

type InsertManyInput struct {
	ArrayData []map[string]any `json:"array-data"`
	TableName string           `json:"table-name"`
}

type InsertManyOutput struct {
	Status string `json:"status"`
}

type UpdateInput struct {
	UpdateData map[string]any `json:"update-data"`
	Filter     string         `json:"filter"`
	TableName  string         `json:"table-name"`
}

type UpdateOutput struct {
	Status string `json:"status"`
}

type SelectInput struct {
	Filter    string   `json:"filter"`
	TableName string   `json:"table-name"`
	Limit     int      `json:"limit"`
	Columns   []string `json:"columns"`
}

type SelectOutput struct {
	Rows   []map[string]any `json:"rows"`
	Status string           `json:"status"`
}

type DeleteInput struct {
	Filter    string `json:"filter"`
	TableName string `json:"table-name"`
}

type DeleteOutput struct {
	Status string `json:"status"`
}

type CreateTableInput struct {
	TableName        string            `json:"table-name"`
	ColumnsStructure map[string]string `json:"columns-structure"`
}

type CreateTableOutput struct {
	Status string `json:"status"`
}

type DropTableInput struct {
	TableName string `json:"table-name"`
}

type DropTableOutput struct {
	Status string `json:"status"`
}

// This function is used to check if the query is valid and not malicious
func isValidQuery(query string) error {
	_, err := sqlparser.Parse(query)
	if err != nil {
		return fmt.Errorf("invalid query filter: %s", err)
	}
	return nil
}

func isValidTableName(tableName string) error {
	if strings.Contains(strings.TrimSpace(tableName), " ") {
		return fmt.Errorf("invalid table name: can only be one word")
	}
	return nil
}

func buildSQLStatementInsert(tableName string, data *map[string]any) (string, map[string]any) {
	sqlStatement := "INSERT INTO " + strings.TrimSpace(tableName) + " ("
	var columns []string
	var placeholders []string
	values := make(map[string]any)

	for dataKey, dataValue := range *data {
		columns = append(columns, dataKey)
		placeholders = append(placeholders, ":"+dataKey)
		values[dataKey] = dataValue
	}

	sqlStatement += strings.Join(columns, ", ") + ") VALUES (" + strings.Join(placeholders, ", ") + ")"

	return sqlStatement, values
}

func buildSQLStatementInsertMany(tableName string, data []map[string]any) (string, map[string]any) {
	sqlStatement := "INSERT INTO " + strings.TrimSpace(tableName) + " ("
	var columns []string
	var placeholders []string
	values := make(map[string]any)

	for dataKey := range data[0] {
		columns = append(columns, dataKey)
	}

	for no, dataMap := range data {
		var placeholder []string
		for dataKey, dataValue := range dataMap {
			modifiedDataKey := fmt.Sprintf("%s%d", dataKey, no)
			placeholder = append(placeholder, ":"+modifiedDataKey)
			values[modifiedDataKey] = dataValue
		}
		placeholders = append(placeholders, "("+strings.Join(placeholder, ", ")+")")
	}

	sqlStatement += strings.Join(columns, ", ") + ") VALUES " + strings.Join(placeholders, ", ")

	return sqlStatement, values
}

func buildSQLStatementUpdate(tableName string, updateData map[string]any, filter string) (string, map[string]any) {
	sqlStatement := "UPDATE " + strings.TrimSpace(tableName) + " SET "
	values := make(map[string]any)

	var setClauses []string
	for col, updateValue := range updateData {
		setClauses = append(setClauses, fmt.Sprintf("%s = :%s", col, col))
		values[col] = updateValue
	}

	sqlStatement += strings.Join(setClauses, ", ")

	if filter != "" {
		sqlStatement += " WHERE (" + filter + ")"
	}

	return sqlStatement, values
}

// limit can be empty, but it will have default value 0
// columns can be empty, if empty it will select all columns
func buildSQLStatementSelect(tableName string, filter string, limit int, columns []string) string {
	sqlStatement := "SELECT "

	var notAll string
	if limit == 0 {
		notAll = ""
	} else {
		notAll = fmt.Sprintf(" LIMIT %d", limit)
	}

	if len(columns) > 0 {
		sqlStatement += strings.Join(columns, ", ")
	} else {
		sqlStatement += "*"
	}

	sqlStatement += " FROM " + strings.TrimSpace(tableName)
	if filter != "" {
		sqlStatement += " WHERE (" + filter + ")"
	}
	sqlStatement += notAll

	return sqlStatement
}

func buildSQLStatementDelete(tableName string, filter string) string {
	sqlStatement := "DELETE FROM " + strings.TrimSpace(tableName)

	if filter != "" {
		sqlStatement += " WHERE (" + filter + ")"
	}

	return sqlStatement
}

// columns is a map of column name and column type and handled in json format to prevent sql injection
func buildSQLStatementCreateTable(tableName string, columnsStructure map[string]string) (string, map[string]any) {
	sqlStatement := "CREATE TABLE " + strings.TrimSpace(tableName) + " ("
	var columnDefs []string
	values := make(map[string]any)

	for colName, colType := range columnsStructure {
		columnDefs = append(columnDefs, fmt.Sprintf("%s %s", colName, colType))
		values[colName] = colType
	}

	sqlStatement += strings.Join(columnDefs, ", ") + ");"
	return sqlStatement, values
}

func buildSQLStatementDropTable(tableName string) string {
	sqlStatement := "DROP TABLE " + strings.TrimSpace(tableName) + ";"
	return sqlStatement
}

func (e *execution) insert(in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct InsertInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}
	err = isValidTableName(inputStruct.TableName)
	if err != nil {
		return nil, err
	}

	sqlStatement, values := buildSQLStatementInsert(inputStruct.TableName, &inputStruct.Data)

	err = isValidQuery(sqlStatement)
	if err != nil {
		return nil, err
	}

	_, err = e.client.NamedExec(sqlStatement, values)

	if err != nil {
		return nil, err
	}

	outputStruct := InsertOutput{
		Status: "Successfully inserted 1 row",
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (e *execution) update(in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct UpdateInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}
	err = isValidTableName(inputStruct.TableName)
	if err != nil {
		return nil, err
	}

	sqlStatement, values := buildSQLStatementUpdate(inputStruct.TableName, inputStruct.UpdateData, inputStruct.Filter)
	err = isValidQuery(sqlStatement)
	if err != nil {
		return nil, err
	}

	res, err := e.client.NamedExec(sqlStatement, values)

	if err != nil {
		return nil, err
	}

	rowsAffected, _ := res.RowsAffected()
	outputStruct := UpdateOutput{
		Status: fmt.Sprintf("Successfully updated %d rows", rowsAffected),
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}
	return output, nil
}

// Queryx is used since we need not only status but also result return
func (e *execution) selects(in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct SelectInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}
	err = isValidTableName(inputStruct.TableName)
	if err != nil {
		return nil, err
	}

	sqlStatement := buildSQLStatementSelect(inputStruct.TableName, inputStruct.Filter, inputStruct.Limit, inputStruct.Columns)
	err = isValidQuery(sqlStatement)
	if err != nil {
		return nil, err
	}

	rows, err := e.client.Queryx(sqlStatement)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []map[string]any

	for rows.Next() {
		rowMap := make(map[string]any)

		err := rows.MapScan(rowMap)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		for key, value := range rowMap {
			switch v := value.(type) {
			case []byte:
				rowMap[key] = string(v)
			}
		}

		result = append(result, rowMap)
	}

	outputStruct := SelectOutput{
		Rows:   result,
		Status: fmt.Sprintf("Successfully selected %d rows", len(result)),
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (e *execution) delete(in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct DeleteInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}
	err = isValidTableName(inputStruct.TableName)
	if err != nil {
		return nil, err
	}

	sqlStatement := buildSQLStatementDelete(inputStruct.TableName, inputStruct.Filter)
	err = isValidQuery(sqlStatement)
	if err != nil {
		return nil, err
	}

	res, err := e.client.NamedExec(sqlStatement, map[string]any{})

	if err != nil {
		return nil, err
	}

	rowsAffected, _ := res.RowsAffected()
	outputStruct := DeleteOutput{
		Status: fmt.Sprintf("Successfully deleted %d rows", rowsAffected),
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (e *execution) createTable(in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct CreateTableInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}
	err = isValidTableName(inputStruct.TableName)
	if err != nil {
		return nil, err
	}

	sqlStatement, values := buildSQLStatementCreateTable(inputStruct.TableName, inputStruct.ColumnsStructure)
	err = isValidQuery(sqlStatement)
	if err != nil {
		return nil, err
	}

	_, err = e.client.NamedExec(sqlStatement, values)

	if err != nil {
		return nil, err
	}

	outputStruct := CreateTableOutput{
		Status: "Successfully created 1 table",
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (e *execution) dropTable(in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct DropTableInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}
	err = isValidTableName(inputStruct.TableName)
	if err != nil {
		return nil, err
	}

	sqlStatement := buildSQLStatementDropTable(inputStruct.TableName)
	err = isValidQuery(sqlStatement)
	if err != nil {
		return nil, err
	}

	_, err = e.client.NamedExec(sqlStatement, map[string]any{})

	if err != nil {
		return nil, err
	}

	outputStruct := DropTableOutput{
		Status: "Successfully dropped 1 table",
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (e *execution) insertMany(in *structpb.Struct) (*structpb.Struct, error) {
	var inputStruct InsertManyInput
	err := base.ConvertFromStructpb(in, &inputStruct)
	if err != nil {
		return nil, err
	}
	err = isValidTableName(inputStruct.TableName)
	if err != nil {
		return nil, err
	}

	sqlStatement, values := buildSQLStatementInsertMany(inputStruct.TableName, inputStruct.ArrayData)
	err = isValidQuery(sqlStatement)
	if err != nil {
		return nil, err
	}

	res, err := e.client.NamedExec(sqlStatement, values)

	if err != nil {
		return nil, err
	}

	rowsAffected, _ := res.RowsAffected()
	outputStruct := InsertOutput{
		Status: fmt.Sprintf("Successfully inserted %d rows", rowsAffected),
	}

	output, err := base.ConvertToStructpb(outputStruct)
	if err != nil {
		return nil, err
	}
	return output, nil
}
