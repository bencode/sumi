package linebreak

import "sort"

// A class is a line break class of UAX #14; classes.go names them.
type class uint8

// outside is the class of what lies before or after the text: sot and eot.
const outside class = 255

// flags are the properties besides the class that the rules look at.
type flags uint8

const (
	eastAsian              flags = 1 << iota // East_Asian_Width F, W or H: $EastAsian
	initial                                  // a QU in General_Category Pi
	final                                    // a QU in General_Category Pf
	unassignedPictographic                   // Extended_Pictographic and unassigned
)

// A span is a run of code points with one class and the same flags.
type span struct {
	lo, hi rune
	c      class
	f      flags
}

func lookup(r rune) (class, flags) {
	i := sort.Search(len(spans), func(i int) bool { return spans[i].hi >= r })
	return spans[i].c, spans[i].f
}

const dottedCircle = '◌' // the [◌] of LB28a

// A unit is a character with the combining marks LB9 gives it, which the rules
// after LB9 take as one character of its class.
type unit struct {
	c   class
	f   flags
	r   rune // the first character
	zwj bool // the last character is a ZWJ, after which LB8a breaks nothing
	at  int  // byte offset in the text
}

// units is LB9 and LB10: a mark joins the character before it, unless that is a
// break or a space, and a mark left alone is AL.
func units(s string) []unit {
	var u []unit
	for at, r := range s {
		c, f := lookup(r)
		zwj := c == ZWJ
		if c == CM || zwj {
			if n := len(u); n > 0 && !in(u[n-1].c, BK, CR, LF, NL, SP, ZW) {
				u[n-1].zwj = zwj
				continue
			}
			c, f = AL, 0
		}
		u = append(u, unit{c, f, r, zwj, at})
	}
	return u
}

// uax14 returns the byte offsets in s where UAX #14 allows a break, mandatory
// breaks among them; neither 0 nor len(s) is one.
func uax14(s string) map[int]bool {
	u := units(s)
	out := map[int]bool{}
	for k := 1; k < len(u); k++ {
		if allowed(u, k) {
			out[u[k].at] = true
		}
	}
	return out
}

// allowed applies the rules from LB4 on, in order, to the position before u[k].
func allowed(u []unit, k int) bool {
	a, b := u[k-1], u[k]
	switch {
	case a.c == BK: // LB4
		return true
	case a.c == CR && b.c == LF: // LB5
		return false
	case in(a.c, CR, LF, NL):
		return true
	case in(b.c, BK, CR, LF, NL): // LB6
		return false
	case in(b.c, SP, ZW): // LB7
		return false
	case classAt(u, spacesBefore(u, k)) == ZW: // LB8
		return true
	case a.zwj: // LB8a
		return false
	case a.c == WJ || b.c == WJ: // LB11
		return false
	case a.c == GL: // LB12
		return false
	case b.c == GL && !in(a.c, SP, BA, HY, HH): // LB12a
		return false
	case in(b.c, CL, CP, EX, SY): // LB13
		return false
	case classAt(u, spacesBefore(u, k)) == OP: // LB14
		return false
	case openingQuote(u, spacesBefore(u, k)): // LB15a
		return false
	case b.c == QU && b.f&final != 0 && in(classAt(u, k+1), SP, GL, WJ, CL, QU, CP, EX, IS, SY, BK, CR, LF, NL, ZW, outside): // LB15b
		return false
	case a.c == SP && b.c == IS && classAt(u, k+1) == NU: // LB15c
		return true
	case b.c == IS: // LB15d
		return false
	case b.c == NS && in(classAt(u, spacesBefore(u, k)), CL, CP): // LB16
		return false
	case b.c == B2 && classAt(u, spacesBefore(u, k)) == B2: // LB17
		return false
	case a.c == SP: // LB18
		return true
	case b.c == QU && b.f&initial == 0, a.c == QU && a.f&final == 0: // LB19
		return false
	case b.c == QU && (!wide(u, k-1) || !wide(u, k+1)), a.c == QU && (!wide(u, k) || !wide(u, k-2)): // LB19a
		return false
	case a.c == CB || b.c == CB: // LB20
		return true
	case in(a.c, HY, HH) && in(b.c, AL, HL) && in(classAt(u, k-2), outside, BK, CR, LF, NL, SP, ZW, CB, GL): // LB20a
		return false
	case in(b.c, BA, HH, HY, NS), a.c == BB: // LB21
		return false
	case in(a.c, HY, HH) && classAt(u, k-2) == HL && b.c != HL: // LB21a
		return false
	case a.c == SY && b.c == HL: // LB21b
		return false
	case b.c == IN: // LB22
		return false
	case in(a.c, AL, HL) && b.c == NU, a.c == NU && in(b.c, AL, HL): // LB23
		return false
	case a.c == PR && in(b.c, ID, EB, EM), in(a.c, ID, EB, EM) && b.c == PO: // LB23a
		return false
	case in(a.c, PR, PO) && in(b.c, AL, HL), in(a.c, AL, HL) && in(b.c, PR, PO): // LB24
		return false
	case number(u, k): // LB25
		return false
	case a.c == JL && in(b.c, JL, JV, H2, H3), in(a.c, JV, H2) && in(b.c, JV, JT), in(a.c, JT, H3) && b.c == JT: // LB26
		return false
	case in(a.c, JL, JV, JT, H2, H3) && b.c == PO, a.c == PR && in(b.c, JL, JV, JT, H2, H3): // LB27
		return false
	case in(a.c, AL, HL) && in(b.c, AL, HL): // LB28
		return false
	case syllable(u, k): // LB28a
		return false
	case a.c == IS && in(b.c, AL, HL): // LB29
		return false
	case in(a.c, AL, HL, NU) && b.c == OP && b.f&eastAsian == 0, a.c == CP && a.f&eastAsian == 0 && in(b.c, AL, HL, NU): // LB30
		return false
	case a.c == RI && b.c == RI && indicatorsBefore(u, k)%2 == 1: // LB30a
		return false
	case b.c == EM && (a.c == EB || a.f&unassignedPictographic != 0): // LB30b
		return false
	}
	return true // LB31
}

// classAt is the class of u[i], outside for the start and end of the text.
func classAt(u []unit, i int) class {
	if i < 0 || i >= len(u) {
		return outside
	}
	return u[i].c
}

// spacesBefore is the index of the last unit before u[k] that is not SP, -1 if none.
func spacesBefore(u []unit, k int) int {
	i := k - 1
	for i >= 0 && u[i].c == SP {
		i--
	}
	return i
}

// openingQuote is the left side of LB15a: u[i] is an initial quote after the start
// of the text, a break, a space, an opening or another quote.
func openingQuote(u []unit, i int) bool {
	if i < 0 || u[i].c != QU || u[i].f&initial == 0 {
		return false
	}
	return i == 0 || in(u[i-1].c, BK, CR, LF, NL, OP, QU, GL, SP, ZW)
}

// wide reports whether u[i] is East Asian; the start and end of the text are not.
func wide(u []unit, i int) bool {
	return i >= 0 && i < len(u) && u[i].f&eastAsian != 0
}

// number is LB25 for the position before u[k].
func number(u []unit, k int) bool {
	// afterNumber reports whether u[i] ends NU (SY | IS)*.
	afterNumber := func(i int) bool {
		for i >= 0 && in(u[i].c, SY, IS) {
			i--
		}
		return classAt(u, i) == NU
	}
	a, b := u[k-1].c, u[k].c
	switch b {
	case PO, PR:
		return afterNumber(k-1) || in(a, CL, CP) && afterNumber(k-2)
	case NU:
		return in(a, PO, PR, HY, IS) || afterNumber(k-1)
	case OP:
		return in(a, PO, PR) && (classAt(u, k+1) == NU || classAt(u, k+1) == IS && classAt(u, k+2) == NU)
	}
	return false
}

// syllable is LB28a for the position before u[k].
func syllable(u []unit, k int) bool {
	// aksara reports whether u[i] is AK, AS or a dotted circle.
	aksara := func(i int) bool {
		return i >= 0 && i < len(u) && (in(u[i].c, AK, AS) || u[i].r == dottedCircle)
	}
	a, b := u[k-1].c, u[k]
	return a == AP && aksara(k) ||
		aksara(k-1) && in(b.c, VF, VI) ||
		aksara(k-2) && a == VI && (b.c == AK || b.r == dottedCircle) ||
		aksara(k-1) && aksara(k) && classAt(u, k+1) == VF
}

// indicatorsBefore counts the RI units that come right before u[k].
func indicatorsBefore(u []unit, k int) int {
	n := 0
	for i := k - 1; i >= 0 && u[i].c == RI; i-- {
		n++
	}
	return n
}

func in(c class, cs ...class) bool {
	for _, x := range cs {
		if c == x {
			return true
		}
	}
	return false
}
