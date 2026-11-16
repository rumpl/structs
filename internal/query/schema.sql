CREATE TABLE packages (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    directory TEXT NOT NULL
);
CREATE TABLE types (
    id INTEGER PRIMARY KEY,
    package_id INTEGER NOT NULL REFERENCES packages(id),
    kind TEXT NOT NULL CHECK (kind IN ('struct', 'interface', 'named', 'alias')),
    name TEXT NOT NULL,
    file TEXT NOT NULL,
    line INTEGER NOT NULL,
    field_count INTEGER,
    method_count INTEGER NOT NULL,
    public_method_count INTEGER,
    private_method_count INTEGER,
    expression TEXT NOT NULL DEFAULT '',
    declaration TEXT NOT NULL DEFAULT '',
    exported INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE methods (
    id INTEGER PRIMARY KEY,
    type_id INTEGER NOT NULL REFERENCES types(id),
    function_id INTEGER REFERENCES functions(id),
    name TEXT NOT NULL,
    declaration TEXT NOT NULL,
    file TEXT NOT NULL,
    line INTEGER NOT NULL,
    exported INTEGER NOT NULL CHECK (exported IN (0, 1))
);
CREATE TABLE embedded_types (
    id INTEGER PRIMARY KEY,
    type_id INTEGER NOT NULL REFERENCES types(id),
    expression TEXT NOT NULL
);
CREATE TABLE fields (
    id INTEGER PRIMARY KEY,
    type_id INTEGER NOT NULL REFERENCES types(id),
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    type_name TEXT NOT NULL,
    type_package TEXT NOT NULL,
    tag TEXT NOT NULL,
    file TEXT NOT NULL,
    line INTEGER NOT NULL,
    embedded INTEGER NOT NULL CHECK (embedded IN (0, 1)),
    exported INTEGER NOT NULL CHECK (exported IN (0, 1))
);
CREATE TABLE imports (
    id INTEGER PRIMARY KEY,
    package_id INTEGER NOT NULL REFERENCES packages(id),
    path TEXT NOT NULL,
    alias TEXT NOT NULL,
    file TEXT NOT NULL,
    line INTEGER NOT NULL
);
CREATE TABLE package_dependencies (
    package_id INTEGER NOT NULL REFERENCES packages(id),
    import_path TEXT NOT NULL,
    PRIMARY KEY (package_id, import_path)
);
CREATE TABLE functions (
    id INTEGER PRIMARY KEY,
    package_id INTEGER NOT NULL REFERENCES packages(id),
    type_id INTEGER REFERENCES types(id),
    kind TEXT NOT NULL CHECK (kind IN ('function', 'method', 'interface_method')),
    name TEXT NOT NULL,
    declaration TEXT NOT NULL,
    file TEXT NOT NULL,
    line INTEGER NOT NULL,
    exported INTEGER NOT NULL,
    receiver TEXT NOT NULL,
    receiver_name TEXT NOT NULL,
    pointer_receiver INTEGER NOT NULL
);
CREATE TABLE parameters (
    id INTEGER PRIMARY KEY,
    function_id INTEGER NOT NULL REFERENCES functions(id),
    role TEXT NOT NULL CHECK (role IN ('parameter', 'result')),
    position INTEGER NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    type_name TEXT NOT NULL,
    type_package TEXT NOT NULL,
    variadic INTEGER NOT NULL,
    file TEXT NOT NULL,
    line INTEGER NOT NULL
);
CREATE TABLE type_parameters (
    id INTEGER PRIMARY KEY,
    type_id INTEGER REFERENCES types(id),
    function_id INTEGER REFERENCES functions(id),
    position INTEGER NOT NULL,
    name TEXT NOT NULL,
    constraint_expression TEXT NOT NULL,
    file TEXT NOT NULL,
    line INTEGER NOT NULL,
    CHECK ((type_id IS NOT NULL) + (function_id IS NOT NULL) = 1)
);
CREATE TABLE package_values (
    id INTEGER PRIMARY KEY,
    package_id INTEGER NOT NULL REFERENCES packages(id),
    kind TEXT NOT NULL CHECK (kind IN ('const', 'var')),
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    initializer TEXT NOT NULL,
    declaration TEXT NOT NULL,
    inherited INTEGER NOT NULL,
    exported INTEGER NOT NULL,
    file TEXT NOT NULL,
    line INTEGER NOT NULL
);
CREATE TABLE type_references (
    id INTEGER PRIMARY KEY,
    package_id INTEGER NOT NULL REFERENCES packages(id),
    owner_kind TEXT NOT NULL CHECK (owner_kind IN ('type', 'field', 'parameter', 'type_parameter', 'value')),
    type_id INTEGER REFERENCES types(id),
    field_id INTEGER REFERENCES fields(id),
    parameter_id INTEGER REFERENCES parameters(id),
    type_parameter_id INTEGER REFERENCES type_parameters(id),
    value_id INTEGER REFERENCES package_values(id),
    name TEXT NOT NULL,
    package TEXT NOT NULL,
    relation TEXT NOT NULL,
    file TEXT NOT NULL,
    line INTEGER NOT NULL,
    CHECK ((type_id IS NOT NULL) + (field_id IS NOT NULL) + (parameter_id IS NOT NULL) + (type_parameter_id IS NOT NULL) + (value_id IS NOT NULL) = 1)
);
CREATE INDEX references_named_type ON type_references(package, name);
CREATE INDEX parameters_function ON parameters(function_id);
CREATE INDEX functions_package ON functions(package_id);
CREATE INDEX fields_type ON fields(type_id);
CREATE INDEX fields_named_type ON fields(type_package, type_name);
CREATE INDEX imports_package ON imports(package_id);
CREATE INDEX dependencies_path ON package_dependencies(import_path);
CREATE INDEX types_package ON types(package_id);
CREATE INDEX methods_type ON methods(type_id);
CREATE INDEX embedded_types_type ON embedded_types(type_id);
CREATE VIEW structs AS
SELECT t.id, t.package_id, p.name AS package, p.directory,
       t.name, t.file, t.line, t.field_count, t.method_count,
       t.public_method_count, t.private_method_count
FROM types t JOIN packages p ON p.id = t.package_id
WHERE t.kind = 'struct';
CREATE VIEW interfaces AS
SELECT t.id, t.package_id, p.name AS package, p.directory,
       t.name, t.file, t.line, t.method_count
FROM types t JOIN packages p ON p.id = t.package_id
WHERE t.kind = 'interface';
