package state

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// TestFixtures reads Mermaid's own state diagrams (testdata/e2e, from Mermaid v12.3.0):
// each must read, and each parent must be a group.
func TestFixtures(t *testing.T) {
	err := filepath.WalkDir("testdata/e2e", func(path string, de fs.DirEntry, err error) error {
		if err != nil || de.IsDir() || !strings.HasSuffix(path, ".mmd") {
			return err
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		body, _ := syntax.Prepare(string(src))
		d, err := Read(body)
		if err != nil {
			t.Errorf("%s: %v", path, err)
			return nil
		}
		groups := map[string]bool{}
		for _, n := range d.Nodes {
			groups[n.ID] = n.Group
		}
		for _, n := range d.Nodes {
			if n.Parent != "" && !groups[n.Parent] {
				t.Errorf("%s: %s has parent %s, which is no group", path, n.ID, n.Parent)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
