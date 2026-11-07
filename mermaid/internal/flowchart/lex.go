package flowchart

import (
	"regexp"
	"sort"
	"unicode/utf8"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// A mode is a start condition of the lexer (flow.jison).
type mode uint8

const (
	initial mode = iota
	stringMode
	markdownMode
	accTitleMode
	accDescrMode
	accBlockMode
	dirMode
	textMode
	ellipseMode
	trapMode
	edgeMode
	thickMode
	dottedMode
	clickMode
	callbackNameMode
	callbackArgsMode
	dataMode
	dataStringMode
)

// A rule is a lexer rule: in the modes of in, a match of it makes a token of kind,
// or none when kind is skip, then pops pop modes and pushes push.
type rule struct {
	in    uint32
	match func(s string) int // the length matched at the start of s, or -1
	kind  syntax.Kind
	pop   int
	push  mode // initial pushes nothing
	graph bool // pushes dirMode the first time it matches
	text  func(s string) string
}

// braceEnd is the end of a line after a }, which flowParser.ts makes a bare }\n.
var braceEnd = regexp.MustCompile(`\}[` + spaces + `]*\n`)

// normalize does what flowParser.ts does to a source before parsing it, and returns
// with it a function that turns an offset in the result into one in src.
func normalize(src string) (string, func(int) int) {
	var out []byte
	var at, dropped []int // where each cut is in out, and all dropped up to it
	last, total := 0, 0
	for _, m := range braceEnd.FindAllStringIndex(src, -1) {
		out = append(append(out, src[last:m[0]+1]...), '\n')
		total += m[1] - m[0] - 2
		at, dropped = append(at, len(out)-1), append(dropped, total)
		last = m[1]
	}
	out = append(out, src[last:]...)
	return string(out), func(i int) int {
		n := sort.SearchInts(at, i+1)
		if n == 0 {
			return i
		}
		return i + dropped[n-1]
	}
}

// Lex splits a flowchart, made ready by syntax.Prepare, into tokens, as the lexer
// Jison makes of flow.jison does: in each mode it applies the first rule that
// matches. When the source runs out it tries the rules once more, as the lexer does,
// which gives accTitle: at the very end its empty value.
func Lex(src string) ([]syntax.Token, error) {
	lines := syntax.LinesOf(src)
	text, origin := normalize(src)
	var ts []syntax.Token
	stack := []mode{initial}
	seenGraph := false
	for i, done := 0, false; !done; {
		done = i == len(text)
		r, n := first(stack[len(stack)-1], text[i:])
		at := lines.Pos(origin(i))
		if n < 0 {
			if done {
				break
			}
			c, _ := utf8.DecodeRuneInString(text[i:])
			return nil, syntax.Errorf(at, "unexpected %q", c)
		}
		if r.kind != skip {
			s := text[i : i+n]
			if r.text != nil {
				s = r.text(s)
			}
			ts = append(ts, syntax.Token{Kind: r.kind, Text: s, Pos: at})
		}
		for range r.pop {
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
		}
		if r.push != initial {
			stack = append(stack, r.push)
		}
		if r.graph && !seenGraph {
			seenGraph = true
			stack = append(stack, dirMode)
		}
		i += n
	}
	return append(ts, syntax.Token{Kind: syntax.EOF, Pos: lines.Pos(len(src))}), nil
}

// first is the first rule of m that matches at the start of s, and the length of its
// match, or -1.
func first(m mode, s string) (rule, int) {
	for _, r := range rules {
		if r.in&(1<<m) == 0 {
			continue
		}
		if n := r.match(s); n >= 0 {
			return r, n
		}
	}
	return rule{}, -1
}
