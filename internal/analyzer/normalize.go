package analyzer

import (
	"sort"

	"github.com/rumpl/structs/internal/report"
)

// All output-order policy lives here. Declaration extraction never sorts.
// Fields, parameters/results, references, and embedded expressions retain their
// source order because their positions and containment order are meaningful.
func (c *packageCollector) normalize() {
	sort.SliceStable(c.types, func(i, j int) bool {
		return definitionLess(c.types[i].named.Definition, c.types[j].named.Definition)
	})
	sort.SliceStable(c.functions, func(i, j int) bool {
		return sourceLess(c.functions[i].File, c.functions[i].Line, c.functions[j].File, c.functions[j].Line)
	})
	sort.SliceStable(c.values, func(i, j int) bool {
		return definitionLess(c.values[i].Definition, c.values[j].Definition)
	})
	sort.SliceStable(c.imports, func(i, j int) bool {
		return sourceLess(c.imports[i].File, c.imports[i].Line, c.imports[j].File, c.imports[j].Line)
	})
	// Each receiver list is sorted once, then shared by both type inventories.
	for _, methods := range c.methods {
		sort.SliceStable(methods, func(i, j int) bool {
			return definitionLess(methodDefinition(&methods[i]), methodDefinition(&methods[j]))
		})
	}

	for i := range c.types {
		methods := c.types[i].interfaceMethods
		sort.SliceStable(methods, func(i, j int) bool {
			return definitionLess(methods[i].Definition, methods[j].Definition)
		})
	}
}

func normalizePackages(packages []report.Package) {
	sort.Slice(packages, func(i, j int) bool {
		a, b := packages[i], packages[j]
		if a.Directory != b.Directory {
			return a.Directory < b.Directory
		}

		return a.Name < b.Name
	})
}

func normalizeDependencies(dependencies []string) { sort.Strings(dependencies) }

func definitionLess(a, b report.Definition) bool {
	if a.Name != b.Name {
		return a.Name < b.Name
	}

	return sourceLess(a.File, a.Line, b.File, b.Line)
}

func sourceLess(aFile string, aLine int, bFile string, bLine int) bool {
	if aFile != bFile {
		return aFile < bFile
	}

	return aLine < bLine
}

func methodDefinition(method *report.Method) report.Definition {
	return report.Definition{
		Name: method.Name,
		File: method.File,
		Line: method.Line,
	}
}
