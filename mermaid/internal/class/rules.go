package class

import (
	"regexp"
	"strings"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// skip is the kind of a rule that makes no token.
const skip syntax.Kind = -1

// The kinds of token a class diagram has, past those all diagrams share. NEWLINE is
// syntax.Newline, and kEnd is the grammar's EOF, a token of its own before syntax.EOF.
const (
	kClassDiagram syntax.Kind = syntax.FirstKind + iota
	kDirectionTB
	kDirectionBT
	kDirectionRL
	kDirectionLR
	kAccTitle
	kAccTitleValue
	kAccDescr
	kAccDescrValue
	kAccBlockValue
	kEdgeState
	kCallbackName
	kCallbackArgs
	kString
	kStyle
	kClassDef
	kNamespace
	kStructStart
	kStructStop
	kEndInStruct
	kClass
	kOpenInStruct
	kMember
	kCSSClass
	kCallback
	kLink
	kClick
	kNoteFor
	kNote
	kAnnotationStart
	kAnnotationEnd
	kHref
	kGeneric
	kBackquote
	kLinkTarget
	kExtension
	kDependency
	kComposition
	kAggregation
	kLollipop
	kLine
	kDottedLine
	kLabel
	kStyleSeparator
	kMinus
	kDot
	kPlus
	kColon
	kComma
	kBrkt
	kPct
	kEquals
	kAlpha
	kSquareOpen
	kSquareClose
	kPunctuation
	kUnicode
	kEnd
)

// A mode is a start condition of the lexer.
type mode uint8

const (
	initial mode = iota
	stringMode
	backquote
	generic
	callbackName
	callbackArgs
	accTitle
	accDescr
	accBlock
	namespaceHead
	namespaceBody
	classHead
	classBody
	modes // how many there are
)

// A rule is a lexer rule: in the modes of in, a match of it makes a token of kind, or
// none when kind is skip, then pops pops modes and pushes push, unless that is initial.
// less gives back what it matched, for the mode it leaves to read.
type rule struct {
	in    uint32
	match func(s string) int // the length matched at the start of s, or -1
	kind  syntax.Kind
	pops  int
	push  mode
	less  bool
}

func in(ms ...mode) uint32 {
	var out uint32
	for _, m := range ms {
		out |= 1 << m
	}
	return out
}

// jsRegexp writes p, a rule of the compiled lexer, for Go, with \s and the dot as
// JavaScript reads them.
func jsRegexp(p string) string {
	var b strings.Builder
	class := false
	for i := 0; i < len(p); i++ {
		c := p[i]
		switch {
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
		default:
			class = c == '[' || class && c != ']'
			b.WriteByte(c)
		}
	}
	return b.String()
}

func re(p string) func(string) int {
	r := regexp.MustCompile(`^(?:` + jsRegexp(p) + `)`)
	return func(s string) int {
		m := r.FindStringIndex(s)
		if m == nil {
			return -1
		}
		return m[1]
	}
}

// unicodeText is the rule for UNICODE_TEXT: letters, in the Basic Multilingual Plane.
const unicodeText = `[` +
	`\x{00aa}\x{00b5}\x{00ba}\x{00c0}-\x{00d6}\x{00d8}-\x{00f6}\x{00f8}-\x{02c1}\x{02c6}-\x{02d1}` +
	`\x{02e0}-\x{02e4}\x{02ec}\x{02ee}\x{0370}-\x{0374}\x{0376}\x{0377}\x{037a}-\x{037d}\x{0386}\x{0388}-` +
	`\x{038a}\x{038c}\x{038e}-\x{03a1}\x{03a3}-\x{03f5}\x{03f7}-\x{0481}\x{048a}-\x{0527}\x{0531}-` +
	`\x{0556}\x{0559}\x{0561}-\x{0587}\x{05d0}-\x{05ea}\x{05f0}-\x{05f2}\x{0620}-\x{064a}\x{066e}\x{066f}` +
	`\x{0671}-\x{06d3}\x{06d5}\x{06e5}\x{06e6}\x{06ee}\x{06ef}\x{06fa}-\x{06fc}\x{06ff}\x{0710}\x{0712}-` +
	`\x{072f}\x{074d}-\x{07a5}\x{07b1}\x{07ca}-\x{07ea}\x{07f4}\x{07f5}\x{07fa}\x{0800}-\x{0815}\x{081a}` +
	`\x{0824}\x{0828}\x{0840}-\x{0858}\x{08a0}\x{08a2}-\x{08ac}\x{0904}-\x{0939}\x{093d}\x{0950}\x{0958}-` +
	`\x{0961}\x{0971}-\x{0977}\x{0979}-\x{097f}\x{0985}-\x{098c}\x{098f}\x{0990}\x{0993}-\x{09a8}` +
	`\x{09aa}-\x{09b0}\x{09b2}\x{09b6}-\x{09b9}\x{09bd}\x{09ce}\x{09dc}\x{09dd}\x{09df}-\x{09e1}\x{09f0}` +
	`\x{09f1}\x{0a05}-\x{0a0a}\x{0a0f}\x{0a10}\x{0a13}-\x{0a28}\x{0a2a}-\x{0a30}\x{0a32}\x{0a33}\x{0a35}` +
	`\x{0a36}\x{0a38}\x{0a39}\x{0a59}-\x{0a5c}\x{0a5e}\x{0a72}-\x{0a74}\x{0a85}-\x{0a8d}\x{0a8f}-\x{0a91}` +
	`\x{0a93}-\x{0aa8}\x{0aaa}-\x{0ab0}\x{0ab2}\x{0ab3}\x{0ab5}-\x{0ab9}\x{0abd}\x{0ad0}\x{0ae0}\x{0ae1}` +
	`\x{0b05}-\x{0b0c}\x{0b0f}\x{0b10}\x{0b13}-\x{0b28}\x{0b2a}-\x{0b30}\x{0b32}\x{0b33}\x{0b35}-\x{0b39}` +
	`\x{0b3d}\x{0b5c}\x{0b5d}\x{0b5f}-\x{0b61}\x{0b71}\x{0b83}\x{0b85}-\x{0b8a}\x{0b8e}-\x{0b90}\x{0b92}-` +
	`\x{0b95}\x{0b99}\x{0b9a}\x{0b9c}\x{0b9e}\x{0b9f}\x{0ba3}\x{0ba4}\x{0ba8}-\x{0baa}\x{0bae}-\x{0bb9}` +
	`\x{0bd0}\x{0c05}-\x{0c0c}\x{0c0e}-\x{0c10}\x{0c12}-\x{0c28}\x{0c2a}-\x{0c33}\x{0c35}-\x{0c39}` +
	`\x{0c3d}\x{0c58}\x{0c59}\x{0c60}\x{0c61}\x{0c85}-\x{0c8c}\x{0c8e}-\x{0c90}\x{0c92}-\x{0ca8}\x{0caa}-` +
	`\x{0cb3}\x{0cb5}-\x{0cb9}\x{0cbd}\x{0cde}\x{0ce0}\x{0ce1}\x{0cf1}\x{0cf2}\x{0d05}-\x{0d0c}\x{0d0e}-` +
	`\x{0d10}\x{0d12}-\x{0d3a}\x{0d3d}\x{0d4e}\x{0d60}\x{0d61}\x{0d7a}-\x{0d7f}\x{0d85}-\x{0d96}\x{0d9a}-` +
	`\x{0db1}\x{0db3}-\x{0dbb}\x{0dbd}\x{0dc0}-\x{0dc6}\x{0e01}-\x{0e30}\x{0e32}\x{0e33}\x{0e40}-\x{0e46}` +
	`\x{0e81}\x{0e82}\x{0e84}\x{0e87}\x{0e88}\x{0e8a}\x{0e8d}\x{0e94}-\x{0e97}\x{0e99}-\x{0e9f}\x{0ea1}-` +
	`\x{0ea3}\x{0ea5}\x{0ea7}\x{0eaa}\x{0eab}\x{0ead}-\x{0eb0}\x{0eb2}\x{0eb3}\x{0ebd}\x{0ec0}-\x{0ec4}` +
	`\x{0ec6}\x{0edc}-\x{0edf}\x{0f00}\x{0f40}-\x{0f47}\x{0f49}-\x{0f6c}\x{0f88}-\x{0f8c}\x{1000}-` +
	`\x{102a}\x{103f}\x{1050}-\x{1055}\x{105a}-\x{105d}\x{1061}\x{1065}\x{1066}\x{106e}-\x{1070}\x{1075}-` +
	`\x{1081}\x{108e}\x{10a0}-\x{10c5}\x{10c7}\x{10cd}\x{10d0}-\x{10fa}\x{10fc}-\x{1248}\x{124a}-\x{124d}` +
	`\x{1250}-\x{1256}\x{1258}\x{125a}-\x{125d}\x{1260}-\x{1288}\x{128a}-\x{128d}\x{1290}-\x{12b0}` +
	`\x{12b2}-\x{12b5}\x{12b8}-\x{12be}\x{12c0}\x{12c2}-\x{12c5}\x{12c8}-\x{12d6}\x{12d8}-\x{1310}` +
	`\x{1312}-\x{1315}\x{1318}-\x{135a}\x{1380}-\x{138f}\x{13a0}-\x{13f4}\x{1401}-\x{166c}\x{166f}-` +
	`\x{167f}\x{1681}-\x{169a}\x{16a0}-\x{16ea}\x{1700}-\x{170c}\x{170e}-\x{1711}\x{1720}-\x{1731}` +
	`\x{1740}-\x{1751}\x{1760}-\x{176c}\x{176e}-\x{1770}\x{1780}-\x{17b3}\x{17d7}\x{17dc}\x{1820}-` +
	`\x{1877}\x{1880}-\x{18a8}\x{18aa}\x{18b0}-\x{18f5}\x{1900}-\x{191c}\x{1950}-\x{196d}\x{1970}-` +
	`\x{1974}\x{1980}-\x{19ab}\x{19c1}-\x{19c7}\x{1a00}-\x{1a16}\x{1a20}-\x{1a54}\x{1aa7}\x{1b05}-` +
	`\x{1b33}\x{1b45}-\x{1b4b}\x{1b83}-\x{1ba0}\x{1bae}\x{1baf}\x{1bba}-\x{1be5}\x{1c00}-\x{1c23}` +
	`\x{1c4d}-\x{1c4f}\x{1c5a}-\x{1c7d}\x{1ce9}-\x{1cec}\x{1cee}-\x{1cf1}\x{1cf5}\x{1cf6}\x{1d00}-` +
	`\x{1dbf}\x{1e00}-\x{1f15}\x{1f18}-\x{1f1d}\x{1f20}-\x{1f45}\x{1f48}-\x{1f4d}\x{1f50}-\x{1f57}` +
	`\x{1f59}\x{1f5b}\x{1f5d}\x{1f5f}-\x{1f7d}\x{1f80}-\x{1fb4}\x{1fb6}-\x{1fbc}\x{1fbe}\x{1fc2}-\x{1fc4}` +
	`\x{1fc6}-\x{1fcc}\x{1fd0}-\x{1fd3}\x{1fd6}-\x{1fdb}\x{1fe0}-\x{1fec}\x{1ff2}-\x{1ff4}\x{1ff6}-` +
	`\x{1ffc}\x{2071}\x{207f}\x{2090}-\x{209c}\x{2102}\x{2107}\x{210a}-\x{2113}\x{2115}\x{2119}-\x{211d}` +
	`\x{2124}\x{2126}\x{2128}\x{212a}-\x{212d}\x{212f}-\x{2139}\x{213c}-\x{213f}\x{2145}-\x{2149}\x{214e}` +
	`\x{2183}\x{2184}\x{2c00}-\x{2c2e}\x{2c30}-\x{2c5e}\x{2c60}-\x{2ce4}\x{2ceb}-\x{2cee}\x{2cf2}\x{2cf3}` +
	`\x{2d00}-\x{2d25}\x{2d27}\x{2d2d}\x{2d30}-\x{2d67}\x{2d6f}\x{2d80}-\x{2d96}\x{2da0}-\x{2da6}` +
	`\x{2da8}-\x{2dae}\x{2db0}-\x{2db6}\x{2db8}-\x{2dbe}\x{2dc0}-\x{2dc6}\x{2dc8}-\x{2dce}\x{2dd0}-` +
	`\x{2dd6}\x{2dd8}-\x{2dde}\x{2e2f}\x{3005}\x{3006}\x{3031}-\x{3035}\x{303b}\x{303c}\x{3041}-\x{3096}` +
	`\x{309d}-\x{309f}\x{30a1}-\x{30fa}\x{30fc}-\x{30ff}\x{3105}-\x{312d}\x{3131}-\x{318e}\x{31a0}-` +
	`\x{31ba}\x{31f0}-\x{31ff}\x{3400}-\x{4db5}\x{4e00}-\x{9fcc}\x{a000}-\x{a48c}\x{a4d0}-\x{a4fd}` +
	`\x{a500}-\x{a60c}\x{a610}-\x{a61f}\x{a62a}\x{a62b}\x{a640}-\x{a66e}\x{a67f}-\x{a697}\x{a6a0}-` +
	`\x{a6e5}\x{a717}-\x{a71f}\x{a722}-\x{a788}\x{a78b}-\x{a78e}\x{a790}-\x{a793}\x{a7a0}-\x{a7aa}` +
	`\x{a7f8}-\x{a801}\x{a803}-\x{a805}\x{a807}-\x{a80a}\x{a80c}-\x{a822}\x{a840}-\x{a873}\x{a882}-` +
	`\x{a8b3}\x{a8f2}-\x{a8f7}\x{a8fb}\x{a90a}-\x{a925}\x{a930}-\x{a946}\x{a960}-\x{a97c}\x{a984}-` +
	`\x{a9b2}\x{a9cf}\x{aa00}-\x{aa28}\x{aa40}-\x{aa42}\x{aa44}-\x{aa4b}\x{aa60}-\x{aa76}\x{aa7a}` +
	`\x{aa80}-\x{aaaf}\x{aab1}\x{aab5}\x{aab6}\x{aab9}-\x{aabd}\x{aac0}\x{aac2}\x{aadb}-\x{aadd}\x{aae0}-` +
	`\x{aaea}\x{aaf2}-\x{aaf4}\x{ab01}-\x{ab06}\x{ab09}-\x{ab0e}\x{ab11}-\x{ab16}\x{ab20}-\x{ab26}` +
	`\x{ab28}-\x{ab2e}\x{abc0}-\x{abe2}\x{ac00}-\x{d7a3}\x{d7b0}-\x{d7c6}\x{d7cb}-\x{d7fb}\x{f900}-` +
	`\x{fa6d}\x{fa70}-\x{fad9}\x{fb00}-\x{fb06}\x{fb13}-\x{fb17}\x{fb1d}\x{fb1f}-\x{fb28}\x{fb2a}-` +
	`\x{fb36}\x{fb38}-\x{fb3c}\x{fb3e}\x{fb40}\x{fb41}\x{fb43}\x{fb44}\x{fb46}-\x{fbb1}\x{fbd3}-\x{fd3d}` +
	`\x{fd50}-\x{fd8f}\x{fd92}-\x{fdc7}\x{fdf0}-\x{fdfb}\x{fe70}-\x{fe74}\x{fe76}-\x{fefc}\x{ff21}-` +
	`\x{ff3a}\x{ff41}-\x{ff5a}\x{ff66}-\x{ffbe}\x{ffc2}-\x{ffc7}\x{ffca}-\x{ffcf}\x{ffd2}-\x{ffd7}` +
	`\x{ffda}-\x{ffdc}` +
	`]`

// every is all the modes, as <*> reads.
const every = uint32(1)<<modes - 1

var (
	inInitial   = in(initial)
	inNamespace = in(initial, namespaceHead, namespaceBody)
	inClass     = in(initial, namespaceBody)
)

// rules are the rules Jison compiles from classDiagram.jison, in their order, which
// decides: a mode takes the first that matches, not the longest. Jison writes a \b after
// a quoted word, and a lookahead with a * on it asks nothing, so neither %% nor the
// values of accTitle and accDescr have one here. Left out are those no text can reach:
// the second %%, the second # and =, [0-9]+ (after \w+), and \s (after \s+, or in a mode
// that takes every character already).
var rules = []rule{
	{in: inInitial, match: re(`.*direction\s+TB[^\n]*`), kind: kDirectionTB},
	{in: inInitial, match: re(`.*direction\s+BT[^\n]*`), kind: kDirectionBT},
	{in: inInitial, match: re(`.*direction\s+RL[^\n]*`), kind: kDirectionRL},
	{in: inInitial, match: re(`.*direction\s+LR[^\n]*`), kind: kDirectionLR},
	{in: inInitial, match: re(`%%[^\n]*(\r?\n?)+`), kind: skip},
	{in: inInitial, match: re(`accTitle\s*:\s*`), kind: kAccTitle, push: accTitle},
	{in: in(accTitle), match: re(`[^\n]*`), kind: kAccTitleValue, pops: 1},
	{in: inInitial, match: re(`accDescr\s*:\s*`), kind: kAccDescr, push: accDescr},
	{in: in(accDescr), match: re(`[^\n]*`), kind: kAccDescrValue, pops: 1},
	{in: inInitial, match: re(`accDescr\s*\{\s*`), kind: skip, push: accBlock},
	{in: in(accBlock), match: re(`[\}]`), kind: skip, pops: 1},
	{in: in(accBlock), match: re(`[^\}]*`), kind: kAccBlockValue},
	{in: inInitial, match: re(`\s*(\r?\n)+`), kind: syntax.Newline},
	{in: inInitial, match: re(`\s+`), kind: skip},
	{in: inInitial, match: re(`classDiagram-v2\b`), kind: kClassDiagram},
	{in: inInitial, match: re(`classDiagram\b`), kind: kClassDiagram},
	{in: inInitial, match: re(`\[\*\]`), kind: kEdgeState},
	{in: inInitial, match: re(`call[\s]+`), kind: skip, push: callbackName},
	{in: in(callbackName), match: re(`\([\s]*\)`), kind: skip, pops: 1},
	{in: in(callbackName), match: re(`\(`), kind: skip, pops: 1, push: callbackArgs},
	{in: in(callbackName), match: re(`[^(]*`), kind: kCallbackName},
	{in: in(callbackArgs), match: re(`\)`), kind: skip, pops: 1},
	{in: in(callbackArgs), match: re(`[^)]*`), kind: kCallbackArgs},
	{in: in(stringMode), match: re(`["]`), kind: skip, pops: 1},
	{in: in(stringMode), match: re(`[^"]*`), kind: kString},
	{in: every, match: re(`["]`), kind: skip, push: stringMode},
	{in: inInitial, match: re(`style\b`), kind: kStyle},
	{in: inInitial, match: re(`classDef\b`), kind: kClassDef},
	{in: inNamespace, match: re(`namespace\b`), kind: kNamespace, push: namespaceHead},
	{in: in(namespaceHead), match: re(`\s*(\r?\n)+`), kind: syntax.Newline, pops: 1},
	{in: in(namespaceHead), match: re(`\s+`), kind: skip},
	{in: in(namespaceHead), match: re(`[{]`), kind: kStructStart, push: namespaceBody},
	{in: in(namespaceHead), match: re(`[}]`), kind: skip, pops: 1, less: true},
	{in: in(namespaceBody), match: re(`[}]`), kind: kStructStop, pops: 1},
	{in: in(namespaceBody), match: re(`$`), kind: kEndInStruct},
	{in: in(namespaceBody), match: re(`\s*(\r?\n)+`), kind: syntax.Newline},
	{in: in(namespaceBody), match: re(`\s+`), kind: skip},
	{in: in(namespaceBody), match: re(`\[\*\]`), kind: kEdgeState},
	{in: inClass, match: re(`class\b`), kind: kClass, push: classHead},
	{in: in(classHead), match: re(`\s*(\r?\n)+`), kind: syntax.Newline, pops: 1},
	{in: in(classHead), match: re(`\s+`), kind: skip},
	{in: in(classHead), match: re(`[}]`), kind: kStructStop, pops: 2},
	{in: in(classHead), match: re(`[{]`), kind: kStructStart, push: classBody},
	{in: in(classBody), match: re(`[}]`), kind: kStructStop, pops: 1},
	{in: in(classBody), match: re(`$`), kind: kEndInStruct},
	{in: in(classBody), match: re(`\[\*\]`), kind: kEdgeState},
	{in: in(classBody), match: re(`[{]`), kind: kOpenInStruct},
	{in: in(classBody), match: re(`[\n]`), kind: skip},
	{in: in(classBody), match: re(`[^{}\n]*`), kind: kMember},
	{in: every, match: re(`cssClass\b`), kind: kCSSClass},
	{in: every, match: re(`callback\b`), kind: kCallback},
	{in: every, match: re(`link\b`), kind: kLink},
	{in: every, match: re(`click\b`), kind: kClick},
	{in: every, match: re(`note for\b`), kind: kNoteFor},
	{in: every, match: re(`note\b`), kind: kNote},
	{in: every, match: re(`<<`), kind: kAnnotationStart},
	{in: every, match: re(`>>`), kind: kAnnotationEnd},
	{in: every, match: re(`href\b`), kind: kHref},
	{in: in(generic), match: re(`[~]`), kind: skip, pops: 1},
	{in: in(generic), match: re(`[^~]*`), kind: kGeneric},
	{in: every, match: re(`~`), kind: skip, push: generic},
	{in: in(backquote), match: re("[`]"), kind: skip, pops: 1},
	{in: in(backquote), match: re("[^`]+"), kind: kBackquote},
	{in: every, match: re("[`]"), kind: skip, push: backquote},
	{in: every, match: re(`_self\b`), kind: kLinkTarget},
	{in: every, match: re(`_blank\b`), kind: kLinkTarget},
	{in: every, match: re(`_parent\b`), kind: kLinkTarget},
	{in: every, match: re(`_top\b`), kind: kLinkTarget},
	{in: every, match: re(`\s*<\|`), kind: kExtension},
	{in: every, match: re(`\s*\|>`), kind: kExtension},
	{in: every, match: re(`\s*>`), kind: kDependency},
	{in: every, match: re(`\s*<`), kind: kDependency},
	{in: every, match: re(`\s*\*`), kind: kComposition},
	{in: every, match: re(`\s*o\b`), kind: kAggregation},
	{in: every, match: re(`\s*\(\)`), kind: kLollipop},
	{in: every, match: re(`--`), kind: kLine},
	{in: every, match: re(`\.\.`), kind: kDottedLine},
	{in: every, match: re(`:{1}[^:\n;]+`), kind: kLabel},
	{in: every, match: re(`:{3}`), kind: kStyleSeparator},
	{in: every, match: re(`-`), kind: kMinus},
	{in: every, match: re(`\.`), kind: kDot},
	{in: every, match: re(`\+`), kind: kPlus},
	{in: inInitial, match: re(`:`), kind: kColon},
	{in: inInitial, match: re(`,`), kind: kComma},
	{in: inInitial, match: re(`#`), kind: kBrkt},
	{in: every, match: re(`%`), kind: kPct},
	{in: every, match: re(`=`), kind: kEquals},
	{in: every, match: re(`\w+`), kind: kAlpha},
	{in: every, match: re(`\[`), kind: kSquareOpen},
	{in: every, match: re(`\]`), kind: kSquareClose},
	{in: every, match: re("[!\"#$%&'*+,-.`?\\\\/]"), kind: kPunctuation},
	{in: every, match: re(unicodeText), kind: kUnicode},
	{in: every, match: re(`$`), kind: kEnd},
}
