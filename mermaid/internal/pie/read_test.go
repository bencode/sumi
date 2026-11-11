package pie

import (
	"fmt"
	"testing"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// What Mermaid 12.3.0 read in Chrome: the title and the sections, or where it failed.
func TestRead(t *testing.T) {
	cases := []struct {
		name, src string
		want      string // the title and the sections, or the error's line and column
	}{
		{"a keyword ends before a space, %% or the end", "pie showDataX", "1:5"},
		{"a section may follow pie on its line", `pie "a": 1`, `"" a=1`},
		{"a whole number before a dot backs off a digit", "pie\n\"a\": 12.", "2:7"},
		{"a title needs a line end after it", "pie titleaccTitle: T", "1:4"},
		{"the last title is the title, even empty", "pie\ntitle A\ntitle\n\"a\": 1", `"" a=1`},
		{"escapes, and a label again is passed over", "pie\n\"a\\tb\\0\\q\": 1\n'a\\tb\\0\\q': 2", "\"\" a\tb\x00q=1"},
		{"values are checked once all is read", "pie\n\"a\": 1\n\"a\": -2", "3:1"},
		{"a directive Mermaid leaves takes the line end", "pie\n\"a\": 1 %%{-}%%\n\"b\": 2", "3:1"},
	}
	for _, c := range cases {
		body, _ := syntax.Prepare(c.src)
		got := ""
		if d, err := Read(body); err != nil {
			se, _ := err.(*syntax.Error)
			got = fmt.Sprintf("%d:%d", se.Line, se.Col)
		} else {
			got = fmt.Sprintf("%q", d.Title)
			for _, s := range d.Sections {
				got += fmt.Sprintf(" %s=%v", s.Label, s.Value)
			}
		}
		if got != c.want {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
	}
}
