// Package linebreak finds where Chrome may wrap a line of text set with
// white-space: break-spaces, as Mermaid sets a label once it wraps.
//
// Blink (NextBreakablePosition in text_break_iterator.cc, Chrome 156) breaks after
// every space, decides between two characters up to U+00FF by a table, and leaves
// the rest to ICU, which follows UAX #14 of the Unicode version Chrome carries, as
// LB1 resolves classes for CSS line-break: normal.
package linebreak

import (
	"strings"
	"unicode/utf8"
)

// Breaks returns the byte offsets in s where a line that starts at s may break, in
// order; neither 0 nor len(s) is one. Chrome starts the rules afresh at the start of
// each line, so text that wraps is asked again from where each line starts.
func Breaks(s string) []int {
	uax := uax14(s)
	var out []int
	var before, a rune
	for i, b := range s {
		if i > 0 && between(before, a, b, uax[i], i+utf8.RuneLen(b) == len(s)) {
			out = append(out, i)
		}
		before, a = a, b
	}
	return out
}

// between reports whether a line may break between a and b, before being the
// character before a, 0 at the start; uax is what UAX #14 says, and last whether b
// ends the text.
func between(before, a, b rune, uax, last bool) bool {
	switch {
	case space(a):
		return true
	case space(b) && !last: // Blink goes on past a space, to break after it
		return false
	case a < '!' || b < '!':
		return false
	case a == '-' && b > 0x7f:
		return uax
	case a <= 0x7f && b <= 0x7f:
		return asciiBreak(before, a, b)
	case a <= 0xff && b <= 0xff:
		return uax14(string(a) + string(b))[len(string(a))]
	}
	return uax
}

// space reports whether a line breaks after r and not before it: the space and the
// tab, as break-spaces has it, and the ideographic space, Blink's other space
// separator.
func space(r rune) bool {
	return r == ' ' || r == '\t' || r == '\u3000'
}

// asciiBreak is Blink's table for two ASCII characters from '!' on, FillAscii in
// character_property_data_generator.cc: after '-' and '?' before most characters,
// and after other punctuation before an opening. Between two other characters up to
// U+00FF, the table is what UAX #14 says of the pair alone.
func asciiBreak(before, a, b rune) bool {
	switch {
	case a == '-' && b >= '0' && b <= '9': // 'ABCD-1234' breaks, '.-1' does not
		return alphanumeric(before)
	case a == '-':
		return !strings.ContainsRune(`!$),./:;?]}`, b)
	case a == '?':
		return !strings.ContainsRune(`!"'),./:;?]}`, b)
	case strings.ContainsRune(`!"#%&)*+,.:;=>\]|}~`, a):
		return strings.ContainsRune(`(<[{`, b)
	}
	return false
}

func alphanumeric(r rune) bool {
	return r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
}
