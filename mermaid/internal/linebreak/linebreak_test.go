package linebreak

import (
	"slices"
	"testing"
)

// Each want is where Chrome breaks the text, set in a block 1px wide with
// white-space: break-spaces.
func TestBreaks(t *testing.T) {
	for _, c := range []struct {
		name, text string
		want       []int
	}{
		{"after every space", "a b  c", []int{2, 4, 5}},
		{"after a tab", "a\tb", []int{2}},
		{"a hyphen before a letter", "well-known", []int{5}},
		{"a hyphen before a digit after a letter", "ABCD-1234", []int{5}},
		{"a hyphen before a digit after punctuation", "a.-1", nil},
		{"a hyphen before a closing", "a-)", nil},
		{"a question mark", "why?not", []int{4}},
		{"a question mark before a quote", "a?\"b", nil},
		{"punctuation before an opening", "a)(b", []int{2}},
		{"a comma before a bracket", "a,[b", []int{2}},
		{"a slash", "and/or", nil},
		{"a period", "a.b", nil},
		{"a percent before a brace", "5%{x}", []int{2}},
		{"a dollar", "a$5", nil},
		{"ideographs", "中文字符", []int{3, 6, 9}},
		{"ideographs and their punctuation", "中文，标点。好", []int{3, 9, 12, 18}},
		{"ideographs and letters", "中文abc中", []int{3, 6, 9}},
		{"small kana", "ぁあァア", []int{3, 6, 9}},
		{"a prolonged sound mark", "カー", []int{3}},
		{"corner brackets", "「引号」中", []int{6, 12}},
		{"East Asian quotes", "中“文”字", []int{3, 12}},
		{"a Latin quote", "a “b” c", []int{2, 10}},
		{"emoji", "😀😀", []int{4}},
		{"an emoji with a modifier", "👍🏽👍🏽", []int{8}},
		{"a ZWJ sequence", "👨\u200d👩\u200d👧x", []int{18}},
		{"flags", "🇨🇳🇺🇸", []int{8}},
		{"combining marks", "éé x", []int{7}},
		{"a zero width space", "a\u200bb", []int{4}},
		{"an ideographic space", "中\u3000文", []int{6}},
		{"an ideographic space after a letter", "a\u3000b", []int{4}},
		{"a no-break space", "a\u00a0b c", []int{5}},
		{"Latin-1 punctuation", "a¿b×c", nil},
		{"guillemets", "«a»b", nil},
		{"a hyphen before a Latin-1 letter", "a-é", []int{2}},
		{"a number", "a1,000.5b", nil},
		{"a number in parentheses", "$(1)x", nil},
		{"Hebrew", "שלום עולם", []int{9}},
		{"Hangul", "한국어 문장", []int{3, 6, 10, 13}},
		{"Thai words split by a space", "ไทย ไทย", []int{10}},
		{"no break before an ideographic space", "\ufffc\u3000a", []int{6}},
		{"a last ideographic space is left to UAX 14", "\ufffc\u3000", []int{3}},
		{"a break after an ideographic space before a space", "\u3000 ", []int{3}},
		{"no break before a last space", "a- ", nil},
		{"a pair up to U+00FF is decided alone", "(1)£", []int{3}},
		{"a break after a zero width space before punctuation", "\u200b;", []int{3}},
		{"no break before a comma", "é,", nil},
		{"no break between two em dashes", "——", nil},
		{"no break before a hyphen", "é-", nil},
		{"no break in a number and its currency", "a 1£", []int{2}},
		{"a dotted circle holds its virama", "◌꧀", nil},
		{"a break before a wide opening bracket", "a【", []int{1}},
		{"an unassigned pictograph holds its modifier", "a\U0001fc00🏻", []int{1}},
		{"an assigned pictograph does not", "a☀🏻", []int{1, 4}},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := Breaks(c.text); !slices.Equal(got, c.want) {
				t.Errorf("Breaks(%q) = %v, want %v", c.text, got, c.want)
			}
		})
	}
}
