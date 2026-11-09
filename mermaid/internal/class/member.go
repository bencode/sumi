package class

import (
	"regexp"
	"strings"

	"github.com/bencode/sumi/mermaid/internal/label"
	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// methodRE is ClassMember's method pattern, with \s and the dot as JavaScript reads them.
var methodRE = regexp.MustCompile(jsRegexp(`([#+~-])?(.+)\((.*)\)([\s$*])?(.*)([$*])?`))

// isTrimmed is whether JavaScript's trim takes r, as it does all of \s but U+0085.
func isTrimmed(r rune) bool { return r != '\u0085' && syntax.IsSpace(r) }

func trim(s string) string { return strings.TrimFunc(s, isTrimmed) }

func isVisibility(s string) bool { return s == "#" || s == "+" || s == "~" || s == "-" }

// newMember is ClassMember of a method or an attribute. A method that its pattern
// does not read leaves parseMember with no id, which throws.
func newMember(input string, method bool, at syntax.Pos) (Member, error) {
	var m Member
	if method {
		g := methodRE.FindStringSubmatch(input)
		if g == nil {
			return m, syntax.Errorf(at, "cannot read the method %q", input)
		}
		if isVisibility(g[1]) {
			m.Visibility = g[1]
		}
		m.ID, m.Parameters, m.ReturnType = g[2], trim(g[3]), trim(g[5])
		m.Classifier = trim(g[4])
		if last := lastByte(m.ReturnType); m.Classifier == "" && isClassifier(last) {
			m.Classifier, m.ReturnType = last, m.ReturnType[:len(m.ReturnType)-1]
		}
	} else {
		first, last := input[:min(1, len(input))], lastByte(input)
		if isVisibility(first) {
			m.Visibility = first
		}
		if isClassifier(last) {
			m.Classifier = last
		}
		m.ID = input[len(m.Visibility) : len(input)-len(m.Classifier)]
	}
	if strings.HasPrefix(m.ID, " ") {
		m.ID = " " + trim(m.ID)
	} else {
		m.ID = trim(m.ID)
	}
	m.Text = memberText(m, method)
	return m, nil
}

func lastByte(s string) string { return s[max(len(s)-1, 0):] }

func isClassifier(s string) bool { return s == "$" || s == "*" }

// memberText is what parseMember leaves in text, which is Markdown: a visibility has
// a backslash so that it is not read as Markdown, and < and > are written as HTML's.
func memberText(m Member, method bool) string {
	s := label.GenericTypes(m.ID)
	if m.Visibility != "" {
		s = `\` + m.Visibility + s
	}
	if method {
		s += "(" + label.GenericTypes(m.Parameters) + ")"
		if m.ReturnType != "" {
			s += " : " + label.GenericTypes(m.ReturnType)
		}
	}
	s = strings.NewReplacer("<", "&lt;", ">", "&gt;").Replace(s)
	if rest, ok := strings.CutPrefix(s, `\&lt;`); ok {
		return "~" + rest
	}
	return s
}
