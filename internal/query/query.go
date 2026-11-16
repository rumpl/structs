// Package query loads analyzed declarations into a transient SQLite database.
package query

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"

	_ "modernc.org/sqlite"

	"github.com/rumpl/structs/internal/report"
)

// Schema describes the complete SQLite schema.
//
//go:embed schema.sql
var Schema string

type execution struct {
	err    error
	result Result
}

func (e *execution) closeRows(rows *sql.Rows) {
	if err := rows.Close(); err != nil {
		e.err = errors.Join(e.err, fmt.Errorf("close query rows: %w", err))
		e.result = Result{}
	}
}

type Result struct {
	Columns []string `json:"columns"`
	Rows    [][]any  `json:"rows"`
}

// Execute loads a private database, executes one read-only query, and closes it.
func Execute(ctx context.Context, data report.Report, statement string) (Result, error) {
	statement, err := singleStatement(statement)
	if err != nil {
		return Result{}, err
	}

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return Result{}, fmt.Errorf("open SQLite: %w", err)
	}

	db.SetMaxOpenConns(1)

	result, err := executeDatabase(ctx, db, data, statement)
	if errClose := db.Close(); errClose != nil {
		err = errors.Join(err, fmt.Errorf("close SQLite: %w", errClose))
	}

	if err != nil {
		return Result{}, err
	}

	return result, nil
}

func executeDatabase(ctx context.Context, db *sql.DB, data report.Report, statement string) (Result, error) {
	if err := load(ctx, db, data); err != nil {
		return Result{}, fmt.Errorf("load SQLite: %w", err)
	}

	if _, err := db.ExecContext(ctx, "PRAGMA query_only = ON"); err != nil {
		return Result{}, err
	}

	rows, err := db.QueryContext(ctx, statement)
	if err != nil {
		return Result{}, fmt.Errorf("query SQLite: %w", err)
	}

	state := &execution{}

	func() {
		defer state.closeRows(rows)

		state.result, state.err = readRows(rows)
		if errRows := rows.Err(); errRows != nil && state.err == nil {
			state.err = fmt.Errorf("query SQLite: %w", errRows)
		}
	}()

	return state.result, state.err
}

func readRows(rows *sql.Rows) (Result, error) {
	columns, err := rows.Columns()
	if err != nil {
		return Result{}, err
	}

	result := Result{Columns: columns, Rows: [][]any{}}
	for rows.Next() {
		values := make([]any, len(columns))

		pointers := make([]any, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}

		if err := rows.Scan(pointers...); err != nil {
			return Result{}, err
		}

		result.Rows = append(result.Rows, values)
	}

	if err := rows.Err(); err != nil {
		return Result{}, fmt.Errorf("query SQLite: %w", err)
	}

	return result, nil
}
