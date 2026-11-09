package class

import (
	"bytes"
	"regexp"
	"strings"

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
	s := genericTypes(m.ID)
	if m.Visibility != "" {
		s = `\` + m.Visibility + s
	}
	if method {
		s += "(" + genericTypes(m.Parameters) + ")"
		if m.ReturnType != "" {
			s += " : " + genericTypes(m.ReturnType)
		}
	}
	s = strings.NewReplacer("<", "&lt;", ">", "&gt;").Replace(s)
	if rest, ok := strings.CutPrefix(s, `\&lt;`); ok {
		return "~" + rest
	}
	return s
}

// genericTypes is parseGenericTypes: the ~ of a generic become < and >, across the
// comma when each side has one.
func genericTypes(input string) string {
	sets := splitCommas(input)
	var out []string
	for i := 0; i < len(sets); i++ {
		set := sets[i]
		if set == "," && i > 0 && i+1 < len(sets) {
			if prev, next := sets[i-1], sets[i+1]; strings.Count(prev, "~") == 1 && strings.Count(next, "~") == 1 {
				set = prev + "," + next
				i++
				out = out[:len(out)-1]
			}
		}
		out = append(out, processSet(set))
	}
	return strings.Join(out, "")
}

// splitCommas is input.split(/(,)/): the parts between commas and the commas.
func splitCommas(input string) []string {
	var out []string
	for i, part := range strings.Split(input, ",") {
		if i > 0 {
			out = append(out, ",")
		}
		out = append(out, part)
	}
	return out
}

// processSet pairs a set's ~ from the outside in; an odd one at its start stays.
func processSet(input string) string {
	n := strings.Count(input, "~")
	if n <= 1 {
		return input
	}
	starting := n%2 != 0 && strings.HasPrefix(input, "~")
	if starting {
		input = input[1:]
	}
	b := []byte(input)
	for first, last := bytes.IndexByte(b, '~'), bytes.LastIndexByte(b, '~'); first != last; first, last = bytes.IndexByte(b, '~'), bytes.LastIndexByte(b, '~') {
		b[first], b[last] = '<', '>'
	}
	if starting {
		return "~" + string(b)
	}
	return string(b)
}
