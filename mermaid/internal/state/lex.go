package state

import (
	"strings"
	"unicode/utf8"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// Lex splits a state diagram into tokens as the lexer Jison makes of
// stateDiagram.jison does: in each mode it applies the first rule that matches. When
// the source runs out it tries the rules once more, which is where the last NL comes
// from.
func Lex(src string) ([]syntax.Token, error) {
	lines := syntax.LinesOf(src)
	var ts []syntax.Token
	stack := []mode{initial}
	for i, done := 0, false; !done; {
		done = i == len(src)
		r, n := first(stack[len(stack)-1], src[i:])
		at := lines.Pos(i)
		if n < 0 {
			if done {
				break
			}
			c, _ := utf8.DecodeRuneInString(src[i:])
			return nil, syntax.Errorf(at, "unexpected %q", c)
		}
		s := src[i : i+n]
		if r.fail {
			return nil, syntax.Errorf(at, "a state's name must be a single word, not %q", trim(s))
		}
		if r.id {
			// processId: an id stops at a %% comment, and one that starts with it is none
			if cut := strings.Index(s, "%%"); cut == 0 {
				i += n
				continue
			} else if cut > 0 {
				s, n = s[:cut], cut
			}
		}
		if r.kind != skip {
			if r.text != nil {
				s = r.text(s)
			}
			ts = append(ts, syntax.Token{Kind: r.kind, Text: s, Pos: at})
		}
		if r.pop && len(stack) > 1 {
			stack = stack[:len(stack)-1]
		}
		if r.push != initial {
			stack = append(stack, r.push)
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
