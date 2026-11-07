package flowchart

import (
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// blockStart is where marked looks for a block that a paragraph cannot go on into.
const blockStart = `(?:\A|\n *\n)(?:[ \t]*\n)*`

// The parts of Markdown readMarkdown leaves out: what starts a block, and tokens marked
// reads before emphasis or leaves as they are.
var (
	blockLine = regexp.MustCompile(`(?m)^ {0,3}(?:` +
		`#{1,6}(?:[\s\p{Z}\x{feff}]|$)|>|~{3,}|\[(?:\\[\s\S]|[^\[\]\\])+\]:|` + // heading, quote, fence, definition
		`(?:[*+-]|[0-9]{1,9}[.)])(?:[ \t]|$)|` + // list
		`(?:(?:-[ \t]*){3,}|(?:_[ \t]*){3,}|(?:\*[ \t]*){3,})$)`) // rule
	underline = regexp.MustCompile(`(?m)\S.*\n {0,3}(?:=+|` + // of a heading, or a table's second row
		`\|?[ \t]*:?-+:?[ \t]*(?:\|[ \t]*:?-+:?[ \t]*)*\|?)[ \t]*$`)
	codeBlock  = regexp.MustCompile(blockStart + `(?: {4}| {0,3}\t)`)
	htmlBlock  = regexp.MustCompile(`(?i)` + blockStart + ` {0,3}<br\s*/?>[ \t]*(?:\n|\z)`)
	hardBreak  = regexp.MustCompile(`(?: {2,}|\\)\n`)
	emphasis   = regexp.MustCompile(`[*_\\]`)
	firstToken = regexp.MustCompile(`(?i)\[|~|://|www\.|@`)
	rawToken   = regexp.MustCompile(`\]\(|~`) // a link or a strikethrough, whose raw text keeps its newlines
)

var (
	paragraphBreak = regexp.MustCompile(blockStart)
	softBreak      = regexp.MustCompile(`\n *`)
	escaped        = regexp.MustCompile(`\\[\p{P}\p{S}]`)
)

const entityBeside = "an entity code beside Markdown"

// sanitized reports whether DOMPurify changes what marked sees in s. It leaves text with
// no < alone. Else it reads entity codes, and writes a no-break space, a space to marked,
// as &nbsp;, punctuation.
func sanitized(s string) bool {
	return strings.Contains(s, "<") && (hasEntity(s) || strings.ContainsRune(s, '\u00a0'))
}

// notRead names the Markdown in s that readMarkdown does not read, if any. What is before
// a * or a _ decides what that does, and marked sees an entity code as a different text
// in a node than on an edge.
func notRead(s string) string {
	marked := emphasis.MatchString(s)
	switch {
	case blockLine.MatchString(s), underline.MatchString(s), codeBlock.MatchString(s), htmlBlock.MatchString(s):
		return "a Markdown block"
	case hardBreak.MatchString(s):
		return "a Markdown hard line break"
	case marked && firstToken.MatchString(s), strings.Contains(s, "\n") && rawToken.MatchString(s):
		return "a Markdown link beside emphasis or a line break"
	case marked && syntax.Decode(s) != s, sanitized(s):
		return entityBeside
	}
	return ""
}

// readMarkdown is what markdownToHTML makes of s, read as readLabel reads it.
func readMarkdown(s string, at syntax.Pos) ([][]Span, error) {
	if what := notRead(s); what != "" {
		return nil, syntax.Unsupported(at, what)
	}
	var lines [][]Span
	for _, p := range paragraphBreak.Split(brTag.ReplaceAllString(s, "<br>"), -1) {
		if strings.Trim(p, " \t") != "" {
			lines = append(lines, closed(lined(inline([]rune(strings.TrimSuffix(p, "\n")), false, false)))...)
		}
	}
	return lines, nil
}

// A piece is what inline makes of a token: text in a style, or a line break.
type piece struct {
	span  Span
	broke bool
}

// lined puts pieces into lines.
func lined(ps []piece) [][]Span {
	lines := [][]Span{}
	var line []Span
	for _, p := range ps {
		if p.broke {
			lines = append(lines, spans(line))
			line = nil
			continue
		}
		for i, s := range softBreak.Split(p.span.Text, -1) {
			if i > 0 {
				lines = append(lines, spans(line))
				line = nil
			}
			line = append(line, Span{s, p.span.Bold, p.span.Italic})
		}
	}
	return append(lines, spans(line))
}

var br = []rune("<br>")

// inline is inlineTokens, for the tokens that are left once notRead has spoken: escape,
// tag, emStrong and text.
func inline(s []rune, bold, italic bool) []piece {
	var out []piece
	text := func(r []rune) { out = append(out, piece{span: Span{string(r), bold, italic}}) }
	masked := mask(s)
	prev, keep := rune(0), false
	for len(s) > 0 {
		if !keep {
			prev = 0
		}
		keep = false
		if s[0] == '\\' && len(s) > len(br) && slices.Equal(s[1:len(br)+1], br) {
			out = append(out, piece{broke: true}) // the < it frees makes a tag again in the HTML
			s = s[len(br)+1:]
			continue
		}
		if s[0] == '\\' && len(s) > 1 && s[1] < unicode.MaxASCII && isPunct(s[1]) {
			text(s[1:2])
			s = s[2:]
			continue
		}
		if len(s) >= len(br) && slices.Equal(s[:len(br)], br) {
			out = append(out, piece{broke: true})
			s = s[len(br):]
			continue
		}
		if n, strong, ok := emStrong(s, masked, prev); ok {
			if strong {
				out = append(out, inline(s[2:n-2], true, italic)...)
			} else {
				out = append(out, inline(s[1:n-1], bold, true)...)
			}
			s = s[n:]
			continue
		}
		n := textEnd(s)
		text(s[:n])
		if s[n-1] != '_' {
			prev = lastUnit(s[n-1])
		}
		keep = true
		s = s[n:]
	}
	return out
}

// lastUnit is c as marked's prevChar has it, one UTF-16 unit: half of a pair is neither
// punctuation nor a letter.
func lastUnit(c rune) rune {
	if c > 0xffff {
		return 0xdc00
	}
	return c
}

// textEnd is the length of the text token at the start of s.
func textEnd(s []rune) int {
	for i := 1; i < len(s); i++ {
		if strings.ContainsRune(`\<*_`, s[i]) {
			return i
		}
	}
	return len(s)
}

// mask is s as emStrong looks for what closes emphasis in it: an escaped character is in
// no delimiter run. marked masks tags too, but a <br> holds no delimiter and starts and
// ends in punctuation either way, and a < that is no tag DOMPurify has written as &lt;.
func mask(s []rune) []rune { return []rune(escaped.ReplaceAllString(string(s), "++")) }

func isPunct(c rune) bool { return unicode.IsPunct(c) || unicode.IsSymbol(c) }

func isPunctSpace(c rune) bool { return isPunct(c) || isSpace(c) }

// opens is emStrongLDelim and the test on what comes before it: the length of the run of
// delimiters at the start of s, or 0 if it opens nothing.
func opens(s []rune, prev rune) int {
	d := s[0]
	if d != '*' && d != '_' {
		return 0
	}
	n := 1
	for n < len(s) && s[n] == d {
		n++
	}
	if n == len(s) || isSpace(s[n]) {
		return 0
	}
	punct := isPunct(s[n])
	if d == '_' && !punct && (unicode.IsLetter(prev) || unicode.IsNumber(prev)) {
		return 0
	}
	if punct && prev != 0 && (prev == '*' || prev == '_' || !isPunctSpace(prev)) {
		return 0
	}
	return n
}

// emStrong is emStrong (marked): the length of the em or strong at the start of s, and
// which. masked is the text s is the end of, masked.
func emStrong(s, masked []rune, prev rune) (length int, strong, ok bool) {
	open := opens(s, prev)
	if open == 0 {
		return 0, false, false
	}
	t := masked[len(masked)-len(s)+open:]
	deep, mid := open, 0
	for from := 0; ; {
		r, found := nextRun(t, from, s[0])
		if !found {
			return 0, false, false
		}
		from = r.end
		switch {
		case r.side == leftRun:
			deep += r.n
			continue
		case r.side == bothRun && open%3 != 0 && (open+r.n)%3 == 0:
			mid += r.n
			continue
		}
		if deep -= r.n; deep > 0 {
			continue
		}
		n := min(r.n, r.n+deep+mid)
		return open + r.at + 1 + n, min(open, n)%2 == 0, true
	}
}

type runSide uint8

const (
	rightRun runSide = iota
	leftRun
	bothRun
)

// A run is a run of delimiters, what emStrongRDelimAst and emStrongRDelimUnd match: at is
// where the character before it is, and end where it ends.
type run struct {
	at, n, end int
	side       runSide
}

// nextRun is the next match of those patterns in t from from that holds a run of d. They
// have lookaheads, so it is written out.
func nextRun(t []rune, from int, d rune) (run, bool) {
	if from == 0 {
		from = skipOther(t, d)
	}
	for i := from; i < len(t); i++ {
		if r, ok := runAt(t, i, d); ok {
			return r, true
		}
	}
	return run{}, false
}

// skipOther is the first branch of the patterns, for a __ or ** around a lone delimiter
// of the other kind: where it ends, or 0.
func skipOther(t []rune, d rune) int {
	other := '*'
	if d == '*' {
		other = '_'
	}
	free := func(i int) int {
		for i < len(t) && t[i] != '*' && t[i] != '_' {
			i++
		}
		return i
	}
	pair := func(i int) bool { return i+1 < len(t) && t[i] == other && t[i+1] == other }
	i := free(0)
	if !pair(i) {
		return 0
	}
	i = free(i + 2)
	if i == len(t) || t[i] != d {
		return 0
	}
	if i = free(i + 1); !pair(i) {
		return 0
	}
	return i
}

// runAt is the match of the patterns' branches with a run of d after t[i].
func runAt(t []rune, i int, d rune) (run, bool) {
	e := i + 1
	for e < len(t) && t[e] == d {
		e++
	}
	r := run{at: i, n: e - i - 1, end: e}
	if r.n == 0 {
		return r, false
	}
	c := t[i]
	end := e == len(t)
	var next rune
	if !end {
		next = t[e]
	}
	punct := c != d && isPunct(c)
	word := !isPunctSpace(c)
	switch {
	case punct && (end || isSpace(next)), word && (end || isPunctSpace(next)):
		r.side = rightRun
	case c != d && isPunctSpace(c) && !end && !isPunctSpace(next), isSpace(c) && !end && isPunct(next):
		r.side = leftRun
	case punct && !end && isPunct(next), d == '*' && word && !end && !isPunctSpace(next):
		r.side = bothRun
	default:
		return r, false
	}
	return r, true
}
