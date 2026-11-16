package query

import (
	"context"
	"slices"

	"github.com/rumpl/structs/internal/report"
)

func (l *loader) loadFunctions(ctx context.Context, packageID int64, functions []report.Function, types map[string][]int64) (map[report.Definition]int64, error) {
	ids := make(map[report.Definition]int64, len(functions))
	for i := range functions {
		f := &functions[i]

		var typeID any
		if matches := types[f.ReceiverName]; len(matches) == 1 {
			typeID = matches[0]
		}

		id, err := l.insert(ctx, insertFunction, packageID, typeID, f.Kind, f.Name, f.Declaration, f.File, f.Line, f.Exported, f.Receiver, f.ReceiverName, f.PointerReceiver)
		if err != nil {
			return nil, err
		}

		if f.Kind == "method" || f.Kind == "interface_method" {
			if _, exists := ids[f.Definition]; !exists {
				ids[f.Definition] = id
			}
		}

		if err := l.typeParameters(ctx, packageID, nil, id, f.TypeParameters); err != nil {
			return nil, err
		}

		if err := l.loadParameters(ctx, packageID, id, "parameter", f.Parameters); err != nil {
			return nil, err
		}

		if err := l.loadParameters(ctx, packageID, id, "result", f.Results); err != nil {
			return nil, err
		}
	}

	return ids, nil
}

func (l *loader) typeParameters(ctx context.Context, packageID int64, typeID, functionID any, params []report.Parameter) error {
	for i := range params {
		p := &params[i]

		id, err := l.insert(ctx, insertTypeParameter, typeID, functionID, p.Position, p.Name, p.Type, p.File, p.Line)
		if err != nil {
			return err
		}

		if err := l.references(ctx, packageID, "type_parameter", id, p.References); err != nil {
			return err
		}
	}

	return nil
}

func (l *loader) loadParameters(ctx context.Context, packageID, functionID int64, role string, params []report.Parameter) error {
	for i := range params {
		p := &params[i]

		id, err := l.insert(ctx, insertParameter, functionID, role, p.Position, p.Name, p.Type, p.TypeName, p.TypePackage, p.Variadic, p.File, p.Line)
		if err != nil {
			return err
		}

		if err := l.references(ctx, packageID, "parameter", id, p.References); err != nil {
			return err
		}
	}

	return nil
}

func (l *loader) loadMethods(ctx context.Context, methods map[int64][]report.Method, functions map[report.Definition]int64) error {
	ids := make([]int64, 0, len(methods))
	for id := range methods {
		ids = append(ids, id)
	}

	slices.Sort(ids)

	for _, typeID := range ids {
		for i := range methods[typeID] {
			m := &methods[typeID][i]

			var functionID any
			if id, ok := functions[report.Definition{Name: m.Name, File: m.File, Line: m.Line}]; ok {
				functionID = id
			}

			if _, err := l.insert(ctx, insertMethod, typeID, functionID, m.Name, m.Declaration, m.File, m.Line, m.Exported); err != nil {
				return err
			}
		}
	}

	return nil
}
