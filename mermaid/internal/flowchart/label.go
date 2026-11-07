package flowchart

import (
	"html"
	"regexp"
	"slices"
	"strings"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// A Span is a run of text in one style.
type Span struct {
	Text         string
	Bold, Italic bool
}

var (
	icon     = regexp.MustCompile(`fa[bklrs]?:fa-[\w-]+`) // replaceIconSubstring
	formula  = regexp.MustCompile(`(?s)\$\$.*\$\$`)       // katexRegex, on HTML, which has its newlines as <br>
	brTag    = regexp.MustCompile(`(?i)<br\s*/?>`)
	anyBreak = regexp.MustCompile(`(?i)</?br\s*/?>`) // lineBreakRegex
	tagStart = regexp.MustCompile(`<[a-zA-Z/!?]`)    // what starts a tag, as HTML reads it
	newLines = regexp.MustCompile(`\\n|\n`)          // nonMarkdownToHTML
)

// readLabel is what a browser shows of l, in lines, spaces as written. Mermaid gives it
// HTML; sumi draws text, so all that HTML may hold here is <br>.
func readLabel(l Label, at syntax.Pos) ([][]Span, error) {
	switch {
	case l.Text == "":
		return nil, nil
	case icon.MatchString(l.Text):
		return nil, syntax.Unsupported(at, "an icon in a label")
	case formula.MatchString(l.Text):
		return nil, syntax.Unsupported(at, "a formula in a label")
	case tagStart.MatchString(brTag.ReplaceAllString(l.Text, " ")):
		return nil, syntax.Unsupported(at, "HTML in a label")
	case l.Kind == "markdown":
		return readMarkdown(l.Text, at)
	}
	var lines [][]Span
	for _, s := range newLines.Split(brTag.ReplaceAllString(l.Text, "\n"), -1) {
		lines = append(lines, spans([]Span{{Text: s}}))
	}
	return closed(lines), nil
}

// readEdgeLabel is readLabel for the label of an edge, whose line breaks the ELK layout
// has made newlines by the time it is drawn: in Markdown they part paragraphs and start
// blocks as a <br> does not.
func readEdgeLabel(l Label, at syntax.Pos) ([][]Span, error) {
	if l.Kind == "markdown" && sanitized(l.Text) {
		return nil, syntax.Unsupported(at, entityBeside)
	}
	l.Text = anyBreak.ReplaceAllString(l.Text, "\n")
	return readLabel(l, at)
}

// closed is lines without the blank line after a last <br>, which takes no room.
func closed(lines [][]Span) [][]Span {
	if n := len(lines); n > 1 && blank(lines[n-1]) {
		return lines[:n-1]
	}
	return lines
}

// blank reports whether a line has nothing HTML would keep.
func blank(line []Span) bool {
	return !slices.ContainsFunc(line, func(s Span) bool { return strings.Trim(s.Text, " \t\n\f\r") != "" })
}

// spans are ss as a line of text: entity codes read, neighbors of one style joined.
func spans(ss []Span) []Span {
	var out []Span
	for _, s := range ss {
		if n := len(out); n > 0 && out[n-1].Bold == s.Bold && out[n-1].Italic == s.Italic {
			out[n-1].Text += s.Text
			continue
		}
		out = append(out, s)
	}
	for i := range out {
		out[i].Text = decode(out[i].Text)
	}
	return slices.DeleteFunc(out, func(s Span) bool { return s.Text == "" })
}

// decode reads the entity codes in s, HTML's before those syntax.Prepare set aside, so
// that #amp;lt; shows as &lt;.
func decode(s string) string { return syntax.Decode(html.UnescapeString(s)) }

// hasEntity reports whether s has an entity code.
func hasEntity(s string) bool { return decode(s) != s }
