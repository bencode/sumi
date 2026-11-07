package flowchart

import (
	"math"
	"slices"
	"testing"
	"unicode/utf8"

	"github.com/bencode/sumi/scene"
)

// Each want is the box Chrome gives the label through createText, kerning off: its
// width, its height and the text of each line that shows something.
func TestSetLabel(t *testing.T) {
	inf := math.Inf(1)
	for _, c := range []struct {
		name, src   string
		wrap, least float64
		w, h        float64
		lines       []string
	}{
		{"a short label keeps its line, as wide as the least width", "flowchart TD\nA[\"short\"]", 120, 120, 120, 21, []string{"short"}},
		{"a label as wide as the width wraps at its spaces", "flowchart TD\nA[\"aaaa bbbb cccc dddd eeee ffff gggg\"]", 120, 120, 120, 63, []string{"aaaa bbbb cccc ", "dddd eeee ffff ", "gggg"}},
		{"a run of spaces is one while the label does not wrap", "flowchart TD\nA@{ label: \"  a   b  \" }", 120, 120, 120, 21, []string{"a b"}},
		{"every space stays once it wraps", "flowchart TD\nA@{ label: \"   aaaa   bbbb   cccc   dddd   eeee   ffff\" }", 120, 120, 120, 63, []string{"   aaaa   bbbb   ", "cccc   dddd   eeee ", "  ffff"}},
		{"a tab goes to the next tab stop", "flowchart TD\nA[\"aaaa\tbbbb\tcccc\tdddd\teeee\tffff\"]", 120, 120, 120, 84, []string{"aaaa\t", "bbbb\tcccc\t", "dddd\t", "eeee\tffff"}},
		{"a tab less than half a space from its stop goes to the one after", "flowchart TD\nA[\"mmri\tmmri\tmmri\tmmri\"]", 120, 120, 120, 63, []string{"mmri\t", "mmri\t", "mmri\tmmri"}},
		{"a tab more than half a space from its stop goes to it", "flowchart TD\nA[\"mmr\tmmr\tmmr\tmmr\tmmr\tmmr\"]", 120, 120, 120, 42, []string{"mmr\tmmr\tmmr\t", "mmr\tmmr\tmmr"}},
		{"a no-break space is a space that holds its words", "flowchart TD\nA[\"aaaa&nbsp;bbbb&nbsp;cccc&nbsp;dddd&nbsp;eeee\"]", 120, 120, 168.140625, 21, []string{"aaaa\u00a0bbbb\u00a0cccc\u00a0dddd\u00a0eeee"}},
		{"a word wider than the width widens the box", "flowchart TD\nA[\"aaaaaaaaaaaaaaaaaaaa\"]", 120, 120, 155.734375, 21, []string{"aaaaaaaaaaaaaaaaaaaa"}},
		{"once widened, later lines wrap at the wider width", "flowchart TD\nA[\"aaaaaaaaaaaaaaaaaaaa bb cc dd ee ff gg hh ii jj kk\"]", 120, 120, 159.625, 63, []string{"aaaaaaaaaaaaaaaaaaaa ", "bb cc dd ee ff gg hh ii jj ", "kk"}},
		{"ideographs break between each other", "flowchart TD\nA[\"中文字符流程图节点标签很长的一段话\"]", 120, 120, 120, 63, []string{"中文字符流程图节", "点标签很长的一段", "话"}},
		{"words not bold", "flowchart TD\nA[\"`iiiii iiiii iiiii iiiii iiiii iiiii iiiii iiiii`\"]", 120, 120, 120, 42, []string{"iiiii iiiii iiiii iiiii iiiii iiiii ", "iiiii iiiii"}},
		{"the same words bold, wider", "flowchart TD\nA[\"`**iiiii iiiii iiiii iiiii iiiii iiiii iiiii iiiii**`\"]", 120, 120, 120, 42, []string{"iiiii iiiii iiiii iiiii iiiii ", "iiiii iiiii iiiii"}},
		{"lines of a <br> wrap on their own", "flowchart TD\nA[\"aaaa bbbb cccc dddd eeee ffff<br>gg hh\"]", 120, 120, 120, 63, []string{"aaaa bbbb cccc ", "dddd eeee ffff", "gg hh"}},
		{"a last line of spaces takes no room while the label does not wrap", "flowchart TD\nA@{ label: \"aaaa<br>  \" }", 120, 120, 120, 21, []string{"aaaa"}},
		{"a last line of spaces takes a line once it wraps", "flowchart TD\nA@{ label: \"aaaa bbbb cccc dddd eeee ffff gggg<br>  \" }", 120, 120, 120, 84, []string{"aaaa bbbb cccc ", "dddd eeee ffff ", "gggg", "  "}},
		{"a line of spaces a <br> ends takes a line", "flowchart TD\nA@{ label: \"aaaa<br>  <br>\" }", 120, 120, 120, 42, []string{"aaaa"}},
		{"an empty line after a last <br> takes no room once it wraps", "flowchart TD\nA[\"aaaa bbbb cccc dddd eeee ffff gggg<br>\"]", 120, 120, 120, 63, []string{"aaaa bbbb cccc ", "dddd eeee ffff ", "gggg"}},
		{"an edge has no least width", "flowchart TD\nA -->|\"short\"| B", 200, 0, 31.125, 21, []string{"short"}},
		{"an edge wraps at 200", "flowchart TD\nA -->|\"aaaa bbbb cccc dddd eeee ffff gggg hhhh iiii jjjj kkkk llll\"| B", 200, 0, 200, 42, []string{"aaaa bbbb cccc dddd eeee ffff ", "gggg hhhh iiii jjjj kkkk llll"}},
		{"a subgraph title never wraps", "flowchart TD\nsubgraph s[\"aaaa bbbb cccc dddd eeee ffff gggg hhhh iiii jjjj kkkk\"]\nend", inf, 0, 322.21875, 21, []string{"aaaa bbbb cccc dddd eeee ffff gggg hhhh iiii jjjj kkkk"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			d, err := readBody(t, c.src)
			if err != nil {
				t.Fatal(err)
			}
			label := d.Nodes[0].Label
			if len(d.Edges) > 0 {
				label = d.Edges[0].Label
			}
			box := setLabel(scene.Arial, label.Lines, c.wrap, c.least)
			var lines []string
			n := 0
			for _, l := range box.Lines {
				if s := joined(l); s != "" {
					lines = append(lines, s)
					n = max(n, utf8.RuneCountInString(s))
				}
			}
			// The table rounds Arial's widths to 1/1000 em, up to 0.007 px a character.
			if math.Abs(box.W-c.w) > 0.007*float64(n)+1.0/64 || box.H != c.h || !slices.Equal(lines, c.lines) {
				t.Errorf("got %v×%v %q, want %v×%v %q", box.W, box.H, lines, c.w, c.h, c.lines)
			}
		})
	}
}
