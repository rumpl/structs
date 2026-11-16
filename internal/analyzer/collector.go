package analyzer

import (
	"go/ast"
	"go/token"

	"github.com/rumpl/structs/internal/report"
)

// collectedType holds one canonical type specification and optional literal
// details. Compatibility report arrays are projections, not a second AST pass.
type collectedType struct {
	fields           []report.Field
	embedded         []string
	interfaceMethods []report.Function
	named            report.NamedType
}

type packageCollector struct {
	*syntaxFormatter

	fset         *token.FileSet
	dependencies map[string]bool
	methods      map[string][]report.Method
	key          packageKey
	types        []collectedType
	functions    []report.Function
	values       []report.Value
	imports      []report.Import
}

var (
	_ ast.Visitor = (*packageCollector)(nil)
	_ ast.Visitor = (*fileCollector)(nil)
)

func newPackageCollector(fset *token.FileSet, key packageKey) *packageCollector {
	return &packageCollector{
		syntaxFormatter: &syntaxFormatter{fset: fset},
		fset:            fset,
		key:             key,
		types:           []collectedType{},
		functions:       []report.Function{},
		values:          []report.Value{},
		imports:         []report.Import{},
		dependencies:    map[string]bool{},
		methods:         map[string][]report.Method{},
	}
}

func (c *packageCollector) Visit(node ast.Node) ast.Visitor {
	file, ok := node.(*ast.File)
	if !ok {
		return nil
	}

	scope, err := fileImports(c.fset, file)
	if err != nil {
		c.err = err

		return nil
	}

	c.imports = append(c.imports, scope.imports...)
	for _, entry := range scope.imports {
		c.dependencies[entry.Path] = true
	}

	return &fileCollector{
		packageCollector: c,
		bindings:         scope.bindings,
	}
}

// The file visitor handles whole declarations and returns nil for them: Walk
// never descends into bodies, initializers, nested types, or import identifiers.
// Each file gets its own import scope; each const group owns inheritance state.
type fileCollector struct {
	*packageCollector

	bindings map[string]string
}

func (c *fileCollector) Visit(node ast.Node) ast.Visitor {
	if c.err != nil {
		return nil
	}

	switch node := node.(type) {
	case *ast.FuncDecl:
		c.visitFunction(node)
	case *ast.GenDecl:
		c.visitDeclaration(node)
	}

	return nil
}

func (c *packageCollector) associateMethods() {
	for i := range c.types {
		typ := &c.types[i].named
		if typ.Kind != report.KindAlias && len(c.methods[typ.Name]) > 0 {
			typ.ReceiverMethods = c.methods[typ.Name]
		}
	}
}

func (c *packageCollector) report() report.Package {
	c.normalize()
	c.associateMethods()

	result := report.Package{
		Name:         c.key.name,
		Directory:    c.key.directory,
		Structs:      []report.Struct{},
		Interfaces:   []report.Interface{},
		Types:        []report.NamedType{},
		Functions:    c.functions,
		Values:       c.values,
		Imports:      c.imports,
		Dependencies: []string{},
	}
	for dependency := range c.dependencies {
		result.Dependencies = append(result.Dependencies, dependency)
	}

	normalizeDependencies(result.Dependencies)

	for i := range c.types {
		typ := &c.types[i]

		result.Types = append(result.Types, typ.named)
		switch typ.named.Kind {
		case report.KindStruct:
			structure := report.Struct{
				Definition:    typ.named.Definition,
				Fields:        typ.fields,
				FieldCount:    len(typ.fields),
				Methods:       typ.named.ReceiverMethods,
				MethodCount:   len(typ.named.ReceiverMethods),
				EmbeddedTypes: typ.embedded,
			}
			for i := range structure.Methods {
				method := &structure.Methods[i]
				if method.Exported {
					structure.PublicMethodCount++
				} else {
					structure.PrivateMethodCount++
				}
			}

			result.Structs = append(result.Structs, structure)
		case report.KindInterface:
			methods := make([]report.Method, 0, len(typ.interfaceMethods))
			for i := range typ.interfaceMethods {
				function := &typ.interfaceMethods[i]
				method := methodOf(function)
				// Legacy interface methods have no receiver fields; the canonical
				// function retains the interface owner for SQL relationships.
				method.Receiver, method.ReceiverName = "", ""
				methods = append(methods, method)
			}

			result.Interfaces = append(result.Interfaces, report.Interface{
				Definition:    typ.named.Definition,
				Methods:       methods,
				MethodCount:   len(methods),
				EmbeddedTypes: typ.embedded,
			})
		}
	}

	return result
}
