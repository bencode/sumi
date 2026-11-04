package scene

// Arial measures text from a table of Arial's advance widths, with no font at
// hand: for SVG meant to be viewed in a browser, and for tests, which get the
// same widths on every machine. An East Asian wide character takes an em,
// another character past ASCII 0.58 of one, and monospaced text 0.6 of an em
// a character (1.2 a wide one). Italic text is measured as upright.
//
// The widths of printable ASCII are those of limu (github.com/firede/limu,
// text/measure.go), which took them from Pele's src/text/metrics.ts
// (github.com/obsidianmd/pele); both are MIT licensed, see LICENSE.
var Arial Measurer = arial{}

// Arial's ascender and descender, in ems, from its hhea table (1854 and 434
// of 2048 units).
const arialAscent, arialDescent = 0.905, 0.212

// Advance widths of printable ASCII in thousandths of an em, two base-36
// digits each.
const (
	arialRegular = "7q7q9vfgfgopij5b9999atg87q997q7qfgfgfgfgfgfgfgfgfgfg7q7qg8g8g8fgs7ijijk2k2ijgzlmk27qdwijfgn5k2lmijlmk2ijgzk2ijq8ijijgz7q7q7qd1fg99fgfgdwfgfg7qfgfg6666dw66n5fgfgfgfg99dw7qfgdwk2dwdwdw9a789ag8"
	arialBold    = "7q99d6fgfgopk26m9999atg87q997q7qfgfgfgfgfgfgfgfgfgfg9999g8g8g8gzr3k2k2k2k2ijgzlmk27qfgk2gzn5k2lmijlmk2ijgzk2ijq8ijijgz997q99g8fg99fggzfggzfg99gzgz7q7qfg7qopgzgzgzgzatfg99gzfglmfgfgdwat7satg8"
)

var arialTables = [2][95]uint16{unpack(arialRegular), unpack(arialBold)}

func unpack(packed string) [95]uint16 {
	digit := func(c byte) uint16 {
		if c <= '9' {
			return uint16(c - '0')
		}
		return uint16(c-'a') + 10
	}
	var out [95]uint16
	for i := range out {
		out[i] = digit(packed[2*i])*36 + digit(packed[2*i+1])
	}
	return out
}

type arial struct{}

func (arial) Measure(s string, f Font) Metrics {
	table := &arialTables[0]
	if f.Bold {
		table = &arialTables[1]
	}
	units := 0
	for _, c := range s {
		units += advance(c, table, f.Mono)
	}
	return Metrics{
		Width:   float64(units) * f.Size / 1000,
		Ascent:  arialAscent * f.Size,
		Descent: arialDescent * f.Size,
	}
}

// advance is c's width in thousandths of an em.
func advance(c rune, table *[95]uint16, mono bool) int {
	switch {
	case c < 32 || isZeroWidth(c):
		return 0
	case c < 127 && mono:
		return 600
	case c < 127:
		return int(table[c-32])
	case isWide(c) && mono:
		return 1200
	case isWide(c):
		return 1000
	case mono:
		return 600
	default:
		return 580
	}
}

// isWide reports whether c is set an em wide: CJK and Hangul, full-width forms,
// and characters past the Basic Multilingual Plane, as emoji.
func isWide(c rune) bool {
	return c >= 0x1100 && c <= 0x115f ||
		c >= 0x2e80 && c <= 0xa4cf ||
		c >= 0xac00 && c <= 0xd7a3 ||
		c >= 0xf900 && c <= 0xfaff ||
		c >= 0xfe30 && c <= 0xfe4f ||
		c >= 0xff00 && c <= 0xff60 ||
		c >= 0xffe0 && c <= 0xffe6 ||
		c > 0xffff
}

func isZeroWidth(c rune) bool {
	return c == 0xad ||
		c >= 0x300 && c <= 0x36f ||
		c >= 0x200b && c <= 0x200f ||
		c >= 0xfe00 && c <= 0xfe0f ||
		c == 0xfeff
}
