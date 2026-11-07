package flowchart

import (
	"errors"
	"strings"
	"testing"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

func readBody(t *testing.T, src string) (*Diagram, error) {
	t.Helper()
	body, _ := syntax.Prepare(src)
	return Read(body)
}

// written is lines as a table has them: a line each, styles as <b> and <i>.
func written(lines [][]Span) string {
	var out []string
	for _, line := range lines {
		var b strings.Builder
		for _, s := range line {
			text := s.Text
			if s.Italic {
				text = "<i>" + text + "</i>"
			}
			if s.Bold {
				text = "<b>" + text + "</b>"
			}
			b.WriteString(text)
		}
		out = append(out, b.String())
	}
	return strings.Join(out, "\n")
}

// The wanted lines are what Chrome shows of these labels.
func TestLabels(t *testing.T) {
	for _, c := range []struct{ name, src, want string }{
		{"a name of HTML", "flowchart TD\nA[\"a&lt;b&copy\"]", "a<b©"},
		{"an entity code", "flowchart TD\nA[\"a#quot;b#9829;\"]", "a\"b♥"},
		{"a number Windows-1252 has", "flowchart TD\nA[\"a#128;\"]", "a€"},
		{"a number past Unicode", "flowchart TD\nA[\"a#4294967361;\"]", "a�"},
		{"a number past an int", "flowchart TD\nA[\"a#18446744073709551681;\"]", "a�"},
		{"a name Mermaid leaves", "flowchart TD\nA[\"#nosuch;\"]", "&nosuch;"},
		{"an entity code in an entity code", "flowchart TD\nA[\"#amp;lt;\"]", "&lt;"},
		{"line breaks", "flowchart TD\nA[\"a<br>b<br/>c<br />d<BR>e\"]", "a\nb\nc\nd\ne"},
		{"a last break", "flowchart TD\nA[\"a<br>\"]", "a"},
		{"two last breaks", "flowchart TD\nA[\"a<br><br>\"]", "a\n"},
		{"a first break", "flowchart TD\nA[\"<br>a\"]", "\na"},
		{"a newline", "flowchart TD\nA[\"a\nb\"]", "a\nb"},
		{"a written newline", "flowchart TD\nA[\"a\\nb\"]", "a\nb"},
		{"a < that is no tag", "flowchart TD\nA[\"a < b <3\"]", "a < b <3"},
		{"stars in text", "flowchart TD\nA[\"**a**\"]", "**a**"},
		{"stars on an edge", "flowchart TD\nA -->|\"**a**\"| B", "**a**"},
		{"stars in a title", "flowchart TD\nsubgraph s[\"**a**\"]\nend", "**a**"},
		{"bold", "flowchart TD\nA[\"`**a**b`\"]", "<b>a</b>b"},
		{"italic in a word", "flowchart TD\nA[\"`a*b*c`\"]", "a<i>b</i>c"},
		{"underscores", "flowchart TD\nA[\"`_a_ __b__`\"]", "<i>a</i> <b>b</b>"},
		{"no underscores in a word", "flowchart TD\nA[\"`a_b_c`\"]", "a_b_c"},
		{"bold italic", "flowchart TD\nA[\"`***a***`\"]", "<b><i>a</i></b>"},
		{"italic in bold", "flowchart TD\nA[\"`**a *b* c**`\"]", "<b>a </b><b><i>b</i></b><b> c</b>"},
		{"bold on an edge", "flowchart TD\nA -->|\"`**a**`\"| B", "<b>a</b>"},
		{"bold in a title", "flowchart TD\nsubgraph s[\"`**a**`\"]\nend", "<b>a</b>"},
		{"stars with spaces", "flowchart TD\nA[\"`** a **`\"]", "** a **"},
		{"a run of three between", "flowchart TD\nA[\"`*a***b*`\"]", "<i>a</i>*<i>b</i>"},
		{"the rule of three", "flowchart TD\nA[\"`**a*b*c**`\"]", "<b>a</b><b><i>b</i></b><b>c</b>"},
		{"an escape", "flowchart TD\nA[\"`\\*a\\*`\"]", "*a*"},
		{"an escaped <br>", "flowchart TD\nA[\"`a\\<br>b`\"]", "a\nb"},
		{"a written newline in Markdown", "flowchart TD\nA[\"`a\\nb`\"]", "a\\nb"},
		{"a newline in Markdown", "flowchart TD\nA[\"`a\n   b`\"]", "a\nb"},
		{"paragraphs", "flowchart TD\nA[\"`a\n\nb`\"]", "a\nb"},
		{"a tab is no blank line", "flowchart TD\nA[\"`a\n\t\nb`\"]", "a\n\t\nb"},
		{"a break in Markdown", "flowchart TD\nA[\"`*a<br>b*`\"]", "<i>a</i>\n<i>b</i>"},
		{"a last break in a paragraph", "flowchart TD\nA[\"`a<br>\n\nb`\"]", "a\nb"},
		{"a link alone", "flowchart TD\nA[\"`[a](b)`\"]", "[a](b)"},
		{"strikethrough alone", "flowchart TD\nA[\"`~~a~~`\"]", "~~a~~"},
		{"after an emoji", "flowchart TD\nA[\"`😀**<3**`\"]", "😀**<3**"},
		{"a lone = is no heading", "flowchart TD\nA[\"`=`\"]", "="},
		{"a < is no tag to emphasis", "flowchart TD\nA[\"`*a<3*>*`\"]", "<i>a<3</i>>*"},
		{"two breaks in a node", "flowchart TD\nA[\"`a<br><br>b`\"]", "a\n\nb"},
		{"two breaks on an edge part paragraphs", "flowchart TD\nA -->|\"`a<br><br>b`\"| B", "a\nb"},
		{"a closing break on an edge", "flowchart TD\nA -->|\"a</br>b\"| B", "a\nb"},
		{"a first break on an edge", "flowchart TD\nA -->|\"`<br>a`\"| B", "a"},
		{"a tab after a last break on an edge", "flowchart TD\nA -->|\"`a<br>\t<br>`\"| B", "a"},
		{"a no-break space is a line", "flowchart TD\nA[\"a<br>&nbsp;\"]", "a\n "},
		{"a title keeps a no-break space beside a <", "flowchart TD\nsubgraph s[\"<br>a \"]\nend", "\na "},
		{"a title trims one otherwise", "flowchart TD\nsubgraph s[\"a \"]\nend", "a"},
		{"only ASCII is escaped", "flowchart TD\nA[\"`\\—*a*`\"]", "\\—<i>a</i>"},
		{"an escaped star closes nothing", "flowchart TD\nA[\"`*a\\*b*`\"]", "<i>a*b</i>"},
		{"text before __", "flowchart TD\nA[\"`a__b__`\"]", "a__b__"},
		{"no opening at a wide space", "flowchart TD\nA[\"`*　a*`\"]", "*　a*"},
		{"no underscore after a letter", "flowchart TD\nA[\"`é_a_`\"]", "é_a_"},
		{"no underscore after a number", "flowchart TD\nA[\"`3_a_`\"]", "3_a_"},
		{"runs of three", "flowchart TD\nA[\"`***a***b`\"]", "<b><i>a</i></b>b"},
		{"a run that goes both ways", "flowchart TD\nA[\"`*-**(**`\"]", "<i>-**(*</i>"},
		{"bold around italic at the end", "flowchart TD\nA[\"`**a *b***`\"]", "<b>a </b><b><i>b</i></b>"},
		{"a lone star in __", "flowchart TD\nA[\"`*__a*__`\"]", "*<b>a*</b>"},
		{"__ in bold", "flowchart TD\nA[\"`**__a**__`\"]", "<b>__a</b>__"},
		{"punctuation in stars", "flowchart TD\nA[\"`*.*`\"]", "<i>.</i>"},
		{"__ in a word closes nothing", "flowchart TD\nA[\"`_a__b c_`\"]", "<i>a__b c</i>"},
		{"a left run after a space", "flowchart TD\nA[\"`*a *.* b*`\"]", "<i>a . b</i>"},
		{"a run between punctuation", "flowchart TD\nA[\"`*a.*.*`\"]", "<i>a.</i>.*"},
		{"__ before a word", "flowchart TD\nA[\"`__a__b`\"]", "__a__b"},
		{"an escape hides what is before it", "flowchart TD\nA[\"`a\\!*.*`\"]", "a!<i>.</i>"},
		{"a lone star in __ further on", "flowchart TD\nA[\"`*a *b* __c*d__ e*`\"]", "<i>a b __c</i>d__ e*"},
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
			if got := written(label.Lines); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

// A <br> on an edge is a newline to Markdown, so these are blocks there.
func TestEdgeLabelsRefused(t *testing.T) {
	for _, label := range []string{"`a<br>- b`", "`a\\<br>b`", "`a  <br>b`", "`a<br>#quot;`"} {
		t.Run(label, func(t *testing.T) {
			_, err := readBody(t, "flowchart TD\nA -->|\""+label+"\"| B")
			var e *syntax.Error
			if !errors.As(err, &e) || !e.IsUnsupported() {
				t.Errorf("got %v", err)
			}
		})
	}
}

func TestLabelsRefused(t *testing.T) {
	for _, label := range []string{
		"<b>a</b>", "a<!-- b -->", "a<b", "</b>", "fa:fa-car", "fab:fa-x y", "$$x^2$$",
		"`# a`", "`> a`", "`- a`", "`1. a`", "`---`", "`~~~`", "`[a]: b`", "`a\n===`", "`a\n--`",
		"`a\n\n    b`", "`a\n\n \tb`", "`<br>`", "`a\n\n<br>\nb`",
		"`a  \nb`", "`a\\\nb`",
		"`*[a](b)*`", "`*a* ~b~`", "`*a* http://b`", "`*a* WWW.b`", "`*a* b@c`", "`[a\nb](c)`", "`~a\nb~`",
		"`*a* #quot;`", "`a<br>#quot;`", "`a<br>&amp;`", "`a<br>\u00a0b`", "$$a\nb$$",
	} {
		t.Run(label, func(t *testing.T) {
			_, err := readBody(t, "flowchart TD\nA[\""+label+"\"]")
			var e *syntax.Error
			if !errors.As(err, &e) || !e.IsUnsupported() {
				t.Errorf("got %v", err)
			}
		})
	}
}
