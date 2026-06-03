// Package internal holds the gen component implementation: writing a rendered
// brick to disk under a workspace root.
package internal

import (
	"os"
	"path/filepath"
)

// File mirrors architecture.File for the writer.
type File struct {
	RelPath   string
	Content   string
	Generated bool
}

// Write materialises plan under <root>/components/<name>. Generated files
// (the tests) are always (over)written. Scaffold files are written only when
// absent, unless force is set. With dryRun nothing touches the disk but the
// would-be paths are still classified.
func Write(root, name string, plan []File, force, dryRun bool) (dir string, written, skipped []string, err error) {
	dir = filepath.Join(root, "components", name)
	for _, f := range plan {
		path := filepath.Join(dir, filepath.FromSlash(f.RelPath))
		if !f.Generated && !force && exists(path) {
			skipped = append(skipped, path)
			continue
		}
		if !dryRun {
			if err = os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return dir, written, skipped, err
			}
			if err = os.WriteFile(path, []byte(f.Content), 0o644); err != nil {
				return dir, written, skipped, err
			}
		}
		written = append(written, path)
	}
	return dir, written, skipped, nil
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
