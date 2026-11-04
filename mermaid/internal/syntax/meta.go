package syntax

import (
	"strings"
	"unicode/utf8"
)

// Meta reads, at src[i:], a statement most diagrams accept: a title or an
// accessible title or description. It returns the token's kind and text, its
// value, and its end, or an end of -1. The rules are those of Mermaid's
// grammars (sequenceDiagram.jison lines 71-78):
//
//	title <text>             to a #, a ; or the end of the line
//	title: <text>            the same, in the older form
//	accTitle: <text>         to the end of the line
//	accDescr: <text>         to the end of the line
//	accDescr { <text> }      across lines, to the }
func Meta(src string, i int) (Kind, string, int) {
	for _, prefix := range []string{"title", "title:"} {
		if !prefixFold(src, i, prefix) {
			continue
		}
		if t, end := title(src, i+len(prefix)); end >= 0 {
			return Title, t, end
		}
	}
	if j, ok := afterFold(src, i, "accTitle"); ok {
		if k, ok := colon(src, j); ok {
			v, end := toLineEnd(src, k)
			return AccTitle, v, end
		}
	}
	if j, ok := afterFold(src, i, "accDescr"); ok {
		if k, ok := colon(src, j); ok {
			v, end := toLineEnd(src, k)
			return AccDescr, v, end
		}
		if k := SkipSpace(src, j); k < len(src) && src[k] == '{' {
			body, _, closed := strings.Cut(src[k+1:], "}")
			end := k + 1 + len(body)
			if closed {
				end++
			}
			return AccDescr, strings.TrimSpace(body), end
		}
	}
	return 0, "", -1
}

// title reads " <text>" at src[i:]: one space, then text up to a #, a ; or a
// line break, at least a character of it.
func title(src string, i int) (string, int) {
	r, n := utf8.DecodeRuneInString(src[i:])
	if n == 0 || !IsSpace(r) {
		return "", -1
	}
	start := i + n
	end := start + strings.IndexAny(src[start:]+"\n", "#;\n")
	if end == start {
		return "", -1
	}
	return strings.TrimSpace(src[start:end]), end
}

func prefixFold(src string, i int, s string) bool {
	return i+len(s) <= len(src) && strings.EqualFold(src[i:i+len(s)], s)
}

func afterFold(src string, i int, s string) (int, bool) {
	return i + len(s), prefixFold(src, i, s)
}

// colon reads \s*:\s* at src[i:].
func colon(src string, i int) (int, bool) {
	j := SkipSpace(src, i)
	if j >= len(src) || src[j] != ':' {
		return i, false
	}
	return SkipSpace(src, j+1), true
}

func toLineEnd(src string, i int) (string, int) {
	line, _, _ := strings.Cut(src[i:], "\n")
	return strings.TrimSpace(line), i + len(line)
}
