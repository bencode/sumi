package syntax

import (
	"html"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Lines turns byte offsets in a source into positions.
type Lines struct {
	src    string
	starts []int // the offset each line starts at
}

// LinesOf indexes the lines of src.
func LinesOf(src string) Lines {
	starts := []int{0}
	for i := range len(src) {
		if src[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	return Lines{src, starts}
}

// Pos is the position of offset i.
func (l Lines) Pos(i int) Pos {
	n := sort.Search(len(l.starts), func(k int) bool { return l.starts[k] > i }) - 1
	return Pos{Line: n + 1, Col: utf8.RuneCountInString(l.src[l.starts[n]:i]) + 1}
}

// IsSpace is whether r is what \s matches in a JavaScript regular expression, and
// what JavaScript's trim takes off: Go's spaces but U+0085, and U+FEFF.
func IsSpace(r rune) bool { return r != '\u0085' && (unicode.IsSpace(r) || r == '\uFEFF') }

// IsWord is whether b is what \w matches.
func IsWord(b byte) bool {
	return b == '_' || '0' <= b && b <= '9' || 'a' <= b && b <= 'z' || 'A' <= b && b <= 'Z'
}

// Keyword is the end of word at src[i:], matched without regard to case, or
// -1. As in Jison, a word that ends in a word character must end the word in
// src too: "end" matches "end:" but not "endpoint".
func Keyword(src string, i int, word string) int {
	end := i + len(word)
	if end > len(src) || !strings.EqualFold(src[i:end], word) {
		return -1
	}
	if IsWord(word[len(word)-1]) && end < len(src) && IsWord(src[end]) {
		return -1
	}
	return end
}

// SkipSpace is the end of the run of spaces at src[i:], line breaks included.
func SkipSpace(src string, i int) int {
	for i < len(src) {
		r, n := utf8.DecodeRuneInString(src[i:])
		if !IsSpace(r) {
			break
		}
		i += n
	}
	return i
}

// JSSpaces is what JavaScript's \s takes, for a character class in a Go pattern, JSBlanks
// the same without the line feed, and JSDot is its dot.
const (
	JSBlanks = `\r\t\x{0b}\f \x{a0}\x{1680}\x{2000}-\x{200a}\x{2028}\x{2029}\x{202f}\x{205f}\x{3000}\x{feff}`
	JSSpaces = `\n` + JSBlanks
	JSDot    = `[^\n\r\x{2028}\x{2029}]`
)

// Mermaid sets each entity code, #9829; or #quot;, aside before it parses a
// diagram, so its # and ; are not read as a comment or the end of a
// statement, and decodes it in the text it ends up in. These two characters,
// from the private use area, take the place of the # and the ;, so the code
// keeps its length in characters and positions after it stay right.
const (
	entityOpen  = '\uE000'
	entityClose = '\uE001'
)

// encodeEntities sets aside each #\w+; in src.
func encodeEntities(src string) string {
	var b strings.Builder
	for {
		i := strings.IndexByte(src, '#')
		if i < 0 {
			break
		}
		j := i + 1
		for j < len(src) && IsWord(src[j]) {
			j++
		}
		if j == i+1 || j == len(src) || src[j] != ';' {
			b.WriteString(src[:i+1])
			src = src[i+1:]
			continue
		}
		b.WriteString(src[:i])
		b.WriteRune(entityOpen)
		b.WriteString(src[i+1 : j])
		b.WriteRune(entityClose)
		src = src[j+1:]
	}
	b.WriteString(src)
	return b.String()
}

// Decode turns the entity codes set aside in s into the characters they
// stand for: #9829; is ♥ and #quot; is ", as HTML names them. A name HTML
// does not know stays as written, as &name;.
func Decode(s string) string { return eachCode(s, entity) }

// Placeholders writes the entity codes set aside in s as Mermaid's
// encodeEntities does, which leaves them so until the drawing is done:
// #9829; is ﬂ°°9829¶ß and #quot; is ﬂ°quot¶ß. A renderer that measures its
// text while drawing measures these.
func Placeholders(s string) string {
	return eachCode(s, func(code string) string {
		if strings.Trim(code, "0123456789") == "" {
			return "ﬂ°°" + code + "¶ß"
		}
		return "ﬂ°" + code + "¶ß"
	})
}

// eachCode is s with each entity code set aside in it written as f writes its
// code, the word between # and ;.
func eachCode(s string, f func(code string) string) string {
	var b strings.Builder
	for {
		i := strings.IndexRune(s, entityOpen)
		if i < 0 {
			break
		}
		j := strings.IndexRune(s[i:], entityClose)
		if j < 0 {
			break
		}
		code := s[i+len(string(entityOpen)) : i+j]
		b.WriteString(s[:i])
		b.WriteString(f(code))
		s = s[i+j+len(string(entityClose)):]
	}
	b.WriteString(s)
	return b.String()
}

func entity(code string) string {
	if strings.Trim(code, "0123456789") != "" {
		return html.UnescapeString("&" + code + ";")
	}
	if n, err := strconv.Atoi(code); err != nil || n > unicode.MaxRune {
		return string(utf8.RuneError)
	}
	return html.UnescapeString("&#" + code + ";")
}
