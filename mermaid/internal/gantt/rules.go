package gantt

import (
	"regexp"
	"strings"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// skip is the kind of a rule that makes no token.
const skip syntax.Kind = -1

// The kinds of token a diagram has, past those all diagrams share. NL is syntax.Newline,
// and kDone is the grammar's EOF, a token of its own before syntax.EOF.
const (
	kGantt syntax.Kind = syntax.FirstKind + iota
	kDirective
	kAccTitle
	kAccTitleValue
	kAccDescr
	kAccDescrValue
	kAccMultiValue
	kHref
	kCallbackName
	kCallbackArgs
	kClick
	kDateFormat
	kInclusiveEndDates
	kTopAxis
	kAxisFormat
	kTickInterval
	kIncludes
	kExcludes
	kTodayMarker
	kWeekday
	kWeekend
	kDate
	kTitle
	kAccDescription
	kSection
	kTaskTxt
	kTaskData
	kColon
	kInvalid
	kDone
)

// A mode is a start condition of the lexer. All but initial are exclusive.
type mode uint8

const (
	initial mode = iota
	click
	href
	callbackName
	callbackArgs
	accTitle
	accDescr
	accMulti
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

// re is a rule's pattern.
func re(p string) func(string) int {
	r := regexp.MustCompile(`^(?:` + p + `)`)
	return func(s string) int {
		m := r.FindStringIndex(s)
		if m == nil {
			return -1
		}
		return m[1]
	}
}

// atEnd matches only where the source has run out.
func atEnd(s string) int {
	if s == "" {
		return 0
	}
	return -1
}

// kw is a keyword, its letters in either case, as the /i of a regular expression without
// /u reads them: unlike Go's (?i), it leaves ſ and K, the Kelvin sign, alone.
func kw(word string) string {
	var b strings.Builder
	for _, c := range word {
		b.WriteString("[" + strings.ToLower(string(c)) + strings.ToUpper(string(c)) + "]")
	}
	return b.String()
}

// JavaScript's \s and its dot.
const (
	sp  = `[` + syntax.JSSpaces + `]`
	dot = syntax.JSDot
)

// rules are the lexer's, in the order of the array Jison compiles from gantt.jison. That
// array differs from the source: a literal ending in a letter gets a \b, the (?!\n|;|#)
// of the accessible title and description loses its ; and #, which leaves a lookahead
// that never holds, repeated none or more times, so nothing, and so does the (?!\{)* of
// a comment. One rule of the array is left out, as it never matches: %[^\n]*, which an
// earlier rule, a % and anything after it, has taken first.
var rules = []rule{
	{in: initial, match: re(`%%\{`), kind: kDirective},
	{in: initial, match: re(kw("accTitle") + sp + `*:` + sp + `*`), kind: kAccTitle, push: accTitle},
	{in: accTitle, match: re(`[^\n]*`), kind: kAccTitleValue, pop: true},
	{in: initial, match: re(kw("accDescr") + sp + `*:` + sp + `*`), kind: kAccDescr, push: accDescr},
	{in: accDescr, match: re(`[^\n]*`), kind: kAccDescrValue, pop: true},
	{in: initial, match: re(kw("accDescr") + sp + `*\{` + sp + `*`), kind: skip, push: accMulti},
	{in: accMulti, match: re(`\}`), kind: skip, pop: true},
	{in: accMulti, match: re(`[^}]*`), kind: kAccMultiValue},
	{in: initial, match: re(`%%[^\n]*`), kind: skip},
	{in: initial, match: re(`[^}]%%*[^\n]*`), kind: skip},
	{in: initial, match: re(`%%*[^\n]*\n*`), kind: skip},
	{in: initial, match: re(`\n+`), kind: syntax.Newline},
	{in: initial, match: re(sp + `+`), kind: skip},
	{in: initial, match: re(kw("href") + sp + `+"`), kind: skip, push: href},
	{in: href, match: re(`"`), kind: skip, pop: true},
	{in: href, match: re(`[^"]*`), kind: kHref},
	{in: initial, match: re(kw("call") + sp + `+`), kind: skip, push: callbackName},
	{in: callbackName, match: re(`\(` + sp + `*\)`), kind: skip, pop: true},
	{in: callbackName, match: re(`\(`), kind: skip, pop: true, push: callbackArgs},
	{in: callbackName, match: re(`[^(]*`), kind: kCallbackName},
	{in: callbackArgs, match: re(`\)`), kind: skip, pop: true},
	{in: callbackArgs, match: re(`[^)]*`), kind: kCallbackArgs},
	{in: initial, match: re(kw("click") + sp + `+`), kind: skip, push: click},
	{in: click, match: re(sp), kind: skip, pop: true},
	{in: click, match: re(`[^` + syntax.JSSpaces + `]*`), kind: kClick},
	{in: initial, match: re(kw("gantt") + `\b`), kind: kGantt},
	{in: initial, match: re(kw("dateFormat") + sp + `[^#\n;]+`), kind: kDateFormat},
	{in: initial, match: re(kw("inclusiveEndDates") + `\b`), kind: kInclusiveEndDates},
	{in: initial, match: re(kw("topAxis") + `\b`), kind: kTopAxis},
	{in: initial, match: re(kw("axisFormat") + sp + `[^#\n;]+`), kind: kAxisFormat},
	{in: initial, match: re(kw("tickInterval") + sp + `[^#\n;]+`), kind: kTickInterval},
	{in: initial, match: re(kw("includes") + sp + `[^#\n;]+`), kind: kIncludes},
	{in: initial, match: re(kw("excludes") + sp + `[^#\n;]+`), kind: kExcludes},
	{in: initial, match: re(kw("todayMarker") + sp + `[^\n;]+`), kind: kTodayMarker},
	{in: initial, match: re(kw("weekday") + sp + `+(?:` + kw("monday") + `|` + kw("tuesday") + `|` + kw("wednesday") + `|` + kw("thursday") + `|` + kw("friday") + `|` + kw("saturday") + `|` + kw("sunday") + `)\b`), kind: kWeekday},
	{in: initial, match: re(kw("weekend") + sp + `+(?:` + kw("friday") + `|` + kw("saturday") + `)\b`), kind: kWeekend},
	{in: initial, match: re(`\d\d\d\d-\d\d-\d\d\b`), kind: kDate},
	{in: initial, match: re(kw("title") + sp + `[^\n]+`), kind: kTitle},
	{in: initial, match: re(kw("accDescription") + sp + `[^#\n;]+`), kind: kAccDescription},
	{in: initial, match: re(kw("section") + sp + `[^\n]+`), kind: kSection},
	{in: initial, match: re(`[^:\n]+`), kind: kTaskTxt},
	{in: initial, match: re(`:[^#\n;]+`), kind: kTaskData},
	{in: initial, match: re(`:`), kind: kColon},
	{in: initial, match: atEnd, kind: kDone},
	{in: initial, match: re(dot), kind: kInvalid},
}
