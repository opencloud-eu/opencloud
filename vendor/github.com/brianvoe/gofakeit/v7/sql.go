package gofakeit

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type SQLOptions struct {
	Table  string  `json:"table" xml:"table"`   // Table name we are inserting into
	Count  int     `json:"count" xml:"count"`   // How many entries (tuples) we're generating
	Fields []Field `json:"fields" xml:"fields"` // The fields to be generated
}

func SQL(so *SQLOptions) (string, error) { return sqlFunc(GlobalFaker, so) }

func (f *Faker) SQL(so *SQLOptions) (string, error) { return sqlFunc(f, so) }

func sqlFunc(f *Faker, so *SQLOptions) (string, error) {
	if so.Table == "" {
		return "", errors.New("must provide table name to generate SQL")
	}
	if len(so.Fields) <= 0 {
		return "", errors.New(("must pass fields in order to generate SQL queries"))
	}
	if so.Count <= 0 {
		return "", errors.New("must have entry count")
	}

	var sb strings.Builder
	sb.WriteString("INSERT INTO " + so.Table + " ")

	// Loop through each field and put together column names
	var cols []string
	for _, f := range so.Fields {
		cols = append(cols, f.Name)
	}
	sb.WriteString("(" + strings.Join(cols, ", ") + ")")

	sb.WriteString(" VALUES ")
	for i := 0; i < so.Count; i++ {
		// Start opening value
		sb.WriteString("(")

		// Now, we need to add all of our fields
		var endStr string
		for ii, field := range so.Fields {
			// Set end of value string
			endStr = ", "
			if ii == len(so.Fields)-1 {
				endStr = ""
			}

			// If autoincrement, add based upon loop
			if field.Function == "autoincrement" {
				sb.WriteString(fmt.Sprintf("%d%s", i+1, endStr))
				continue
			}

			// Get the function info for the field
			funcInfo := GetFuncLookup(field.Function)
			if funcInfo == nil {
				return "", errors.New("invalid function, " + field.Function + " does not exist")
			}

			// Generate the value
			val, err := funcInfo.Generate(f, &field.Params, funcInfo)
			if err != nil {
				return "", err
			}

			// Convert the output value to the proper SQL type
			convertType := sqlConvertType(funcInfo.Output, val)

			// If its the last field, we need to close the value
			sb.WriteString(convertType + endStr)
		}

		// If its the last value, we need to close the value
		if i == so.Count-1 {
			sb.WriteString(");")
		} else {
			sb.WriteString("),")
		}
	}

	return sb.String(), nil
}

// sqlConvertType will take in a type and value and convert it to the proper SQL type
func sqlConvertType(t string, val any) string {
	switch t {
	case "string":
		return `'` + fmt.Sprintf("%v", val) + `'`
	case "[]byte":
		return `'` + fmt.Sprintf("%s", val) + `'`
	default:
		return fmt.Sprintf("%v", val)
	}
}

func addDatabaseSQLLookup() {
	AddFuncLookup("sql", Info{
		Display:     "SQL",
		Category:    "database",
		Description: "Command in SQL used to add new data records into a database table",
		Example: `INSERT INTO people 
	(id, first_name, price, age, created_at) 
VALUES 
	(1, 'Markus', 804.92, 21, '1937-01-30 07:58:01'),
	(2, 'Santino', 235.13, 40, '1964-07-07 22:25:40');`,
		Output:      "string",
		ContentType: "application/sql",
		Aliases: []string{
			"insert command", "database query", "sql statement", "record insert", "data query",
		},
		Keywords: []string{
			"database", "insert", "command", "records", "table", "tuples", "rows", "data", "values", "query",
		},
		Params: []Param{
			{Field: "table", Display: "Table", Type: "string", Description: "Name of the table to insert into"},
			{Field: "count", Display: "Count", Type: "int", Default: "100", Description: "Number of inserts to generate"},
			{Field: "fields", Display: "Fields", Type: "[]Field", Description: "Fields containing key name and function to run in json format"},
		},
		Generate: func(f *Faker, m *MapParams, info *Info) (any, error) {
			so := SQLOptions{}

			table, err := info.GetString(m, "table")
			if err != nil {
				return nil, err
			}
			so.Table = table

			count, err := info.GetInt(m, "count")
			if err != nil {
				return nil, err
			}
			so.Count = count

			fieldsStr, err := info.GetStringArray(m, "fields")
			if err != nil {
				return nil, err
			}

			// Check to make sure fields has length
			if len(fieldsStr) > 0 {
				so.Fields = make([]Field, len(fieldsStr))

				for i, f := range fieldsStr {
					// Unmarshal fields string into fields array
					err = json.Unmarshal([]byte(f), &so.Fields[i])
					if err != nil {
						return nil, err
					}
				}
			}

			return sqlFunc(f, &so)
		},
	})

}
