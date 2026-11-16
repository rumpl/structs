package analyzer

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/printer"
	"go/scanner"
	"go/token"
	"strings"

	"github.com/rumpl/structs/internal/report"
)

func definitionOf(fset *token.FileSet, name *ast.Ident) report.Definition {
	pos := fset.PositionFor(name.Pos(), false)

	return report.Definition{
		Name: name.Name,
		File: pos.Filename,
		Line: pos.Line,
	}
}

func methodOf(function *report.Function) report.Method {
	return report.Method{
		MethodDeclaration:   report.MethodDeclaration{Name: function.Name, Declaration: function.Declaration, File: function.File},
		Line:                function.Line,
		Exported:            function.Exported,
		Signature:           function.Signature,
		SignatureParameters: function.SignatureParameters,
		PointerReceiver:     function.PointerReceiver,
	}
}

func receiverName(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.StarExpr:
		return receiverName(expr.X)
	case *ast.IndexExpr:
		return receiverName(expr.X)
	case *ast.IndexListExpr:
		return receiverName(expr.X)
	case *ast.ParenExpr:
		return receiverName(expr.X)
	default:
		return ""
	}
}

// syntaxFormatter retains the first extraction error because ast.Visitor has
// no error return. collectFile checks it before publishing any collected data.
type syntaxFormatter struct {
	fset *token.FileSet
	err  error
}

func (f *syntaxFormatter) format(node ast.Node) string {
	if f.err != nil {
		return ""
	}
	// These common expressions have unambiguous spelling. Avoid constructing a
	// printer and tabwriter for every simple field/parameter type.
	switch node := node.(type) {
	case *ast.Ident:
		return node.Name
	case *ast.SelectorExpr:
		return f.format(node.X) + "." + node.Sel.Name
	case *ast.StarExpr:
		return "*" + f.format(node.X)
	case *ast.Ellipsis:
		return "..." + f.format(node.Elt)
	}

	var out bytes.Buffer

	err := printer.Fprint(&out, f.fset, node)
	if err != nil {
		f.err = fmt.Errorf("format Go declaration: %w", err)

		return ""
	}

	return out.String()
}

// sourceParseError preserves every diagnostic and the parser's original error.
type sourceParseError struct {
	cause       error
	diagnostics string
}

func (err *sourceParseError) Error() string { return "parse Go source: " + err.diagnostics }

func (err *sourceParseError) Unwrap() error { return err.cause }

func parseError(path string, err error) error {
	var diagnostics scanner.ErrorList
	if !errors.As(err, &diagnostics) {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	messages := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		messages = append(messages, diagnostic.Error())
	}

	return &sourceParseError{cause: err, diagnostics: strings.Join(messages, "\n")}
}
