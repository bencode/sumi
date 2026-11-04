package scene

import (
	"math"
	"slices"
	"testing"
)

func TestWrap(t *testing.T) {
	f := Font{Size: 10}
	width := func(s string) float64 { return Arial.Measure(s, f).Width }
	tests := []struct {
		name  string
		s     string
		width float64
		want  []string
	}{
		{"at spaces", "the quick brown fox", width("brown fox"), []string{"the quick", "brown fox"}},
		{"between CJK characters", "中文按字断行", 25, []string{"中文", "按字", "断行"}},
		{"not before closing punctuation", "你好，世界", 25, []string{"你好，", "世界"}},
		{"at <br> and newlines", "a<br>b<BR />c\nd", 0, []string{"a", "b", "c", "d"}},
		{"long word on a line of its own", "a extraordinarily b", 20, []string{"a", "extraordinarily", "b"}},
		{"no width, no breaks", "aaa bbb  ccc", 0, []string{"aaa bbb ccc"}},
		{"before a word with an ellipsis", "aa wait… bb", 30, []string{"aa", "wait…", "bb"}},
		{"not at a no-break space", "100\u00a0km away", 25, []string{"100\u00a0km", "away"}},
		{"at CRLF", "line one\r\nline two", 0, []string{"line one", "line two"}},
		{"empty", "", 10, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Wrap(Arial, tt.s, f, tt.width); !slices.Equal(got, tt.want) {
				t.Errorf("Wrap(%q, %v) = %q, want %q", tt.s, tt.width, got, tt.want)
			}
		})
	}
}

func TestSetTextCenters(t *testing.T) {
	f := Font{Size: 14}
	at := Point{100, 50}
	text, box := SetText(Arial, "a short line<br>and a longer second line", f, 0, Center, at)
	if len(text.Lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(text.Lines))
	}
	for _, l := range text.Lines {
		mid := l.X + Arial.Measure(l.S, f).Width/2
		if math.Abs(mid-at.X) > 1e-9 {
			t.Errorf("line %q is centered on %v, want %v", l.S, mid, at.X)
		}
	}
	if mid := box.Y + box.H/2; math.Abs(mid-at.Y) > 1e-9 {
		t.Errorf("block is centered down on %v, want %v", mid, at.Y)
	}
	if d := text.Lines[1].Baseline - text.Lines[0].Baseline; math.Abs(d-f.Size*LineHeight) > 1e-9 {
		t.Errorf("baselines are %v apart, want %v", d, f.Size*LineHeight)
	}
}
