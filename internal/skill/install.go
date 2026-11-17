// Package skill installs the agent instructions bundled with the executable.
package skill

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed SKILL.md
var instructions []byte

var errEmptyHome = errors.New("home directory is empty")

// Install writes the bundled skill below home. Identical files are left alone;
// other existing files and symlinks are never overwritten.
func Install(home string) (string, error) {
	if strings.TrimSpace(home) == "" {
		return "", errEmptyHome
	}

	path := filepath.Join(home, ".agents", "skills", "structs", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("create skill directory: %w", err)
	}

	file, err := os.OpenFile(filepath.Clean(path), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return existingSkill(path)
	}

	if err != nil {
		return "", fmt.Errorf("create skill: %w", err)
	}

	_, errWrite := file.Write(instructions)

	err = errors.Join(errWrite, file.Close())
	if err != nil {
		return "", errors.Join(fmt.Errorf("write skill: %w", err), os.Remove(path))
	}

	return path, nil
}

func existingSkill(path string) (string, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("inspect existing skill: %w", err)
	}

	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("skill path is not a regular file %q: %w", path, os.ErrExist)
	}

	content, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return "", fmt.Errorf("read existing skill: %w", err)
	}

	if !bytes.Equal(content, instructions) {
		return "", fmt.Errorf("skill %q differs from the bundled version; back it up and remove it before reinstalling: %w", path, os.ErrExist)
	}

	return path, nil
}
