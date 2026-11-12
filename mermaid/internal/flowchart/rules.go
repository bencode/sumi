package flowchart

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// The kinds of token a flowchart has, past those all diagrams share. Newline is NEWLINE
// alone: a ; is kSemi, as the first line tells them apart.
const (
	kAccTitle = syntax.FirstKind + iota
	kAccTitleValue
	kAccDescr
	kAccDescrValue
	kAccBlock
	kData
	kCallbackName
	kCallbackArgs
	kMarkdown
	kString
	kStyle
	kDefault
	kLinkStyle
	kInterpolate
	kClassDef
	kClass
	kHref
	kClick
	kGraph
	kSubgraph
	kEnd
	kLinkTarget
	kNoDir
	kDir
	kDirection
	kLinkID
	kNum
	kBrkt
	kStyleSep
	kColon
	kAmp
	kSemi
	kComma
	kMult
	kLink
	kStartLink
	kEdgeText
	kEllipseEnd
	kEllipseStart
	kText
	kStadiumEnd
	kStadiumStart
	kSubroutineEnd
	kSubroutineStart
	kVertexProps
	kTagEnd
	kCylinderEnd
	kCylinderStart
	kDoubleCircleEnd
	kDoubleCircleStart
	kTrapEnd
	kInvTrapEnd
	kTrapStart
	kInvTrapStart
	kTagStart
	kUp
	kSep
	kDown
	kNodeString
	kMinus
	kUnicode
	kPipe
	kPE
	kPS
	kSQE
	kSQS
	kDiamondStop
	kDiamondStart
	kSpace
	skip syntax.Kind = -1
)

// spaces is what \s matches in JavaScript, for a character class; blanks is the
// same without line breaks, for a space between tokens.
const (
	blanks = "\\t\\x{0b}\\f \\x{a0}\\x{1680}\\x{2000}-\\x{200a}\\x{2028}\\x{2029}\\x{202f}\\x{205f}\\x{3000}\\x{feff}"
	spaces = "\\n\\r" + blanks
)

func in(ms ...mode) uint32 {
	var set uint32
	for _, m := range ms {
		set |= 1 << m
	}
	return set
}

// everywhere is what Jison's <*> means: every mode.
const everywhere = ^uint32(0)

// re makes a matcher of a pattern, which reads \s and \S as JavaScript does.
func re(pattern string) func(string) int {
	pattern = strings.ReplaceAll(pattern, `\s`, "["+spaces+"]")
	pattern = strings.ReplaceAll(pattern, `\S`, "[^"+spaces+"]")
	r := regexp.MustCompile(`\A(?:` + pattern + `)`)
	return func(s string) int {
		if m := r.FindStringIndex(s); m != nil {
			return m[1]
		}
		return -1
	}
}

// directionLine matches the rest of a line that says "direction" and d anywhere in it,
// so a line that only mentions one is taken for a direction statement too.
func directionLine(d string) func(string) int {
	return re(`[^\n\r\x{2028}\x{2029}]*direction\s+` + d + `[^\n]*`)
}

// linkID is [^\s"[\](){}|]+@(?=[^{"]): an edge's id, and the @ after it.
func linkID(s string) int {
	run := strings.IndexFunc(s, func(c rune) bool { return syntax.IsSpace(c) || strings.ContainsRune(`"[](){}|`, c) })
	if run < 0 {
		run = len(s)
	}
	for i := run - 1; i > 0; i-- {
		if s[i] == '@' && i+1 < len(s) && s[i+1] != '{' && s[i+1] != '"' {
			return i + 1
		}
	}
	return -1
}

// edgeChar is [^-]|-(?!-)+: a character of an edge's text.
func edgeChar(s string) int {
	if s == "" || strings.HasPrefix(s, "--") {
		return -1
	}
	_, n := utf8.DecodeRuneInString(s)
	return n
}

// trapText is \/(?!\])|\\(?!\])|[^\\\[\]\(\)\{\}\/]+: text in a trapezoid. The rules
// before it take a / or a \ with a ] after it, so it need not look.
func trapText(s string) int {
	if s == "" {
		return -1
	}
	if s[0] == '/' || s[0] == '\\' {
		return 1
	}
	if n := strings.IndexAny(s, `\[](){}/`); n != 0 {
		if n < 0 {
			return len(s)
		}
		return n
	}
	return -1
}

// nodeString is a run of ([A-Za-z0-9!"\#$%&'*+\.`?\\_\/]|-(?=[^\>\-\.])). Mermaid's
// grammar has a third branch, for =, that Jison compiles to one that never matches.
func nodeString(s string) int {
	i := 0
	for i < len(s) {
		c := s[i]
		if c == '-' {
			if i+1 == len(s) || strings.IndexByte(">-.", s[i+1]) >= 0 {
				break
			}
		} else if !syntax.IsWord(c) && strings.IndexByte("!\"#$%&'*+.`?\\/", c) < 0 {
			break
		}
		i++
	}
	if i == 0 {
		return -1
	}
	return i
}

// letter is one of the characters of letters.
func letter(s string) int {
	c, n := utf8.DecodeRuneInString(s)
	if n == 0 || !unicode.Is(letters, c) {
		return -1
	}
	return n
}

// breaks is what the lexer makes of a line break in a string in @{...}.
var breaks = regexp.MustCompile(`\n[` + spaces + `]*`)

// The sets of modes the rules are in; start is the first mode alone.
var (
	start     = in(initial)
	linkEnds  = in(initial, edgeMode)
	thickEnds = in(initial, thickMode)
	dotEnds   = in(initial, dottedMode)
	inText    = in(textMode)
	inTrap    = in(trapMode)
)

// rules are the rules of flow.jison as Jison compiles them, which is not always as
// they are written: =(?!=) and \.(?!-) lose what follows the ?!, and never match. In
// the grammar's order, without those that an earlier rule always takes first, and
// without <<EOF>>: a source ends with a line break, so no statement waits for it.
var rules = []rule{
	{in: start, match: re(`accTitle\s*:\s*`), kind: kAccTitle, push: accTitleMode},
	{in: in(accTitleMode), match: re(`[^\n]*`), kind: kAccTitleValue, pop: 1},
	{in: start, match: re(`accDescr\s*:\s*`), kind: kAccDescr, push: accDescrMode},
	{in: in(accDescrMode), match: re(`[^\n]*`), kind: kAccDescrValue, pop: 1},
	{in: start, match: re(`accDescr\s*\{\s*`), kind: skip, push: accBlockMode},
	{in: in(accBlockMode), match: re(`\}`), kind: skip, pop: 1},
	{in: in(accBlockMode), match: re(`[^\}]*`), kind: kAccBlock},
	{in: start, match: re(`@\{`), kind: kData, push: dataMode, text: func(string) string { return "" }},
	{in: in(dataMode), match: re(`"`), kind: kData, push: dataStringMode},
	{in: in(dataStringMode), match: re(`"`), kind: kData, pop: 1},
	{in: in(dataStringMode), match: re(`[^"]+`), kind: kData, text: func(s string) string { return breaks.ReplaceAllString(s, "<br/>") }},
	{in: in(dataMode), match: re(`[^}^"]+`), kind: kData},
	{in: in(dataMode), match: re(`\}`), kind: skip, pop: 1},
	{in: start, match: re(`call\s+`), kind: skip, push: callbackNameMode},
	{in: in(callbackNameMode), match: re(`\(\s*\)`), kind: skip, pop: 1},
	{in: in(callbackNameMode), match: re(`\(`), kind: skip, pop: 1, push: callbackArgsMode},
	{in: in(callbackNameMode), match: re(`[^(]*`), kind: kCallbackName},
	{in: in(callbackArgsMode), match: re(`\)`), kind: skip, pop: 1},
	{in: in(callbackArgsMode), match: re(`[^)]*`), kind: kCallbackArgs},
	{in: in(markdownMode), match: re("[^`\"]+"), kind: kMarkdown},
	{in: in(markdownMode), match: re("`\""), kind: skip, pop: 1},
	{in: everywhere, match: re("\"`"), kind: skip, push: markdownMode},
	{in: in(stringMode), match: re(`[^"]+`), kind: kString},
	{in: in(stringMode), match: re(`"`), kind: skip, pop: 1},
	{in: everywhere, match: re(`"`), kind: skip, push: stringMode},
	{in: start, match: re(`style\b`), kind: kStyle},
	{in: start, match: re(`default\b`), kind: kDefault},
	{in: start, match: re(`linkStyle\b`), kind: kLinkStyle},
	{in: start, match: re(`interpolate\b`), kind: kInterpolate},
	{in: start, match: re(`classDef\b`), kind: kClassDef},
	{in: start, match: re(`class\b`), kind: kClass},
	{in: start, match: re(`href\s`), kind: kHref},
	{in: start, match: re(`click\s+`), kind: skip, push: clickMode},
	{in: in(clickMode), match: re(`\s`), kind: skip, pop: 1},
	{in: in(clickMode), match: re(`\S*`), kind: kClick},
	{in: start, match: re(`flowchart-elk\b`), kind: kGraph, graph: true},
	{in: start, match: re(`swimlane-beta\b`), kind: kGraph, graph: true},
	{in: start, match: re(`graph\b`), kind: kGraph, graph: true},
	{in: start, match: re(`flowchart\b`), kind: kGraph, graph: true},
	{in: start, match: re(`subgraph\b`), kind: kSubgraph},
	{in: start, match: re(`end\b\s*`), kind: kEnd},
	{in: start, match: re(`_self\b`), kind: kLinkTarget},
	{in: start, match: re(`_blank\b`), kind: kLinkTarget},
	{in: start, match: re(`_parent\b`), kind: kLinkTarget},
	{in: start, match: re(`_top\b`), kind: kLinkTarget},
	{in: in(dirMode), match: re(`(\r?\n)*\s*\n`), kind: kNoDir, pop: 1},
	{in: in(dirMode), match: re(`\s*LR\b`), kind: kDir, pop: 1},
	{in: in(dirMode), match: re(`\s*RL\b`), kind: kDir, pop: 1},
	{in: in(dirMode), match: re(`\s*TB\b`), kind: kDir, pop: 1},
	{in: in(dirMode), match: re(`\s*BT\b`), kind: kDir, pop: 1},
	{in: in(dirMode), match: re(`\s*TD\b`), kind: kDir, pop: 1},
	{in: in(dirMode), match: re(`\s*BR\b`), kind: kDir, pop: 1},
	{in: in(dirMode), match: re(`\s*<`), kind: kDir, pop: 1},
	{in: in(dirMode), match: re(`\s*>`), kind: kDir, pop: 1},
	{in: in(dirMode), match: re(`\s*\^`), kind: kDir, pop: 1},
	{in: in(dirMode), match: re(`\s*v\b`), kind: kDir, pop: 1},
	{in: start, match: directionLine("TB"), kind: kDirection, text: value("TB")},
	{in: start, match: directionLine("BT"), kind: kDirection, text: value("BT")},
	{in: start, match: directionLine("RL"), kind: kDirection, text: value("RL")},
	{in: start, match: directionLine("LR"), kind: kDirection, text: value("LR")},
	{in: start, match: directionLine("TD"), kind: kDirection, text: value("TD")},
	{in: start, match: linkID, kind: kLinkID},
	{in: start, match: re(`[0-9]+`), kind: kNum},
	{in: start, match: re(`#`), kind: kBrkt},
	{in: start, match: re(`:::`), kind: kStyleSep},
	{in: start, match: re(`:`), kind: kColon},
	{in: start, match: re(`&`), kind: kAmp},
	{in: start, match: re(`;`), kind: kSemi},
	{in: start, match: re(`,`), kind: kComma},
	{in: start, match: re(`\*`), kind: kMult},
	{in: linkEnds, match: re(`\s*[xo<]?--+[-xo>]\s*`), kind: kLink, pop: 1},
	{in: start, match: re(`\s*[xo<]?--\s*`), kind: kStartLink, push: edgeMode},
	{in: in(edgeMode), match: edgeChar, kind: kEdgeText},
	{in: thickEnds, match: re(`\s*[xo<]?==+[=xo>]\s*`), kind: kLink, pop: 1},
	{in: start, match: re(`\s*[xo<]?==\s*`), kind: kStartLink, push: thickMode},
	{in: in(thickMode), match: re(`[^=]`), kind: kEdgeText},
	{in: dotEnds, match: re(`\s*[xo<]?-?\.+-[xo>]?\s*`), kind: kLink, pop: 1},
	{in: start, match: re(`\s*[xo<]?-\.\s*`), kind: kStartLink, push: dottedMode},
	{in: in(dottedMode), match: re(`[^.]`), kind: kEdgeText},
	{in: everywhere, match: re(`\s*~~~+\s*`), kind: kLink},
	{in: in(ellipseMode), match: re(`[-/)]\)`), kind: kEllipseEnd, pop: 1},
	{in: in(ellipseMode), match: re(`[^()\[\]{}]|!\)+`), kind: kText},
	{in: everywhere, match: re(`\(-`), kind: kEllipseStart, push: ellipseMode},
	{in: inText, match: re(`\]\)`), kind: kStadiumEnd, pop: 1},
	{in: everywhere, match: re(`\(\[`), kind: kStadiumStart, push: textMode},
	{in: inText, match: re(`\]\]`), kind: kSubroutineEnd, pop: 1},
	{in: everywhere, match: re(`\[\[`), kind: kSubroutineStart, push: textMode},
	{in: start, match: re(`\[\|`), kind: kVertexProps},
	{in: start, match: re(`>`), kind: kTagEnd, push: textMode},
	{in: inText, match: re(`\)\]`), kind: kCylinderEnd, pop: 1},
	{in: everywhere, match: re(`\[\(`), kind: kCylinderStart, push: textMode},
	{in: inText, match: re(`\)\)\)`), kind: kDoubleCircleEnd, pop: 1},
	{in: everywhere, match: re(`\(\(\(`), kind: kDoubleCircleStart, push: textMode},
	{in: inTrap, match: re(`[\\(?=\])]\]`), kind: kTrapEnd, pop: 1},
	{in: inTrap, match: re(`/\]`), kind: kInvTrapEnd, pop: 1},
	{in: inTrap, match: trapText, kind: kText},
	{in: everywhere, match: re(`\[/`), kind: kTrapStart, push: trapMode},
	{in: everywhere, match: re(`\[\\`), kind: kInvTrapStart, push: trapMode},
	{in: start, match: re(`<`), kind: kTagStart},
	{in: start, match: re(`\^`), kind: kUp},
	{in: start, match: re(`\\\|`), kind: kSep},
	{in: start, match: re(`v\b`), kind: kDown},
	{in: start, match: nodeString, kind: kNodeString},
	{in: start, match: re(`-`), kind: kMinus},
	{in: start, match: letter, kind: kUnicode},
	{in: inText, match: re(`\|`), kind: kPipe, pop: 1},
	{in: everywhere, match: re(`\|`), kind: kPipe, push: textMode},
	{in: inText, match: re(`\)`), kind: kPE, pop: 1},
	{in: everywhere, match: re(`\(`), kind: kPS, push: textMode},
	{in: inText, match: re(`\]`), kind: kSQE, pop: 1},
	{in: everywhere, match: re(`\[`), kind: kSQS, push: textMode},
	{in: inText, match: re(`\}`), kind: kDiamondStop, pop: 1},
	{in: everywhere, match: re(`\{`), kind: kDiamondStart, push: textMode},
	{in: inText, match: re(`[^\[\](){}|"]+`), kind: kText},
	{in: start, match: re(`(\r?\n)+`), kind: syntax.Newline},
	{in: start, match: re(`[` + blanks + `]+`), kind: kSpace},
}

func value(d string) func(string) string { return func(string) string { return d } }
