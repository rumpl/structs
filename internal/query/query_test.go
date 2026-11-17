package query_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rumpl/structs/internal/query"
	"github.com/rumpl/structs/internal/report"
)

const (
	samplePackage   = "sample"
	sampleStruct    = "Box"
	sampleInterface = "Reader"
	nameColumn      = "name"
)

func sample() report.Report {
	return report.Report{Packages: []report.Package{{
		Name: samplePackage, Directory: "/sample",
		Structs: []report.Struct{{
			Name: sampleStruct, File: "/sample/types.go", Line: 2, FieldCount: 3, MethodCount: 2, PublicMethodCount: 1, PrivateMethodCount: 1,
			Methods: []report.Method{{Name: "Get", Declaration: "func (Box) Get() int", Exported: true}, {Name: "hidden"}}, EmbeddedTypes: []string{"Base"},
		}},
		Interfaces: []report.Interface{{Name: sampleInterface, MethodCount: 1, Methods: []report.Method{{Name: readMethod, Exported: true}}, EmbeddedTypes: []string{"io.Closer"}}},
	}}}
}

func executeQuery(t *testing.T, statement string) query.Result {
	t.Helper()

	result, err := query.Execute(context.Background(), sample(), statement)
	require.NoError(t, err)

	return result
}

func TestLoadedTypeMetadata(t *testing.T) {
	t.Parallel()
	result := executeQuery(t, "SELECT name,field_count,public_method_count,private_method_count FROM structs")
	assert.Equal(t, []string{nameColumn, "field_count", "public_method_count", "private_method_count"}, result.Columns)
	assert.Equal(t, [][]any{{sampleStruct, int64(3), int64(1), int64(1)}}, result.Rows)
}

func TestMethodJoinAndVisibility(t *testing.T) {
	t.Parallel()
	result := executeQuery(t, "SELECT s.name,m.name,m.exported FROM structs s JOIN methods m ON m.type_id=s.id ORDER BY m.name")
	assert.Equal(t, [][]any{{sampleStruct, "Get", int64(1)}, {sampleStruct, "hidden", int64(0)}}, result.Rows)
}

func TestInterfaceCountsAreNullable(t *testing.T) {
	t.Parallel()
	result := executeQuery(t, "SELECT name,method_count,field_count,public_method_count FROM types WHERE kind='interface'")
	assert.Equal(t, [][]any{{sampleInterface, int64(1), nil, nil}}, result.Rows)
}

func TestEmbeddingRowsRetainOwners(t *testing.T) {
	t.Parallel()
	result := executeQuery(t, "SELECT t.name,e.expression FROM embedded_types e JOIN types t ON t.id=e.type_id ORDER BY t.name")
	assert.Equal(t, [][]any{{sampleStruct, "Base"}, {sampleInterface, "io.Closer"}}, result.Rows)
}

func TestCTEAggregatesPackages(t *testing.T) {
	t.Parallel()
	result := executeQuery(t, "WITH counts AS (SELECT package_id,count(*) n FROM types GROUP BY package_id) SELECT p.name,n FROM counts JOIN packages p ON p.id=counts.package_id")
	assert.Equal(t, [][]any{{samplePackage, int64(2)}}, result.Rows)
}

func TestSQLQuotedSemicolons(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name, sql string
		want      [][]any
	}{
		{"literal", "SELECT 'semi;colon',1.25,NULL; -- trailing", [][]any{{"semi;colon", 1.25, nil}}},
		{"escaped quote", "/* leading */ SELECT 'it''s;ok' AS [a;b]", [][]any{{"it's;ok"}}},
		{"identifier", "SELECT `name` FROM structs -- trailing", [][]any{{sampleStruct}}},
		{"values", "VALUES (1),(2)", [][]any{{int64(1)}, {int64(2)}}},
	} {
		t.Run(test.name, func(t *testing.T) { t.Parallel(); assert.Equal(t, test.want, executeQuery(t, test.sql).Rows) })
	}
}

func TestEmptyQueryResultPreservesColumns(t *testing.T) {
	t.Parallel()
	result := executeQuery(t, "SELECT name FROM structs WHERE field_count>100")
	assert.Equal(t, []string{nameColumn}, result.Columns)
	assert.NotNil(t, result.Rows)
	assert.Empty(t, result.Rows)
}

func TestDuplicateColumns(t *testing.T) {
	t.Parallel()
	result := executeQuery(t, "SELECT name,name FROM structs")
	assert.Equal(t, []string{nameColumn, nameColumn}, result.Columns)
	assert.Equal(t, [][]any{{sampleStruct, sampleStruct}}, result.Rows)
}

func TestRejectInvalidOrMutatingSQL(t *testing.T) {
	t.Parallel()

	for _, statement := range []string{"", "-- comment", "/* comment */;", "SELECT 'unclosed", "SELECT 1 /* unclosed", "SELECT 1; SELECT 2", "SELECT 1;;", "SELECT 1; PRAGMA query_only=OFF", "DROP TABLE structs", "DELETE FROM types", "PRAGMA query_only=OFF", "ATTACH DATABASE '/tmp/structs-unsafe.db' AS external", "SELECT * FROM missing", "WITH x AS (SELECT 1) DELETE FROM types", "SELECT load_extension('/tmp/no-such-extension')"} {
		t.Run(statement, func(t *testing.T) {
			t.Parallel()

			result, err := query.Execute(context.Background(), sample(), statement)
			require.Error(t, err)
			assert.Empty(t, result.Rows)
		})
	}
}

func TestDatabasesAreIndependent(t *testing.T) {
	t.Parallel()
	populated := executeQuery(t, "SELECT count(*) FROM structs")
	empty, err := query.Execute(context.Background(), report.Report{}, "SELECT count(*) FROM structs")
	require.NoError(t, err)
	assert.Equal(t, [][]any{{int64(1)}}, populated.Rows)
	assert.Equal(t, [][]any{{int64(0)}}, empty.Rows)
}

func TestCanceledQuery(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := query.Execute(ctx, sample(), "SELECT name FROM structs")
	assert.ErrorIs(t, err, context.Canceled)
}
