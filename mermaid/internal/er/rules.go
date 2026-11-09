package er

import (
	"regexp"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// skip is the kind of a rule that makes no token.
const skip syntax.Kind = -1

// The kinds of token a diagram has, past those all diagrams share. NEWLINE is
// syntax.Newline, and kDone is the grammar's EOF, a token of its own before syntax.EOF.
const (
	kDiagram syntax.Kind = syntax.FirstKind + iota
	kDirectionTB
	kDirectionBT
	kDirectionRL
	kDirectionLR
	kAccTitle
	kAccTitleValue
	kAccDescr
	kAccDescrValue
	kAccMultiValue
	kEntityName
	kWord
	kBlockStart
	kBlockStop
	kBrkt
	kComma
	kStyleSeparator
	kColon
	kAttributeKey
	kAttributeWord
	kComment
	kSquareOpen
	kSquareClose
	kStyle
	kClassDef
	kClass
	kSubgraph
	kEnd
	kOnlyOne
	kZeroOrOne
	kZeroOrMore
	kOneOrMore
	kMdParent
	kDecimal
	kEntityOne
	kNum
	kIdentifying
	kNonIdentifying
	kStyleText
	kSemi
	kUnicodeText
	kChar // a character no other rule took
	kDone
)

// A mode is a start condition of the lexer. All but initial are exclusive.
type mode uint8

const (
	initial mode = iota
	block
	blockQuote
	accTitle
	accDescr
	accMulti
	styleMode
)

// A rule is a lexer rule: in the mode in, a match of it makes a token of kind, or none
// when kind is skip, then pops the mode if pop is set and pushes push, unless that is
// initial.
type rule struct {
	in    mode
	match func(s string) int // the length matched at the start of s, or -1
	kind  syntax.Kind
	pop   bool
	push  mode
}

// re is a rule's pattern, case insensitive as the lexer's are all.
func re(p string) func(string) int {
	r := regexp.MustCompile(`(?i)^(?:` + p + `)`)
	return func(s string) int {
		m := r.FindStringIndex(s)
		if m == nil {
			return -1
		}
		return m[1]
	}
}

// ahead is re for a rule with a lookahead, which Go's patterns do not have: p, followed
// by what follow matches.
func ahead(p, follow string) func(string) int {
	match, next := re(p), re(follow)
	return func(s string) int {
		n := match(s)
		if n < 0 || next(s[n:]) < 0 {
			return -1
		}
		return n
	}
}

// atEnd matches only where the source has run out.
func atEnd(s string) int {
	if s == "" {
		return 0
	}
	return -1
}

// JavaScript's \s and its dot, and the range of a pattern with ￿ at its top, which
// takes each half of a surrogate pair and so every character past U+FFFF as well.
const (
	sp      = `[` + syntax.JSSpaces + `]`
	nonSp   = `[^` + syntax.JSSpaces + `]`
	dot     = syntax.JSDot
	highest = `\x{c0}-\x{10ffff}`
)

// rules are the lexer's, in the order of the array Jison compiles, which adds a \b to a
// literal ending in a letter. Each mode tries only its own. Two rules of that array are
// left out, as they never match: the second for a #, and block's line break, which \s+
// has taken first.
var rules = []rule{
	{in: initial, match: re(`accTitle` + sp + `*:` + sp + `*`), kind: kAccTitle, push: accTitle},
	{in: accTitle, match: re(`[^\n]*`), kind: kAccTitleValue, pop: true},
	{in: initial, match: re(`accDescr` + sp + `*:` + sp + `*`), kind: kAccDescr, push: accDescr},
	{in: accDescr, match: re(`[^\n]*`), kind: kAccDescrValue, pop: true},
	{in: initial, match: re(`accDescr` + sp + `*\{` + sp + `*`), kind: skip, push: accMulti},
	{in: accMulti, match: re(`[\}]`), kind: skip, pop: true},
	{in: accMulti, match: re(`[^\}]*`), kind: kAccMultiValue},
	{in: initial, match: re(dot + `*direction` + sp + `+TB[^\n]*`), kind: kDirectionTB},
	{in: initial, match: re(dot + `*direction` + sp + `+BT[^\n]*`), kind: kDirectionBT},
	{in: initial, match: re(dot + `*direction` + sp + `+RL[^\n]*`), kind: kDirectionRL},
	{in: initial, match: re(dot + `*direction` + sp + `+LR[^\n]*`), kind: kDirectionLR},
	{in: initial, match: re(`[ \t\r]+`), kind: skip},
	{in: initial, match: re(`[\n]+`), kind: syntax.Newline},
	{in: initial, match: re(`"[^"%\r\n\x{0b}\x{08}\\]+"`), kind: kEntityName},
	{in: initial, match: re(`"[^"]*"`), kind: kWord},
	{in: initial, match: re(`erDiagram\b`), kind: kDiagram},
	{in: initial, match: re(`\{`), kind: kBlockStart, push: block},
	{in: initial, match: re(`#`), kind: kBrkt},
	{in: initial, match: re(`,`), kind: kComma},
	{in: initial, match: re(`:::`), kind: kStyleSeparator},
	{in: initial, match: re(`:`), kind: kColon},

	{in: block, match: re(sp + `+`), kind: skip},
	{in: block, match: re(`\b((?:PK)|(?:FK)|(?:UK))\b`), kind: kAttributeKey},
	{in: block, match: re(`(` + nonSp + `*)[~]` + dot + `*[~](` + nonSp + `*)`), kind: kAttributeWord},
	{in: block, match: re(`[\*A-Za-z_` + highest + `][A-Za-z0-9\-_\[\]\(\)\.,` + highest + `\*]*`), kind: kAttributeWord},
	{in: block, match: re("[`]"), kind: skip, push: blockQuote},
	{in: blockQuote, match: re("[^`]+"), kind: kAttributeWord},
	{in: blockQuote, match: re("[`]"), kind: skip, pop: true},
	{in: block, match: re(`"[^"]*"`), kind: kComment},
	{in: block, match: re(`\}`), kind: kBlockStop, pop: true},
	{in: block, match: re(dot), kind: kChar},

	{in: initial, match: re(`\[`), kind: kSquareOpen},
	{in: initial, match: re(`\]`), kind: kSquareClose},
	{in: initial, match: re(`style\b`), kind: kStyle, push: styleMode},

	{in: styleMode, match: re(`[\n]+`), kind: syntax.Newline, pop: true},
	{in: styleMode, match: re(sp + `+`), kind: skip},
	{in: styleMode, match: re(`:`), kind: kColon},
	{in: styleMode, match: re(`,`), kind: kComma},
	{in: styleMode, match: re(`#`), kind: kBrkt},
	{in: styleMode, match: re(`([^\x00-\x7F]|\w|-|\*)+`), kind: kStyleText},
	{in: styleMode, match: re(`;`), kind: kSemi},

	{in: initial, match: re(`classDef\b`), kind: kClassDef, push: styleMode},
	{in: initial, match: re(`class\b`), kind: kClass},
	{in: initial, match: re(`subgraph\b`), kind: kSubgraph},
	{in: initial, match: re(`end\b` + sp + `*`), kind: kEnd},
	{in: initial, match: re(`one or zero\b`), kind: kZeroOrOne},
	{in: initial, match: re(`one or more\b`), kind: kOneOrMore},
	{in: initial, match: re(`one or many\b`), kind: kOneOrMore},
	{in: initial, match: re(`1\+`), kind: kOneOrMore},
	{in: initial, match: re(`\|o\b`), kind: kZeroOrOne},
	{in: initial, match: re(`zero or one\b`), kind: kZeroOrOne},
	{in: initial, match: re(`zero or more\b`), kind: kZeroOrMore},
	{in: initial, match: re(`zero or many\b`), kind: kZeroOrMore},
	{in: initial, match: re(`0\+`), kind: kZeroOrMore},
	{in: initial, match: re(`\}o\b`), kind: kZeroOrMore},
	{in: initial, match: re(`many\(0\)`), kind: kZeroOrMore},
	{in: initial, match: re(`many\(1\)`), kind: kOneOrMore},
	{in: initial, match: re(`many\b`), kind: kZeroOrMore},
	{in: initial, match: re(`\}\|`), kind: kOneOrMore},
	{in: initial, match: re(`one\b`), kind: kOnlyOne},
	{in: initial, match: re(`only one\b`), kind: kOnlyOne},
	{in: initial, match: re(`[0-9]+\.[0-9]+`), kind: kDecimal},
	{in: initial, match: ahead(`1`, sp+`+[A-Za-z_"']`), kind: kOnlyOne},
	{in: initial, match: ahead(`1`, sp+`+[0-9]`), kind: kOnlyOne},
	{in: initial, match: ahead(`1`, `--|\.\.|\.-|-\.`), kind: kOnlyOne},
	{in: initial, match: re(`1\b`), kind: kEntityOne},
	{in: initial, match: re(`[0-9]+`), kind: kNum},
	{in: initial, match: re(`\|\|`), kind: kOnlyOne},
	{in: initial, match: re(`o\|`), kind: kZeroOrOne},
	{in: initial, match: re(`o\{`), kind: kZeroOrMore},
	{in: initial, match: re(`\|\{`), kind: kOneOrMore},
	{in: initial, match: ahead(`u`, `[\.\-\|]`), kind: kMdParent},
	{in: initial, match: re(`\.\.`), kind: kNonIdentifying},
	{in: initial, match: re(`--`), kind: kIdentifying},
	{in: initial, match: re(`to\b`), kind: kIdentifying},
	{in: initial, match: re(`optionally to\b`), kind: kNonIdentifying},
	{in: initial, match: re(`\.-`), kind: kNonIdentifying},
	{in: initial, match: re(`-\.`), kind: kNonIdentifying},
	{in: initial, match: re(`([^\x00-\x7F]|\w|-|\*|\.)+`), kind: kUnicodeText},
	{in: initial, match: re(dot), kind: kChar},
	{in: initial, match: atEnd, kind: kDone},
}
