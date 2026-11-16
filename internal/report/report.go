// Package report defines the format-independent results of source analysis.
package report

// Report contains packages ordered by directory and package name.
type Report struct {
	Packages []Package `json:"packages"`
}

// Package identifies a Go package within a source directory.
type Package struct {
	Name         string      `json:"name"`
	Directory    string      `json:"directory"`
	Structs      []Struct    `json:"structs"`
	Interfaces   []Interface `json:"interfaces"`
	Dependencies []string    `json:"dependencies"`
	Imports      []Import    `json:"imports"`
	Types        []NamedType `json:"types"`
	Functions    []Function  `json:"functions"`
	Values       []Value     `json:"values"`
}

// Definition identifies a named, package-level type.
type Definition struct {
	Name string `json:"name"`
	File string `json:"file"`
	Line int    `json:"line"`
}

// MethodDeclaration identifies a method's name, source, and declaration text.
type MethodDeclaration struct {
	Name        string `json:"name"`
	Declaration string `json:"declaration"`
	File        string `json:"file"`
}

// Method describes a declaration, without its documentation or body.
type Method struct {
	Signature
	MethodDeclaration
	SignatureParameters

	Line            int  `json:"line"`
	Exported        bool `json:"exported"`
	PointerReceiver bool `json:"pointer_receiver"`
}

// Import records a source import. Alias is empty for an implicit package name.
type Import struct {
	Path  string `json:"path"`
	Alias string `json:"alias"`
	File  string `json:"file"`
	Line  int    `json:"line"`
}

// Field is one declared field, including embedded and blank fields.
// TypePackage is a syntactic import-path association, not resolved type identity.
type Field struct {
	Name        string          `json:"name"`
	Type        string          `json:"type"`
	TypeName    string          `json:"type_name"`
	TypePackage string          `json:"type_package"`
	Tag         string          `json:"tag"`
	File        string          `json:"file"`
	References  []TypeReference `json:"references"`
	Line        int             `json:"line"`
	Embedded    bool            `json:"embedded"`
	Exported    bool            `json:"exported"`
}

// Struct includes directly declared methods for both value and pointer receivers.
type Struct struct {
	Definition

	Fields             []Field  `json:"fields"`
	Methods            []Method `json:"methods"`
	EmbeddedTypes      []string `json:"embedded_types"`
	FieldCount         int      `json:"field_count"`
	MethodCount        int      `json:"method_count"`
	PublicMethodCount  int      `json:"public_method_count"`
	PrivateMethodCount int      `json:"private_method_count"`
}

// Interface includes explicitly declared methods, not inherited methods.
// EmbeddedTypes also includes type-set terms and constraints.
type Interface struct {
	Definition

	Methods       []Method `json:"methods"`
	EmbeddedTypes []string `json:"embedded_types"`
	MethodCount   int      `json:"method_count"`
}
