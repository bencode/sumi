package syntax

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/bencode/sumi/scene"
)

// Color reads s as a CSS color, as a browser reads the fill Mermaid writes
// out for it: one of CSS's names, transparent, or rgb(), rgba(), hsl() or
// hsla(), their arguments apart by commas, or by spaces with the alpha after
// a slash. Case does not matter. Hex colors are left out, as Mermaid's
// lines end at a #.
func Color(s string) (scene.Color, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "transparent" {
		return scene.Color{}, true
	}
	if hex, ok := colorNames[s]; ok {
		return scene.Color{R: uint8(hex >> 16), G: uint8(hex >> 8), B: uint8(hex), A: 255}, true
	}
	fn, rest, ok := strings.Cut(s, "(")
	args, ok2 := strings.CutSuffix(rest, ")")
	if !ok || !ok2 {
		return scene.Color{}, false
	}
	vs, legacy, ok := arguments(args)
	if !ok {
		return scene.Color{}, false
	}
	a := 1.0
	if len(vs) == 4 {
		if a, ok = alpha(vs[3], legacy); !ok {
			return scene.Color{}, false
		}
	}
	var r, g, b float64
	switch fn {
	case "rgb", "rgba":
		r, g, b, ok = rgb(vs[:3], legacy)
	case "hsl", "hsla":
		r, g, b, ok = hsl(vs[:3], legacy)
	default:
		ok = false
	}
	if !ok {
		return scene.Color{}, false
	}
	return scene.Color{R: byte255(r), G: byte255(g), B: byte255(b), A: byte255(a)}, true
}

// arguments are a color function's three or four arguments, and whether
// they are apart by commas, the legacy way.
func arguments(s string) ([]string, bool, bool) {
	if strings.Contains(s, ",") {
		vs := strings.Split(s, ",")
		for i, v := range vs {
			vs[i] = strings.TrimSpace(v)
			if vs[i] == "" || strings.ContainsAny(vs[i], " \t/") {
				return nil, true, false
			}
		}
		return vs, true, len(vs) == 3 || len(vs) == 4
	}
	main, a, slash := strings.Cut(s, "/")
	vs := strings.Fields(main)
	if len(vs) != 3 {
		return nil, false, false
	}
	if slash {
		a = strings.TrimSpace(a)
		if a == "" || strings.ContainsAny(a, " \t/") {
			return nil, false, false
		}
		vs = append(vs, a)
	}
	return vs, false, true
}

// rgb is the red, green and blue, from 0 to 1, of an rgb()'s arguments:
// numbers out of 255 or percentages, which the comma syntax does not let
// mix.
func rgb(vs []string, legacy bool) (float64, float64, float64, bool) {
	var out [3]float64
	for i, v := range vs {
		n, unit, ok := number(v, legacy)
		if !ok || unit != "" && unit != "%" || legacy && (unit == "%") != strings.HasSuffix(vs[0], "%") {
			return 0, 0, 0, false
		}
		if unit == "%" {
			out[i] = n / 100
		} else {
			out[i] = n / 255
		}
	}
	return out[0], out[1], out[2], true
}

// degrees are how many degrees one of each unit a hue may have is; a hue
// with no unit is in degrees.
var degrees = map[string]float64{"": 1, "deg": 1, "grad": 0.9, "rad": 180 / math.Pi, "turn": 360}

// hsl is the red, green and blue of an hsl()'s hue, an angle, and its
// saturation and lightness, percentages, or in the space syntax numbers
// out of 100 too (CSS Color 4, section 7.1).
func hsl(vs []string, legacy bool) (float64, float64, float64, bool) {
	h, unit, ok := number(vs[0], legacy)
	deg, known := degrees[unit]
	if !ok || !known {
		return 0, 0, 0, false
	}
	var sl [2]float64
	for i, v := range vs[1:] {
		n, unit, ok := number(v, legacy)
		if !ok || unit != "%" && (legacy || unit != "") {
			return 0, 0, 0, false
		}
		sl[i] = clamp(n / 100)
	}
	s, l := sl[0], sl[1]
	h = math.Mod(math.Mod(h*deg, 360)+360, 360)
	f := func(n float64) float64 {
		k := math.Mod(n+h/30, 12)
		return l - s*min(l, 1-l)*max(-1, min(k-3, 9-k, 1))
	}
	return f(0), f(8), f(4), true
}

// alpha is an alpha from 0 to 1: a number, or a percentage.
func alpha(v string, legacy bool) (float64, bool) {
	n, unit, ok := number(v, legacy)
	switch unit {
	case "":
		return n, ok
	case "%":
		return n / 100, ok
	}
	return 0, false
}

var cssNumber = regexp.MustCompile(`^([+-]?(?:\d+|\d*\.\d+)(?:e[+-]?\d+)?)(%|[a-z]*)$`)

// number reads a CSS number and its unit, % or a word, if it has one. The
// space syntax takes none for a number, zero.
func number(v string, legacy bool) (float64, string, bool) {
	if v == "none" && !legacy {
		return 0, "", true
	}
	m := cssNumber.FindStringSubmatch(v)
	if m == nil {
		return 0, "", false
	}
	n, err := strconv.ParseFloat(m[1], 64)
	return n, m[2], err == nil
}

func clamp(x float64) float64 { return min(max(x, 0), 1) }

func byte255(x float64) uint8 { return uint8(math.Round(clamp(x) * 255)) }
