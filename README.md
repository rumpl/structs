# structs

Explore Go codebases with searchable reports and SQL queries.

Find large structs, inspect function signatures, locate fields by type, and see which packages depend on what. Get JSON for scripts and agents, Markdown for documentation, or an interactive HTML report to browse.

## Install

Requires Go 1.22 or later. From this checkout:

```sh
go build -o bin/structs ./cmd/structs
mkdir -p ~/bin
ln -s "$PWD/bin/structs" ~/bin/structs
```

Make sure `~/bin` is on your PATH.

If you have [Task](https://taskfile.dev) installed, run `task install` instead.

## Quick start

```sh
# Report on the current directory.
structs

# Browse an interactive report.
structs --format html . > report.html

# Find the structs with the most fields.
structs 'SELECT package, name, field_count FROM structs ORDER BY field_count DESC LIMIT 20' .
```

Pass directories or individual Go files. With no paths, `structs` uses the current directory.

```sh
structs ./internal ./pkg
structs ./types.go ./methods.go
structs --include-tests .
```

## Agent skill

Install the bundled CLI guide and SQL schema for your agents:

```sh
structs skill install
```

This writes `$HOME/.agents/skills/structs/SKILL.md` without downloading anything. Reinstalling the same version leaves the file unchanged. A different existing skill is preserved; back it up and remove it before installing the bundled version.

## Reports

```sh
structs --format json . > report.json
structs --format markdown . > report.md
structs --format html . > report.html
```

Reports include:

- **Structs:** fields, tags, embedded types, method declarations, and public/private method counts.
- **Interfaces:** method declarations, counts, and embedded types.
- **Functions and methods:** receivers, parameters, results, and generic constraints.
- **Named types and aliases:** declarations and referenced types.
- **Constants and variables:** declarations, type annotations, and initializer expressions.
- **Packages:** import paths, aliases, and import locations.

Definitions include source files and line numbers.

### HTML

Open the generated file in your browser. The report works offline and includes:

- Search across types, fields, methods, and file paths.
- All / Structs / Interfaces filters.
- A collapsible directory → Go file → type sidebar.
- Field and method counts, visibility badges, and expandable declarations.
- Links to individual types.
- Separate sections for package dependencies and additional declarations.

## Ask questions with SQL

Pass a quoted query followed by the paths you want to explore:

```sh
structs 'SELECT name, field_count FROM structs WHERE field_count > 10' .
```

You can also use the explicit `query` command:

```sh
structs query 'SELECT name, method_count FROM interfaces ORDER BY method_count DESC' ./internal
```

### Which structs hold a context?

```sh
structs query "
SELECT s.package, s.name AS struct, f.name AS field, f.type, f.file, f.line
FROM structs s
JOIN fields f ON f.type_id = s.id
WHERE f.type_package = 'context' AND f.type_name = 'Context'
ORDER BY f.file, f.line
" .
```

To include context references inside collections, callbacks, and other nested field types:

```sh
structs query "
SELECT s.name AS struct, f.name AS field, f.type, r.relation
FROM structs s
JOIN fields f ON f.type_id = s.id
JOIN type_references r ON r.field_id = f.id
WHERE r.package = 'context' AND r.name = 'Context'
ORDER BY f.file, f.line
" .
```

### Which functions accept context after the first parameter?

```sh
structs query "
SELECT f.name, f.receiver, p.position, f.file, f.line
FROM functions f
JOIN parameters p ON p.function_id = f.id
WHERE p.role = 'parameter'
  AND p.type_package = 'context'
  AND p.type_name = 'Context'
  AND p.position > 0
ORDER BY f.file, f.line
" .
```

### Which packages import net/http?

```sh
structs query "
SELECT p.name, p.directory
FROM packages p
JOIN package_dependencies d ON d.package_id = p.id
WHERE d.import_path = 'net/http'
ORDER BY p.directory
" .
```

### What methods does a type declare?

```sh
structs query "
SELECT t.name, m.declaration, m.file, m.line
FROM types t
JOIN methods m ON m.type_id = t.id
WHERE t.name = 'Server'
ORDER BY m.file, m.line
" .
```

### Tables and schema

| Table or view | Contents |
| --- | --- |
| `packages` | Package names and directories |
| `structs` | Structs with field and method counts |
| `interfaces` | Interfaces with method counts |
| `types` | All named types and aliases |
| `fields` | Field names, types, tags, and visibility |
| `methods` | Methods linked to their types |
| `functions` | Functions, receiver methods, and interface methods |
| `parameters` | Function inputs and results |
| `type_parameters` | Generic parameters and constraints |
| `type_references` | Referenced types and where they occur within declarations |
| `package_values` | Package constants and variables |
| `embedded_types` | Embedded types and interface constraints |
| `imports` | Import declarations, aliases, and locations |
| `package_dependencies` | Unique direct import paths for each package |

Print every column and relationship:

```sh
structs query --schema
```

### Query output

JSON is the default. Results contain ordered columns and positional rows:

```json
{
  "columns": ["name", "field_count"],
  "rows": [["Server", 8], ["Config", 5]]
}
```

Use Markdown or HTML for a results table:

```sh
structs query --format markdown 'SELECT * FROM interfaces' .
structs query --format html 'SELECT * FROM structs' . > results.html
```

## Project tasks

With [Task](https://taskfile.dev) installed:

```sh
task build    # Build bin/structs.
task install  # Build and link the executable into ~/bin.
task test     # Run race tests and go vet.
task fmt      # Format with gofumpt and format-structs (both must be on PATH).
task lint            # Run rubocop-go and golangci-lint.
task lint:rubocop    # Run every cop from rubocop-go v1.3.0.
task lint:golangci   # Run golangci-lint v2.15.0.
```

Linting requires Go 1.26.1 or later, or automatic Go toolchain downloads enabled. The pinned linters are downloaded on their first run. All 48 cops are enabled in `.rubocop-go.yml`; individual Go-version requirements still apply. Offenses and skipped whole-program analysis fail the task.

`.golangci.yml` enables all golangci-lint linters except deprecated duplicates, `depguard` (no project import allowlist is defined), and `noinlineerr` (inline error handling is allowed). JSON tags are checked against the existing snake_case naming convention. It includes tests and generated files, enables additional vet/staticcheck/gocritic checks, and reports issues without per-linter limits or blanket exclusions. The broad configuration currently reports an existing lint backlog; `task lint` fails until those findings are addressed.

## Help

```sh
structs --help
structs query --help
```
