package filecache

import (
	"os"
	"path/filepath"
)

func LoadSidebarTree(dir string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "sidebar.json"))
	if err != nil {
		return "[]", nil // default: empty tree
	}
	return string(raw), nil
}

func SaveSidebarTree(dir string, tree string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "sidebar.json"), []byte(tree), 0o644)
}
