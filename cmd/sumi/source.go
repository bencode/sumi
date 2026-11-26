package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/bencode/sumi/mermaid"
	"github.com/bencode/sumi/scene"
)

// A diagram is one Mermaid source in a file: a fenced mermaid block of a
// Markdown file, or the whole of any other file.
type diagram struct {
	src     string
	first   int      // the file's line of the source's first line, from 1
	end     int      // the file's line it ends at: its closing fence, or its last line
	lines   []string // the file's lines the source was taken from, as written
	indents []int    // how many leading spaces each of those lines lost
}

// diagrams are the diagrams of the file name, whose text is text.
func diagrams(name string, text string) []diagram {
	lines := strings.Split(strings.NewReplacer("\r\n", "\n", "\r", "\n").Replace(text), "\n")
	ext := strings.ToLower(filepath.Ext(name))
	if ext != ".md" && ext != ".markdown" {
		return []diagram{block(lines, 1, len(lines), 0)}
	}
	var ds []diagram
	for i := 0; i < len(lines); i++ {
		indent, fence, info, ok := openFence(lines[i])
		if !ok {
			continue
		}
		end := i + 1
		for end < len(lines) && !closesFence(lines[end], fence) {
			end++
		}
		if word, _, _ := strings.Cut(strings.TrimSpace(info), " "); word == "mermaid" {
			ds = append(ds, block(lines[i+1:end], i+2, min(end+1, len(lines)), indent))
		}
		i = end
	}
	return ds
}

// block is the diagram of lines, from the file's line first to its line end,
// each line losing up to indent leading spaces as CommonMark strips them from
// a fence's content.
func block(lines []string, first, end, indent int) diagram {
	d := diagram{first: first, end: end, lines: lines, indents: make([]int, len(lines))}
	body := make([]string, len(lines))
	for i, l := range lines {
		n := len(l) - len(strings.TrimLeft(l, " "))
		d.indents[i] = min(n, indent)
		body[i] = l[d.indents[i]:]
	}
	d.src = strings.Join(body, "\n")
	return d
}

// openFence reads line as the opening of a fenced code block: its indent, its
// fence of backticks or tildes, and its info string.
func openFence(line string) (indent int, fence, info string, ok bool) {
	rest := strings.TrimLeft(line, " ")
	indent = len(line) - len(rest)
	if indent > 3 || rest == "" || (rest[0] != '`' && rest[0] != '~') {
		return 0, "", "", false
	}
	n := len(rest) - len(strings.TrimLeft(rest, rest[:1]))
	fence, info = rest[:n], rest[n:]
	if n < 3 || (fence[0] == '`' && strings.Contains(info, "`")) {
		return 0, "", "", false
	}
	return indent, fence, info, true
}

// closesFence is whether line closes a block opened with fence.
func closesFence(line, fence string) bool {
	rest := strings.TrimLeft(line, " ")
	if len(line)-len(rest) > 3 {
		return false
	}
	run := strings.TrimLeft(rest, fence[:1])
	return len(rest)-len(run) >= len(fence) && strings.TrimSpace(run) == ""
}

// A result is a diagram laid out, or the error that stopped it.
type result struct {
	diagram
	scene *scene.Scene
	err   error
}

// check lays out each diagram of the file name.
func check(name, text string) []result {
	var rs []result
	for _, d := range diagrams(name, text) {
		s, err := mermaid.Layout(d.src, mermaid.Options{})
		if err == nil && (s.Width <= 0 || s.Height <= 0) {
			err = &mermaid.Error{Line: 1, Col: 1, Msg: "diagram draws nothing"}
		}
		rs = append(rs, result{diagram: d, scene: s, err: err})
	}
	return rs
}

// report says where in the file name the error of r is and what it is, and
// under that the line it is on with a caret at its column.
func (r result) report(name string) string {
	var e *mermaid.Error
	if !errors.As(r.err, &e) {
		return fmt.Sprintf("%s:%d: %v\n", name, r.first, r.err)
	}
	i := e.Line - 1
	if i < 0 || i >= len(r.lines) {
		return fmt.Sprintf("%s:%d:1: %s\n", name, min(r.first+max(i, 0), r.end), e.Msg)
	}
	line := r.lines[i]
	col := e.Col + r.indents[i]
	return fmt.Sprintf("%s:%d:%d: %s\n    %s\n    %s^\n", name, r.first+i, col, e.Msg, line, padTo(line, col))
}

// padTo is the blank that lines up under line up to its column col, counted in
// runes: tabs stay tabs, and wide characters take two columns, as a terminal shows them.
func padTo(line string, col int) string {
	var b strings.Builder
	for _, c := range []rune(line)[:max(0, min(col-1, len([]rune(line))))] {
		switch {
		case c == '\t':
			b.WriteByte('\t')
		case isWide(c):
			b.WriteString("  ")
		default:
			b.WriteByte(' ')
		}
	}
	return b.String()
}

func isWide(c rune) bool {
	return unicode.In(c, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul) ||
		(c >= 0x3000 && c <= 0x303f) || (c >= 0xff00 && c <= 0xff60)
}
