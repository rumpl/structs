package query

import (
	"context"
	"errors"
	"fmt"

	"github.com/rumpl/structs/internal/report"
)

var errReferenceOwner = errors.New("unknown reference owner")

const (
	referenceColumns   = "package_id,owner_kind,type_id,field_id,parameter_id,type_parameter_id,value_id,name,package,relation,file,line"
	referenceBatchSize = 128
	referenceWidth     = 12
)

// References have no dependent rows, so batching them does not change IDs or
// relationships. Bounded batches reduce SQLite prepare/execute overhead while
// retaining context cancellation and transaction rollback on any failure.
func (l *loader) references(ctx context.Context, packageID int64, kind string, id int64, refs []report.TypeReference) error {
	var typeID, fieldID, parameterID, typeParameterID, valueID any

	switch kind {
	case "type":
		typeID = id
	case "field":
		fieldID = id
	case "parameter":
		parameterID = id
	case "type_parameter":
		typeParameterID = id
	case "value":
		valueID = id
	default:
		return fmt.Errorf("%w %q", errReferenceOwner, kind)
	}

	for _, ref := range refs {
		l.referenceValues = append(l.referenceValues, packageID, kind, typeID, fieldID, parameterID, typeParameterID, valueID, ref.Name, ref.Package, ref.Relation, ref.File, ref.Line)
		if len(l.referenceValues) == referenceBatchSize*referenceWidth {
			err := l.flushReferences(ctx)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// referenceInsert accepts only a bounded row count. Every byte is fixed SQL or
// a placeholder; source declarations and user queries are always bound values.
func referenceInsert(count int) string {
	statement := []byte("INSERT INTO type_references(" + referenceColumns + ") VALUES ")

	for i := range count {
		if i > 0 {
			statement = append(statement, ',')
		}

		statement = append(statement, "(?,?,?,?,?,?,?,?,?,?,?,?)"...)
	}

	return string(statement)
}

func (l *loader) flushReferences(ctx context.Context) error {
	count := len(l.referenceValues) / referenceWidth
	if count == 0 {
		return nil
	}

	statement := referenceInsert(count)
	if _, err := l.tx.ExecContext(ctx, statement, l.referenceValues...); err != nil {
		return err
	}

	clear(l.referenceValues)
	l.referenceValues = l.referenceValues[:0]

	return nil
}
