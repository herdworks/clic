package clic

// ColumnListable returns display values for a column.
type ColumnListable interface {
	ColumnValues(field string) []string
}

// Comparable returns a value used for comparison.
type Comparable interface {
	ComparedBy(field string) string
}
