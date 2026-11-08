package label

import (
	"math"
	"strings"

	"github.com/bencode/sumi/scene"
)

// Texts sets b's lines in its box, the box's top left at top: each line centered
// across the box (text-align: center), lineHeight tall, its baseline where Blink
// puts it: half the leading, floored (CalculateLeadingSpace), above the font's
// ascent. A span is a Line of the Text of its font, so a line of bold and plain
// words is in two Texts. Spaces at a span's ends are left out and the span set past
// those it starts with, so that a backend that drops them, as SVG's <text> does,
// sets it in the same place. Every span is in ink.
func Texts(m scene.Measurer, b Box, top scene.Point, ink scene.Paint) []scene.Text {
	var out []scene.Text
	index := map[scene.Font]int{}
	for i, l := range b.Lines {
		x := top.X + (b.W-advance(m, 0, l))/2
		lineTop := top.Y + lineHeight*float64(i)
		for _, s := range l {
			lead := len(s.Text) - len(strings.TrimLeft(s.Text, " "))
			at := advance(m, x, []Span{{s.Text[:lead], s.Bold, s.Italic}})
			x = advance(m, x, []Span{s})
			word := strings.TrimRight(s.Text[lead:], " ")
			if word == "" {
				continue
			}
			f := scene.Font{Size: fontSize, Bold: s.Bold, Italic: s.Italic}
			j, ok := index[f]
			if !ok {
				j = len(out)
				index[f] = j
				out = append(out, scene.Text{Font: f, Paint: ink})
			}
			a, d := Extent(m, f)
			out[j].Lines = append(out[j].Lines, scene.Line{S: word, X: at, Baseline: lineTop + math.Floor((lineHeight-a-d)/2) + a})
		}
	}
	return out
}

// Extent is how far f reaches above and below the baseline as Blink lays text out:
// SimpleFontData rounds both to whole pixels.
func Extent(m scene.Measurer, f scene.Font) (ascent, descent float64) {
	met := m.Measure("", f)
	return math.Round(met.Ascent), math.Round(met.Descent)
}
