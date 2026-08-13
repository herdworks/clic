package clic

import (
	"bytes"
	"strings"
	"testing"
)

type testColumns map[string][]string

func (columns testColumns) ColumnValues(field string) []string {
	return columns[field]
}

func TestWriteListColumns(t *testing.T) {
	var output bytes.Buffer
	err := WriteList(&output, ColumnarFormat, testColumns{
		"name":        {"one", "long-name"},
		"description": {"First.", "Second."},
	}, Field{Key: "name", Header: "Name"}, Field{Key: "description", Header: "Description"})
	if err != nil {
		t.Fatal(err)
	}
	want := "Name       Description\n----       -----------\none        First.\nlong-name  Second.\n"
	if got := output.String(); got != want {
		t.Fatalf("WriteList() = %q, want %q", got, want)
	}
}

func TestWriteListJSON(t *testing.T) {
	var output bytes.Buffer
	err := WriteList(&output, JSONFormat, testColumns{
		"name":        {"one"},
		"description": {"First."},
	}, Field{Key: "name"}, Field{Key: "description"})
	if err != nil {
		t.Fatal(err)
	}
	want := "[{\"description\":\"First.\",\"name\":\"one\"}]\n"
	if got := output.String(); got != want {
		t.Fatalf("WriteList() = %q, want %q", got, want)
	}
}

func TestWriteListEmpty(t *testing.T) {
	fields := []Field{{Key: "name", Header: "Name"}, {Key: "description", Header: "Description"}}
	rows := testColumns{"name": {}, "description": {}}
	for _, test := range []struct {
		name   string
		format Format
		want   string
	}{
		{name: "columns", format: ColumnarFormat, want: "Name  Description\n----  -----------\n"},
		{name: "json", format: JSONFormat, want: "[]\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := WriteList(&output, test.format, rows, fields...); err != nil {
				t.Fatal(err)
			}
			if got := output.String(); got != test.want {
				t.Fatalf("WriteList() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestWriteListRejectsInvalidFields(t *testing.T) {
	rows := testColumns{"name": {"one"}, "description": {"First.", "Second."}}
	for _, test := range []struct {
		name   string
		fields []Field
		want   string
	}{
		{name: "none", want: "at least one field"},
		{name: "empty key", fields: []Field{{Header: "Name"}}, want: "empty key"},
		{name: "empty header", fields: []Field{{Key: "name"}}, want: "empty header"},
		{name: "duplicate", fields: []Field{{Key: "name", Header: "Name"}, {Key: "name", Header: "Again"}}, want: "duplicate"},
		{name: "mismatched", fields: []Field{{Key: "name", Header: "Name"}, {Key: "description", Header: "Description"}}, want: "has 2 values, want 1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			err := WriteList(&output, ColumnarFormat, rows, test.fields...)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("WriteList() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestWriteListRejectsUnsupportedFormat(t *testing.T) {
	var output bytes.Buffer
	err := WriteList(&output, Format("yaml"), testColumns{"name": {"one"}}, Field{Key: "name", Header: "Name"})
	if err == nil || !strings.Contains(err.Error(), "unsupported list format") {
		t.Fatalf("WriteList() error = %v", err)
	}
}

func TestWriteJSON(t *testing.T) {
	var output bytes.Buffer
	if err := WriteJSON(&output, struct {
		Enabled bool `json:"enabled"`
		Count   int  `json:"count"`
	}{Enabled: true, Count: 3}); err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), "{\"enabled\":true,\"count\":3}\n"; got != want {
		t.Fatalf("WriteJSON() = %q, want %q", got, want)
	}
}
