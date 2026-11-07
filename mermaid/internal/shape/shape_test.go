package shape

import (
	"math"
	"slices"
	"testing"

	"github.com/bencode/sumi/scene"
)

// What is expected is what Mermaid gave in Chrome: for a label box it measured, the
// size updateNodeBounds took, where it moved the label's top left to, and the
// extent of the outline it drew.

// ends are where an outline's segments end.
func ends(segs []scene.Seg) []scene.Point {
	var out []scene.Point
	for _, s := range segs {
		if s.Op == scene.CubeTo {
			out = append(out, s.Pts[2])
		} else if s.Op != scene.Close {
			out = append(out, s.Pts[0])
		}
	}
	return out
}

func extent(segs []scene.Seg) (lo, hi scene.Point) {
	lo, hi = scene.Point{X: math.Inf(1), Y: math.Inf(1)}, scene.Point{X: math.Inf(-1), Y: math.Inf(-1)}
	for _, p := range ends(segs) {
		lo = scene.Point{X: min(lo.X, p.X), Y: min(lo.Y, p.Y)}
		hi = scene.Point{X: max(hi.X, p.X), Y: max(hi.Y, p.Y)}
	}
	return lo, hi
}

// near allows for the stadium's ends: Go's math.Sin and math.Cos are at times one
// ulp off V8's.
func near(a, b scene.Point) bool {
	return math.Abs(a.X-b.X) < 1e-9 && math.Abs(a.Y-b.Y) < 1e-9
}

func TestSet(t *testing.T) {
	tests := []struct {
		kind   Kind
		w, h   float64 // the label box
		W, H   float64
		label  scene.Point
		lo, hi scene.Point
	}{
		{Rect, 0, 0, 32, 24, scene.Point{X: 0, Y: 0}, scene.Point{X: -16, Y: -12}, scene.Point{X: 16, Y: 12}},
		{Rect, 449.28125, 21, 481.28125, 45, scene.Point{X: -224.640625, Y: -10.5}, scene.Point{X: -240.640625, Y: -22.5}, scene.Point{X: 240.640625, Y: 22.5}},
		{Rect, 120, 126, 152, 150, scene.Point{X: -60, Y: -63}, scene.Point{X: -76, Y: -75}, scene.Point{X: 76, Y: 75}},
		{Rounded, 0, 0, 30, 30, scene.Point{X: 0, Y: 0}, scene.Point{X: -15, Y: -15}, scene.Point{X: 15, Y: 15}},
		{Rounded, 475.703125, 21, 505.703125, 51, scene.Point{X: -237.8515625, Y: -10.5}, scene.Point{X: -252.8515625, Y: -25.5}, scene.Point{X: 252.8515625, Y: 25.5}},
		{Rounded, 120, 147, 150, 177, scene.Point{X: -60, Y: -73.5}, scene.Point{X: -75, Y: -88.5}, scene.Point{X: 75, Y: 88.5}},
		{Stadium, 0, 0, 45.9876708984375, 24, scene.Point{X: 0, Y: 0}, scene.Point{X: -22.993834594408256, Y: -12}, scene.Point{X: 22.993834594408256, Y: 12}},
		{Stadium, 475.703125, 21, 526.9299926757812, 45, scene.Point{X: -237.8515625, Y: -10.5}, scene.Point{X: -263.4650023645155, Y: -22.5}, scene.Point{X: 263.4650023645155, Y: 22.5}},
		{Stadium, 120, 147, 256.5, 171, scene.Point{X: -60, Y: -73.5}, scene.Point{X: -128.25, Y: -85.5}, scene.Point{X: 128.25, Y: 85.5}},
		{Stadium, 120, 105, 214.10789489746094, 129, scene.Point{X: -60, Y: -52.5}, scene.Point{X: -107.05394659582164, Y: -64.5}, scene.Point{X: 107.05394659582164, Y: 64.5}},
		{Subroutine, 0, 0, 44, 12, scene.Point{X: 0, Y: 0}, scene.Point{X: -22, Y: -6}, scene.Point{X: 22, Y: 6}},
		{Subroutine, 502.140625, 21, 546.140625, 33, scene.Point{X: -251.0703125, Y: -10.5}, scene.Point{X: -273.0703125, Y: -16.5}, scene.Point{X: 273.0703125, Y: 16.5}},
		{Subroutine, 120, 126, 164, 138, scene.Point{X: -60, Y: -63}, scene.Point{X: -82, Y: -69}, scene.Point{X: 82, Y: 69}},
		{Cylinder, 0, 0, 24, 36.08053970336914, scene.Point{X: 0, Y: 10}, scene.Point{X: -12, Y: -18.04026845637584}, scene.Point{X: 12, Y: 18.040268456375838}},
		{Cylinder, 314.890625, 21, 338.890625, 99.79048919677734, scene.Point{X: -157.4453125, Y: -0.5}, scene.Point{X: -169.4453125, Y: -49.89524739802621}, scene.Point{X: 169.4453125, Y: 49.89524739802621}},
		{Cylinder, 120, 126, 144, 190.14869689941406, scene.Point{X: -60, Y: -53}, scene.Point{X: -72, Y: -95.07434944237919}, scene.Point{X: 72, Y: 95.07434944237919}},
		{Circle, 0, 0, 32, 32, scene.Point{X: 0, Y: 0}, scene.Point{X: -16, Y: -16}, scene.Point{X: 16, Y: 16}},
		{Circle, 443.171875, 21, 475.6691589355469, 475.6691589355469, scene.Point{X: -221.5859375, Y: -10.5}, scene.Point{X: -237.83457281892268, Y: -237.83457281892268}, scene.Point{X: 237.83457281892268, Y: 237.83457281892268}},
		{Circle, 120, 126, 206, 206, scene.Point{X: -60, Y: -63}, scene.Point{X: -103, Y: -103}, scene.Point{X: 103, Y: 103}},
		{DoubleCircle, 0, 0, 56, 56, scene.Point{X: 0, Y: 0}, scene.Point{X: -28, Y: -28}, scene.Point{X: 28, Y: 28}},
		{DoubleCircle, 515.34375, 21, 571.7714233398438, 571.7714233398438, scene.Point{X: -257.671875, Y: -10.5}, scene.Point{X: -285.8857211363507, Y: -285.8857211363507}, scene.Point{X: 285.8857211363507, Y: 285.8857211363507}},
		{DoubleCircle, 120, 126, 230, 230, scene.Point{X: -60, Y: -63}, scene.Point{X: -115, Y: -115}, scene.Point{X: 115, Y: 115}},
		{Diamond, 0, 0, 30, 30, scene.Point{X: 0, Y: 0}, scene.Point{X: -14.5, Y: -15}, scene.Point{X: 15.5, Y: 15}},
		{Diamond, 515.34375, 21, 566.34375, 566.34375, scene.Point{X: -257.671875, Y: -10.5}, scene.Point{X: -282.671875, Y: -283.171875}, scene.Point{X: 283.671875, Y: 283.171875}},
		{Diamond, 120, 126, 276, 276, scene.Point{X: -60, Y: -63}, scene.Point{X: -137.5, Y: -138}, scene.Point{X: 138.5, Y: 138}},
		{Hexagon, 0, 0, 72, 70, scene.Point{X: 0, Y: 0}, scene.Point{X: -36, Y: -35}, scene.Point{X: 36, Y: 35}},
		{Hexagon, 488.921875, 21, 572.921875, 91, scene.Point{X: -244.4609375, Y: -10.5}, scene.Point{X: -286.4609375, Y: -45.5}, scene.Point{X: 286.4609375, Y: 45.5}},
		{Hexagon, 120, 126, 264, 196, scene.Point{X: -60, Y: -63}, scene.Point{X: -132, Y: -98}, scene.Point{X: 132, Y: 98}},
		{Odd, 0, 0, 48, 24, scene.Point{X: 3, Y: 0}, scene.Point{X: -24, Y: -12}, scene.Point{X: 24, Y: 12}},
		{Odd, 528.5625, 21, 581.8125, 45, scene.Point{X: -258.65625, Y: -10.5}, scene.Point{X: -290.90625, Y: -22.5}, scene.Point{X: 290.90625, Y: 22.5}},
		{Odd, 120, 126, 199.5, 150, scene.Point{X: -41.25, Y: -63}, scene.Point{X: -99.75, Y: -75}, scene.Point{X: 99.75, Y: 75}},
		{Trapezoid, 0, 0, 45, 15, scene.Point{X: 0, Y: 0}, scene.Point{X: -22.5, Y: -7.5}, scene.Point{X: 22.5, Y: 7.5}},
		{Trapezoid, 454.828125, 21, 520.828125, 36, scene.Point{X: -227.4140625, Y: -10.5}, scene.Point{X: -260.4140625, Y: -18}, scene.Point{X: 260.4140625, Y: 18}},
		{Trapezoid, 120, 126, 291, 141, scene.Point{X: -60, Y: -63}, scene.Point{X: -145.5, Y: -70.5}, scene.Point{X: 145.5, Y: 70.5}},
		{InvTrapezoid, 0, 0, 90, 30, scene.Point{X: 0, Y: 0}, scene.Point{X: -45, Y: -15}, scene.Point{X: 45, Y: 15}},
		{InvTrapezoid, 314.890625, 21, 425.890625, 51, scene.Point{X: -157.4453125, Y: -10.5}, scene.Point{X: -212.9453125, Y: -25.5}, scene.Point{X: 212.9453125, Y: 25.5}},
		{InvTrapezoid, 120, 126, 336, 156, scene.Point{X: -60, Y: -63}, scene.Point{X: -168, Y: -78}, scene.Point{X: 168, Y: 78}},
		{LeanRight, 0, 0, 45, 15, scene.Point{X: 0, Y: 0}, scene.Point{X: -22.5, Y: -7.5}, scene.Point{X: 22.5, Y: 7.5}},
		{LeanRight, 528.5625, 21, 594.5625, 36, scene.Point{X: -264.28125, Y: -10.5}, scene.Point{X: -297.28125, Y: -18}, scene.Point{X: 297.28125, Y: 18}},
		{LeanRight, 120, 126, 291, 141, scene.Point{X: -60, Y: -63}, scene.Point{X: -145.5, Y: -70.5}, scene.Point{X: 145.5, Y: 70.5}},
		{LeanLeft, 0, 0, 45, 15, scene.Point{X: 0, Y: 0}, scene.Point{X: -22.5, Y: -7.5}, scene.Point{X: 22.5, Y: 7.5}},
		{LeanLeft, 373.1875, 21, 439.1875, 36, scene.Point{X: -186.59375, Y: -10.5}, scene.Point{X: -219.59375, Y: -18}, scene.Point{X: 219.59375, Y: 18}},
		{LeanLeft, 120, 126, 291, 141, scene.Point{X: -60, Y: -63}, scene.Point{X: -145.5, Y: -70.5}, scene.Point{X: 145.5, Y: 70.5}},
	}
	for _, tt := range tests {
		n := Set(tt.kind, tt.w, tt.h, 15)
		if n.W != tt.W || n.H != tt.H {
			t.Errorf("%v %v x %v: size %v x %v, want %v x %v", tt.kind, tt.w, tt.h, n.W, n.H, tt.W, tt.H)
		}
		if got := (scene.Point{X: n.Label.X - tt.w/2, Y: n.Label.Y - tt.h/2}); got != tt.label {
			t.Errorf("%v %v x %v: label at %v, want %v", tt.kind, tt.w, tt.h, got, tt.label)
		}
		if lo, hi := extent(n.Outline); !near(lo, tt.lo) || !near(hi, tt.hi) {
			t.Errorf("%v %v x %v: outline from %v to %v, want %v to %v", tt.kind, tt.w, tt.h, lo, hi, tt.lo, tt.hi)
		}
	}
}

// A cylinder's outline goes as Mermaid's path: under the middle of the top
// ellipse and back over it, down the left, under the bottom one, up the right.
func TestCylinderOutline(t *testing.T) {
	const w, h = 314.890625, 21
	want := []scene.Point{{X: -169.4453125, Y: -31.631749132675402}, {X: 0, Y: -13.368250867324598}, {X: 169.4453125, Y: -31.631749132675402}, {X: 0, Y: -49.89524739802621}, {X: -169.4453125, Y: -31.631749132675402}, {X: -169.4453125, Y: 31.631749132675402}, {X: 0, Y: 49.89524739802621}, {X: 169.4453125, Y: 31.631749132675402}, {X: 169.4453125, Y: -31.631749132675402}}
	if got := ends(Set(Cylinder, w, h, 15).Outline); !slices.EqualFunc(got, want, near) {
		t.Errorf("%v x %v: %v, want %v", w, h, got, want)
	}
}

// Chrome's label boxes show which shapes Mermaid widens a label in: a circle's
// label of a few letters stays narrower than the least width.
func TestWidensLabel(t *testing.T) {
	want := map[Kind]bool{
		Rect:         true,
		Rounded:      true,
		Stadium:      true,
		Subroutine:   true,
		Cylinder:     true,
		Circle:       false,
		DoubleCircle: false,
		Diamond:      true,
		Hexagon:      true,
		Odd:          true,
		Trapezoid:    true,
		InvTrapezoid: true,
		LeanRight:    true,
		LeanLeft:     true,
	}
	for k, w := range want {
		if k.WidensLabel() != w {
			t.Errorf("%v: WidensLabel %v, want %v", k, k.WidensLabel(), w)
		}
	}
}
