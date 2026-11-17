---
name: structs
description: Query Go source with the structs CLI using in-memory SQLite. Use for structs, interfaces, nested type references, functions/methods, parameters/results, generic constraints, named types/aliases, package constants/variables, fields/tags, imports, and source locations. Includes the complete query schema and SQL recipes; no schema-discovery command is required.
---

# structs: Go declaration queries

Use this CLI for syntax-based codebase exploration without building the target or resolving dependencies. Prefer focused SQL projections/aggregates over dumping a full report. Verify semantic conclusions in source: this is not type checking or usage analysis.

## Executable and invocation

Locate the executable with `command -v structs`. If it is not on PATH, check `~/bin/structs` or ask for its installed location. To build from a source checkout, run `go build -o bin/structs ./cmd/structs` in that checkout, not the target repository. Requires Go 1.22+, no CGO.

This skill is bundled with the binary. Install it with `structs skill install`; it writes `$HOME/.agents/skills/structs/SKILL.md`. An identical existing skill is left unchanged. A different existing file is preserved: back it up and remove it before installing the bundled version.

```sh
structs query 'SELECT name, field_count FROM structs ORDER BY field_count DESC LIMIT 20' /path/to/project
structs 'SELECT name, method_count FROM interfaces ORDER BY method_count DESC LIMIT 20' /path/to/project
structs query --include-tests --format markdown 'SELECT * FROM structs LIMIT 20' ./pkg ./cmd
```

- Paths follow a single quoted SQL argument. Directories are recursive; multiple paths or individual `.go` files work. Omitted paths mean the current directory. Prefer an explicit target path.
- Shorthand recognizes SELECT/WITH/VALUES/EXPLAIN followed by whitespace, case-insensitively. Use `query` for SQL starting with comments.
- Each invocation scans, loads a fresh private in-memory SQLite database, executes one read-only statement, and closes it. No persistent files/cache/session. CTEs, joins, aggregates, window functions, and SQLite functions work. One trailing semicolon is accepted; multiple statements, writes, ATTACH, and PRAGMA commands are not.
- Results are materialized in memory. Project only needed columns, use LIMIT, and add ORDER BY when ordering matters.
- Quote SQL as one shell argument. Do not interpolate untrusted input into SQL or shell commands. For mixed quotes, use a quoted heredoc:

```sh
structs query "$(cat <<'SQL'
SELECT name, field_count FROM structs
WHERE name LIKE '%Config%'
ORDER BY field_count DESC, file, line LIMIT 30
SQL
)" /path/to/project
```

## Complete query schema (no discovery step needed)

The following covers every table/view and every queryable column. INTEGER booleans use 0/1. `id` columns are integer primary keys. Non-key columns are non-NULL unless marked `?`; a `?` suffix below means nullable (not part of the name). TEXT absence is generally `''`, not NULL.

```text
packages
  id INTEGER PK
  name TEXT
  directory TEXT

types
  id INTEGER PK
  package_id INTEGER -> packages.id
  kind TEXT: struct | interface | named | alias
  name TEXT
  file TEXT
  line INTEGER
  field_count INTEGER?
  method_count INTEGER
  public_method_count INTEGER?
  private_method_count INTEGER?
  expression TEXT
  declaration TEXT
  exported INTEGER

structs VIEW
  id INTEGER -> types.id
  package_id INTEGER -> packages.id
  package TEXT                       -- packages.name
  directory TEXT                     -- packages.directory
  name TEXT
  file TEXT
  line INTEGER
  field_count INTEGER?
  method_count INTEGER
  public_method_count INTEGER?
  private_method_count INTEGER?
  -- Only types.kind = 'struct'; no expression/declaration/exported columns.

interfaces VIEW
  id INTEGER -> types.id
  package_id INTEGER -> packages.id
  package TEXT
  directory TEXT
  name TEXT
  file TEXT
  line INTEGER
  method_count INTEGER
  -- Only types.kind = 'interface'.

fields
  id INTEGER PK
  type_id INTEGER -> types.id
  name TEXT
  type TEXT                          -- original formatted type expression
  type_name TEXT                     -- outer named type through */()/generic wrappers
  type_package TEXT                  -- syntactically associated import path
  tag TEXT                           -- decoded tag, without Go quotes
  file TEXT
  line INTEGER
  embedded INTEGER
  exported INTEGER

methods
  id INTEGER PK
  type_id INTEGER -> types.id
  function_id INTEGER? -> functions.id
  name TEXT
  declaration TEXT
  file TEXT
  line INTEGER
  exported INTEGER

functions
  id INTEGER PK
  package_id INTEGER -> packages.id
  type_id INTEGER? -> types.id
  kind TEXT: function | method | interface_method
  name TEXT
  declaration TEXT
  file TEXT
  line INTEGER
  exported INTEGER
  receiver TEXT                      -- e.g. *Box[T], '' for free function
  receiver_name TEXT                 -- e.g. Box
  pointer_receiver INTEGER

parameters
  id INTEGER PK
  function_id INTEGER -> functions.id
  role TEXT: parameter | result
  position INTEGER                   -- zero-based, separately within each role
  name TEXT                          -- '' if unnamed
  type TEXT
  type_name TEXT
  type_package TEXT
  variadic INTEGER
  file TEXT
  line INTEGER

type_parameters
  id INTEGER PK
  type_id INTEGER? -> types.id
  function_id INTEGER? -> functions.id
  position INTEGER                   -- zero-based
  name TEXT
  constraint_expression TEXT
  file TEXT
  line INTEGER
  -- Exactly one of type_id/function_id is non-NULL.

package_values
  id INTEGER PK
  package_id INTEGER -> packages.id
  kind TEXT: const | var
  name TEXT
  type TEXT                          -- explicit type only, '' if inferred
  initializer TEXT                   -- full RHS list, not evaluated/per-name value
  declaration TEXT
  inherited INTEGER                  -- omitted const RHS inherited from prior spec
  exported INTEGER
  file TEXT
  line INTEGER

type_references
  id INTEGER PK
  package_id INTEGER -> packages.id
  owner_kind TEXT: type | field | parameter | type_parameter | value
  type_id INTEGER? -> types.id
  field_id INTEGER? -> fields.id
  parameter_id INTEGER? -> parameters.id
  type_parameter_id INTEGER? -> type_parameters.id
  value_id INTEGER? -> package_values.id
  name TEXT
  package TEXT                       -- import path, NOT local package name
  relation TEXT                      -- slash-separated syntactic containment path
  file TEXT
  line INTEGER
  -- Exactly one owner FK is non-NULL, corresponding to owner_kind.

embedded_types
  id INTEGER PK
  type_id INTEGER -> types.id
  expression TEXT

imports
  id INTEGER PK
  package_id INTEGER -> packages.id
  path TEXT
  alias TEXT                         -- '' for implicit name; also '_' or '.'
  file TEXT
  line INTEGER

package_dependencies
  package_id INTEGER -> packages.id
  import_path TEXT
  PRIMARY KEY (package_id, import_path)
```

Use this schema directly. `structs query --schema` remains an **optional** troubleshooting check for a different/stale installed version, not a prerequisite. `--help` and `query --help` explain flags.

### Join and interpretation rules

- Package identity is absolute directory plus package name, not name alone. `packages.name` is NOT a Go import path. Do not join dependency paths to it.
- IDs are scan-local; join by foreign keys, not names. Do not persist IDs between scans.
- `types` inventories every package-level type spec. `kind='alias'` takes precedence even for `type A = struct{...}`. `kind='named'` covers other definitions such as `type Count int`, `type Handler func()`, and `type Copy Existing`. Struct/interface detail tables cover non-alias literal struct/interface declarations only.
- `functions` includes free functions, methods on **any** named receiver type, and explicitly declared interface methods. To query only free functions filter `kind='function'`; do not count `methods` plus `functions` as disjoint sets. `methods.function_id` links the legacy method inventory to structured signatures.
- `functions.type_id` is NULL for free functions, missing receiver definitions, or ambiguous same-name build variants. `receiver_name` retains syntactic receiver identity; do not attach an ambiguous method to an arbitrary type.
- `parameters.role` distinguishes inputs/results; positions exclude the receiver. Grouped names produce separate rows. Variadics retain `...T` and `variadic=1`. Unnamed entries have an empty name. Type parameters/constraints have a separate table; generic receiver bindings are retained in `receiver` text, not reported as fresh function type parameters.
- `types.field_count` is NULL except for literal structs. Public/private counts are NULL for interfaces/aliases; named non-struct types count their own declared receiver methods.
- `fields.type_name`/`type_package` identify the outer named type only (including pointer/parenthesis/generic wrappers). Composite slices/maps/functions/etc. have empty direct-type columns. Use `type_references` for nested references.
- Reference `relation` starts at `direct` for a bare named type. Nested steps replace `direct` and compose using `/`: `pointer`, `slice_element`, `array_element`, `map_key`, `map_value`, `channel_element`, `variadic_element`, `type_argument`, `parameter`, `result`, `constraint`, `struct_field`, `interface_member`, `approximation`, `union_term`. Example: `map[string]*http.Client` has `string` at `map_key` and `http.Client` at `map_value/pointer`.
- References include builtin names and type parameters with `package=''`. Empty package means unassociated, not proven builtin. Explicit aliases are file-local; implicit import names are inferred from the path's final component. Dot imports, unusual declared package names, shadowing, and local/transitive aliases are not type-resolved.
- References are per syntactic occurrence and owner. A struct's type-level references overlap with its field-level references; interface type-level references overlap method parameter references. Filter owner_kind or join via one owner FK to avoid double counting. Grouped parameter/field names share syntax and each receive references.
- Value references come from explicit type annotations only, never guessed from initializers. Initializers are whole RHS lists shared by all names in that spec (`var a,b=pair()`), not evaluated values. Omitted const specs inherit previous expressions/types, flagged with `inherited=1`; `iota` is not evaluated. A reference in an inherited annotation points to its original physical source occurrence.

## SQL recipes

### Inventory, definitions, and large structs

```sh
structs query 'SELECT kind, count(*) AS count FROM types GROUP BY kind ORDER BY kind' .
structs query 'SELECT package, name, field_count, method_count, file, line FROM structs ORDER BY field_count DESC, file, line LIMIT 30' .
structs query "SELECT kind, name, expression, file, line FROM types WHERE name='Config' ORDER BY file,line" .
structs query 'SELECT package, directory, count(*) AS structs, sum(field_count) AS fields FROM structs GROUP BY package,directory ORDER BY structs DESC LIMIT 30' .
```

### Direct context fields vs context anywhere in a field type

```sh
# Direct context field, including pointers, embeddings, and import aliases.
structs query "SELECT s.package,s.name AS struct,f.name AS field,f.type,f.file,f.line FROM structs s JOIN fields f ON f.type_id=s.id WHERE f.type_package='context' AND f.type_name='Context' ORDER BY f.file,f.line" .

# Includes slices/maps/channels/callbacks/generic arguments/nested anonymous types.
structs query "SELECT s.name AS struct,f.name AS field,f.type,r.relation,f.file,f.line FROM structs s JOIN fields f ON f.type_id=s.id JOIN type_references r ON r.field_id=f.id WHERE r.package='context' AND r.name='Context' ORDER BY f.file,f.line,r.id" .

structs query "SELECT s.name,f.name,f.tag FROM structs s JOIN fields f ON f.type_id=s.id WHERE f.tag<>'' ORDER BY s.file,f.line" .
```

A callback mentioning context does not store a context value in the same sense as a direct field; preserve the relation in your explanation. No type-reference result by itself proves a design defect.

### Functions with context after the first parameter; error results

```sh
structs query "SELECT f.kind,f.receiver_name,f.name,p.position,f.file,f.line FROM functions f JOIN parameters p ON p.function_id=f.id WHERE f.exported=1 AND p.role='parameter' AND p.type_package='context' AND p.type_name='Context' AND p.position>0 ORDER BY f.file,f.line" .

structs query "SELECT DISTINCT f.kind,f.receiver_name,f.name,f.file,f.line FROM functions f JOIN parameters p ON p.function_id=f.id WHERE p.role='result' AND p.type='error' ORDER BY f.file,f.line" .

structs query "SELECT f.name,p.role,p.position,p.name,p.type,p.variadic FROM functions f JOIN parameters p ON p.function_id=f.id WHERE f.name='Serve' ORDER BY f.id,p.role,p.position" .

structs query "SELECT f.name,f.receiver,f.pointer_receiver FROM functions f WHERE f.kind='method' ORDER BY f.file,f.line LIMIT 30" .
```

`p.type='error'` is an exact syntactic match, not resolved builtin identity or error-implementation checking.

### Aliases, generics, and globals

```sh
structs query "SELECT name,expression,file,line FROM types WHERE kind='alias' ORDER BY file,line" .
structs query 'SELECT t.name,tp.name AS parameter,tp.constraint_expression FROM types t JOIN type_parameters tp ON tp.type_id=t.id ORDER BY t.id,tp.position' .
structs query "SELECT v.name,v.type,v.initializer,v.file,v.line FROM package_values v JOIN type_references r ON r.value_id=v.id WHERE v.kind='var' AND r.package='context' AND r.name='Context' ORDER BY v.file,v.line" .
structs query "SELECT name,type,initializer,inherited FROM package_values WHERE kind='const' ORDER BY file,line" .
```

### Dependencies and embeddings

```sh
structs query 'SELECT p.name,p.directory,d.import_path FROM packages p JOIN package_dependencies d ON d.package_id=p.id ORDER BY p.directory,d.import_path' .
structs query "SELECT p.name,p.directory FROM packages p JOIN package_dependencies d ON d.package_id=p.id WHERE d.import_path='net/http' ORDER BY p.directory" .
structs query 'SELECT path,alias,file,line FROM imports ORDER BY path,file,line' .
structs query 'SELECT t.kind,t.name,e.expression,t.file,t.line FROM embedded_types e JOIN types t ON t.id=e.type_id ORDER BY t.file,t.line,e.id' .
```

Dependencies include blank/dot imports and packages with no type declarations. They are direct source imports, not resolved transitive/module dependencies. Embedded expressions are not expanded method sets.

## Outputs

Default JSON query result preserves ordered and duplicate column names:

```json
{"columns":["name","field_count"],"rows":[["Box",3]]}
```

Rows are positional arrays, not objects. Use aliases for ambiguous columns. Empty results retain columns with `rows: []`; SQL NULL becomes `null`, numbers remain numbers, blobs become base64 strings. An empty result is not a command failure.

```sh
structs query --format markdown 'SELECT * FROM interfaces LIMIT 20' .
structs query --format html 'SELECT * FROM functions LIMIT 50' . > /tmp/function-query.html
structs --format json .
structs --format markdown ./internal
structs --format html . > /tmp/go-report.html
```

Path-only commands generate full reports. JSON has `packages` containing `structs`, `interfaces`, `types` (every type spec), `functions`, `values`, `dependencies`, and `imports`. Signatures expose parameter/result/type-parameter arrays; field/parameter/type/value records contain `references`. Full Markdown/HTML include declarations/signatures/globals as well as the original type reports. HTML has interactive search/filter and a collapsible directory → Go file → type tree for literal structs/interfaces, with matching counts, Expand all / Collapse all, and active-link highlighting. Search opens matching branches and restores expansion state when cleared; extended declarations and package dependencies are separate sections. SQL HTML is a result table. Avoid overwriting user files without permission.

## Scope and troubleshooting

- Only package-level declarations. No function-local types/values, anonymous function inventory, bodies, call graph, usage tracking, value/type inference, evaluated constants, alias expansion, interface conformance, or transitive dependency resolution.
- Value/pointer methods count direct declarations only, not promoted/inherited methods. Interface methods exclude inherited members. Named type methods do not imply interface implementation.
- Scans all selected build variants regardless of GOOS/GOARCH/build tags, including generated files. Duplicates may be mutually exclusive declarations; inspect directory/file/line rather than deduplicating by name.
- `_test.go` excluded unless `--include-tests`; external test packages stay separate.
- `vendor` always excluded, even explicit vendor paths/files. Hidden directories, `testdata`, and `node_modules` skipped recursively but can be explicitly selected. Symlinks not followed.
- Individual file selection does not load sibling method/type files. Prefer package directories for complete direct-declaration inventories.
- Fields count each grouped name and blank field; embedded fields count once. Nested members do not increase the outer field count.
- Source files are absolute; physical 1-based lines ignore `//line` remapping. Read returned locations to verify behavior. Treat paths as data and quote them when opening files.
- Missing paths/syntax errors produce nonzero status and stderr diagnostics, not partial reports. Fix/narrow source rather than interpreting a failure as an empty inventory. Empty scans are valid.
- Unknown tables/columns: confirm the binary is current; the full schema is above. `query --schema` is optional for drift troubleshooting.
- Use the same owner kind when comparing reference counts. Use EXISTS or DISTINCT for inventory questions to avoid multiplying rows through joins.
- Quote the SQL argument. Prefer a few focused queries; every invocation rescans and reloads.
