package syntax

import (
	"fmt"
	"math"
	"math/big"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

// scalar is a YAML scalar written on one line, as Mermaid shows the title it
// is (extractFrontMatter, frontmatter.ts): js-yaml reads it with JSON_SCHEMA,
// and title.toString() shows it, a false value showing none. It is false for a
// value js-yaml does not read, such as a quote that is not closed.
func scalar(v string) (string, bool) {
	v = strings.TrimLeft(v, " \t")
	if v == "" || v[0] != '\'' && v[0] != '"' {
		if i := commentAt(v); i >= 0 {
			v = v[:i]
		}
		return plain(strings.TrimRight(v, " \t")), true
	}
	s, rest, ok := quoted(v)
	if after := strings.TrimLeft(rest, " \t"); !ok || after != "" && after[0] != '#' {
		return "", false
	}
	return s, true
}

// commentAt is where a comment starts in the plain scalar v: a # first, or
// after a space or a tab.
func commentAt(v string) int {
	for i := range len(v) {
		if v[i] == '#' && (i == 0 || v[i-1] == ' ' || v[i-1] == '\t') {
			return i
		}
	}
	return -1
}

// quoted reads the quoted scalar v opens with: in single quotes, two quotes
// are one; in double quotes, a backslash starts an escape. It returns the text
// and what follows the closing quote, where a comment may start at once.
func quoted(v string) (string, string, bool) {
	q := v[0]
	var b strings.Builder
	for i := 1; i < len(v); i++ {
		c := v[i]
		if c == q && q == '\'' && i+1 < len(v) && v[i+1] == '\'' {
			b.WriteByte('\'')
			i++
		} else if c == q {
			return b.String(), v[i+1:], true
		} else if c == '\\' && q == '"' {
			s, n, ok := escape(v[i+1:])
			if !ok {
				return "", "", false
			}
			b.WriteString(s)
			i += n
		} else {
			b.WriteByte(c)
		}
	}
	return "", "", false
}

// escapes are YAML's escapes of one character in a double-quoted scalar.
var escapes = map[byte]string{
	'0': "\x00", 'a': "\a", 'b': "\b", 't': "\t", '\t': "\t", 'n': "\n", 'v': "\v", 'f': "\f",
	'r': "\r", 'e': "\x1b", ' ': " ", '"': `"`, '/': "/", '\\': `\`,
	'N': "\u0085", '_': " ", 'L': " ", 'P': " ",
}

// hexDigits are the digits of YAML's escapes of a code point.
var hexDigits = map[byte]int{'x': 2, 'u': 4, 'U': 8}

// escape reads the escape at the start of s, after its backslash, returning
// its text and its length.
func escape(s string) (string, int, bool) {
	if s == "" {
		return "", 0, false
	}
	if e, ok := escapes[s[0]]; ok {
		return e, 1, true
	}
	n, ok := hexDigits[s[0]]
	if !ok || len(s) <= n {
		return "", 0, false
	}
	r, err := strconv.ParseUint(s[1:1+n], 16, 32)
	if err != nil {
		return "", 0, false
	}
	return string(utf8.AppendRune(nil, rune(r))), 1 + n, true
}

var (
	nulls  = []string{"", "~", "null", "Null", "NULL"}
	falses = []string{"false", "False", "FALSE"}
	trues  = []string{"true", "True", "TRUE"}
	// The patterns of js-yaml's int and float, in its JSON_SCHEMA.
	yamlInt   = regexp.MustCompile(`^[-+]?(?:0b[01]+|0x[0-9a-fA-F]+|0o[0-7]+|[0-9]+)$`)
	yamlFloat = regexp.MustCompile(`^(?:[-+]?[0-9]+(?:\.[0-9]*)?(?:[eE][-+]?[0-9]+)?|\.[0-9]+(?:[eE][-+]?[0-9]+)?|[-+]?\.(?:inf|Inf|INF)|\.(?:nan|NaN|NAN))$`)
)

// plain is the title of a plain scalar: none for null, false and a number
// that is 0 or NaN, true for true, a number as JavaScript prints it, and the
// text otherwise.
func plain(v string) string {
	if slices.Contains(nulls, v) || slices.Contains(falses, v) {
		return ""
	}
	if slices.Contains(trues, v) {
		return "true"
	}
	f, ok := yamlNumber(v)
	if !ok {
		return v
	}
	if f == 0 || math.IsNaN(f) {
		return ""
	}
	return numberText(f)
}

// yamlNumber is the number a plain scalar is, if it is one: a whole number in
// binary, hexadecimal, octal or decimal, or a float, .inf or .nan. A number
// too large to hold is a string.
func yamlNumber(v string) (float64, bool) {
	if yamlInt.MatchString(v) {
		digits, sign := strings.TrimLeft(v, "+-"), 1.0
		if v[0] == '-' {
			sign = -1
		}
		base := 10
		if len(digits) > 2 && digits[0] == '0' && strings.ContainsRune("bxo", rune(digits[1])) {
			base = map[byte]int{'b': 2, 'x': 16, 'o': 8}[digits[1]]
			digits = digits[2:]
		}
		n, _ := new(big.Int).SetString(digits, base)
		f, _ := new(big.Float).SetInt(n).Float64()
		return sign * f, !math.IsInf(f, 0)
	}
	if !yamlFloat.MatchString(v) {
		return 0, false
	}
	special := strings.TrimLeft(strings.ToLower(v), "+-")
	if special == ".inf" || special == ".nan" {
		f := math.Inf(1)
		if special == ".nan" {
			f = math.NaN()
		}
		if v[0] == '-' {
			f = -f
		}
		return f, true
	}
	f, err := strconv.ParseFloat(v, 64)
	return f, err == nil
}

// numberText is Number::toString of ECMAScript: the fewest digits that read
// back as f, in fixed notation from 1e-6 up to 1e21, and as 1.5e-7 or 1e+21
// beyond.
func numberText(f float64) string {
	if math.IsInf(f, 1) {
		return "Infinity"
	}
	if math.IsInf(f, -1) {
		return "-Infinity"
	}
	mantissa, exp, _ := strings.Cut(strconv.FormatFloat(f, 'e', -1, 64), "e")
	e, _ := strconv.Atoi(exp)
	if -7 < e && e < 21 {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	return fmt.Sprintf("%se%+d", mantissa, e)
}
