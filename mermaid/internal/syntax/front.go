package syntax

import (
	"strings"
)

// Prepare readies a diagram's source for its lexer as Mermaid does
// (preprocessDiagram, preprocess.ts, and Diagram.fromText, Diagram.ts): it
// turns CRLF and CR into LF, takes off the front matter, drops %%{...}%%
// directives and the lines that are %% comments, sets entity codes aside
// (see Decode), and ends the source with a line break. A %% comment after
// something on its line is left for the lexer. It returns the source and
// the front matter's title, if any. Whatever it drops leaves its line
// breaks behind, so positions in the source it returns are those in src.
//
// Only the front matter's title is read; the rest of it, config included,
// is not supported yet and passed over, as are directives.
func Prepare(src string) (body, title string) {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = strings.ReplaceAll(src, "\r", "\n")
	src, title = frontMatter(src)
	return encodeEntities(stripComments(stripDirectives(src))) + "\n", title
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

// frontTitle is the value of a top-level title: key, on a line of its own,
// with its quotes taken off.
func frontTitle(lines []string, indent string) string {
	for _, line := range lines {
		v, ok := strings.CutPrefix(strings.TrimPrefix(line, indent), "title:")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
			v = v[1 : len(v)-1]
		}
		return v
	}
	return ""
}

// stripDirectives drops each %%{...}%%, which may span lines, keeping its
// line breaks.
func stripDirectives(src string) string {
	var b strings.Builder
	for {
		i := strings.Index(src, "%%{")
		if i < 0 {
			break
		}
		end := len(src)
		if j := strings.Index(src[i:], "}%%"); j >= 0 {
			end = i + j + len("}%%")
		}
		b.WriteString(src[:i])
		b.WriteString(strings.Repeat("\n", strings.Count(src[i:end], "\n")))
		src = src[end:]
	}
	b.WriteString(src)
	return b.String()
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
