package gantt

import (
	"strings"
	"unicode/utf8"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// Lex splits a diagram into tokens as the lexer Jison makes of gantt.jison does. When
// the source runs out it tries the rules once more, which is where kDone comes from. It
// starts past the spaces Mermaid trims before it parses, which here include the lines
// syntax.Prepare has blanked.
func Lex(src string) ([]syntax.Token, error) {
	lines := syntax.LinesOf(src)
	var ts []syntax.Token
	stack := []mode{initial}
	for i, done := len(src)-len(strings.TrimLeftFunc(src, isJSSpace)), false; !done; {
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
		if r.kind != skip {
			ts = append(ts, syntax.Token{Kind: r.kind, Text: src[i : i+n], Pos: at})
		}
		if r.pop {
			stack = stack[:max(len(stack)-1, 1)]
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
		if r.in != m {
			continue
		}
		if n := r.match(s); n >= 0 {
			return r, n
		}
	}
	return rule{}, -1
}
