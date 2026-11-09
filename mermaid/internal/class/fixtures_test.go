package class

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// TestFixtures reads Mermaid's own class diagrams (testdata/e2e, from Mermaid v12.3.0):
// each must read, and each parent must be a namespace. Those Mermaid itself refuses, with
// an escaped << where a statement starts, are not in testdata.
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
		namespaces := map[string]bool{}
		for _, n := range d.Nodes {
			namespaces[n.ID] = n.Kind == Namespace
		}
		for _, n := range d.Nodes {
			if n.Parent != "" && !namespaces[n.Parent] {
				t.Errorf("%s: %s has parent %s, which is no namespace", path, n.ID, n.Parent)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
