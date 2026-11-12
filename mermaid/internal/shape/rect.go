package shape

import (
	"math"

	"github.com/bencode/sumi/scene"
)

// themeRadius rounds a roundedRect's corners: the radius of Mermaid's default
// theme, redux-color.
const themeRadius = 12

// rect is squareRect: 16 to each side of the label and 12 above and below in the
// neo look.
func rect(w, h, _ float64) Node {
	return rectNode(w+16*2, h+12*2, 0)
}

// rounded is roundedRect: padding all round the label.
func rounded(w, h, padding float64) Node {
	return rectNode(w+padding*2, h+padding*2, themeRadius)
}

// Box is a square-cornered node of a size the diagram worked out, as classBox's is.
func Box(w, h float64) Node { return rectNode(w, h, 0) }

func rectNode(w, h, radius float64) Node {
	return Node{
		W:         w,
		H:         h,
		Outline:   scene.RoundRect(scene.Rect{X: -w / 2, Y: -h / 2, W: w, H: h}, radius),
		Intersect: meetRect,
	}
}

// meetRect is intersect-rect.js.
func meetRect(c scene.Point, w, h float64, p scene.Point) scene.Point {
	dx, dy := p.X-c.X, p.Y-c.Y
	hw, hh := w/2, h/2
	var sx, sy float64
	if math.Abs(dy)*hw > math.Abs(dx)*hh {
		if dy < 0 {
			hh = -hh
		}
		if dy != 0 {
			sx = hh * dx / dy
		}
		sy = hh
	} else {
		if dx < 0 {
			hw = -hw
		}
		sx = hw
		if dx != 0 {
			sy = hw * dy / dx
		}
	}
	return scene.Point{X: c.X + sx, Y: c.Y + sy}
}
