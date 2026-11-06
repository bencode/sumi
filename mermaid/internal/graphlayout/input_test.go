package graphlayout

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bencode/sumi/mermaid/internal/elk"
)

// Every graph here is one Mermaid laid out, and what is expected is what Mermaid
// handed ELK for it, unless it says otherwise.

func leaf(id, parent string) Node { return Node{ID: id, Parent: parent} }

func group(id, parent string, direction elk.Direction, title *elk.Size) Node {
	return Node{ID: id, Parent: parent, Direction: direction, Title: title, Padding: 8}
}

func edge(id, start, end string) Edge { return Edge{ID: id, Start: start, End: end} }

func labelled(id, start, end string, width, height float64) Edge {
	return Edge{id, start, end, &elk.Size{Width: width, Height: height}}
}

var directionNames = map[elk.Direction]string{elk.Down: "down", elk.Up: "up", elk.Right: "right", elk.Left: "left"}

var hierarchyNames = map[elk.Hierarchy]string{elk.Include: "include", elk.Separate: "separate"}

func writeNodes(b *strings.Builder, all []elk.InputNode, depth int) {
	for _, n := range all {
		fmt.Fprintf(b, "\n%s%s", strings.Repeat("  ", depth), n.ID)
		if n.Width != 0 || n.Height != 0 {
			fmt.Fprintf(b, " %vx%v", n.Width, n.Height)
		}
		if name := hierarchyNames[n.Hierarchy]; name != "" {
			b.WriteString(" " + name)
		}
		if name := directionNames[n.Direction]; name != "" {
			b.WriteString(" " + name)
		}
		if n.Title != nil {
			fmt.Fprintf(b, " title %vx%v", n.Title.Width, n.Title.Height)
		}
		if n.MinWidth != 0 {
			fmt.Fprintf(b, " min %v", n.MinWidth)
		}
		writeNodes(b, n.Children, depth+1)
	}
}

func describe(in elk.Input, reversed map[string]bool) string {
	var b strings.Builder
	b.WriteString(directionNames[in.Direction])
	writeNodes(&b, in.Nodes, 0)
	for _, e := range in.Edges {
		fmt.Fprintf(&b, "\n%s: %s -> %s", e.ID, e.Source, e.Target)
		if e.Label != nil {
			fmt.Fprintf(&b, " label %vx%v", e.Label.Width, e.Label.Height)
		}
		if reversed[e.ID] {
			b.WriteString(" reversed")
		}
	}
	return b.String()
}

func TestElkInput(t *testing.T) {
	cases := []struct {
		name string
		data Data
		want string
	}{
		// Mermaid falls back on DOWN itself; no diagram in the captures leaves it out.
		{"no direction reads as down", Data{Nodes: []Node{{ID: "a", Width: 10, Height: 20}}}, "down\na 10x20"},
		{"feedback edges turn round and move", Data{elk.Up, []Node{
			group("s1", "", elk.Undefined, &elk.Size{Width: 21, Height: 21}),
			group("s0", "", elk.Up, &elk.Size{Width: 3.890625, Height: 21}),
			leaf("n1", "s0"),
			leaf("n2", "s0"),
			leaf("n0", ""),
		}, []Edge{
			labelled("L_s0_n0_0", "s0", "n0", 42.828125, 21),
			labelled("L_s1_n0_0", "s1", "n0", 42.828125, 21),
			labelled("L_n0_s1_0", "n0", "s1", 42.828125, 21),
			edge("L_n0_n1_0", "n0", "n1"),
		}}, `up
s1
s0 include up title 3.890625x19 min 11.890625
  n1 include
  n2
n0 include
L_n0_n1_0: n1 -> n0 reversed
L_s0_n0_0: s0 -> n0 label 42.828125x21
L_s1_n0_0: s1 -> n0 label 42.828125x21
L_n0_s1_0: n0 -> s1 label 42.828125x21`},
		{"edges to a border and inside it", Data{elk.Right, []Node{
			group("s0", "", elk.Undefined, &elk.Size{Width: 31.125, Height: 21}),
			leaf("n1", "s0"),
			leaf("n2", "s0"),
			leaf("n0", ""),
		}, []Edge{
			edge("L_n2_n1_0", "n2", "n1"),
			edge("L_n1_n1_0", "n1", "n1"),
			edge("L_n2_s0_0", "n2", "s0"),
		}}, `right
s0 include title 31.125x19 min 39.125
  n1
  n2 include
n0
L_n2_n1_0: n2 -> n1
L_n1_n1_0: n1 -> n1
L_n2_s0_0: n2 -> s0`},
		{"empty groups and edges into depth", Data{elk.Left, []Node{
			group("s3", "", elk.Undefined, &elk.Size{Width: 105, Height: 21}),
			group("s2", "", elk.Down, &elk.Size{Width: 3.890625, Height: 21}),
			group("s0", "", elk.Undefined, &elk.Size{Width: 31.125, Height: 21}),
			group("s1", "s0", elk.Undefined, &elk.Size{Width: 35, Height: 21}),
			leaf("n2", "s1"),
			leaf("n0", ""),
			leaf("n1", ""),
		}, []Edge{
			edge("L_n0_s1_0", "n0", "s1"),
			edge("L_n0_n1_0", "n0", "n1"),
			edge("L_n1_n2_0", "n1", "n2"),
			edge("L_n2_s3_0", "n2", "s3"),
			edge("L_n0_n2_0", "n0", "n2"),
			edge("L_n2_n0_0", "n2", "n0"),
			edge("L_n0_s2_0", "n0", "s2"),
		}}, `left
s3 include
s2
s0 include title 31.125x19 min 39.125
  s1 include title 35x19 min 43
    n2 include
n0 include
n1 include
L_n0_n2_0: n0 -> n2
L_n2_n0_0: n0 -> n2 reversed
L_n0_s1_0: n0 -> s1
L_n0_n1_0: n0 -> n1
L_n1_n2_0: n1 -> n2
L_n2_s3_0: n2 -> s3
L_n0_s2_0: n0 -> s2`},
		{"a subgraph without a title", Data{elk.Down, []Node{
			group("s0", "", elk.Undefined, nil),
			leaf("a", "s0"),
			leaf("b", "s0"),
			leaf("c", ""),
		}, []Edge{
			edge("L_a_b_0", "a", "b"),
			edge("L_b_c_0", "b", "c"),
		}}, `down
s0 include min 8
  a
  b include
c include
L_a_b_0: a -> b
L_b_c_0: b -> c`},
		{"edges of a subgraph and of its nodes", Data{elk.Right, []Node{
			group("s1", "", elk.Undefined, &elk.Size{Width: 31.125, Height: 21}),
			group("s0", "", elk.Up, &elk.Size{Width: 31.125, Height: 21}),
			leaf("n1", "s0"),
			leaf("n2", "s0"),
			leaf("n0", "s1"),
		}, []Edge{
			edge("L_s0_s1_0", "s0", "s1"),
			edge("L_n1_n2_0", "n1", "n2"),
		}}, `right
s1 title 31.125x19 min 39.125
  n0
s0 separate up title 31.125x19 min 39.125
  n1
  n2
L_s0_s1_0: s0 -> s1
L_n1_n2_0: n1 -> n2`},
		{"the search starts at sources", Data{elk.Left, []Node{
			group("s1", "", elk.Right, &elk.Size{Width: 147, Height: 21}),
			group("s0", "", elk.Undefined, &elk.Size{Width: 31.125, Height: 21}),
			leaf("n1", "s0"),
			leaf("n3", "s1"),
			leaf("n4", "s1"),
			leaf("n0", ""),
			leaf("n2", ""),
			leaf("n5", ""),
		}, []Edge{
			labelled("L_n4_n4_0", "n4", "n4", 25.6875, 21),
			edge("L_n3_n4_0", "n3", "n4"),
			edge("L_n0_n5_0", "n0", "n5"),
			edge("L_n2_s1_0", "n2", "s1"),
			edge("L_n0_n3_0", "n0", "n3"),
			edge("L_s0_n1_0", "s0", "n1"),
			edge("L_n4_n0_0", "n4", "n0"),
		}}, `left
s1 include right title 147x19 min 155
  n3 include
  n4 include
s0 include title 31.125x19 min 39.125
  n1 include
n0 include
n2
n5
L_n4_n4_0: n4 -> n4 label 25.6875x21
L_n3_n4_0: n3 -> n4
L_n0_n5_0: n0 -> n5
L_n2_s1_0: n2 -> s1
L_n0_n3_0: n3 -> n0 reversed
L_s0_n1_0: s0 -> n1
L_n4_n0_0: n4 -> n0`},
		{"edges stop including at their ancestor", Data{elk.Down, []Node{
			group("s0", "", elk.Undefined, &elk.Size{Width: 189, Height: 21}),
			group("s1", "s0", elk.Undefined, &elk.Size{Width: 31.125, Height: 21}),
			group("s2", "s1", elk.Undefined, &elk.Size{Width: 3.890625, Height: 21}),
			leaf("n4", "s2"),
			leaf("n5", "s2"),
			leaf("n6", "s2"),
			leaf("n1", "s1"),
			leaf("n2", "s1"),
			leaf("n3", "s1"),
			leaf("n7", "s1"),
			leaf("n9", "s1"),
			leaf("n0", "s0"),
			leaf("n8", ""),
		}, []Edge{
			labelled("L_n4_n6_0", "n4", "n6", 25.6875, 21),
			edge("L_n6_n4_0", "n6", "n4"),
			labelled("L_n6_n6_0", "n6", "n6", 25.6875, 21),
			labelled("L_n3_n7_0", "n3", "n7", 25.6875, 21),
			edge("L_s2_n1_0", "s2", "n1"),
			edge("L_s1_n7_0", "s1", "n7"),
		}}, `down
s0 title 189x19 min 197
  s1 include title 31.125x19 min 39.125
    s2 title 3.890625x19 min 11.890625
      n4
      n5
      n6
    n1
    n2
    n3
    n7 include
    n9
  n0
n8
L_n4_n6_0: n4 -> n6 label 25.6875x21
L_n6_n4_0: n6 -> n4
L_n6_n6_0: n6 -> n6 label 25.6875x21
L_n3_n7_0: n3 -> n7 label 25.6875x21
L_s2_n1_0: s2 -> n1
L_s1_n7_0: s1 -> n7`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			in, reversed := elkInput(c.data)
			if got := describe(in, reversed); got != c.want {
				t.Errorf("got\n%s\nwant\n%s", got, c.want)
			}
		})
	}
}
