package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errDiagnosticWrite = errors.New("diagnostic write failed")

type failedDiagnostics struct{}

func (failedDiagnostics) Write([]byte) (int, error) { return 0, errDiagnosticWrite }

func TestRunPreservesDiagnosticFailure(t *testing.T) {
	t.Parallel()

	err := run([]string{"--flag-that-does-not-exist"}, failedDiagnostics{})
	require.ErrorIs(t, err, errDiagnosticWrite)
	assert.ErrorContains(t, err, "unknown flag")
}
