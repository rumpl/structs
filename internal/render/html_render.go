package render

import (
	"bytes"
	"fmt"
	"html/template"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/rumpl/structs/internal/report"
)

type htmlFragment struct {
	data   any
	err    error
	output **bytes.Buffer
	name   string
}

// escapeFragment is the single trust boundary for already-rendered fragments.
// The input is produced exclusively by html/template in renderFragment; source
// strings are escaped there before this function can receive them.
func escapedFragment(fragment *bytes.Buffer) template.HTML {
	// #nosec G203 -- only buffers rendered by html/template are passed here; never raw report values.
	return template.HTML(fragment.String())
}

func writeHTML(out *bytes.Buffer, input report.Report) error {
	return writeHTMLWithWorkers(out, input, min(runtime.GOMAXPROCS(0), 8))
}

func writeHTMLWithWorkers(out *bytes.Buffer, input report.Report, workers int) error {
	data, err := htmlData(input)
	if err != nil {
		return err
	}
	// Embedded repository assets are literal template source, not user content.
	source := strings.NewReplacer("/* REPORT_STYLES */", reportCSS, "/* REPORT_SCRIPT */", reportJS).Replace(htmlTemplate)

	html, err := template.New(FormatHTML).Funcs(template.FuncMap{
		"filename": filepath.Base, "methodSignature": methodSignature, "singleLineSignature": singleLineSignature, "fragment": escapedFragment,
	}).Parse(source)
	if err != nil {
		return fmt.Errorf("parse HTML report template: %w", err)
	}

	jobs := htmlJobs(&data, input)
	renderFragments(html, jobs, workers)

	for i := range jobs {
		if jobs[i].err != nil {
			return jobs[i].err
		}
	}

	return html.Execute(out, data)
}

func htmlJobs(data *htmlReport, input report.Report) []htmlFragment {
	data.DependencyPackages = make([]*bytes.Buffer, len(input.Packages))
	data.DeclarationPackages = make([]*bytes.Buffer, len(input.Packages))

	jobs := make([]htmlFragment, 0, len(input.Packages)*4)
	for si := range data.Sections {
		for pi := range data.Sections[si].Packages {
			pkg := &data.Sections[si].Packages[pi]
			jobs = append(jobs, htmlFragment{name: "type-package", data: pkg, output: &pkg.Content})
		}
	}

	for i := range input.Packages {
		pkg := &input.Packages[i]
		jobs = append(jobs, htmlFragment{name: "dependency-package", data: pkg, output: &data.DependencyPackages[i]}, htmlFragment{name: "declaration-package", data: pkg, output: &data.DeclarationPackages[i]})
	}

	return jobs
}

func renderFragments(html *template.Template, jobs []htmlFragment, workers int) {
	workers = max(1, min(workers, len(jobs)))

	var group sync.WaitGroup

	queue := make(chan int)

	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()

			for index := range queue {
				job := &jobs[index]

				fragment := &bytes.Buffer{}
				if job.err = html.ExecuteTemplate(fragment, job.name, job.data); job.err == nil {
					*job.output = fragment
				}
			}
		}()
	}

	for i := range jobs {
		queue <- i
	}

	close(queue)
	group.Wait()
}
