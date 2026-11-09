package state

import (
	"regexp"
	"strings"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// skip is the kind of a rule that makes no token.
const skip syntax.Kind = -1

// The kinds of token a state diagram has, past those all diagrams share. NL is
// syntax.Newline.
const (
	kSD syntax.Kind = syntax.FirstKind + iota
	kClick
	kHref
	kString
	kDefault
	kDirection
	kScale
	kWidth
	kAccTitle
	kAccTitleValue
	kAccDescr
	kAccDescrValue
	kAccBlockValue
	kClassDef
	kClassDefID
	kDefaultClassDef
	kClassDefStyles
	kClass
	kClassIDs
	kClassName
	kStyle
	kStyleIDs
	kStyleStyles
	kFork
	kJoin
	kChoice
	kAs
	kID
	kStateDescr
	kComposite
	kStructStart
	kStructStop
	kNote
	kLeftOf
	kRightOf
	kNoteText
	kHideEmpty
	kEdgeState
	kDescr
	kArrow
	kConcurrent
	kStyleSep
	kInvalid
)

// A mode is a start condition of the lexer.
type mode uint8

const (
	initial mode = iota
	structMode
	stateMode
	stateStringMode
	stateIDMode
	scaleMode
	accTitleMode
	accDescrMode
	accBlockMode
	classDefMode
	classDefIDMode
	classMode
	classStyleMode
	styleMode
	styleDefMode
	noteMode
	noteIDMode
	noteTextMode
	floatingNoteMode
	floatingNoteIDMode
)

// A rule is a lexer rule: in the modes of in, a match of it makes a token of kind, or
// none when kind is skip, then pops a mode if pop is set and pushes push, unless that
// is initial. id is processId, and fail is the error of a state name of two words.
type rule struct {
	in       uint32
	match    func(s string) int // the length matched at the start of s, or -1
	kind     syntax.Kind
	pop      bool
	push     mode
	text     func(s string) string
	id, fail bool
}

func in(ms ...mode) uint32 {
	var out uint32
	for _, m := range ms {
		out |= 1 << m
	}
	return out
}

// goRegexp writes p, a rule of the compiled lexer without its lookaheads, for Go:
// \s and the dot as JavaScript reads them, and letters in either case, as the /i of
// a regular expression without /u reads them, which leaves ſ and K alone.
func goRegexp(p string) string {
	var b strings.Builder
	class := false
	for i := 0; i < len(p); i++ {
		c := p[i]
		switch {
		case strings.HasPrefix(p[i:], `[\s\S]`):
			b.WriteString(`[\x{0}-\x{10ffff}]`)
			i += 5
		case c == '\\' && p[i+1] == 's' && class:
			b.WriteString(syntax.JSSpaces)
			i++
		case c == '\\' && p[i+1] == 's':
			b.WriteString("[" + syntax.JSSpaces + "]")
			i++
		case c == '\\':
			b.WriteString(p[i : i+2])
			i++
		case c == '.' && !class:
			b.WriteString(syntax.JSDot)
		case !class && (c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'):
			b.WriteString("[" + strings.ToLower(string(c)) + strings.ToUpper(string(c)) + "]")
		default:
			class = c == '[' || class && c != ']'
			b.WriteByte(c)
		}
	}
	return b.String()
}

func re(p string) func(string) int {
	r := regexp.MustCompile(`^(?:` + goRegexp(p) + `)`)
	return func(s string) int {
		m := r.FindStringIndex(s)
		if m == nil {
			return -1
		}
		return m[1]
	}
}

// comment is %%(?!\{)[^\n]*.
func comment(s string) int {
	if !strings.HasPrefix(s, "%%") || strings.HasPrefix(s, "%%{") {
		return -1
	}
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return i
	}
	return len(s)
}

func trim(s string) string {
	return strings.TrimFunc(s, func(r rune) bool { return r != '\u0085' && syntax.IsSpace(r) })
}

// without trims s of its last n bytes and then of its spaces.
func without(n int) func(string) string {
	return func(s string) string { return trim(s[:len(s)-n]) }
}

// afterTwo is substr(2).trim(): a note's text, without whatever its first two
// characters are, a colon and a space as a rule.
func afterTwo(s string) string {
	r := []rune(s)
	return trim(string(r[min(2, len(r)):]))
}

// directionOf is the rule of a line that names the direction dir, wherever on it.
func directionOf(dir string) string { return `.*direction\s+` + dir + `[^\n]*` }

func value(v string) func(string) string { return func(string) string { return v } }

var (
	lineModes = in(initial, structMode, stateMode)
	bothModes = in(initial, structMode)
)

// rules are the compiled rules in their order, with those no mode reaches left out.
var rules = []rule{
	{in: in(initial), match: re(`click\b`), kind: kClick},
	{in: in(initial), match: re(`href\b`), kind: kHref},
	{in: in(initial), match: re(`"[^"]*"`), kind: kString},
	{in: in(initial), match: re(`default\b`), kind: kDefault},
	{in: in(initial), match: re(directionOf("TB")), kind: kDirection, text: value("TB")},
	{in: in(initial), match: re(directionOf("BT")), kind: kDirection, text: value("BT")},
	{in: in(initial), match: re(directionOf("RL")), kind: kDirection, text: value("RL")},
	{in: in(initial), match: re(directionOf("LR")), kind: kDirection, text: value("LR")},
	{in: in(initial), match: re(`[\n]+`), kind: syntax.Newline},
	{in: in(initial), match: re(`[\s]+`), kind: skip},
	{in: in(structMode, stateMode), match: re(`[` + syntax.JSBlanks + `]+`), kind: skip},
	{in: lineModes, match: re(`#[^\n]*`), kind: skip},
	{in: lineModes, match: comment, kind: skip},
	{in: in(initial), match: re(`scale\s+`), kind: kScale, push: scaleMode},
	{in: in(scaleMode), match: re(`\d+`), kind: kWidth},
	{in: in(scaleMode), match: re(`\s+width\b`), kind: skip, pop: true},
	{in: in(initial), match: re(`accTitle\s*:\s*`), kind: kAccTitle, push: accTitleMode},
	{in: in(accTitleMode), match: re(`[^\n]*`), kind: kAccTitleValue, pop: true},
	{in: in(initial), match: re(`accDescr\s*:\s*`), kind: kAccDescr, push: accDescrMode},
	{in: in(accDescrMode), match: re(`[^\n]*`), kind: kAccDescrValue, pop: true},
	{in: in(initial), match: re(`accDescr\s*\{\s*`), kind: skip, push: accBlockMode},
	{in: in(accBlockMode), match: re(`[\}]`), kind: skip, pop: true},
	{in: in(accBlockMode), match: re(`[^\}]*`), kind: kAccBlockValue},
	{in: bothModes, match: re(`classDef\s+`), kind: kClassDef, push: classDefMode},
	{in: in(classDefMode), match: re(`DEFAULT\s+`), kind: kDefaultClassDef, pop: true, push: classDefIDMode},
	{in: in(classDefMode), match: re(`\w+\s+`), kind: kClassDefID, pop: true, push: classDefIDMode},
	{in: in(classDefIDMode), match: re(`[^\n]*`), kind: kClassDefStyles, pop: true},
	{in: bothModes, match: re(`class\s+`), kind: kClass, push: classMode},
	{in: in(classMode), match: re(`(\w+)+((,\s*\w+)*)`), kind: kClassIDs, pop: true, push: classStyleMode},
	{in: in(classStyleMode), match: re(`[^\n]*`), kind: kClassName, pop: true},
	{in: bothModes, match: re(`style\s+`), kind: kStyle, push: styleMode},
	{in: in(styleMode), match: re(`[\w,]+\s+`), kind: kStyleIDs, pop: true, push: styleDefMode},
	{in: in(styleDefMode), match: re(`[^\n]*`), kind: kStyleStyles, pop: true},
	{in: bothModes, match: re(`state\s+`), kind: skip, push: stateMode},
	{in: in(stateMode), match: re(`.*<<fork>>`), kind: kFork, pop: true, text: without(8)},
	{in: in(stateMode), match: re(`.*<<join>>`), kind: kJoin, pop: true, text: without(8)},
	{in: in(stateMode), match: re(`.*<<choice>>`), kind: kChoice, pop: true, text: without(10)},
	{in: in(stateMode), match: re(`.*\[\[fork\]\]`), kind: kFork, pop: true, text: without(8)},
	{in: in(stateMode), match: re(`.*\[\[join\]\]`), kind: kJoin, pop: true, text: without(8)},
	{in: in(stateMode), match: re(`.*\[\[choice\]\]`), kind: kChoice, pop: true, text: without(10)},
	{in: in(structMode), match: re(directionOf("TB")), kind: kDirection, text: value("TB")},
	{in: in(structMode), match: re(directionOf("BT")), kind: kDirection, text: value("BT")},
	{in: in(structMode), match: re(directionOf("RL")), kind: kDirection, text: value("RL")},
	{in: in(structMode), match: re(directionOf("LR")), kind: kDirection, text: value("LR")},
	{in: in(stateMode), match: re(`["]`), kind: skip, push: stateStringMode},
	{in: in(stateMode), match: re(`\s*as\s+`), kind: kAs, push: stateIDMode},
	{in: in(stateIDMode), match: re(`[^\n\{]*`), kind: kID, pop: true, id: true},
	{in: in(stateStringMode), match: re(`["]`), kind: skip, pop: true},
	{in: in(stateStringMode), match: re(`[^"]*`), kind: kStateDescr},
	{in: in(stateMode), match: re(`\w+\s+\w+.*?\{`), fail: true},
	{in: in(stateMode), match: re(`[^\n\s\{]+`), kind: kComposite},
	{in: in(stateMode), match: re(`\n`), kind: skip, pop: true},
	{in: in(initial, stateMode), match: re(`\{`), kind: kStructStart, pop: true, push: structMode},
	{in: in(structMode), match: re(`\}`), kind: kStructStop, pop: true},
	{in: in(structMode), match: re(`[\n]`), kind: skip},
	{in: bothModes, match: re(`note\s+`), kind: kNote, push: noteMode},
	{in: in(noteMode), match: re(`left of\b`), kind: kLeftOf, pop: true, push: noteIDMode},
	{in: in(noteMode), match: re(`right of\b`), kind: kRightOf, pop: true, push: noteIDMode},
	{in: in(noteMode), match: re(`"`), kind: skip, pop: true, push: floatingNoteMode},
	{in: in(floatingNoteMode), match: re(`\s*as\s*`), kind: kAs, pop: true, push: floatingNoteIDMode},
	{in: in(floatingNoteMode), match: re(`["]`), kind: skip},
	{in: in(floatingNoteMode), match: re(`[^"]*`), kind: kNoteText},
	{in: in(floatingNoteIDMode), match: re(`[^\n]*`), kind: kID, pop: true, id: true},
	{in: in(noteIDMode), match: re(`\s*[^:\n\s\-]+`), kind: kID, pop: true, push: noteTextMode, id: true},
	{in: in(noteTextMode), match: re(`\s*:[^:\n;]+`), kind: kNoteText, pop: true, text: afterTwo},
	{in: in(noteTextMode), match: re(`[\s\S]*?\n\s*end note\b`), kind: kNoteText, pop: true, text: without(8)},
	{in: in(initial), match: re(`stateDiagram\s+`), kind: kSD},
	{in: in(initial), match: re(`stateDiagram-v2\s+`), kind: kSD},
	{in: in(initial), match: re(`hide empty description\b`), kind: kHideEmpty},
	{in: bothModes, match: re(`\[\*\]`), kind: kEdgeState},
	{in: bothModes, match: re(`[^:\n\s\-\{]+`), kind: kID, id: true},
	{in: bothModes, match: re(`\s*:(?:[^:\n;]|:[^:\n;])+`), kind: kDescr, text: trim},
	{in: bothModes, match: re(`-->`), kind: kArrow},
	{in: in(structMode), match: re(`--`), kind: kConcurrent},
	{in: bothModes, match: re(`:::`), kind: kStyleSep},
	{in: in(initial), match: re(`$`), kind: syntax.Newline},
	{in: in(initial), match: re(`.`), kind: kInvalid},
}
