package label

import (
	"math"
	"slices"
	"strings"

	"github.com/bencode/sumi/mermaid/internal/linebreak"
	"github.com/bencode/sumi/scene"
)

// The font Chrome gives a label of Mermaid's default theme, the line height
// addHtmlSpan sets, and CSS's tab-size of 8 spaces.
const (
	fontSize   = 14
	lineHeight = 1.5 * fontSize
	tabSize    = 8
)

// epsilon is how far apart two widths may be and still be one: a sum of widths in
// floating point is off by far less than the 1/64 px Chrome lays text out in.
const epsilon = 1e-9

// A Box is a label as a browser sets it: its lines once wrapped, and its size.
type Box struct {
	Lines [][]Span // spaces as the browser keeps them
	W, H  float64
}

// Set sets lines in a box wrap wide, no narrower than least, as addHtmlSpan does.
// A label narrower than wrap keeps its lines, set with white-space: nowrap; one that
// reaches wrap is set again with white-space: break-spaces in a table as wide as wrap
// or its widest piece, whichever is wider. wrap is math.Inf(1) for a label that never
// wraps.
func Set(m scene.Measurer, lines [][]Span, wrap, least float64) Box {
	// An empty line after a last <br> takes no room, and a line of spaces there takes
	// none until the label wraps. A line a <br> ends takes one either way.
	n := len(lines)
	lastBreak := n > 1 && joined(lines[n-1]) == ""
	if lastBreak {
		lines = lines[:n-1]
	}
	kept := make([][]Span, len(lines))
	w := 0.0
	for i, l := range lines {
		kept[i] = collapse(l)
		w = math.Max(w, snap(advance(m, 0, kept[i])))
	}
	if w < wrap {
		if k := len(kept); !lastBreak && k > 1 && len(kept[k-1]) == 0 {
			kept = kept[:k-1]
		}
		return Box{kept, math.Max(w, least), lineHeight * float64(len(kept))}
	}
	for _, l := range lines {
		for _, piece := range breakLine(m, l, 0) {
			wrap = math.Max(wrap, snap(advance(m, 0, piece)))
		}
	}
	var out [][]Span
	for _, l := range lines {
		out = append(out, breakLine(m, l, wrap)...)
	}
	return Box{out, wrap, lineHeight * float64(len(out))}
}

// snap is a width as Chrome gives a box, rounded up to 1/64 px: what addHtmlSpan
// compares with the width it wraps at.
func snap(w float64) float64 {
	return math.Ceil(w*64-epsilon) / 64
}

// collapse is a line as white-space: nowrap shows it: each run of spaces and tabs
// one space, which the span the run starts in keeps, and none at either end.
func collapse(line []Span) []Span {
	var out []Span
	gap := true // a space here goes: at the start, or after another
	for _, s := range line {
		var b strings.Builder
		for _, r := range s.Text {
			switch {
			case r != ' ' && r != '\t':
				b.WriteRune(r)
				gap = false
			case !gap:
				b.WriteByte(' ')
				gap = true
			}
		}
		out = append(out, Span{b.String(), s.Bold, s.Italic})
	}
	for i := len(out) - 1; gap && i >= 0; i-- {
		if out[i].Text != "" {
			out[i].Text = strings.TrimSuffix(out[i].Text, " ")
			break
		}
	}
	return slices.DeleteFunc(out, func(s Span) bool { return s.Text == "" })
}

// breakLine wraps a line width wide with white-space: break-spaces, as Chrome's line
// breaker does: each line takes the most that fits up to a break, the first piece if
// nothing fits, and the break rules start again from where each line starts.
func breakLine(m scene.Measurer, line []Span, width float64) [][]Span {
	text := joined(line)
	var out [][]Span
	for start := 0; ; {
		end, x := start, 0.0
		for _, o := range append(linebreak.Breaks(text[start:]), len(text)-start) {
			next := advance(m, x, cut(line, end, start+o))
			if end > start && next > width+epsilon {
				break
			}
			end, x = start+o, next
		}
		out = append(out, cut(line, start, end))
		if end == len(text) {
			return out
		}
		start = end
	}
}

// advance is where spans set from x end. A tab goes to the next tab stop, a stop
// every tabSize spaces from the start of the line, or to the one after if that is
// less than half a space away, as Blink's Font::TabWidth has it.
func advance(m scene.Measurer, x float64, spans []Span) float64 {
	for _, s := range spans {
		f := scene.Font{Size: fontSize, Bold: s.Bold, Italic: s.Italic}
		space := m.Measure(" ", f).Width
		for i, run := range strings.Split(s.Text, "\t") {
			if i > 0 {
				next := (math.Floor(x/(tabSize*space)) + 1) * tabSize * space
				if next-x < space/2-epsilon {
					next += tabSize * space
				}
				x = next
			}
			x += m.Measure(run, f).Width
		}
	}
	return x
}

func joined(line []Span) string {
	var b strings.Builder
	for _, s := range line {
		b.WriteString(s.Text)
	}
	return b.String()
}

// cut is the part of line from byte i to byte j of its text.
func cut(line []Span, i, j int) []Span {
	var out []Span
	at := 0
	for _, s := range line {
		lo, hi := max(i-at, 0), min(j-at, len(s.Text))
		if lo < hi {
			out = append(out, Span{s.Text[lo:hi], s.Bold, s.Italic})
		}
		at += len(s.Text)
	}
	return out
}
