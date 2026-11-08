package sequence

import (
	"strings"
	"unicode/utf8"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// idRules read a name after participant, actor, activate and the like
// (sequenceDiagram.jison lines 26-39): it ends before " as ", before @{, or
// at the end of the line.
var idRules = []rule{
	{scan: token(skip, lineSpaces)},
	{scan: token(skip, hashComment)},
	{scan: token(kConfig, settings(true)), next: change(0, aliasMode)},
	{scan: token(kConfig, settings(false)), next: change(1)},
	{scan: token(kName, idName(true, beforeConfig))},
	{scan: token(kName, idName(true, beforeAs)), next: change(0, aliasMode)},
	{scan: token(kName, idName(false, atLineEnd)), next: change(1)},
	{scan: token(kInvalid, invalidName), next: change(1)},
	{scan: token(kInvalid, upToLine), next: change(1)},
}

// aliasRules read the as after a name, or go back to the start (lines
// 42-43).
var aliasRules = []rule{
	{scan: token(skip, lineSpaces)},
	{scan: token(skip, hashComment)},
	{scan: token(kAs, kw("as")), next: change(2, lineMode)},
	{scan: token(syntax.Newline, func(_ string, i int) int { return i }), next: change(2)},
}

// lineRules read the rest of a line, to a # or a ; (line 58).
var lineRules = []rule{
	{scan: token(skip, lineSpaces)},
	{scan: token(skip, hashComment)},
	{scan: token(kRest, func(src string, i int) int { return i + upTo(src[i:], "#;\n") }), next: change(1)},
}

// kw matches word as a keyword: see syntax.Keyword.
func kw(word string) func(string, int) int {
	return func(src string, i int) int { return syntax.Keyword(src, i, word) }
}

func lit(s string) func(string, int) int {
	return func(src string, i int) int {
		if strings.HasPrefix(src[i:], s) {
			return i + len(s)
		}
		return -1
	}
}

// oneOf matches the first of ss that src[i:] starts with, ignoring case.
func oneOf(ss []string) func(string, int) int {
	return func(src string, i int) int {
		for _, s := range ss {
			if i+len(s) <= len(src) && strings.EqualFold(src[i:i+len(s)], s) {
				return i + len(s)
			}
		}
		return -1
	}
}

// run matches one or more of the bytes in set.
func run(set string) func(string, int) int {
	return func(src string, i int) int {
		j := i
		for j < len(src) && strings.IndexByte(set, src[j]) >= 0 {
			j++
		}
		if j == i {
			return -1
		}
		return j
	}
}

// spaces matches \s+, line breaks included.
func spaces(src string, i int) int {
	if j := syntax.SkipSpace(src, i); j > i {
		return j
	}
	return -1
}

// lineSpaces matches ((?!\n)\s)+.
func lineSpaces(src string, i int) int {
	j := i
	for j < len(src) {
		r, n := utf8.DecodeRuneInString(src[j:])
		if r == '\n' || !syntax.IsSpace(r) {
			break
		}
		j += n
	}
	if j == i {
		return -1
	}
	return j
}

func hashComment(src string, i int) int {
	if !strings.HasPrefix(src[i:], "#") {
		return -1
	}
	return upToLine(src, i)
}

// percentComment and percentAfter match a comment that syntax.Prepare left:
// one after something on its line, a bare %%, or a %%{ that is no directive.
// Lines 28-29 write \%%(?!\{), which Jison compiles to one % and that lookahead.
func percentComment(src string, i int) int {
	if !strings.HasPrefix(src[i:], "%") || strings.HasPrefix(src[i:], "%{") {
		return -1
	}
	return upToLine(src, i)
}

func percentAfter(src string, i int) int {
	_, n := utf8.DecodeRuneInString(src[i:])
	if src[i] == '}' || !strings.HasPrefix(src[i+n:], "%%") {
		return -1
	}
	return upToLine(src, i)
}

// upToLine matches [^\n]+.
func upToLine(src string, i int) int {
	j := i + upTo(src[i:], "\n")
	if j == i {
		return -1
	}
	return j
}

func anyRune(src string, i int) int {
	_, n := utf8.DecodeRuneInString(src[i:])
	return i + n
}

// upTo is the length of s up to the first byte of set, or all of s.
func upTo(s, set string) int {
	if j := strings.IndexAny(s, set); j >= 0 {
		return j
	}
	return len(s)
}

// name matches a participant's name where a statement starts, as the ACTOR
// rule of sequenceDiagram.jison (line 84) does:
//
//	[^/\\+()<\->:\n,;]+ ( (?!arrowish) -* [^+<\->:\n,;]+ )*
//
// So a name holds spaces, and a - or a / when what follows is not the start
// of an arrow.
func name(src string, i int) int {
	j := i + span(src[i:], `/\+()<->:`+"\n,;")
	if j == i {
		return -1
	}
	for {
		k := j
		if startsWithFold(src[k:], arrowish) {
			return j
		}
		for k < len(src) && src[k] == '-' {
			k++
		}
		n := span(src[k:], "+<->:\n,;")
		if n == 0 {
			return j
		}
		j = k + n
	}
}

// arrowish are the starts of arrows a name stops before.
var arrowish = []string{
	"-x", "--x", "-)", "--)", `-|\`, `-\`, "-/", "-//", "-|/", "/|-", `\|-`, "//-", `\\-`, "/|-", `--|\`, "--", "()",
}

// span is the length of s up to the first byte of stop.
func span(s, stop string) int { return upTo(s, stop) }

func startsWithFold(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if len(s) >= len(p) && strings.EqualFold(s[:len(p)], p) {
			return true
		}
	}
	return false
}

// idName matches a name in idMode: a run of [^<>:\n,;@], with no spaces in
// it when noSpace, as long as it is followed by what look accepts, giving
// back characters from its end until it is (lines 35-37).
func idName(noSpace bool, look func(src string, j int) bool) func(string, int) int {
	return func(src string, i int) int {
		j := i
		for j < len(src) {
			r, n := utf8.DecodeRuneInString(src[j:])
			if strings.ContainsRune("<>:\n,;@", r) || noSpace && syntax.IsSpace(r) {
				break
			}
			j += n
		}
		for k := j; k > i; {
			if look(src, k) {
				return k
			}
			_, n := utf8.DecodeLastRuneInString(src[:k])
			k -= n
		}
		return -1
	}
}

// beforeConfig is \s*@\{.
func beforeConfig(src string, j int) bool {
	return strings.HasPrefix(src[syntax.SkipSpace(src, j):], "@{")
}

// beforeAs is \s+as\s, ignoring case.
func beforeAs(src string, j int) bool {
	k := lineSpaces(src, j)
	if k < 0 || k+3 > len(src) || !strings.EqualFold(src[k:k+2], "as") {
		return false
	}
	r, _ := utf8.DecodeRuneInString(src[k+2:])
	return syntax.IsSpace(r)
}

// atLineEnd is \s*[\n;#] or the end of src.
func atLineEnd(src string, j int) bool {
	if j == len(src) {
		return true
	}
	k := syntax.SkipSpace(src, j)
	return strings.Contains(src[j:k], "\n") || k < len(src) && (src[k] == ';' || src[k] == '#')
}

// invalidName matches a name with a < in it, to the end of the line
// (line 38).
func invalidName(src string, i int) int {
	j := i + upTo(src[i:], "<>:\n,;@")
	if j == len(src) || src[j] != '<' {
		return -1
	}
	return j + 1 + upTo(src[j+1:], "\n")
}

// settings matches @{...}, a participant's settings (lines 31-34), which may
// span lines; when as is true, only if " as " follows it.
func settings(as bool) func(string, int) int {
	return func(src string, i int) int {
		if !strings.HasPrefix(src[i:], "@{") {
			return -1
		}
		end := len(src)
		if j := strings.IndexByte(src[i:], '}'); j >= 0 {
			end = i + j + 1
		}
		if as != beforeAs(src, end) {
			return -1
		}
		return end
	}
}
