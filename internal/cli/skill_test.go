package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const installSubcommand = "install"

var errSkillTest = errors.New("skill command failure")

type failedSkillOutput struct{}

func (failedSkillOutput) Write([]byte) (int, error) { return 0, errSkillTest }

func TestSkillInstallCommand(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	command := newSkillCommand(func() (string, error) { return home, nil })

	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{installSubcommand})
	require.NoError(t, command.Execute())

	path := filepath.Join(home, ".agents", "skills", "structs", "SKILL.md")
	assert.Equal(t, path+"\n", output.String())
	assert.FileExists(t, path)
}

func TestSkillInstallPropagatesHomeError(t *testing.T) {
	t.Parallel()

	command := newSkillCommand(func() (string, error) { return "", errSkillTest })
	command.SetArgs([]string{installSubcommand})
	assert.ErrorIs(t, command.Execute(), errSkillTest)
}

func TestSkillInstallPropagatesOutputError(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	command := newSkillCommand(func() (string, error) { return home, nil })
	command.SetOut(failedSkillOutput{})
	command.SetArgs([]string{installSubcommand})
	assert.ErrorIs(t, command.Execute(), errSkillTest)
}

func TestSkillInstallRejectsArguments(t *testing.T) {
	t.Parallel()

	command := newSkillCommand(func() (string, error) { return t.TempDir(), nil })
	command.SetArgs([]string{installSubcommand, "unexpected"})
	assert.Error(t, command.Execute())
}

func TestRootRegistersSkillCommand(t *testing.T) {
	t.Parallel()

	command, err := NewCommand()
	require.NoError(t, err)

	var output bytes.Buffer
	command.SetOut(&output)
	command.SetArgs([]string{"skill", installSubcommand, "--help"})
	require.NoError(t, command.Execute())
	assert.Contains(t, output.String(), "structs skill install")
}

func TestSkillInstallPreservesExistingFile(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	path := filepath.Join(home, ".agents", "skills", "structs", "SKILL.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte("custom"), 0o600))

	command := newSkillCommand(func() (string, error) { return home, nil })

	var output strings.Builder
	command.SetOut(&output)
	command.SetArgs([]string{installSubcommand})
	require.ErrorIs(t, command.Execute(), os.ErrExist)
	assert.Empty(t, output.String())
}
