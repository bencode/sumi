package shape_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/bencode/sumi/mermaid/internal/elk"
	"github.com/bencode/sumi/mermaid/internal/graphlayout"
	"github.com/bencode/sumi/mermaid/internal/shape"
	"github.com/bencode/sumi/scene"
)

// Each graph here is one Mermaid laid out in Chrome, with what it measured for each
// label; what is expected is what Mermaid gave its painter for it, to 0.01 px.
// Chrome's sizes are float32, so a point of a line can lie a hair from the next:
// two points that print alike are one.

func node(id string, k shape.Kind, w, h float64) graphlayout.Node {
	s := shape.Set(k, w, h, 15)
	return graphlayout.Node{ID: id, Width: s.W, Height: s.H, Intersect: func(b graphlayout.Box, p elk.Vec) elk.Vec {
		q := s.Intersect(scene.Point{X: b.X, Y: b.Y}, b.Width, b.Height, scene.Point{X: p.X, Y: p.Y})
		return elk.Vec{X: q.X, Y: q.Y}
	}}
}

func describe(r graphlayout.Result) string {
	var lines []string
	for _, n := range r.Nodes {
		lines = append(lines, fmt.Sprintf("%s %.2f,%.2f %.2fx%.2f", n.ID, n.X, n.Y, n.Width, n.Height))
	}
	for _, e := range r.Edges {
		var b strings.Builder
		b.WriteString(e.ID + ":")
		last := ""
		for _, p := range e.Points {
			if at := fmt.Sprintf(" %.2f,%.2f", p.X, p.Y); at != last {
				b.WriteString(at)
				last = at
			}
		}
		if e.Label != nil {
			fmt.Fprintf(&b, " label %.2f,%.2f", e.Label.X, e.Label.Y)
		}
		lines = append(lines, b.String())
	}
	return strings.Join(lines, "\n")
}

func TestLayout(t *testing.T) {
	tests := []struct {
		name string
		data graphlayout.Data
		want string
	}{
		{
			"Circle, Cylinder, DoubleCircle, InvTrapezoid, LeanLeft, LeanRight, Subroutine",
			graphlayout.Data{Direction: elk.Down, Nodes: []graphlayout.Node{
				node("n0", shape.Cylinder, 120, 21),
				node("n1", shape.InvTrapezoid, 120, 21),
				node("n2", shape.Circle, 120, 42),
				node("n3", shape.LeanRight, 120, 42),
				node("n4", shape.Subroutine, 120, 42),
				node("n5", shape.LeanLeft, 120, 42),
				node("n6", shape.DoubleCircle, 120, 42),
			}, Edges: []graphlayout.Edge{
				{ID: "L_n0_n1_0", Start: "n0", End: "n1", Label: &elk.Size{Width: 41.265625, Height: 21}, StartMarker: "none", EndMarker: "arrow_point"},
				{ID: "L_n0_n2_0", Start: "n0", End: "n2", StartMarker: "none", EndMarker: "arrow_point"},
				{ID: "L_n2_n3_0", Start: "n2", End: "n3", StartMarker: "none", EndMarker: "arrow_point"},
				{ID: "L_n1_n4_0", Start: "n1", End: "n4", StartMarker: "none", EndMarker: "arrow_point"},
				{ID: "L_n4_n5_0", Start: "n4", End: "n5", StartMarker: "none", EndMarker: "arrow_point"},
				{ID: "L_n0_n6_0", Start: "n0", End: "n6", StartMarker: "none", EndMarker: "arrow_point"},
				{ID: "L_n5_n1_0", Start: "n5", End: "n1", StartMarker: "none", EndMarker: "arrow_point"},
				{ID: "L_n3_n1_0", Start: "n3", End: "n1", StartMarker: "none", EndMarker: "arrow_point"},
				{ID: "L_n1_n0_0", Start: "n1", End: "n0", StartMarker: "none", EndMarker: "arrow_point"},
				{ID: "L_n5_n4_0", Start: "n5", End: "n4", StartMarker: "none", EndMarker: "arrow_point"},
			}},
			"n0 171.38,54.57 144.00x85.15\nn1 129.17,502.79 231.00x51.00\nn2 233.62,248.72 159.14x159.14\nn3 233.62,408.79 207.00x57.00\nn4 94.00,595.29 164.00x54.00\nn5 117.38,690.79 207.00x57.00\nn6 444.76,248.72 183.14x183.14\nL_n0_n1_0: 134.18,96.03 134.18,97.15 134.18,117.15 76.42,117.15 76.42,408.79 76.42,477.29 label 76.42,248.72\nL_n0_n2_0: 183.77,96.96 183.77,97.15 183.77,137.15 233.62,137.15 233.62,169.15\nL_n2_n3_0: 233.62,328.29 233.62,380.29\nL_n1_n4_0: 94.00,528.29 94.00,568.29\nL_n4_n5_0: 70.00,622.29 70.00,642.29 70.62,642.29 70.62,662.29\nL_n0_n6_0: 208.57,96.03 208.57,97.15 208.57,117.15 444.76,117.15 444.76,157.15\nL_n5_n1_0: 164.12,662.29 164.12,642.29 196.00,642.29 196.00,595.29 196.00,548.29 164.33,548.29 164.33,528.29\nL_n3_n1_0: 233.62,437.29 233.62,457.29 181.92,457.29 181.92,477.29\nL_n1_n0_0: 129.17,477.29 129.17,457.29 109.12,457.29 109.12,408.79 109.12,248.72 109.12,137.15 158.97,137.15 158.97,97.15\nL_n5_n4_0: 117.38,662.29 117.38,642.29 118.00,642.29 118.00,622.29",
		},
		{
			"Hexagon, Rect, Subroutine",
			graphlayout.Data{Direction: elk.Right, Nodes: []graphlayout.Node{
				node("n0", shape.Hexagon, 120, 21),
				node("n1", shape.Hexagon, 120, 21),
				node("n2", shape.Subroutine, 120, 42),
				node("n3", shape.Rect, 120, 21),
			}, Edges: []graphlayout.Edge{
				{ID: "L_n0_n1_0", Start: "n0", End: "n1", StartMarker: "none", EndMarker: "arrow_point"},
				{ID: "L_n1_n2_0", Start: "n1", End: "n2", StartMarker: "none", EndMarker: "arrow_point"},
				{ID: "L_n1_n3_0", Start: "n1", End: "n3", StartMarker: "none", EndMarker: "arrow_point"},
				{ID: "L_n3_n1_0", Start: "n3", End: "n1", StartMarker: "none", EndMarker: "arrow_point"},
			}},
			"n0 114.00,110.58 204.00x91.00\nn1 358.00,110.58 204.00x91.00\nn2 602.00,39.00 164.00x54.00\nn3 596.00,128.50 152.00x45.00\nL_n0_n1_0: 216.00,110.58 256.00,110.58\nL_n1_n2_0: 449.86,92.83 460.00,92.83 480.00,92.83 480.00,39.00 520.00,39.00\nL_n1_n3_0: 460.00,110.58 500.00,110.58 500.00,124.33 520.00,124.33\nL_n3_n1_0: 520.00,132.67 480.00,132.67 480.00,128.33 460.00,128.33 449.86,128.33",
		},
		{
			"Cylinder, LeanLeft, Rounded",
			graphlayout.Data{Direction: elk.Down, Nodes: []graphlayout.Node{
				node("n0", shape.Rounded, 120, 42),
				node("n1", shape.LeanLeft, 120, 21),
				node("n2", shape.LeanLeft, 120, 21),
				node("n3", shape.Cylinder, 120, 21),
			}, Edges: []graphlayout.Edge{
				{ID: "L_n0_n1_0", Start: "n0", End: "n1", StartMarker: "none", EndMarker: "arrow_point"},
				{ID: "L_n1_n2_0", Start: "n1", End: "n2", StartMarker: "none", EndMarker: "arrow_point"},
				{ID: "L_n2_n3_0", Start: "n2", End: "n3", StartMarker: "none", EndMarker: "arrow_point"},
				{ID: "L_n1_n2_2", Start: "n1", End: "n2", StartMarker: "none", EndMarker: "arrow_point"},
				{ID: "L_n1_n0_0", Start: "n1", End: "n0", StartMarker: "none", EndMarker: "arrow_point"},
				{ID: "L_n2_n1_0", Start: "n2", End: "n1", StartMarker: "none", EndMarker: "arrow_point"},
			}},
			"n0 105.00,48.00 150.00x72.00\nn1 105.00,142.00 186.00x36.00\nn2 105.00,218.00 186.00x36.00\nn3 105.00,318.57 144.00x85.15\nL_n0_n1_0: 83.33,84.00 83.33,104.00 77.33,104.00 77.33,124.00\nL_n1_n2_0: 63.50,160.00 63.50,200.00\nL_n2_n3_0: 105.00,236.00 105.00,276.00\nL_n1_n2_2: 105.00,160.00 105.00,200.00\nL_n1_n0_0: 132.67,124.00 132.67,104.00 126.67,104.00 126.67,84.00\nL_n2_n1_0: 146.50,200.00 146.50,160.00",
		},
		{
			"Diamond, LeanLeft, Odd, Rounded",
			graphlayout.Data{Direction: elk.Down, Nodes: []graphlayout.Node{
				node("n0", shape.Odd, 120, 21),
				node("n1", shape.Diamond, 120, 21),
				node("n2", shape.LeanLeft, 120, 21),
				node("n3", shape.Rounded, 120, 42),
			}, Edges: []graphlayout.Edge{
				{ID: "L_n0_n1_0", Start: "n0", End: "n1", StartMarker: "none", EndMarker: "arrow_point"},
				{ID: "L_n0_n2_0", Start: "n0", End: "n2", Label: &elk.Size{Width: 41.265625, Height: 21}, StartMarker: "none", EndMarker: "arrow_point"},
				{ID: "L_n0_n3_0", Start: "n0", End: "n3", StartMarker: "none", EndMarker: "arrow_point"},
				{ID: "L_n0_n1_2", Start: "n0", End: "n1", StartMarker: "none", EndMarker: "arrow_point"},
			}},
			"n0 183.33,34.50 173.25x45.00\nn1 97.50,202.50 171.00x171.00\nn2 244.00,346.00 186.00x36.00\nn3 379.27,153.00 150.00x72.00\nL_n0_n1_0: 137.36,57.00 137.36,77.00 72.33,77.00 72.33,117.00 72.33,142.17\nL_n0_n2_0: 198.66,57.00 198.66,97.00 244.00,97.00 244.00,328.00 label 243.63,202.50\nL_n0_n3_0: 229.31,57.00 229.31,77.00 379.27,77.00 379.27,117.00\nL_n0_n1_2: 168.01,57.00 168.01,97.00 122.67,97.00 122.67,117.00 122.67,142.17",
		},
		{
			"Circle, DoubleCircle, Rounded, Trapezoid",
			graphlayout.Data{Direction: elk.Left, Nodes: []graphlayout.Node{
				node("n0", shape.Trapezoid, 120, 42),
				node("n1", shape.DoubleCircle, 24.125, 21),
				node("n2", shape.Rounded, 120, 21),
				node("n3", shape.Circle, 24.125, 21),
			}, Edges: []graphlayout.Edge{
				{ID: "L_n0_n1_0", Start: "n0", End: "n1", StartMarker: "none", EndMarker: "arrow_point"},
				{ID: "L_n0_n2_0", Start: "n0", End: "n2", StartMarker: "none", EndMarker: "arrow_point"},
				{ID: "L_n0_n3_0", Start: "n0", End: "n3", StartMarker: "none", EndMarker: "arrow_point"},
				{ID: "L_n2_n1_0", Start: "n2", End: "n1", StartMarker: "none", EndMarker: "arrow_point"},
				{ID: "L_n2_n0_0", Start: "n2", End: "n0", StartMarker: "none", EndMarker: "arrow_point"},
				{ID: "L_n2_n0_2", Start: "n2", End: "n0", StartMarker: "none", EndMarker: "arrow_point"},
			}},
			"n0 453.48,80.83 207.00x57.00\nn1 55.99,55.99 87.98x87.98\nn2 214.98,79.24 150.00x51.00\nn3 257.99,176.73 63.98x63.98\nL_n0_n1_0: 370.40,68.49 349.98,68.49 329.98,68.49 329.98,44.66 214.98,44.66 99.98,44.66\nL_n0_n2_0: 367.32,74.66 349.98,74.66 309.98,74.66 309.98,71.49 289.98,71.49\nL_n0_n3_0: 358.07,93.16 349.98,93.16 329.98,93.16 329.98,176.73 289.98,176.73\nL_n2_n1_0: 139.98,79.24 119.98,79.24 119.98,67.32 99.98,67.32\nL_n2_n0_0: 289.98,79.24 329.98,79.24 329.98,80.83 349.98,80.83 364.23,80.83\nL_n2_n0_2: 289.98,86.99 349.98,86.99 361.15,86.99",
		},
	}
	for _, tt := range tests {
		if got := describe(graphlayout.Layout(tt.data)); got != tt.want {
			t.Errorf("%s:\n got %s\nwant %s", tt.name, got, tt.want)
		}
	}
}
