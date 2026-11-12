package gantt

import (
	"strings"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// The tokens of V8's date scanner. It reads a string the same way whatever its parser is
// doing, so the tokens are read up front. A character past U+FFFF is two units to V8 and
// one rune here; it is only ever part of a word, which then matches no keyword either
// way, and a word's length counts only for keywords.

type tokenKind uint8

const (
	tEnd tokenKind = iota
	tNumber
	tSymbol
	tWord
	tSpace
	tUnknown
)

// wordKind is what a keyword names, wNone for a word that is none.
type wordKind uint8

const (
	wNone wordKind = iota
	wMonth
	wZone
	wTimeSep
	wAmPm
)

// A dateToken's value is a number's value, a symbol's character, or a keyword's: a month
// from 1, an offset in hours, or the hours am or pm add. Its length is a number's digits,
// leading zeros too, or a word's characters.
type dateToken struct {
	kind   tokenKind
	value  int
	length int
	word   wordKind
}

func (t dateToken) isNumber(length int) bool { return t.kind == tNumber && t.length == length }
func (t dateToken) isSymbol(c rune) bool     { return t.kind == tSymbol && t.value == int(c) }
func (t dateToken) isSign() bool             { return t.isSymbol('+') || t.isSymbol('-') }
func (t dateToken) isZ() bool                { return t.word == wZone && t.length == 1 && t.value == 0 }

type keyword struct {
	word  wordKind
	value int
}

// keywords are V8's, by the first three letters of a word, lower case. A month's name may
// go on past them; another word must be the keyword itself.
var keywords = map[string]keyword{
	"jan": {wMonth, 1}, "feb": {wMonth, 2}, "mar": {wMonth, 3}, "apr": {wMonth, 4},
	"may": {wMonth, 5}, "jun": {wMonth, 6}, "jul": {wMonth, 7}, "aug": {wMonth, 8},
	"sep": {wMonth, 9}, "oct": {wMonth, 10}, "nov": {wMonth, 11}, "dec": {wMonth, 12},
	"am": {wAmPm, 0}, "pm": {wAmPm, 12},
	"ut": {wZone, 0}, "utc": {wZone, 0}, "z": {wZone, 0}, "gmt": {wZone, 0},
	"cdt": {wZone, -5}, "cst": {wZone, -6}, "edt": {wZone, -4}, "est": {wZone, -5},
	"mdt": {wZone, -6}, "mst": {wZone, -7}, "pdt": {wZone, -7}, "pst": {wZone, -8},
	"t": {wTimeSep, 0},
}

// lookup is the keyword of a word. V8 lowers a character by setting its 0x20 bit, which
// makes a letter only of A to Z.
func lookup(word []rune) keyword {
	prefix := []rune{}
	for _, r := range word[:min(len(word), 3)] {
		if 'A' <= r && r <= 'Z' {
			r += 'a' - 'A'
		}
		prefix = append(prefix, r)
	}
	if k, ok := keywords[string(prefix)]; ok && (len(word) <= 3 || k.word == wMonth) {
		return k
	}
	return keyword{}
}

// dateTokens are the tokens of s and the end: numbers, of up to nine digits past the
// leading zeros; the symbols :-+.); words, the runs of characters from A up that are not
// spaces; a space, one character each; and anything else, a run in parentheses being
// one. V8 reads a NUL as the end.
func dateTokens(s string) []dateToken {
	s, _, _ = strings.Cut(s, "\x00")
	rs := []rune(s)
	var ts []dateToken
	for i := 0; i < len(rs); {
		j, c := i+1, rs[i]
		switch {
		case '0' <= c && c <= '9':
			n, digits := 0, 0
			for j = i; j < len(rs) && '0' <= rs[j] && rs[j] <= '9'; j++ {
				if digits > 0 || rs[j] != '0' {
					if digits < 9 {
						n = n*10 + int(rs[j]-'0')
					}
					digits++
				}
			}
			ts = append(ts, dateToken{kind: tNumber, value: n, length: j - i})
		case strings.ContainsRune(":-+.)", c):
			ts = append(ts, dateToken{kind: tSymbol, value: int(c), length: 1})
		case c >= 'A' && !isBlank(c):
			for j < len(rs) && rs[j] >= 'A' && !isBlank(rs[j]) {
				j++
			}
			k := lookup(rs[i:j])
			ts = append(ts, dateToken{kind: tWord, value: k.value, length: j - i, word: k.word})
		case syntax.IsSpace(c):
			ts = append(ts, dateToken{kind: tSpace, length: 1})
		case c == '(':
			// To the parenthesis that closes it, or the end.
			for depth := 1; j < len(rs) && depth > 0; j++ {
				if rs[j] == '(' {
					depth++
				} else if rs[j] == ')' {
					depth--
				}
			}
			ts = append(ts, dateToken{kind: tUnknown})
		default:
			ts = append(ts, dateToken{kind: tUnknown})
		}
		i = j
	}
	return append(ts, dateToken{kind: tEnd})
}

// isBlank is V8's IsWhiteSpace: JavaScript's \s but the ends of lines, which are spaces
// only between words.
func isBlank(r rune) bool { return syntax.IsSpace(r) && !strings.ContainsRune("\n\r\u2028\u2029", r) }
