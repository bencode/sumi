package scene

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// A Font is how text is set. It names no typeface: the Measurer, and the
// backend that draws, choose one for its kind.
type Font struct {
	Size   float64
	Bold   bool
	Italic bool
	Mono   bool
}

// Metrics are the measures of a line of text: its advance width, and how far
// its font reaches above and below the baseline.
type Metrics struct {
	Width           float64
	Ascent, Descent float64
}

// A Measurer measures text as the backend that draws it will set it, so a
// diagram is laid out in the font it is drawn in.
type Measurer interface {
	Measure(s string, f Font) Metrics
}

// A Text is text laid out in lines, each placed on its baseline.
type Text struct {
	Font  Font
	Lines []Line
	Paint Paint
}

// A Line is a line of text, starting at X on the baseline Baseline.
type Line struct {
	S        string
	X        float64
	Baseline float64
}

// Align is how the lines of a text lie about the point they are set at.
type Align uint8

const (
	Center Align = iota // each line centered on it: the default
	Left                // each line starting at it
	Right               // each line ending at it
)

// LineHeight is the height of a line of text, as a multiple of its size.
const LineHeight = 1.2

var breakTag = regexp.MustCompile(`(?i)<br\s*/?>`)

// Wrap breaks s into lines no wider than width, as f sets them in m: at a
// newline or a <br> always, and where a line would grow past width, at a space
// or beside a wide (CJK) character, though not before closing punctuation. A
// word wider than width takes a line of its own, unbroken. A width of 0 or
// less breaks lines at newlines and <br> only. Runs of spaces count as one.
func Wrap(m Measurer, s string, f Font, width float64) []string {
	if s == "" {
		return nil
	}
	var lines []string
	s = strings.ReplaceAll(breakTag.ReplaceAllString(s, "\n"), "\r\n", "\n")
	for _, para := range strings.Split(s, "\n") {
		lines = append(lines, wrapPara(m, para, f, width)...)
	}
	return lines
}

// A unit is a piece of text a line breaks around: a word, or a wide
// character. space is whether spaces come before it.
type unit struct {
	s     string
	space bool
}

func wrapPara(m Measurer, para string, f Font, width float64) []string {
	var lines []string
	line := ""
	for _, u := range units(para) {
		next := u.s
		if line != "" && u.space {
			next = line + " " + u.s
		} else if line != "" {
			next = line + u.s
		}
		if line != "" && width > 0 && !noBreakBefore(u.s) && m.Measure(next, f).Width > width {
			lines = append(lines, line)
			next = u.s
		}
		line = next
	}
	return append(lines, line)
}

func units(s string) []unit {
	var out []unit
	var word strings.Builder
	space := false
	flush := func() {
		if word.Len() > 0 {
			out = append(out, unit{word.String(), space})
			word.Reset()
			space = false
		}
	}
	for _, c := range s {
		switch {
		case c == ' ' || c == '\t': // a no-break space stays in its word
			flush()
			space = true
		case isWide(c):
			flush()
			out = append(out, unit{string(c), space})
			space = false
		default:
			word.WriteRune(c)
		}
	}
	flush()
	return out
}

// noBreakBefore reports whether s starts with punctuation a line must not
// start with.
func noBreakBefore(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return strings.ContainsRune("，。、；：？！）」』》】〉…・", r)
}

// SetText lays out s as Wrap breaks it at width, lines aligned about at.X as a says,
// the block of lines centered down on at.Y, in the theme's text color. It
// returns the text and the box it fills.
func SetText(m Measurer, s string, f Font, width float64, a Align, at Point) (Text, Rect) {
	t := Text{Font: f, Paint: Use(TextColor)}
	lines := Wrap(m, s, f, width)
	lh := f.Size * LineHeight
	top := at.Y - lh*float64(len(lines))/2
	widest := 0.0
	for i, s := range lines {
		met := m.Measure(s, f)
		widest = max(widest, met.Width)
		t.Lines = append(t.Lines, Line{
			S:        s,
			X:        at.X - met.Width*shift(a),
			Baseline: top + lh*float64(i) + (lh-met.Ascent-met.Descent)/2 + met.Ascent,
		})
	}
	box := Rect{X: at.X - widest*shift(a), Y: top, W: widest, H: lh * float64(len(lines))}
	return t, box
}

// shifts is how much of a line's width lies before the point it is aligned at.
var shifts = [...]float64{Center: 0.5, Left: 0, Right: 1}

func shift(a Align) float64 {
	if int(a) < len(shifts) {
		return shifts[a]
	}
	return 0
}
