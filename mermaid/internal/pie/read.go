package pie

import (
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// A Diagram is what pieDb holds once a diagram is read.
type Diagram struct {
	ShowData bool
	// Title, AccTitle and AccDescr are the last of each the diagram writes; populateCommonDb
	// sets none that is empty, so an empty Title leaves the front matter's.
	Title, AccTitle, AccDescr string
	Sections                  []Section // in the order written, each label once
}

// A Section is a slice of the pie as addSection keeps it.
type Section struct {
	At    syntax.Pos
	Label string  // with the entity codes syntax.Prepare set aside
	Value float64 // as Number reads it: -0 stays, and too many digits are +Inf
}

// Read reads a diagram's body, prepared by syntax.Prepare, as pie.langium's parser and
// pieDb do:
//
//	Pie     := NEWLINE* "pie" "showData"? (Meta EOL | Section | NEWLINE)* EOF
//	Section := STRING ":" NUMBER EOL
//	EOL     := NEWLINE+ | EOF
//
// where a Meta is a title, an accTitle or an accDescr.
func Read(body string) (*Diagram, error) {
	ts, err := lex(body)
	if err != nil {
		return nil, err
	}
	for ok := true; ok; _, ts, ok = syntax.Accept(ts, syntax.Newline) {
	}
	if _, ts, err = syntax.Expect(ts, kPie, `"pie"`); err != nil {
		return nil, err
	}
	d := &Diagram{}
	_, ts, d.ShowData = syntax.Accept(ts, kShowData)
	metas := map[syntax.Kind]syntax.Token{}
	var sections []Section
	for ts[0].Kind != syntax.EOF {
		switch t := ts[0]; t.Kind {
		case syntax.Newline:
			ts = ts[1:]
		case syntax.Title, syntax.AccTitle, syntax.AccDescr:
			// Chevrotain takes TitleAndAccessibilities only when a line end or the end
			// follows, which it looks two tokens ahead to see.
			if k := ts[1].Kind; k != syntax.Newline && k != syntax.EOF {
				return nil, syntax.Errorf(t.Pos, "%s must end its line", syntax.Describe(t))
			}
			metas[t.Kind] = t
			if ts, err = eol(ts[1:]); err != nil {
				return nil, err
			}
		case kString:
			var s Section
			if s, ts, err = section(ts); err != nil {
				return nil, err
			}
			sections = append(sections, s)
		default:
			return nil, syntax.Unexpected(t, "a section, a title or the end")
		}
	}
	// pieDb's addSection, once the whole diagram is parsed.
	seen := map[string]bool{}
	for _, s := range sections {
		if s.Value < 0 {
			return nil, syntax.Errorf(s.At, "%q has invalid value: %s. Negative values are not allowed in pie charts",
				s.Label, strconv.FormatFloat(s.Value, 'g', -1, 64))
		}
		if !seen[s.Label] {
			seen[s.Label] = true
			d.Sections = append(d.Sections, s)
		}
	}
	for _, m := range []struct {
		kind  syntax.Kind
		field *string
	}{{syntax.Title, &d.Title}, {syntax.AccTitle, &d.AccTitle}, {syntax.AccDescr, &d.AccDescr}} {
		t, ok := metas[m.kind]
		if !ok {
			continue
		}
		// sanitizeText leaves a text without a < as it is.
		if *m.field = metaValue(t); strings.Contains(*m.field, "<") {
			return nil, syntax.Unsupported(t.Pos, "a < in a title or a description")
		}
	}
	return d, nil
}

// section reads STRING ":" NUMBER EOL.
func section(ts []syntax.Token) (Section, []syntax.Token, error) {
	label := ts[0]
	_, ts, err := syntax.Expect(ts[1:], kColon, `":"`)
	if err != nil {
		return Section{}, nil, err
	}
	num, ts, err := syntax.Expect(ts, kNumber, "a number")
	if err != nil {
		return Section{}, nil, err
	}
	v, err := strconv.ParseFloat(num.Text, 64)
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return Section{}, nil, err
	}
	ts, err = eol(ts)
	return Section{At: label.Pos, Label: unquote(label.Text), Value: v}, ts, err
}

// eol reads NEWLINE+ | EOF, leaving the EOF.
func eol(ts []syntax.Token) ([]syntax.Token, error) {
	if ts[0].Kind == syntax.EOF {
		return ts, nil
	}
	if _, _, ok := syntax.Accept(ts, syntax.Newline); !ok {
		return nil, syntax.Unexpected(ts[0], "a line break")
	}
	for ok := true; ok; _, ts, ok = syntax.Accept(ts, syntax.Newline) {
	}
	return ts, nil
}

// unquote is a STRING's value, as Langium's convertString gives it: the quotes off, and
// each backslash off the character after it, which some escapes turn into another.
func unquote(s string) string {
	var b strings.Builder
	for i := 1; i < len(s)-1; i++ {
		if s[i] != '\\' {
			b.WriteByte(s[i])
			continue
		}
		i++
		if c := strings.IndexByte("bfnrtv0", s[i]); c >= 0 {
			b.WriteByte("\b\f\n\r\t\v\x00"[c])
		} else {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// metaValue is a title's, an accTitle's or an accDescr's value, as runCommonConverter
// (valueConverter.ts) takes it with the patterns of matcher.ts: what follows the keyword,
// or its colon, trimmed and with its runs of spaces and tabs made one; or the lines of an
// accDescr between braces.
func metaValue(t syntax.Token) string {
	s := strings.TrimLeft(t.Text, " \t")
	switch t.Kind {
	case syntax.Title:
		s = strings.TrimPrefix(s, "title")
	case syntax.AccTitle:
		s = strings.TrimLeft(strings.TrimPrefix(s, "accTitle"), " \t")[1:]
	case syntax.AccDescr:
		s = strings.TrimLeft(strings.TrimPrefix(s, "accDescr"), " \t")
		if !strings.HasPrefix(s, ":") {
			s = s[strings.IndexByte(s, '{')+1 : len(s)-1]
			return oneSpace(trimLineEnds(trimLineStarts(s)))
		}
		s = s[1:]
	}
	return oneSpace(strings.TrimFunc(s, syntax.IsSpace))
}

// trimLineStarts is s.replace(/^\s*/gm, ""): each run of \s that starts a line goes, line
// ends and the lines they end with it. No two line ends are left side by side, so the
// .replace(/[\n\r]{2,}/gm, '\n') runCommonConverter ends with, and commonDb's
// .replace(/\n\s+/g, '\n') after it, find nothing.
func trimLineStarts(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if n := spaces(s[i:]); n > 0 && (i == 0 || endsLine(s[:i])) {
			i += n
			continue
		}
		_, n := utf8.DecodeRuneInString(s[i:])
		b.WriteString(s[i : i+n])
		i += n
	}
	return b.String()
}

// trimLineEnds is s.replace(/\s+$/gm, ""): from each run of \s, as much as ends at a line
// end or at the end of s goes, the line end itself staying.
func trimLineEnds(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		run := s[i : i+spaces(s[i:])]
		if i+len(run) == len(s) {
			break
		}
		if k := lastLineEnd(run); k > 0 {
			i += k
			continue
		}
		b.WriteString(run)
		i += len(run)
		_, n := utf8.DecodeRuneInString(s[i:])
		b.WriteString(s[i : i+n])
		i += n
	}
	return b.String()
}

// oneSpace is s.replace(/[\t ]{2,}/gm, ' ').
func oneSpace(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if n := blanks(s[i:]); n >= 2 {
			b.WriteByte(' ')
			i += n
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// endsLine is whether s ends with a line end, after which ^ matches in a multiline pattern.
func endsLine(s string) bool {
	r, _ := utf8.DecodeLastRuneInString(s)
	return isLineEnd(r)
}

// lastLineEnd is the offset of the last line end in a run of \s past its first character,
// before which $ matches, or -1.
func lastLineEnd(run string) int {
	k := -1
	for i, r := range run {
		if i > 0 && isLineEnd(r) {
			k = i
		}
	}
	return k
}
