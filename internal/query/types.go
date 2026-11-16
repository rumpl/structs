package query

import (
	"context"
	"go/ast"
	"slices"

	"github.com/rumpl/structs/internal/report"
)

type packageIndex struct {
	structs    map[report.Definition]*report.Struct
	interfaces map[report.Definition]*report.Interface
	ids        map[string][]int64
	methods    map[int64][]report.Method
	types      []report.NamedType
}

// Preserve canonical declarations, with compatibility support for reports
// assembled directly by callers using only the struct/interface detail arrays.
func indexPackage(pkg *report.Package) *packageIndex {
	index := &packageIndex{types: slices.Clone(pkg.Types), structs: map[report.Definition]*report.Struct{}, interfaces: map[report.Definition]*report.Interface{}, ids: map[string][]int64{}, methods: map[int64][]report.Method{}}

	known := map[report.Definition]bool{}
	for i := range index.types {
		known[index.types[i].Definition] = true
	}

	for i := range pkg.Structs {
		typ := &pkg.Structs[i]

		index.structs[typ.Definition] = typ
		if !known[typ.Definition] {
			index.types = append(index.types, report.NamedType{Definition: typ.Definition, Kind: report.KindStruct, Exported: ast.IsExported(typ.Name)})
		}
	}

	for i := range pkg.Interfaces {
		typ := &pkg.Interfaces[i]

		index.interfaces[typ.Definition] = typ
		if !known[typ.Definition] {
			index.types = append(index.types, report.NamedType{Definition: typ.Definition, Kind: report.KindInterface, Exported: ast.IsExported(typ.Name)})
		}
	}

	return index
}

type typeDetails struct {
	fields   any
	public   any
	private  any
	methods  []report.Method
	embedded []string
	count    int
}

func (index *packageIndex) details(typ *report.NamedType) typeDetails {
	details := typeDetails{methods: typ.ReceiverMethods, count: len(typ.ReceiverMethods)}
	if typ.Kind == report.KindNamed {
		public, private := 0, 0

		for i := range details.methods {
			if details.methods[i].Exported {
				public++
			} else {
				private++
			}
		}

		details.public, details.private = public, private
	}

	if s, ok := index.structs[typ.Definition]; ok && typ.Kind == report.KindStruct {
		details.fields, details.public, details.private = s.FieldCount, s.PublicMethodCount, s.PrivateMethodCount
		details.count, details.methods, details.embedded = s.MethodCount, s.Methods, s.EmbeddedTypes
	}

	if i, ok := index.interfaces[typ.Definition]; ok && typ.Kind == report.KindInterface {
		details.count, details.methods, details.embedded = i.MethodCount, i.Methods, i.EmbeddedTypes
	}

	return details
}

func (l *loader) loadTypes(ctx context.Context, packageID int64, index *packageIndex) error {
	for i := range index.types {
		typ := &index.types[i]
		details := index.details(typ)

		id, err := l.insert(ctx, insertType, packageID, typ.Kind, typ.Name, typ.File, typ.Line, details.fields, details.count, details.public, details.private, typ.Expression, typ.Declaration, typ.Exported)
		if err != nil {
			return err
		}

		index.ids[typ.Name] = append(index.ids[typ.Name], id)

		index.methods[id] = details.methods
		if err := l.loadTypeMetadata(ctx, packageID, id, typ, details.embedded); err != nil {
			return err
		}

		if s, ok := index.structs[typ.Definition]; ok && typ.Kind == report.KindStruct {
			if err := l.loadFields(ctx, packageID, id, s.Fields); err != nil {
				return err
			}
		}
	}

	return nil
}

func (l *loader) loadTypeMetadata(ctx context.Context, packageID, id int64, typ *report.NamedType, embedded []string) error {
	if err := l.references(ctx, packageID, "type", id, typ.References); err != nil {
		return err
	}

	if err := l.typeParameters(ctx, packageID, id, nil, typ.TypeParameters); err != nil {
		return err
	}

	for _, expression := range embedded {
		if _, err := l.insert(ctx, insertEmbedded, id, expression); err != nil {
			return err
		}
	}

	return nil
}

func (l *loader) loadFields(ctx context.Context, packageID, typeID int64, fields []report.Field) error {
	for i := range fields {
		f := &fields[i]

		id, err := l.insert(ctx, insertField, typeID, f.Name, f.Type, f.TypeName, f.TypePackage, f.Tag, f.File, f.Line, f.Embedded, f.Exported)
		if err != nil {
			return err
		}

		if err := l.references(ctx, packageID, "field", id, f.References); err != nil {
			return err
		}
	}

	return nil
}
