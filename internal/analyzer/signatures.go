package analyzer

import (
	"go/ast"
	"go/token"

	"github.com/rumpl/structs/internal/report"
)

func (c *fileCollector) parameters(list *ast.FieldList) []report.Parameter {
	result := []report.Parameter{}
	if list == nil {
		return result
	}

	for _, field := range list.List {
		reference := fieldType(field.Type, c.bindings)
		_, variadic := field.Type.(*ast.Ellipsis)
		base := report.Parameter{
			Type:        c.format(field.Type),
			TypeName:    reference.name,
			TypePackage: reference.packagePath,
			Variadic:    variadic,
			References:  typeReferences(c.fset, field.Type, c.bindings),
		}

		appendEntry := func(name string, position token.Pos) {
			entry := base
			pos := c.fset.PositionFor(position, false)
			entry.Name, entry.Position = name, len(result)
			entry.File, entry.Line = pos.Filename, pos.Line
			result = append(result, entry)
		}
		if len(field.Names) == 0 {
			appendEntry("", field.Pos())
		} else {
			for _, name := range field.Names {
				appendEntry(name.Name, name.Pos())
			}
		}
	}

	return result
}

func (c *fileCollector) signatureParameters(typ *ast.FuncType) report.SignatureParameters {
	return report.SignatureParameters{Parameters: c.parameters(typ.Params), Results: c.parameters(typ.Results), TypeParameters: c.parameters(typ.TypeParams)}
}

func (c *fileCollector) signatureOf(receiver *ast.FieldList) report.Signature {
	signature := report.Signature{}

	if receiver != nil && len(receiver.List) > 0 {
		expression := receiver.List[0].Type
		signature.Receiver = c.format(expression)
		signature.ReceiverName = receiverName(expression)
	}

	return signature
}
