package query

// Identifiers and placeholder counts are fixed source, never query input.
const (
	insertPackage       = "INSERT INTO packages(name,directory) VALUES (?,?)"
	insertImport        = "INSERT INTO imports(package_id,path,alias,file,line) VALUES (?,?,?,?,?)"
	insertDependency    = "INSERT INTO package_dependencies(package_id,import_path) VALUES (?,?)"
	insertType          = "INSERT INTO types(package_id,kind,name,file,line,field_count,method_count,public_method_count,private_method_count,expression,declaration,exported) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)"
	insertEmbedded      = "INSERT INTO embedded_types(type_id,expression) VALUES (?,?)"
	insertField         = "INSERT INTO fields(type_id,name,type,type_name,type_package,tag,file,line,embedded,exported) VALUES (?,?,?,?,?,?,?,?,?,?)"
	insertFunction      = "INSERT INTO functions(package_id,type_id,kind,name,declaration,file,line,exported,receiver,receiver_name,pointer_receiver) VALUES (?,?,?,?,?,?,?,?,?,?,?)"
	insertParameter     = "INSERT INTO parameters(function_id,role,position,name,type,type_name,type_package,variadic,file,line) VALUES (?,?,?,?,?,?,?,?,?,?)"
	insertTypeParameter = "INSERT INTO type_parameters(type_id,function_id,position,name,constraint_expression,file,line) VALUES (?,?,?,?,?,?,?)"
	insertMethod        = "INSERT INTO methods(type_id,function_id,name,declaration,file,line,exported) VALUES (?,?,?,?,?,?,?)"
	insertValue         = "INSERT INTO package_values(package_id,kind,name,type,initializer,declaration,inherited,exported,file,line) VALUES (?,?,?,?,?,?,?,?,?,?)"
)
