package schema

// StructType represents a struct type.
type StructType struct {
	// Name is the name of the struct type.
	Name string

	// Fields is the list of fields in the struct. ObjectKind fields are not allowed.
	// It is a COMPATIBLE change to add new fields to an unsealed struct,
	// but it is an INCOMPATIBLE change to add new fields to a sealed struct.
	Fields []Field

	// Sealed is true if it is an INCOMPATIBLE change to add new fields to the struct.
	// It is a COMPATIBLE change to change an unsealed struct to sealed, but it is
	// an INCOMPATIBLE change to change a sealed struct to unsealed.
	Sealed bool
}
