// Package cli wires command-line options to discovery, analysis, and rendering.
package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/rumpl/structs/internal/analyzer"
	"github.com/rumpl/structs/internal/query"
	"github.com/rumpl/structs/internal/render"
	"github.com/rumpl/structs/internal/report"
	"github.com/rumpl/structs/internal/source"
)

var errSchemaArguments = errors.New("--schema does not accept a query or paths")

type options struct {
	format       string
	includeTests bool
}

func (o *options) analyze(paths []string) (report.Report, error) {
	if err := render.ValidateFormat(o.format); err != nil {
		return report.Report{}, err
	}

	files, err := source.Discover(paths, source.Options{
		IncludeTests: o.includeTests,
	})
	if err != nil {
		return report.Report{}, err
	}

	return analyzer.Analyze(files)
}

func (o *options) query(cmd *cobra.Command, statement string, paths []string) error {
	data, err := o.analyze(paths)
	if err != nil {
		return err
	}

	result, err := query.Execute(cmd.Context(), data, statement)
	if err != nil {
		return err
	}

	return render.WriteQuery(cmd.OutOrStdout(), o.format, result)
}

// NewCommand creates an independent command suitable for execution or testing.
func NewCommand() (*cobra.Command, error) {
	opts := &options{}
	command := &cobra.Command{
		Use:           "structs [SQL] [path ...]",
		Args:          cobra.ArbitraryArgs,
		Short:         "Report or query Go structs, interfaces, and their methods",
		Long:          "Recursively scan Go source directories or individual .go files.\nDefaults to the current directory. Counts directly declared methods only.\nPass a quoted SELECT or WITH query to query an in-memory SQLite database.",
		Example:       "  structs .\n  structs --format html . > report.html\n  structs 'SELECT name, field_count FROM structs ORDER BY field_count DESC' .\n  structs query --schema",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && looksLikeSQL(args[0]) {
				return opts.query(cmd, args[0], args[1:])
			}

			data, err := opts.analyze(args)
			if err != nil {
				return err
			}

			return render.Write(cmd.OutOrStdout(), opts.format, data)
		},
	}
	command.PersistentFlags().StringVarP(&opts.format, "format", "f", render.FormatJSON, "Output format: json, markdown, html")
	command.PersistentFlags().BoolVar(&opts.includeTests, "include-tests", false, "Include _test.go files")

	err := command.RegisterFlagCompletionFunc("format", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return []string{
			render.FormatJSON,
			render.FormatMarkdown,
			render.FormatHTML,
		}, cobra.ShellCompDirectiveNoFileComp
	})
	if err != nil {
		return nil, fmt.Errorf("register format completion: %w", err)
	}

	command.AddCommand(newQueryCommand(opts), newSkillCommand(os.UserHomeDir))

	return command, nil
}

func newQueryCommand(opts *options) *cobra.Command {
	var schema bool

	command := &cobra.Command{
		Use:     "query SQL [path ...]",
		Short:   "Run a read-only SQLite query over Go declarations",
		Example: "  structs query 'SELECT * FROM structs WHERE field_count > 5' .\n  structs query --schema",
		Args: func(cmd *cobra.Command, args []string) error {
			if schema {
				if len(args) != 0 {
					return errSchemaArguments
				}

				return nil
			}

			return cobra.MinimumNArgs(1)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if schema {
				_, err := fmt.Fprint(cmd.OutOrStdout(), query.Schema)

				return err
			}

			return opts.query(cmd, args[0], args[1:])
		},
	}
	command.Flags().BoolVar(&schema, "schema", false, "Print the SQLite schema without scanning source")

	return command
}

// Require whitespace after the SQL keyword, so paths named select or with still
// work. The explicit query command also accepts comments before the statement.
func looksLikeSQL(value string) bool {
	index := strings.IndexFunc(strings.TrimSpace(value), unicode.IsSpace)
	value = strings.TrimSpace(value)

	if index < 0 {
		return false
	}

	switch strings.ToUpper(value[:index]) {
	case "SELECT", "WITH", "VALUES", "EXPLAIN":
		return true
	default:
		return false
	}
}
