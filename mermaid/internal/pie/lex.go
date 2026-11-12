package pie

import (
	"strings"
	"unicode/utf8"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// The kinds of token Langium makes of pie.langium, beside syntax's Newline, Title,
// AccTitle and AccDescr.
const (
	kPie = syntax.FirstKind + iota
	kShowData
	kColon
	kNumber
	kString
	skip // a token Chevrotain hides from the parser
)

// A rule is a token's pattern: the length of its match at the start of a source, or -1.
type rule struct {
	kind  syntax.Kind
	match func(s string) int
}

// rules are in the order Langium gives Chevrotain (token-builder.js, buildTokens): the
// keywords, longest first, then the terminals the entry rule reaches, pie.langium's before
// common.langium's, then the hidden ones. INT_PIE and FLOAT_PIE make no tokens of their own,
// and ID, INT, FLOAT and BOOLEAN none at all, as no parser rule reaches them.
var rules = []rule{
	{kShowData, keyword("showData")},
	{kPie, keyword("pie")},
	{kColon, func(s string) int { return prefix(s, ":") }},
	{kNumber, number},
	{syntax.AccDescr, accDescr},
	{syntax.AccTitle, accTitle},
	{syntax.Title, title},
	{kString, quoted},
	{syntax.Newline, newline},
	{skip, blanks},
	{skip, yaml},
	{skip, directive},
	{skip, comment},
}

// lex splits a diagram into tokens as Chevrotain does with rules: at each place the first
// that matches, not the longest. It starts past the spaces Mermaid trims before it parses
// (cleanupComments, preprocess.ts), which include the lines syntax.Prepare has blanked, and
// stops at the first character none matches, which is the first error Mermaid reports,
// lexer errors coming before the parser's.
func lex(src string) ([]syntax.Token, error) {
	lines := syntax.LinesOf(src)
	var ts []syntax.Token
	for i := spaces(src); i < len(src); {
		k, n := first(src[i:])
		if n < 0 {
			c, _ := utf8.DecodeRuneInString(src[i:])
			return nil, syntax.Errorf(lines.Pos(i), "unexpected %q", c)
		}
		if k != skip {
			ts = append(ts, syntax.Token{Kind: k, Text: src[i : i+n], Pos: lines.Pos(i)})
		}
		i += n
	}
	return append(ts, syntax.Token{Kind: syntax.EOF, Pos: lines.Pos(len(src))}), nil
}

// first is the kind of the first rule that matches at the start of s, and the length of
// its match, or -1. A rule matches with a length above 0.
func first(s string) (syntax.Kind, int) {
	for _, r := range rules {
		if n := r.match(s); n > 0 {
			return r.kind, n
		}
	}
	return 0, -1
}

// keyword matches word, followed by the end, %% or a space: Mermaid adds
// (?:(?=%%)|(?!\S)) to the keywords (AbstractMermaidTokenBuilder, tokenBuilder.ts).
func keyword(word string) func(string) int {
	return func(s string) int {
		if !strings.HasPrefix(s, word) {
			return -1
		}
		if rest := s[len(word):]; rest != "" && !strings.HasPrefix(rest, "%%") && !startsSpace(rest) {
			return -1
		}
		return len(word)
	}
}

// number matches NUMBER_PIE, -?[0-9]+\.[0-9]+(?!\.) or else -?(0|[1-9][0-9]*)(?!\.),
// backing off as a regular expression does: a run of digits followed by a dot gives up its
// last digit, which is then followed by a digit.
func number(s string) int {
	sign := max(prefix(s, "-"), 0)
	whole := digits(s[sign:])
	if whole == 0 {
		return -1
	}
	end := sign + whole
	if end < len(s) && s[end] == '.' {
		frac := digits(s[end+1:])
		switch e := end + 1 + frac; {
		case frac > 0 && (e == len(s) || s[e] != '.'):
			return e
		case frac > 1:
			return e - 1
		}
	}
	switch {
	case s[sign] == '0' && (sign+1 == len(s) || s[sign+1] != '.'):
		return sign + 1
	case end == len(s) || s[end] != '.':
		return end
	case whole > 1:
		return end - 1
	}
	return -1
}

// accDescr matches [\t ]*accDescr(?:[\t ]*:([^\n\r]*?(?=%%)|[^\n\r]*)|\s*{([^}]*)}).
func accDescr(s string) int {
	i := blankStart(s, "accDescr")
	if i < 0 {
		return -1
	}
	if j := i + blanks(s[i:]); strings.HasPrefix(s[j:], ":") {
		return j + 1 + upToComment(s[j+1:])
	}
	j := i + spaces(s[i:])
	if !strings.HasPrefix(s[j:], "{") {
		return -1
	}
	if e := strings.IndexByte(s[j+1:], '}'); e >= 0 {
		return j + 1 + e + 1
	}
	return -1
}

// accTitle matches [\t ]*accTitle[\t ]*:(?:[^\n\r]*?(?=%%)|[^\n\r]*).
func accTitle(s string) int {
	i := blankStart(s, "accTitle")
	if i < 0 {
		return -1
	}
	if i += blanks(s[i:]); !strings.HasPrefix(s[i:], ":") {
		return -1
	}
	return i + 1 + upToComment(s[i+1:])
}

// title matches [\t ]*title(?:[\t ][^\n\r]*?(?=%%)|[\t ][^\n\r]*|): "titleX" is the title
// alone, and the X another token.
func title(s string) int {
	i := blankStart(s, "title")
	if i < 0 {
		return -1
	}
	if i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		return i + 1 + upToComment(s[i+1:])
	}
	return i
}

// quoted matches STRING, "([^"\\]|\\.)*"|'([^'\\]|\\.)*': a backslash takes the character
// after it, which is not a line end, as the dot does not match one.
func quoted(s string) int {
	if s == "" || s[0] != '"' && s[0] != '\'' {
		return -1
	}
	for i := 1; i < len(s); {
		switch s[i] {
		case s[0]:
			return i + 1
		case '\\':
			r, n := utf8.DecodeRuneInString(s[i+1:])
			if n == 0 || isLineEnd(r) {
				return -1
			}
			i += 1 + n
		default:
			i++
		}
	}
	return -1
}

// newline matches \r?\n.
func newline(s string) int {
	if strings.HasPrefix(s, "\r\n") {
		return 2
	}
	return prefix(s, "\n")
}

// yaml matches ---[\t ]*\r?\n(?:[\S\s]*?\r?\n)?---(?:\r?\n|(?!\S)). With the optional
// group, the closing --- is tried at the start of each line from the third on, the
// shortest first; without it, at the start of the second.
func yaml(s string) int {
	if !strings.HasPrefix(s, "---") {
		return -1
	}
	i := 3 + blanks(s[3:])
	n := newline(s[i:])
	if n < 0 {
		return -1
	}
	body := i + n
	for j := body; ; {
		nl := strings.IndexByte(s[j:], '\n')
		if nl < 0 {
			break
		}
		j += nl + 1
		if e := closing(s[j:]); e >= 0 {
			return j + e
		}
	}
	if e := closing(s[body:]); e >= 0 {
		return body + e
	}
	return -1
}

// closing matches ---(?:\r?\n|(?!\S)).
func closing(s string) int {
	if !strings.HasPrefix(s, "---") {
		return -1
	}
	if n := newline(s[3:]); n > 0 {
		return 3 + n
	}
	if s[3:] == "" || startsSpace(s[3:]) {
		return 3
	}
	return -1
}

// directive matches [\t ]*%%{[\S\s]*?}%%(?:\r?\n|(?!\S)): the first }%% that a line end, a
// space or the end follows.
func directive(s string) int {
	i := blanks(s)
	if !strings.HasPrefix(s[i:], "%%{") {
		return -1
	}
	for j := i + 3; ; {
		e := strings.Index(s[j:], "}%%")
		if e < 0 {
			return -1
		}
		k := j + e + 3
		if n := newline(s[k:]); n > 0 {
			return k + n
		}
		if k == len(s) || startsSpace(s[k:]) {
			return k
		}
		j += e + 1
	}
}

// comment matches [\t ]*%%[^\n\r]*.
func comment(s string) int {
	i := blanks(s)
	if !strings.HasPrefix(s[i:], "%%") {
		return -1
	}
	return i + 2 + strings.IndexAny(s[i+2:]+"\n", "\n\r")
}

// upToComment is the length of [^\n\r]*?(?=%%)|[^\n\r]*: up to the first %% on the line,
// or else to its end.
func upToComment(s string) int {
	line := s[:strings.IndexAny(s+"\n", "\n\r")]
	if c := strings.Index(line, "%%"); c >= 0 {
		return c
	}
	return len(line)
}

// blankStart is the end of [\t ]*word at the start of s, or -1.
func blankStart(s, word string) int {
	i := blanks(s)
	if !strings.HasPrefix(s[i:], word) {
		return -1
	}
	return i + len(word)
}

// blanks is the length of the run of [\t ] at the start of s.
func blanks(s string) int {
	return len(s) - len(strings.TrimLeft(s, " \t"))
}

// digits is the length of the run of [0-9] at the start of s.
func digits(s string) int {
	return len(s) - len(strings.TrimLeft(s, "0123456789"))
}

// prefix is the length of p if s starts with it, or -1.
func prefix(s, p string) int {
	if !strings.HasPrefix(s, p) {
		return -1
	}
	return len(p)
}

// spaces is the length of the run of what JavaScript's \s matches at the start of s.
func spaces(s string) int {
	return len(s) - len(strings.TrimLeftFunc(s, syntax.IsSpace))
}

// startsSpace is whether s starts with what JavaScript's \s matches.
func startsSpace(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return syntax.IsSpace(r)
}

// isLineEnd is whether r ends a line for a JavaScript regular expression.
func isLineEnd(r rune) bool { return r == '\n' || r == '\r' || r == '\u2028' || r == '\u2029' }
