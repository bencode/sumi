package flowchart

import (
	"regexp"
	"strings"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// shapes are the names @{ shape } can give the shapes the grammar can write
// (shapes.ts). Mermaid knows more, which sumi does not draw yet.
var shapes = map[string]Shape{
	"rect": Rect, "proc": Rect, "process": Rect, "rectangle": Rect,
	"rounded": Round, "event": Round,
	"stadium": Stadium, "terminal": Stadium, "pill": Stadium,
	"fr-rect": Subroutine, "subprocess": Subroutine, "subproc": Subroutine, "framed-rectangle": Subroutine, "subroutine": Subroutine,
	"cyl": Cylinder, "db": Cylinder, "database": Cylinder, "cylinder": Cylinder,
	"circle": Circle, "circ": Circle,
	"dbl-circ": DoubleCircle, "double-circle": DoubleCircle, "doublecircle": DoubleCircle,
	"diam": Diamond, "decision": Diamond, "diamond": Diamond, "question": Diamond,
	"hex": Hexagon, "hexagon": Hexagon, "prepare": Hexagon,
	"odd":    Odd,
	"trap-b": Trapezoid, "priority": Trapezoid, "trapezoid-bottom": Trapezoid, "trapezoid": Trapezoid,
	"trap-t": InvTrapezoid, "manual": InvTrapezoid, "trapezoid-top": InvTrapezoid, "inv-trapezoid": InvTrapezoid,
	"lean-r": LeanRight, "lean-right": LeanRight, "in-out": LeanRight,
	"lean-l": LeanLeft, "lean-left": LeanLeft, "out-in": LeanLeft,
}

// A metadata is what an @{...} says, by key. The value of a key in typed is a string.
type metadata map[string]string

// typed are the keys whose values Resolve reads.
var typed = map[string]bool{"shape": true, "label": true, "labelType": true}

// nonPrintable is what js-yaml refuses anywhere in a stream.
var nonPrintable = regexp.MustCompile(`[\x00-\x08\x0b\x0c\x0e-\x1f\x7f-\x84\x86-\x9f\x{fffe}\x{ffff}]`)

// readMetadata reads the YAML of an @{...} (addVertex): a flow mapping if it is on one
// line, and a block mapping if not. It takes the part of YAML people write there, keys
// and scalars, and leaves the rest as not supported.
func readMetadata(d *Data) (metadata, error) {
	if nonPrintable.MatchString(d.Text) {
		return nil, syntax.Errorf(d.Pos, "a character YAML does not take in @{...}")
	}
	read := flowEntries
	if strings.Contains(d.Text, "\n") {
		read = blockEntries
	}
	entries, ok := read(d.Text)
	if !ok {
		return nil, syntax.Unsupported(d.Pos, "this YAML in @{...}")
	}
	m := metadata{}
	for _, e := range entries {
		if _, dup := m[e.key]; dup {
			return nil, syntax.Errorf(d.Pos, "%q is given twice", e.key)
		}
		if typed[e.key] && !e.value.isString {
			return nil, syntax.Unsupported(d.Pos, e.key+" that is not a string")
		}
		m[e.key] = e.value.text
	}
	return m, nil
}

// An entry is a key and its value.
type entry struct {
	key   string
	value scalar
}

// A scalar is a YAML value.
type scalar struct {
	text     string
	isString bool
}

// yamlBlanks are what separates tokens in YAML.
const yamlBlanks = " \t"

func strip(s string) string { return strings.Trim(s, yamlBlanks) }

// flowEntries reads the entries of a flow mapping, without its braces.
func flowEntries(s string) ([]entry, bool) {
	var entries []entry
	for s = strip(s); s != ""; {
		e, rest, ok := readEntry(s, true)
		if !ok {
			return nil, false
		}
		entries = append(entries, e)
		s = strip(rest)
		if s == "" {
			break
		}
		if s[0] != ',' {
			return nil, false
		}
		s = strip(s[1:])
	}
	return entries, true
}

// blockEntries reads a block mapping: an entry on each line, all with the same indent.
// One with none is not a mapping: YAML reads it as null, and addVertex fails on that.
func blockEntries(s string) ([]entry, bool) {
	var entries []entry
	indent := -1
	for _, line := range strings.Split(s, "\n") {
		if strip(line) == "" {
			continue
		}
		body := strings.TrimLeft(line, " ")
		if indent >= 0 && len(line)-len(body) != indent {
			return nil, false
		}
		indent = len(line) - len(body)
		e, rest, ok := readEntry(body, false)
		if !ok || strip(rest) != "" {
			return nil, false
		}
		entries = append(entries, e)
	}
	return entries, len(entries) > 0
}

// readEntry reads key: value from the start of s. In a flow mapping a word alone is a key
// with no value, and so is shape:circle, as a colon with no blank after it is part of
// the word.
func readEntry(s string, flow bool) (entry, string, bool) {
	if flow {
		end := plainEnd(s, flow)
		if word := strip(s[:end]); word != "" && plain(word, flow) {
			return entry{key: word}, s[end:], true
		}
	}
	key, rest, ok := readKey(s, flow)
	if !ok {
		return entry{}, s, false
	}
	v, rest, ok := readScalar(strip(rest), flow)
	return entry{key, v}, rest, ok
}

// readKey reads a key and its colon, which has a blank after it or ends s. After a quoted
// key in a flow mapping it needs neither.
func readKey(s string, flow bool) (string, string, bool) {
	if s != "" && (s[0] == '"' || s[0] == '\'') {
		v, rest, ok := readScalar(s, flow)
		if !ok || !strings.HasPrefix(rest, ":") || !flow && !endsKey(rest[1:]) {
			return "", s, false
		}
		return v.text, rest[1:], true
	}
	i := strings.Index(s, ":")
	if i <= 0 || !plain(s[:i], flow) || i > plainEnd(s, flow) || !endsKey(s[i+1:]) {
		return "", s, false
	}
	return strip(s[:i]), s[i+1:], true
}

// endsKey reports whether what follows a colon lets it end a key.
func endsKey(rest string) bool {
	return rest == "" || strings.ContainsRune(yamlBlanks, rune(rest[0]))
}

// notString is what YAML's JSON schema reads as a number, a boolean or null, and what is
// close to one: a string that only looks like a number is not worth telling apart here.
var notString = regexp.MustCompile(`^(?:|~|null|Null|NULL|true|True|TRUE|false|False|FALSE|[-+]?\.?[0-9][0-9a-zA-Z_.+-]*|[-+]?\.(?:inf|Inf|INF)|\.(?:nan|NaN|NAN))$`)

// plainEnd is where a plain scalar at the start of s ends: at a comma in a flow mapping.
func plainEnd(s string, flow bool) int {
	if i := strings.IndexByte(s, ','); flow && i >= 0 {
		return i
	}
	return len(s)
}

// readScalar reads a scalar from the start of s.
func readScalar(s string, flow bool) (scalar, string, bool) {
	if strings.HasPrefix(s, `"`) {
		end := strings.Index(s[1:], `"`)
		if end < 0 || strings.Contains(s[1:1+end], `\`) {
			return scalar{}, s, false
		}
		return scalar{s[1 : 1+end], true}, s[end+2:], true
	}
	if strings.HasPrefix(s, "'") {
		return readQuoted(s)
	}
	end := plainEnd(s, flow)
	text := strip(s[:end])
	if !plain(text, flow) {
		return scalar{}, s, false
	}
	return scalar{text, !notString.MatchString(text)}, s[end:], true
}

// readQuoted reads a single quoted string, in which two quotes in a row are one.
func readQuoted(s string) (scalar, string, bool) {
	var b strings.Builder
	for i := 1; i < len(s); i++ {
		if s[i] != '\'' {
			b.WriteByte(s[i])
			continue
		}
		if i+1 == len(s) || s[i+1] != '\'' {
			return scalar{b.String(), true}, s[i+1:], true
		}
		b.WriteByte('\'')
		i++
	}
	return scalar{}, s, false
}

// indicator is a colon that ends a key, or a # that starts a comment.
var indicator = regexp.MustCompile(`:[ \t]|:$|[ \t]#`)

// plain reports whether s is a plain scalar with nothing in it that YAML reads as more
// than text. A flow mapping takes fewer characters.
func plain(s string, flow bool) bool {
	if s != "" && strings.ContainsRune("&*!|>'\"%@`#?[]{}-,:\t", rune(s[0])) {
		return false
	}
	if indicator.MatchString(s) {
		return false
	}
	return !flow || !strings.ContainsAny(s, "[]{}")
}
