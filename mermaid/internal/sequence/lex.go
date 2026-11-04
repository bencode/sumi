package sequence

import (
	"strings"
	"unicode/utf8"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// The kinds of token a sequence diagram has, past those all diagrams share.
const (
	kSD = syntax.FirstKind + iota
	kParticipant
	kActorKw
	kCreate
	kDestroy
	kBox
	kLoop
	kRect
	kOpt
	kAlt
	kElse
	kPar
	kParOver
	kAnd
	kCritical
	kOption
	kBreak
	kEnd
	kLeftOf
	kRightOf
	kMenu // links, link, properties, details
	kOver
	kNote
	kActivate
	kDeactivate
	kAutonumber
	kOff
	kComma
	kNum
	kName   // a participant's name
	kAs     // as, before an alias
	kRest   // the rest of a line: a block's title, an alias
	kConfig // @{...}, a participant's settings
	kArrow
	kHalfArrow // one of the sixteen half arrows, as -|\ or //-
	kText      // : and the text after it
	kPlus
	kMinus
	kCentral // ()
	kInvalid
)

// skip is the kind of a rule that produces no token.
const skip syntax.Kind = -1

// A mode is a start condition of the lexer (sequenceDiagram.jison line 17):
// the rules it applies depend on the keyword before.
type mode uint8

const (
	initial   mode = iota
	idMode         // after participant, actor, activate...: a name
	aliasMode      // after a name and before as
	lineMode       // after as or a block's keyword: the rest of the line
)

// A rule is a lexer rule: it scans a token at src[i:], returning its kind,
// its text and its end, or an end of -1, and changes the mode stack after.
type rule struct {
	scan func(src string, i int) (syntax.Kind, string, int)
	next func(ms []mode) []mode
}

// token makes a rule's scan from a matcher, which returns the end of what it
// matches at src[i:], or -1.
func token(k syntax.Kind, match func(src string, i int) int) func(string, int) (syntax.Kind, string, int) {
	return func(src string, i int) (syntax.Kind, string, int) {
		end := match(src, i)
		if end < 0 {
			return 0, "", -1
		}
		return k, src[i:end], end
	}
}

// change pops n modes off the stack, then pushes ms.
func change(n int, push ...mode) func([]mode) []mode {
	return func(stack []mode) []mode {
		stack = stack[:max(len(stack)-n, 0):max(len(stack)-n, 0)]
		return append(stack, push...)
	}
}

// Lex splits a sequence diagram, made ready by syntax.Prepare, into tokens,
// as the lexer of sequenceDiagram.jison does: in each mode it applies the
// first rule that matches, in the grammar's order. The tokens end with an
// EOF.
func Lex(src string) ([]syntax.Token, error) {
	lines := syntax.LinesOf(src)
	var ts []syntax.Token
	var ms []mode
	for i := 0; ; {
		cur := initial
		if len(ms) > 0 {
			cur = ms[len(ms)-1]
		}
		if i == len(src) && cur == initial {
			at := lines.Pos(i)
			return append(ts, syntax.Token{Kind: syntax.Newline, Pos: at}, syntax.Token{Kind: syntax.EOF, Pos: at}), nil
		}
		k, text, end, r := first(rules[cur], src, i)
		if end < 0 {
			if i == len(src) {
				return append(ts, syntax.Token{Kind: syntax.EOF, Pos: lines.Pos(i)}), nil
			}
			c, _ := utf8.DecodeRuneInString(src[i:])
			return nil, syntax.Errorf(lines.Pos(i), "unexpected %q", c)
		}
		if k != skip {
			ts = append(ts, syntax.Token{Kind: k, Text: text, Pos: lines.Pos(i)})
		}
		if r.next != nil {
			ms = r.next(ms)
		}
		i = end
	}
}

func first(rs []rule, src string, i int) (syntax.Kind, string, int, rule) {
	for _, r := range rs {
		if k, text, end := r.scan(src, i); end >= 0 {
			return k, text, end, r
		}
	}
	return 0, "", -1, rule{}
}

var rules = [...][]rule{initial: initialRules(), idMode: idRules, aliasMode: aliasRules, lineMode: lineRules}

// initialRules are the rules outside any mode, in the grammar's order
// (sequenceDiagram.jison lines 23-126).
func initialRules() []rule {
	toID, toLine := change(0, idMode), change(0, lineMode)
	r := []rule{
		{scan: token(syntax.Newline, run("\n"))},
		{scan: token(skip, spaces)},
		{scan: token(skip, hashComment)},
		{scan: token(skip, percentComment)},
		{scan: token(skip, percentAfter)},
		{scan: token(kNum, num)},
		{scan: token(kBox, kw("box")), next: toLine},
		{scan: token(kParticipant, kw("participant")), next: toID},
		{scan: token(kActorKw, kw("actor")), next: toID},
		{scan: token(kCreate, kw("create"))},
		{scan: token(kDestroy, kw("destroy")), next: toID},
	}
	for _, b := range []struct {
		word string
		k    syntax.Kind
	}{
		{"loop", kLoop}, {"rect", kRect}, {"opt", kOpt}, {"alt", kAlt}, {"else", kElse},
		{"par", kPar}, {"par_over", kParOver}, {"and", kAnd}, {"critical", kCritical},
		{"option", kOption}, {"break", kBreak},
	} {
		r = append(r, rule{scan: token(b.k, kw(b.word)), next: toLine})
	}
	r = append(r,
		rule{scan: token(kEnd, kw("end"))},
		rule{scan: token(kLeftOf, kw("left of"))},
		rule{scan: token(kRightOf, kw("right of"))},
		rule{scan: token(kMenu, menu)},
		rule{scan: token(kOver, kw("over"))},
		rule{scan: token(kNote, kw("note"))},
		rule{scan: token(kActivate, kw("activate")), next: toID},
		rule{scan: token(kDeactivate, kw("deactivate")), next: toID},
		rule{scan: syntax.Meta},
		rule{scan: token(kSD, kw("sequenceDiagram"))},
		rule{scan: token(kAutonumber, kw("autonumber"))},
		rule{scan: token(kOff, kw("off"))},
		rule{scan: token(kComma, lit(","))},
		rule{scan: token(syntax.Newline, lit(";"))},
		rule{scan: token(kName, name)},
		rule{scan: token(kArrow, oneOf(arrows))},
		rule{scan: token(kHalfArrow, oneOf(halfArrows))},
		rule{scan: token(kText, text)},
		rule{scan: token(kPlus, lit("+"))},
		rule{scan: token(kMinus, lit("-"))},
		rule{scan: token(kCentral, lit("()"))},
		rule{scan: token(kInvalid, anyRune)},
	)
	return r
}

// The arrows, in the grammar's order, which is the order to try them in
// (lines 85-94); and the half arrows (lines 96-117), which are not supported
// yet.
var (
	arrows     = []string{"->>", "<<->>", "-->>", "<<-->>", "->", "-->", "-x", "--x", "-)", "--)"}
	halfArrows = []string{
		`--|\`, `--|/`, `--\\`, `--//`, `/|--`, `\|--`, `//--`, `\\--`,
		`-|\`, `-|/`, `-\\`, `-//`, `/|-`, `\|-`, `//-`, `\\-`,
	}
)

// num is a number, followed by a space or a line break (line 30).
func num(src string, i int) int {
	j := i
	for j < len(src) && '0' <= src[j] && src[j] <= '9' {
		j++
	}
	ends := []int{}
	if j < len(src) && src[j] == '.' {
		k := j + 1
		for k < len(src) && k < j+3 && '0' <= src[k] && src[k] <= '9' {
			k++
		}
		for e := k; e > j+1; e-- {
			ends = append(ends, e)
		}
	}
	if j > i {
		ends = append(ends, j)
	}
	for _, e := range ends {
		if e < len(src) && (src[e] == ' ' || src[e] == '\n') {
			return e
		}
	}
	return -1
}

// text is a : and the text after it, to a #, a ; or a line break (line 120).
func text(src string, i int) int {
	if !strings.HasPrefix(src[i:], ":") {
		return -1
	}
	return i + 1 + upTo(src[i+1:], "#;\n")
}

// menu is a keyword of an actor's menu, followed by what can start a name
// (lines 63-66).
func menu(src string, i int) int {
	for _, w := range []string{"links", "link", "properties", "details"} {
		if !strings.EqualFold(src[i:min(i+len(w), len(src))], w) {
			continue
		}
		j := i + len(w)
		k := j
		for k < len(src) && (src[k] == ' ' || src[k] == '\t') {
			k++
		}
		if k > j && k < len(src) {
			if r, _ := utf8.DecodeRuneInString(src[k:]); !syntax.IsSpace(r) && !strings.ContainsRune(`/\+()<>:,;-`, r) {
				return j
			}
		}
	}
	return -1
}
