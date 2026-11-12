package syntax

import (
	"regexp"
	"strings"
)

// Prepare readies a diagram's source for its lexer as Mermaid does
// (preprocessDiagram, preprocess.ts, and Diagram.fromText, Diagram.ts): it
// turns CRLF and CR into LF, makes the quotes of HTML attributes single, takes
// off the front matter, drops %%{...}%% directives and the lines that are %%
// comments, drops the ; after a color, sets entity codes aside (see Decode),
// and ends the source with a line break. A %% comment after something on its
// line is left for the lexer. It returns the source and the front matter's
// title, if any. Whatever it drops leaves its line breaks behind, so lines in
// the source it returns are those in src.
//
// Only the front matter's title is read; the rest of it, config included,
// is not supported yet and passed over, as are directives.
func Prepare(src string) (body, title string) {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = strings.ReplaceAll(src, "\r", "\n")
	src = htmlTag.ReplaceAllStringFunc(src, func(tag string) string {
		return htmlAttr.ReplaceAllString(tag, "='${1}'")
	})
	src, title = frontMatter(src)
	return encodeEntities(dropColorEnds(stripComments(stripDirectives(src)))) + "\n", title
}

// An attribute of an HTML tag has its double quotes made single, which a parser would
// take for the end of a string (cleanupText, preprocess.ts).
var (
	htmlTag  = regexp.MustCompile(`<\w+[^>]*>`)
	htmlAttr = regexp.MustCompile(`="([^"]*)"`)
)

// A statement that sets a color loses the ; that ends it, so that the color is
// not read as an entity code. The patterns are encodeEntities' in utils.ts,
// with . and \S as JavaScript reads them.
var colorStatements = []*regexp.Regexp{
	regexp.MustCompile(`style` + jsDot + `*:` + jsNonSpace + `*#` + jsDot + `*;`),
	regexp.MustCompile(`classDef` + jsDot + `*:` + jsNonSpace + `*#` + jsDot + `*;`),
}

const (
	jsDot      = `[^\n\r\x{2028}\x{2029}]`
	jsSpaces   = `\t\n\x{0b}\f\r \x{a0}\x{1680}\x{2000}-\x{200a}\x{2028}\x{2029}\x{202f}\x{205f}\x{3000}\x{feff}`
	jsSpace    = `[` + jsSpaces + `]`
	jsNonSpace = `[^` + jsSpaces + `]`
)

// dropColorEnds takes the last ; off each statement that sets a color, so
// columns after it on its line are one short.
func dropColorEnds(src string) string {
	for _, re := range colorStatements {
		src = re.ReplaceAllStringFunc(src, func(s string) string { return s[:len(s)-1] })
	}
	return src
}

// frontMatter takes off the YAML block between two --- lines that src may
// open with, as frontMatterRegex (diagram-api/regexes.ts line 10) matches it:
// the closing --- has just the opening one's indent, and a line break after
// it.
func frontMatter(src string) (string, string) {
	lines := strings.SplitAfter(src, "\n")
	indent, ok := opening(lines[0])
	if !ok {
		return src, ""
	}
	for j := 1; j < len(lines); j++ {
		if closing(lines[j], indent) && strings.HasSuffix(lines[j], "\n") {
			rest := strings.Join(lines[j+1:], "")
			return strings.Repeat("\n", j+1) + rest, frontTitle(lines[1:j], indent)
		}
	}
	return src, ""
}

// opening is whether line opens front matter: spaces and tabs, ---, then
// nothing but spaces. It returns the indent, which the closing line must
// have.
func opening(line string) (string, bool) {
	indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	return indent, closing(line, indent)
}

// closing is whether line is indent, then ---, then nothing but spaces.
func closing(line, indent string) bool {
	rest, ok := strings.CutPrefix(line, indent+"---")
	return ok && strings.TrimSpace(rest) == ""
}

// frontTitle is the title a top-level title key gives, its value on its line
// read as a YAML scalar. As in YAML, the colon after the key is followed by a
// space, a tab or the line's end: title:F is no key but a string. A value
// js-yaml does not read, which fails in Mermaid, is the title as written.
func frontTitle(lines []string, indent string) string {
	for _, line := range lines {
		v, ok := strings.CutPrefix(strings.TrimPrefix(line, indent), "title:")
		if !ok || v != "" && !strings.ContainsAny(v[:1], " \t\n") {
			continue
		}
		v = strings.TrimSuffix(v, "\n")
		if s, ok := scalar(v); ok {
			return s
		}
		return strings.Trim(v, " \t")
	}
	return ""
}

// stripDirectives drops each directive, which may span lines. The text on either
// side of one joins into a line, and the directive's line breaks go after that
// line, so that later lines keep their numbers. A %%{ that is no directive stays
// for the lexer.
func stripDirectives(src string) string {
	var b strings.Builder
	for {
		i := strings.Index(src, "%%{")
		if i < 0 {
			break
		}
		n := directiveLen(src[i:])
		if n < 0 {
			b.WriteString(src[:i+len("%%{")])
			src = src[i+len("%%{"):]
			continue
		}
		b.WriteString(src[:i])
		src = withBreaks(src[i+n:], strings.Count(src[i:i+n], "\n"))
	}
	b.WriteString(src)
	return b.String()
}

// withBreaks is s with n more line breaks at the end of its first line.
func withBreaks(s string, n int) string {
	end := strings.IndexByte(s, '\n')
	if end < 0 {
		end = len(s)
	}
	return s[:end] + strings.Repeat("\n", n) + s[end:]
}

// directiveHead and directiveTail are directiveRegex (utils.ts) around its last
// group, whose lookahead Go has no way to write. The group is a word or else
// directiveArgs, so that a word after the first ends a directive early.
var (
	directiveHead = regexp.MustCompile(`^%%\{` + jsSpace + `*(?:\w+` + jsSpace + `*:|\w+)` + jsSpace + `*(\w+)?`)
	directiveTail = regexp.MustCompile(`^` + jsSpace + `*(?:\}%%)?`)
)

// directiveLen is the length of the directive s starts with, or -1.
func directiveLen(s string) int {
	m := directiveHead.FindStringSubmatchIndex(s)
	if m == nil {
		return -1
	}
	n := m[1]
	if m[2] < 0 {
		n += directiveArgs(s[n:])
	}
	return n + len(directiveTail.FindString(s[n:]))
}

// directiveArgs is the length of s up to }%% or a line separator, which is where
// (?:(?!}%%).|\r?\n)* stops, there being no \r left by now.
func directiveArgs(s string) int {
	n := len(s)
	if i := strings.Index(s, "}%%"); i >= 0 {
		n = i
	}
	if i := strings.IndexAny(s[:n], "\u2028\u2029"); i >= 0 {
		n = i
	}
	return n
}

// stripComments drops each line that is a %% comment, as Mermaid's
// cleanupComments (diagram-api/comments.ts) does: spaces, %%, and at least
// one more character, but not %%{. The line break stays.
func stripComments(src string) string {
	lines := strings.Split(src, "\n")
	for i, l := range lines {
		c, ok := strings.CutPrefix(strings.TrimLeftFunc(l, IsSpace), "%%")
		if ok && c != "" && !strings.HasPrefix(c, "{") {
			lines[i] = ""
		}
	}
	return strings.Join(lines, "\n")
}
