package mermaid_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/bencode/sumi/mermaid"
	"github.com/bencode/sumi/scene"
)

func TestLayout(t *testing.T) {
	s, err := mermaid.Layout("---\ntitle: Greeting\n---\nsequenceDiagram\nAlice->>Bob: Hello", mermaid.Options{})
	if err != nil {
		t.Fatal(err)
	}
	hasTitle := slices.ContainsFunc(s.Items, func(it scene.Item) bool {
		return it.Kind == scene.Label && len(it.Texts) == 1 && it.Texts[0].Lines[0].S == "Greeting"
	})
	if !hasTitle || s.Width <= 0 || s.Height <= 0 {
		t.Errorf("scene %vx%v, title shown: %v", s.Width, s.Height, hasTitle)
	}
}

func TestLayoutErrors(t *testing.T) {
	tests := []struct {
		name, src, want string
		unsupported     bool
	}{
		{"an unknown kind", "%% a comment\npie\n  \"a\": 1", `line 2:1: diagram type "pie" is not supported yet`, true},
		{"a mistake, counted from the top of the front matter", "---\ntitle: x\n---\nsequenceDiagram\nloop\nend\nend",
			`line 7:1: unexpected "end", expected a statement`, false},
	}
	for _, tt := range tests {
		_, err := mermaid.Layout(tt.src, mermaid.Options{})
		var e *mermaid.Error
		if !errors.As(err, &e) || err.Error() != tt.want || e.IsUnsupported() != tt.unsupported {
			t.Errorf("%s: error %v, want %s", tt.name, err, tt.want)
		}
	}
}
