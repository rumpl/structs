package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"sync"
)

type fileResult struct {
	collector *packageCollector
	err       error
}

// collectFiles bounds both concurrency and live syntax trees. Each worker has
// its own FileSet and collector, avoiding shared AST/file-scope mutation. The
// result slots preserve input order, including which syntax error is returned.
func collectFilesWithWorkers(paths []string, workers int) []fileResult {
	results := make([]fileResult, len(paths))

	workers = min(workers, len(paths))
	if workers <= 1 {
		for i, path := range paths {
			results[i] = collectFile(path)
		}

		return results
	}

	jobs := make(chan int)

	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()

			for index := range jobs {
				results[index] = collectFile(paths[index])
			}
		}()
	}

	for i := range paths {
		jobs <- i
	}

	close(jobs)
	group.Wait()

	return results
}

func collectFile(path string) fileResult {
	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, path, nil, parser.AllErrors|parser.SkipObjectResolution)
	if err != nil {
		return fileResult{err: parseError(path, err)}
	}

	key := packageKey{directory: filepath.Dir(path), name: file.Name.Name}
	collector := newPackageCollector(fset, key)
	ast.Walk(collector, file)

	if collector.err != nil {
		return fileResult{err: collector.err}
	}
	// ASTs and FileSets are no longer needed after extracting source locations.
	collector.fset = nil
	collector.syntaxFormatter = nil

	return fileResult{collector: collector}
}

func (c *packageCollector) merge(file *packageCollector) {
	c.types = append(c.types, file.types...)
	c.functions = append(c.functions, file.functions...)
	c.values = append(c.values, file.values...)

	c.imports = append(c.imports, file.imports...)
	for path := range file.dependencies {
		c.dependencies[path] = true
	}

	for receiver, methods := range file.methods {
		c.methods[receiver] = append(c.methods[receiver], methods...)
	}
}
