package report

// TypeReference records a syntactic named-type occurrence. Relation is a
// slash-separated path from its owner (e.g. map_value/pointer).
type TypeReference struct {
	Name     string `json:"name"`
	Package  string `json:"package"`
	Relation string `json:"relation"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}

// Parameter represents one parameter, result, or type parameter; positions are
// zero-based within its role. Unnamed parameters have an empty name.
type Parameter struct {
	Name        string          `json:"name"`
	Type        string          `json:"type"`
	TypeName    string          `json:"type_name"`
	TypePackage string          `json:"type_package"`
	File        string          `json:"file"`
	References  []TypeReference `json:"references"`
	Position    int             `json:"position"`
	Line        int             `json:"line"`
	Variadic    bool            `json:"variadic"`
}

type Signature struct {
	Receiver     string `json:"receiver"`
	ReceiverName string `json:"receiver_name"`
}

// SignatureParameters groups positional signature data separately from receiver
// text, keeping embedded report records compact without changing their JSON.
type SignatureParameters struct {
	Parameters     []Parameter `json:"parameters"`
	Results        []Parameter `json:"results"`
	TypeParameters []Parameter `json:"type_parameters"`
}

// FunctionDeclaration contains the textual identity of a callable.
type FunctionDeclaration struct {
	Kind        string `json:"kind"`
	Declaration string `json:"declaration"`
}

type Function struct {
	Signature
	SignatureParameters
	FunctionDeclaration
	Definition

	Exported        bool `json:"exported"`
	PointerReceiver bool `json:"pointer_receiver"`
}

// NamedType inventories every package-level type specification, including
// aliases. Literal struct/interface details remain in the existing arrays.
type NamedType struct {
	Definition

	Kind            string          `json:"kind"`
	Expression      string          `json:"expression"`
	Declaration     string          `json:"declaration"`
	ReceiverMethods []Method        `json:"receiver_methods"`
	TypeParameters  []Parameter     `json:"type_parameters"`
	References      []TypeReference `json:"references"`
	Exported        bool            `json:"exported"`
}

// Value describes a package-level const/var. Initializer is the complete RHS
// expression list for its specification, not a type-checked per-name value.
type Value struct {
	Definition

	Kind        string          `json:"kind"`
	Type        string          `json:"type"`
	Initializer string          `json:"initializer"`
	Declaration string          `json:"declaration"`
	References  []TypeReference `json:"references"`
	Inherited   bool            `json:"inherited"`
	Exported    bool            `json:"exported"`
}
