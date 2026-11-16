package query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/rumpl/structs/internal/report"
)

type loader struct {
	tx              *sql.Tx
	statements      map[string]*sql.Stmt
	referenceValues []any
}

func (l *loader) insert(ctx context.Context, statement string, values ...any) (int64, error) {
	stmt := l.statements[statement]
	if stmt == nil {
		var err error

		stmt, err = l.tx.PrepareContext(ctx, statement)
		if err != nil {
			return 0, err
		}

		l.statements[statement] = stmt
	}

	res, err := stmt.ExecContext(ctx, values...)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func load(ctx context.Context, db *sql.DB, data report.Report) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	l := &loader{tx: tx, statements: map[string]*sql.Stmt{}}
	err = l.loadReport(ctx, data)

	err = errors.Join(err, l.closeStatements())
	if err == nil {
		if errCommit := tx.Commit(); errCommit != nil {
			err = fmt.Errorf("commit SQLite load: %w", errCommit)
		}
	}

	if errRollback := tx.Rollback(); errRollback != nil && !errors.Is(errRollback, sql.ErrTxDone) {
		err = errors.Join(err, fmt.Errorf("rollback SQLite load: %w", errRollback))
	}

	return err
}

func (l *loader) loadReport(ctx context.Context, data report.Report) error {
	if _, err := l.tx.ExecContext(ctx, Schema); err != nil {
		return err
	}

	for i := range data.Packages {
		if err := l.loadPackage(ctx, &data.Packages[i]); err != nil {
			return err
		}
	}

	return l.flushReferences(ctx)
}

func (l *loader) closeStatements() error {
	var result error

	for statement, stmt := range l.statements {
		if err := stmt.Close(); err != nil {
			result = errors.Join(result, fmt.Errorf("close insert: %w", err))
		}

		delete(l.statements, statement)
	}

	return result
}

func (l *loader) loadPackage(ctx context.Context, pkg *report.Package) error {
	packageID, err := l.insert(ctx, insertPackage, pkg.Name, pkg.Directory)
	if err != nil {
		return err
	}

	if errImports := l.loadImports(ctx, packageID, pkg); errImports != nil {
		return errImports
	}

	index := indexPackage(pkg)
	if errTypes := l.loadTypes(ctx, packageID, index); errTypes != nil {
		return errTypes
	}

	functions, err := l.loadFunctions(ctx, packageID, pkg.Functions, index.ids)
	if err != nil {
		return err
	}

	if err := l.loadMethods(ctx, index.methods, functions); err != nil {
		return err
	}

	return l.loadValues(ctx, packageID, pkg.Values)
}

func (l *loader) loadImports(ctx context.Context, packageID int64, pkg *report.Package) error {
	for _, entry := range pkg.Imports {
		if _, err := l.insert(ctx, insertImport, packageID, entry.Path, entry.Alias, entry.File, entry.Line); err != nil {
			return err
		}
	}

	for _, dependency := range pkg.Dependencies {
		if _, err := l.insert(ctx, insertDependency, packageID, dependency); err != nil {
			return err
		}
	}

	return nil
}

func (l *loader) loadValues(ctx context.Context, packageID int64, values []report.Value) error {
	for i := range values {
		v := &values[i]

		id, err := l.insert(ctx, insertValue, packageID, v.Kind, v.Name, v.Type, v.Initializer, v.Declaration, v.Inherited, v.Exported, v.File, v.Line)
		if err != nil {
			return err
		}

		if err := l.references(ctx, packageID, "value", id, v.References); err != nil {
			return err
		}
	}

	return nil
}
