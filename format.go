package clic

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ColumnListable returns display values for a column.
type ColumnListable interface {
	ColumnValues(field string) []string
}

// Comparable returns a value used for comparison.
type Comparable interface {
	ComparedBy(field string) string
}

// Field identifies one value in a formatted list.
type Field struct {
	Key    string
	Header string
}

// FieldValues contains one display-value column per requested field.
type FieldValues [][]string

// Format identifies a list output format.
type Format string

const (
	// ColumnarFormat writes an aligned, human-readable table.
	ColumnarFormat Format = "columns"
	// JSONFormat writes a JSON array of string-valued records.
	JSONFormat Format = "json"

	columnSeparator = "  "
)

// WriteList writes rows in the requested format using the supplied fields.
func WriteList(writer io.Writer, format Format, rows ColumnListable, fields ...Field) (err error) {
	var values FieldValues
	if values, err = fieldValues(rows, fields); err != nil {
		return err
	}
	switch format {
	case ColumnarFormat:
		return writeColumnarList(writer, fields, values)
	case JSONFormat:
		return writeJSONList(writer, fields, values)
	default:
		return fmt.Errorf("unsupported list format %q", format)
	}
}

// WriteJSON writes value as JSON followed by a newline.
func WriteJSON(writer io.Writer, value any) error {
	return json.NewEncoder(writer).Encode(value)
}

func fieldValues(rows ColumnListable, fields []Field) (values FieldValues, err error) {
	if len(fields) == 0 {
		return nil, fmt.Errorf("list format requires at least one field")
	}
	values = make(FieldValues, len(fields))
	keys := make(map[string]struct{}, len(fields))
	for index, field := range fields {
		if field.Key == "" {
			return nil, fmt.Errorf("list format field %d has an empty key", index)
		}
		if _, exists := keys[field.Key]; exists {
			return nil, fmt.Errorf("list format contains duplicate field %q", field.Key)
		}
		keys[field.Key] = struct{}{}
		values[index] = rows.ColumnValues(field.Key)
		if index > 0 && len(values[index]) != len(values[0]) {
			return nil, fmt.Errorf("list format field %q has %d values, want %d", field.Key, len(values[index]), len(values[0]))
		}
	}
	return values, nil
}

func writeColumnarList(writer io.Writer, fields []Field, values FieldValues) error {
	widths := make([]int, len(fields))
	for index, field := range fields {
		if field.Header == "" {
			return fmt.Errorf("list format field %q has an empty header", field.Key)
		}
		widths[index] = len(field.Header)
		for _, value := range values[index] {
			widths[index] = max(widths[index], len(value))
		}
	}
	for index, field := range fields {
		if index > 0 {
			if _, err := io.WriteString(writer, columnSeparator); err != nil {
				return err
			}
		}
		if err := writeColumnValue(writer, widths[index], field.Header, index == len(fields)-1); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(writer, "\n"); err != nil {
		return err
	}
	for index, field := range fields {
		if index > 0 {
			if _, err := io.WriteString(writer, columnSeparator); err != nil {
				return err
			}
		}
		if err := writeColumnValue(writer, widths[index], strings.Repeat("-", len(field.Header)), index == len(fields)-1); err != nil {
			return err
		}
	}
	if _, err := io.WriteString(writer, "\n"); err != nil {
		return err
	}
	for row := range values[0] {
		for column := range fields {
			if column > 0 {
				if _, err := io.WriteString(writer, columnSeparator); err != nil {
					return err
				}
			}
			if err := writeColumnValue(writer, widths[column], values[column][row], column == len(fields)-1); err != nil {
				return err
			}
		}
		if _, err := io.WriteString(writer, "\n"); err != nil {
			return err
		}
	}
	return nil
}

func writeColumnValue(writer io.Writer, width int, value string, final bool) error {
	if final {
		_, err := io.WriteString(writer, value)
		return err
	}
	_, err := fmt.Fprintf(writer, "%-*s", width, value)
	return err
}

func writeJSONList(writer io.Writer, fields []Field, values FieldValues) error {
	records := make([]map[string]string, len(values[0]))
	for row := range records {
		records[row] = make(map[string]string, len(fields))
		for column, field := range fields {
			records[row][field.Key] = values[column][row]
		}
	}
	return WriteJSON(writer, records)
}
