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
// label; what is expected is what Mermaid gave its painter for it.

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
		lines = append(lines, fmt.Sprintf("%s %v,%v %vx%v", n.ID, n.X, n.Y, n.Width, n.Height))
	}
	for _, e := range r.Edges {
		var b strings.Builder
		b.WriteString(e.ID + ":")
		for _, p := range e.Points {
			fmt.Fprintf(&b, " %v,%v", p.X, p.Y)
		}
		if e.Label != nil {
			fmt.Fprintf(&b, " label %v,%v", e.Label.X, e.Label.Y)
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
			"n0 171.375,54.57434844970703 144x85.14869689941406\nn1 129.16666666666669,502.7864227294922 231x51\nn2 233.625,248.71755981445312 159.13772583007812x159.13772583007812\nn3 233.625,408.7864227294922 207x57\nn4 94,595.2864227294922 164x54\nn5 117.375,690.7864227294922 207x57\nn6 444.7627258300781,248.71755981445312 183.13772583007812x183.13772583007812\nL_n0_n1_0: 134.175,96.031449941358 134.175,97.14869689941406 134.175,117.14869689941406 76.41666666666669,117.14869689941406 76.41666666666669,408.7864227294922 76.41666666666669,477.2864227294922 label 76.41666666666669,248.71755981445312\nL_n0_n2_0: 183.77499999999998,96.96286125864572 183.77499999999998,97.14869689941406 183.77499999999998,137.14869689941406 233.625,137.14869689941406 233.625,169.14869689941406\nL_n2_n3_0: 233.625,328.2864201157044 233.625,328.2864227294922 233.625,380.2864227294922\nL_n1_n4_0: 94.00000000000003,528.2864227294922 94,568.2864227294922\nL_n4_n5_0: 70,622.2864227294922 70,642.2864227294922 70.625,642.2864227294922 70.625,662.2864227294922\nL_n0_n6_0: 208.575,96.031449941358 208.575,97.14869689941406 208.575,117.14869689941406 444.7627258300781,117.14869689941406 444.7627258300781,157.14869689941406\nL_n5_n1_0: 164.125,662.2864227294922 164.125,642.2864227294922 196,642.2864227294922 196,595.2864227294922 196,548.2864227294922 164.33333333333337,548.2864227294922 164.33333333333337,528.2864227294922\nL_n3_n1_0: 233.625,437.2864227294922 233.625,457.2864227294922 181.91666666666669,457.2864227294922 181.91666666666669,477.2864227294922\nL_n1_n0_0: 129.16666666666669,477.2864227294922 129.16666666666669,457.2864227294922 109.125,457.2864227294922 109.125,408.7864227294922 109.125,248.71755981445312 109.125,137.14869689941406 158.975,137.14869689941406 158.975,97.14869689941406\nL_n5_n4_0: 117.375,662.2864227294922 117.375,642.2864227294922 118,642.2864227294922 118,622.2864227294922",
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
			"n0 114,110.58333333333334 204x91\nn1 358,110.58333333333334 204x91\nn2 602,39 164x54\nn3 596,128.5 152x45\nL_n0_n1_0: 216,110.58333333333334 256,110.58333333333334\nL_n1_n2_0: 449.8570613861084,92.83333333333334 460,92.83333333333334 480,92.83333333333334 480,39 520,39\nL_n1_n3_0: 460,110.58333333333334 500,110.58333333333334 500,124.33333333333334 520,124.33333333333334\nL_n3_n1_0: 520,132.66666666666666 480,132.66666666666666 480,128.33333333333334 460,128.33333333333334 449.8570613861084,128.33333333333334",
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
			"n0 105.00000000000003,48 150x72\nn1 105,142 186x36\nn2 105,218 186x36\nn3 105,318.57434844970703 144x85.14869689941406\nL_n0_n1_0: 83.33333333333334,84 83.33333333333334,104 77.33333333333334,104 77.33333333333334,124\nL_n1_n2_0: 63.5,160 63.5,200\nL_n2_n3_0: 105,236 105,276\nL_n1_n2_2: 105,160 105,200\nL_n1_n0_0: 132.66666666666669,124 132.66666666666669,104 126.66666666666669,104 126.66666666666669,84\nL_n2_n1_0: 146.5,200 146.5,160",
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
			"n0 183.33333333333334,34.5 173.25x45\nn1 97.5,202.5 171x171\nn2 244,346 186x36\nn3 379.265625,153 150x72\nL_n0_n1_0: 137.35833333333335,57 137.35833333333335,77 72.33333333333333,77 72.33333333333333,117 72.33333333333333,142.16673374176025\nL_n0_n2_0: 198.65833333333333,57 198.65833333333333,97 244,97 244,328 label 243.6328125,202.5\nL_n0_n3_0: 229.30833333333334,57 229.30833333333334,77 379.265625,77 379.265625,117\nL_n0_n1_2: 168.00833333333333,57 168.00833333333333,97 122.66666666666666,97 122.66666666666666,117 122.66666666666666,142.16673374176025",
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
			"n0 453.484619140625,80.82564290364584 207x57\nn1 55.9923095703125,55.9923095703125 87.984619140625x87.984619140625\nn2 214.984619140625,79.2423095703125 150x51\nn3 257.99231147766113,176.7346172332764 63.984615325927734x63.984615325927734\nL_n0_n1_0: 370.40131521224976,68.4923095703125 349.984619140625,68.4923095703125 329.984619140625,68.4923095703125 329.984619140625,44.66153971354167 214.984619140625,44.66153971354167 99.984619140625,44.66153971354167\nL_n0_n2_0: 367.31795930862427,74.65897623697917 349.984619140625,74.65897623697917 309.984619140625,74.65897623697917 309.984619140625,71.4923095703125 289.984619140625,71.4923095703125\nL_n0_n3_0: 358.06799030303955,93.15897623697916 349.984619140625,93.15897623697916 329.984619140625,93.15897623697916 329.984619140625,176.73461723327637 289.984619140625,176.73461723327637\nL_n2_n1_0: 139.984619140625,79.2423095703125 119.984619140625,79.2423095703125 119.984619140625,67.32307942708334 99.984619140625,67.32307942708334\nL_n2_n0_0: 289.984619140625,79.2423095703125 329.984619140625,79.2423095703125 329.984619140625,80.82564290364584 349.984619140625,80.82564290364584 364.2347021102905,80.82564290364584\nL_n2_n0_2: 289.984619140625,86.9923095703125 349.984619140625,86.9923095703125 361.15134620666504,86.9923095703125",
		},
	}
	for _, tt := range tests {
		if got := describe(graphlayout.Layout(tt.data)); got != tt.want {
			t.Errorf("%s:\n got %s\nwant %s", tt.name, got, tt.want)
		}
	}
}
