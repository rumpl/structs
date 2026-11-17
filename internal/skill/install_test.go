package skill

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func skillPath(home string) string {
	return filepath.Join(home, ".agents", "skills", "structs", "SKILL.md")
}

func TestInstallWritesEmbeddedSkill(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	path, err := Install(home)
	require.NoError(t, err)
	assert.Equal(t, skillPath(home), path)
	content, err := os.ReadFile(filepath.Clean(path))
	require.NoError(t, err)
	assert.Equal(t, instructions, content)
	assert.Contains(t, string(content), "name: structs")
	assert.Contains(t, string(content), "type_references")
}

func TestInstallIdenticalSkillIsIdempotent(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	path, err := Install(home)
	require.NoError(t, err)
	before, err := os.Stat(path)
	require.NoError(t, err)
	installed, err := Install(home)
	require.NoError(t, err)
	after, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, path, installed)
	assert.Equal(t, before.ModTime(), after.ModTime())
}

func TestInstallPreservesCustomSkill(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	path, err := Install(home)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte("custom skill"), 0o600))

	_, err = Install(home)
	require.ErrorIs(t, err, os.ErrExist)
	content, err := os.ReadFile(filepath.Clean(path))
	require.NoError(t, err)
	assert.Equal(t, "custom skill", string(content))
}

func TestInstallRejectsSymlink(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	path := skillPath(home)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))

	target := filepath.Join(home, "target")
	require.NoError(t, os.WriteFile(target, instructions, 0o600))
	require.NoError(t, os.Symlink(target, path))

	_, err := Install(home)
	assert.ErrorIs(t, err, os.ErrExist)
}

func TestInstallRejectsEmptyHome(t *testing.T) {
	t.Parallel()

	_, err := Install("")
	assert.ErrorIs(t, err, errEmptyHome)
}

func TestInstallPropagatesDirectoryFailure(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(home, ".agents"), []byte("file"), 0o600))
	_, err := Install(home)
	assert.ErrorContains(t, err, "create skill directory")
}
