package analyzer

import (
	"go/ast"
	"go/token"
	"strings"

	"github.com/rumpl/structs/internal/report"
)

func (c *fileCollector) visitDeclaration(decl *ast.GenDecl) {
	switch decl.Tok {
	case token.TYPE:
		for _, spec := range decl.Specs {
			if typ, ok := spec.(*ast.TypeSpec); ok {
				c.visitType(typ)
			}
		}
	case token.CONST, token.VAR:
		c.visitValues(decl)
	}
}

func (c *fileCollector) visitFunction(decl *ast.FuncDecl) {
	// Copy only to remove documentation/body without mutating the parsed AST.
	syntax := *decl
	syntax.Doc, syntax.Body = nil, nil

	kind := "function"
	if decl.Recv != nil {
		kind = "method"
	}

	function := report.Function{
		Definition: definitionOf(c.fset, decl.Name),
		Kind:       kind, Declaration: c.format(&syntax),
		Exported:            ast.IsExported(decl.Name.Name),
		Signature:           c.signatureOf(decl.Recv),
		SignatureParameters: c.signatureParameters(decl.Type),
	}

	if decl.Recv != nil && len(decl.Recv.List) > 0 {
		_, function.PointerReceiver = decl.Recv.List[0].Type.(*ast.StarExpr)
	}

	c.functions = append(c.functions, function)
	if kind == "method" && function.ReceiverName != "" {
		c.methods[function.ReceiverName] = append(c.methods[function.ReceiverName], methodOf(&function))
	}
}

func typeKind(spec *ast.TypeSpec) string {
	if spec.Assign.IsValid() {
		return report.KindAlias
	}

	switch spec.Type.(type) {
	case *ast.StructType:
		return report.KindStruct
	case *ast.InterfaceType:
		return report.KindInterface
	default:
		return report.KindNamed
	}
}

func (c *fileCollector) visitType(spec *ast.TypeSpec) {
	typ := collectedType{
		named: report.NamedType{
			Definition:      definitionOf(c.fset, spec.Name),
			Kind:            typeKind(spec),
			Expression:      c.format(spec.Type),
			Declaration:     "type " + c.format(spec),
			Exported:        ast.IsExported(spec.Name.Name),
			ReceiverMethods: []report.Method{},
			TypeParameters:  c.parameters(spec.TypeParams),
			References:      typeReferences(c.fset, spec.Type, c.bindings),
		},
		embedded: []string{},
	}
	// Aliases retain their expression/references but do not become additional
	// struct/interface definitions or acquire methods from an aliased type.
	if typ.named.Kind != report.KindAlias {
		switch body := spec.Type.(type) {
		case *ast.StructType:
			typ.fields = c.analyzeFields(body)
			for i := range typ.fields {
				field := &typ.fields[i]
				if field.Embedded {
					typ.embedded = append(typ.embedded, field.Type)
				}
			}
		case *ast.InterfaceType:
			c.visitInterface(body, &typ)
		}
	}

	c.types = append(c.types, typ)
}

func (c *fileCollector) visitInterface(body *ast.InterfaceType, owner *collectedType) {
	for _, field := range body.Methods.List {
		if len(field.Names) == 0 {
			owner.embedded = append(owner.embedded, c.format(field.Type))

			continue
		}

		typ, ok := field.Type.(*ast.FuncType)
		if !ok {
			continue
		}

		signature := c.signatureOf(nil)
		parameters := c.signatureParameters(typ)
		signature.Receiver, signature.ReceiverName = owner.named.Name, owner.named.Name

		declaration := strings.TrimPrefix(c.format(typ), "func")
		for _, name := range field.Names {
			function := report.Function{
				Definition: definitionOf(c.fset, name),
				Kind:       "interface_method", Declaration: name.Name + declaration,
				Exported:            ast.IsExported(name.Name),
				Signature:           signature,
				SignatureParameters: parameters,
			}
			owner.interfaceMethods = append(owner.interfaceMethods, function)
			c.functions = append(c.functions, function)
		}
	}
}

// Const inheritance belongs to a single declaration group, not the visitor's
// file/package state. Initializers are printed, never traversed or evaluated.
func (c *fileCollector) visitValues(decl *ast.GenDecl) {
	var (
		previousType   ast.Expr
		previousValues []ast.Expr
	)

	for _, raw := range decl.Specs {
		spec, ok := raw.(*ast.ValueSpec)
		if !ok {
			continue
		}

		typ, values := spec.Type, spec.Values

		inherited := decl.Tok == token.CONST && typ == nil && len(values) == 0
		if inherited {
			typ, values = previousType, previousValues
		}

		previousType, previousValues = typ, values

		parts := make([]string, 0, len(values))
		for _, value := range values {
			parts = append(parts, c.format(value))
		}

		expression := ""
		if typ != nil {
			expression = c.format(typ)
		}

		references := typeReferences(c.fset, typ, c.bindings)

		declaration := decl.Tok.String() + " " + c.format(spec)
		for _, name := range spec.Names {
			c.values = append(c.values, report.Value{
				Definition:  definitionOf(c.fset, name),
				Kind:        decl.Tok.String(),
				Type:        expression,
				Initializer: strings.Join(parts, ", "),
				Declaration: declaration,
				Inherited:   inherited,
				Exported:    ast.IsExported(name.Name),
				References:  references,
			})
		}
	}
}
