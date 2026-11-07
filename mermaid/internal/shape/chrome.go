package shape

import (
	"math"
	"strconv"
	"strings"
)

// Mermaid sizes a node by the getBBox of the element it draws, so a size is a
// float32 that went through Chrome: JS writes each number of the element's
// attributes as its shortest string, Blink reads the string back as a float32,
// and gfx::RectF holds the box. The functions here follow those steps.

// jsNumber is v as JS's Number.prototype.toString writes it.
func jsNumber(v float64) string {
	if v == 0 {
		return "0"
	}
	if v < 0 {
		return "-" + jsNumber(-v)
	}
	e := strconv.FormatFloat(v, 'e', -1, 64)
	mant, exp, _ := strings.Cut(e, "e")
	digits := strings.Replace(mant, ".", "", 1)
	k := len(digits)
	n, _ := strconv.Atoi(exp)
	n++ // v = 0.digits × 10^n
	switch {
	case k <= n && n <= 21:
		return digits + strings.Repeat("0", n-k)
	case 0 < n && n <= 21:
		return digits[:n] + "." + digits[n:]
	case -6 < n && n <= 0:
		return "0." + strings.Repeat("0", -n) + digits
	}
	sign := "+"
	if n-1 < 0 {
		sign = "-"
	}
	m := digits[:1]
	if k > 1 {
		m += "." + digits[1:]
	}
	return m + "e" + sign + strconv.Itoa(abs(n-1))
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// svgNumber reads s as Blink reads a number of an SVG path's d or a polygon's
// points: GenericParseNumber<float> adds the digits up in float32, which does not
// always give the nearest float32. s is one jsNumber wrote.
func svgNumber(s string) float32 {
	var sign float32 = 1
	if s[0] == '-' {
		sign, s = -1, s[1:]
	}
	mant, exp, hasExp := strings.Cut(s, "e")
	ip, fp, _ := strings.Cut(mant, ".")
	var integer float32
	var mult float32 = 1
	for i := len(ip) - 1; i >= 0; i-- {
		integer += float32(mult * float32(ip[i]-'0'))
		mult *= 10
	}
	var decimal float32
	var frac float32 = 1
	for i := range len(fp) {
		frac *= float32(0.1)
		decimal += float32(float32(fp[i]-'0') * frac)
	}
	number := float32(integer+decimal) * sign
	if hasExp {
		e, _ := strconv.Atoi(exp)
		number *= float32(math.Pow(10, float64(e)))
	}
	return number
}

// svg is v as Blink holds it once JS has written it into an SVG path or points.
func svg(v float64) float32 {
	return svgNumber(jsNumber(v))
}

// A rectF is gfx::RectF, which holds a getBBox.
type rectF struct{ x, y, w, h float32 }

// trivial is gfx::SizeF's kTrivial: a width or height up to it is held as 0.
const trivial = 8 * 0x1p-23

func size(v float32) float32 {
	if v <= trivial {
		return 0
	}
	return v
}

// boundingRect is gfx::BoundingRect: the rect of b, each side one step wider
// when the float32 difference falls short of the far edge.
func boundingRect(b box) rectF {
	return rectF{b.l, b.t, size(span(b.l, b.r)), size(span(b.t, b.b))}
}

func span(lo, hi float32) float32 {
	w := hi - lo
	if lo+w != hi {
		w = math.Nextafter32(w, float32(math.Inf(1)))
	}
	return w
}
