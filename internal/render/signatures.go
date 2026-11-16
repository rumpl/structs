package render

import (
	"go/scanner"
	"go/token"
	"strings"
)

// methodSignature strips a balanced receiver using Go tokens, without building
// a second AST and invoking the printer for every method in the HTML report.
func methodSignature(declaration string) string {
	if !strings.HasPrefix(declaration, "func ") {
		return singleLineSignature(declaration)
	}

	start := len("func ")
	for start < len(declaration) && (declaration[start] == ' ' || declaration[start] == '\t') {
		start++
	}

	if start == len(declaration) || declaration[start] != '(' {
		return singleLineSignature(declaration)
	}

	fset := token.NewFileSet()
	file := fset.AddFile("", -1, len(declaration))

	var lexer scanner.Scanner
	lexer.Init(file, []byte(declaration), nil, 0)

	depth := 0

	for {
		pos, tok, _ := lexer.Scan()
		switch tok {
		case token.EOF:
			return singleLineSignature(declaration)
		case token.LPAREN:
			depth++
		case token.RPAREN:
			depth--
			if depth == 0 {
				return singleLineSignature("func " + strings.TrimSpace(declaration[file.Offset(pos)+1:]))
			}
		}
	}
}

// singleLineSignature normalizes whitespace without changing raw string tags
// or other quoted literals. Implicit semicolons inside anonymous types become
// explicit when a newline is removed; the final implicit semicolon is omitted.
func singleLineSignature(source string) string {
	if !strings.ContainsAny(source, "\n\r\t") {
		return source
	}

	fset := token.NewFileSet()
	file := fset.AddFile("", -1, len(source))

	var lexer scanner.Scanner
	lexer.Init(file, []byte(source), nil, 0)

	out := make([]byte, 0, len(source))

	end := 0
	previous := token.ILLEGAL
	pendingComma := false
	pendingSemicolon := false

	for {
		pos, tok, literal := lexer.Scan()
		if tok == token.EOF {
			break
		}

		if tok == token.SEMICOLON && literal == "\n" {
			pendingSemicolon = true

			continue
		}

		start := file.Offset(pos)

		if pendingComma {
			if tok != token.RPAREN && tok != token.RBRACK && tok != token.RBRACE {
				out = append(out, ',')
			}

			pendingComma = false
		}

		if tok == token.COMMA {
			pendingComma = true
			end = start + 1
			previous = tok

			continue
		}

		if pendingSemicolon {
			out = append(out, ';', ' ')
		} else if start > end && len(out) > 0 && previous != token.LPAREN && previous != token.LBRACK && tok != token.RPAREN && tok != token.RBRACK {
			out = append(out, ' ')
		}

		pendingSemicolon = false

		if literal == "" {
			literal = tok.String()
		}

		out = append(out, literal...)
		end = start + len(literal)
		previous = tok
	}

	return string(out)
}
