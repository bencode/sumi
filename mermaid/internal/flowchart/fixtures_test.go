package flowchart

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// TestFixtures reads Mermaid's own flowchart test diagrams (testdata/e2e, from Mermaid
// v12.3.0): each must read, or be refused as not supported yet. Each parent of those
// that read must be a group.
func TestFixtures(t *testing.T) {
	var ok, unsupported int
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
		var e *syntax.Error
		if errors.As(err, &e) && e.IsUnsupported() {
			unsupported++
			return nil
		}
		if err != nil {
			t.Errorf("%s: %v", path, err)
			return nil
		}
		ok++
		checkParents(t, path, d)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d read, %d not supported yet", ok, unsupported)
}

func checkParents(t *testing.T, path string, d *Diagram) {
	groups := map[string]bool{}
	for _, n := range d.Nodes {
		groups[n.ID] = n.Group
	}
	for _, n := range d.Nodes {
		if n.Parent != "" && !groups[n.Parent] {
			t.Errorf("%s: %s has parent %s, which is no group", path, n.ID, n.Parent)
		}
	}
}
