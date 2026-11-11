package mermaid_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/bencode/sumi/mermaid"
	"github.com/bencode/sumi/scene"
)

func TestLayout(t *testing.T) {
	tests := []struct {
		name, src, title string
		nodes, edges     int
	}{
		{"a sequence diagram", "---\ntitle: Greeting\n---\nsequenceDiagram\nAlice->>Bob: Hello", "Greeting", 0, 0},
		{"a flowchart", "---\ntitle: Flow\n---\nflowchart LR\n  A --> B --> C", "Flow", 3, 2},
		{"a state diagram", "---\ntitle: States\n---\nstateDiagram-v2\n  [*] --> A\n  A --> [*]", "States", 3, 2},
		{"flowchart-elk, which Mermaid 12 draws as a flowchart", "flowchart-elk TD\n  A --> B", "", 2, 1},
		{"a gantt diagram", "---\ntitle: Plan\n---\ngantt\ndateFormat YYYY-MM-DD\nA :2014-01-01, 3d", "Plan", 1, 0},
		{"a pie chart", "---\ntitle: Pets\n---\npie\n  \"Dogs\": 3\n  \"Cats\": 2", "Pets", 0, 0},
	}
	for _, tt := range tests {
		s, err := mermaid.Layout(tt.src, mermaid.Options{})
		if err != nil {
			t.Errorf("%s: %v", tt.name, err)
			continue
		}
		count := func(k scene.Kind) int {
			n := 0
			for _, it := range s.Items {
				if it.Kind == k {
					n++
				}
			}
			return n
		}
		hasTitle := tt.title == "" || slices.ContainsFunc(s.Items, func(it scene.Item) bool {
			return it.Kind == scene.Label && len(it.Texts) == 1 && it.Texts[0].Lines[0].S == tt.title
		})
		if !hasTitle || s.Width <= 0 || s.Height <= 0 || tt.nodes > 0 && (count(scene.Node) != tt.nodes || count(scene.Edge) != tt.edges) {
			t.Errorf("%s: scene %vx%v, %d nodes, %d edges, title shown: %v", tt.name, s.Width, s.Height, count(scene.Node), count(scene.Edge), hasTitle)
		}
	}
}

// A gantt diagram is as wide as the options say, 1200 where they do not.
func TestGanttWidth(t *testing.T) {
	for _, w := range []struct{ given, want float64 }{{0, 1200}, {800, 800}} {
		s, err := mermaid.Layout("gantt\ndateFormat YYYY-MM-DD\nA :2014-01-01, 3d", mermaid.Options{Width: w.given})
		if err != nil {
			t.Errorf("width %v: %v", w.given, err)
		} else if s.Width != w.want {
			t.Errorf("width %v: scene %v wide, want %v", w.given, s.Width, w.want)
		}
	}
}

func TestLayoutErrors(t *testing.T) {
	tests := []struct {
		name, src, want string
		unsupported     bool
	}{
		{"an unknown kind", "%% a comment\njourney\n  title T", `line 2:1: diagram type "journey" is not supported yet`, true},
		{"a mistake, counted from the top of the front matter", "---\ntitle: x\n---\nsequenceDiagram\nloop\nend\nend",
			`line 7:1: unexpected "end", expected a statement`, false},
		{"a link to an edge's id, which Mermaid's ELK does not find", "flowchart LR\n  A e1@--> B\n  B --> e1",
			`line 3:4: the link ends at "e1", an edge's id, not a node`, false},
		{"a style sumi does not draw", "flowchart LR\n  A\n  style A fill:#f9f,opacity:0.5",
			`line 3:3: opacity in a style is not supported yet`, true},
		{"a font sumi does not draw", "flowchart LR\n  A\n  style A font-size:20px",
			`line 3:3: font-size in a style is not supported yet`, true},
		{"a color sumi does not read", "flowchart LR\n  A:::c\n  classDef c fill:currentColor",
			`line 3:3: fill:currentColor in a style is not supported yet`, true},
	}
	for _, tt := range tests {
		_, err := mermaid.Layout(tt.src, mermaid.Options{})
		var e *mermaid.Error
		if !errors.As(err, &e) || err.Error() != tt.want || e.IsUnsupported() != tt.unsupported {
			t.Errorf("%s: error %v, want %s", tt.name, err, tt.want)
		}
	}
}
