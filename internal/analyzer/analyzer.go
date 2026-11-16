// Package analyzer extracts package-level declarations from Go syntax trees.
package analyzer

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/rumpl/structs/internal/report"
)

type packageKey struct {
	directory string
	name      string
}

// Analyze parses each selected file once. Collection is scoped to package-level
// declarations; method association and output normalization happen afterward.
// Build constraints and type checking are deliberately outside this analyzer.
func Analyze(paths []string) (report.Report, error) {
	return analyzeWithWorkers(paths, min(runtime.GOMAXPROCS(0), 16))
}

func analyzeWithWorkers(paths []string, workers int) (report.Report, error) {
	unique := make([]string, 0, len(paths))

	seen := make(map[string]bool, len(paths))
	for _, path := range paths {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return report.Report{}, fmt.Errorf("resolve %q: %w", path, err)
		}

		if !seen[absolute] {
			seen[absolute] = true
			unique = append(unique, absolute)
		}
	}

	collectors := make(map[packageKey]*packageCollector)

	for _, file := range collectFilesWithWorkers(unique, workers) {
		if file.err != nil {
			return report.Report{}, file.err
		}

		collector := file.collector
		if existing := collectors[collector.key]; existing != nil {
			existing.merge(collector)
		} else {
			collectors[collector.key] = collector
		}
	}

	result := report.Report{Packages: make([]report.Package, 0, len(collectors))}
	for _, collector := range collectors {
		result.Packages = append(result.Packages, collector.report())
	}

	normalizePackages(result.Packages)

	return result, nil
}
