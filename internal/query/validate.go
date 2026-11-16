package query

import (
	"errors"
	"strings"
)

var (
	errUnterminatedComment = errors.New("unterminated SQL comment")
	errMultipleStatements  = errors.New("expected one SQL query")
	errNotReadOnly         = errors.New("expected a read-only SELECT, WITH, VALUES, or EXPLAIN query")
	errUnterminatedQuote   = errors.New("unterminated SQL quote")
	errEmptyQuery          = errors.New("SQL query is empty")
)

// singleStatement recognizes SQLite quoting and comments, rejecting commands
// that could disable query_only as well as multiple statements.
func singleStatement(text string) (string, error) {
	end := -1
	hasSQL := false

	for i := 0; i < len(text); {
		trivia, err := skipTrivia(text, i)
		if err != nil {
			return "", err
		}

		if trivia.skipped {
			i = trivia.next

			continue
		}

		if text[i] == ';' {
			if end >= 0 {
				return "", errMultipleStatements
			}

			end = i
			i++

			continue
		}

		if end >= 0 {
			return "", errMultipleStatements
		}

		if !hasSQL {
			if !readOnlyKeyword(text[i:]) {
				return "", errNotReadOnly
			}

			hasSQL = true
		}

		next, err := skipQuoted(text, i)
		if err != nil {
			return "", err
		}

		i = next
	}

	if !hasSQL {
		return "", errEmptyQuery
	}

	if end >= 0 {
		text = text[:end]
	}

	return text, nil
}

type triviaResult struct {
	next    int
	skipped bool
}

func skipTrivia(text string, offset int) (triviaResult, error) {
	switch {
	case strings.ContainsRune(" \t\r\n\f", rune(text[offset])):
		return triviaResult{next: offset + 1, skipped: true}, nil
	case strings.HasPrefix(text[offset:], "--"):
		for offset < len(text) && text[offset] != '\n' {
			offset++
		}

		return triviaResult{next: offset, skipped: true}, nil
	case strings.HasPrefix(text[offset:], "/*"):
		end := strings.Index(text[offset+2:], "*/")
		if end < 0 {
			return triviaResult{}, errUnterminatedComment
		}

		return triviaResult{next: offset + end + 4, skipped: true}, nil
	default:
		return triviaResult{next: offset}, nil
	}
}

func readOnlyKeyword(text string) bool {
	end := 0
	for end < len(text) && ((text[end] >= 'a' && text[end] <= 'z') || (text[end] >= 'A' && text[end] <= 'Z')) {
		end++
	}

	switch strings.ToUpper(text[:end]) {
	case "SELECT", "WITH", "VALUES", "EXPLAIN":
		return true
	default:
		return false
	}
}

func skipQuoted(text string, offset int) (int, error) {
	quote := text[offset]
	if quote != '\'' && quote != '"' && quote != '`' && quote != '[' {
		return offset + 1, nil
	}

	if quote == '[' {
		quote = ']'
	}

	for i := offset + 1; i < len(text); i++ {
		if text[i] != quote {
			continue
		}

		if quote != ']' && i+1 < len(text) && text[i+1] == quote {
			i++

			continue
		}

		return i + 1, nil
	}

	return offset, errUnterminatedQuote
}
