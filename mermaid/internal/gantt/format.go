package gantt

import (
	"errors"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// The dates of a diagram are read and written with the formats of dayjs 1.11.21, with
// the customParseFormat, advancedFormat and isoWeek plugins Mermaid loads and the
// English locale, as this file does.

var (
	months   = [...]string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
	weekdays = [...]string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
)

// ordinal is the English ordinal of n, as the locale writes it: 1st, 2nd, 11th.
func ordinal(n int) string {
	suffix := "th"
	if v := n % 100; v%10 >= 1 && v%10 <= 3 && (v < 11 || v > 13) {
		suffix = [...]string{"st", "nd", "rd"}[v%10-1]
	}
	return strconv.Itoa(n) + suffix
}

// The plugins Mermaid leaves out are those that write a week of the year, a week year
// and the name of a time zone: a format with one of their tokens fails as it does in
// Mermaid, as a call of a function the date does not have.
var missing = map[string]string{"w": "week", "ww": "week", "wo": "week", "gggg": "weekYear", "z": "offsetName", "zzz": "offsetName"}

var (
	advancedTokens = regexp.MustCompile(`\[[^\]]+]|Q|wo|ww|w|WW|W|zzz|z|gggg|GGGG|Do|X|x|k{1,2}|S`)
	formatTokens   = regexp.MustCompile(`\[([^\]]+)]|YYYY|YY|M{1,4}|D{1,2}|d{1,4}|H{1,2}|h{1,2}|a|A|m{1,2}|s{1,2}|Z{1,2}|SSS`)
)

// format writes t, a time in the diagram's zone, as dayjs's format(layout) does: the
// tokens of advancedFormat first, then those of dayjs itself, over what the first left.
// An empty layout is dayjs's default.
func format(t time.Time, layout string) (string, error) {
	if layout == "" {
		layout = "YYYY-MM-DDTHH:mm:ssZ"
	}
	var err error
	layout = advancedTokens.ReplaceAllStringFunc(layout, func(tok string) string {
		if name, ok := missing[tok]; ok {
			err = errors.New("this." + name + " is not a function")
		}
		return advanced(t, tok)
	})
	if err != nil {
		return "", err
	}
	return formatTokens.ReplaceAllStringFunc(layout, func(tok string) string {
		if tok[0] == '[' {
			return tok[1 : len(tok)-1]
		}
		return basic(t, tok)
	}), nil
}

func advanced(t time.Time, tok string) string {
	year, week := t.ISOWeek()
	switch tok {
	case "Q":
		return strconv.Itoa((int(t.Month()) + 2) / 3)
	case "Do":
		return "[" + ordinal(t.Day()) + "]"
	case "GGGG":
		return strconv.Itoa(year)
	case "W", "WW":
		return pad(strconv.Itoa(week), len(tok))
	case "k", "kk":
		h := t.Hour()
		if h == 0 {
			h = 24
		}
		return pad(strconv.Itoa(h), len(tok))
	case "X":
		return strconv.FormatInt(int64(math.Floor(float64(t.UnixMilli())/1000)), 10)
	case "x":
		return strconv.FormatInt(t.UnixMilli(), 10)
	}
	return tok
}

func basic(t time.Time, tok string) string {
	h, w := t.Hour(), t.Weekday()
	switch tok {
	case "YY":
		y := strconv.Itoa(t.Year())
		return y[max(len(y)-2, 0):]
	case "YYYY":
		return pad(strconv.Itoa(t.Year()), 4)
	case "M", "MM":
		return pad(strconv.Itoa(int(t.Month())), len(tok))
	case "MMM":
		return months[t.Month()-1][:3]
	case "MMMM":
		return months[t.Month()-1]
	case "D", "DD":
		return pad(strconv.Itoa(t.Day()), len(tok))
	case "d":
		return strconv.Itoa(int(w))
	case "dd", "ddd":
		return weekdays[w][:len(tok)]
	case "dddd":
		return weekdays[w]
	case "H", "HH":
		return pad(strconv.Itoa(h), len(tok))
	case "h", "hh":
		return pad(strconv.Itoa((h+11)%12+1), len(tok))
	case "a", "A":
		m := "AM"
		if h >= 12 {
			m = "PM"
		}
		if tok == "a" {
			return strings.ToLower(m)
		}
		return m
	case "m", "mm":
		return pad(strconv.Itoa(t.Minute()), len(tok))
	case "s", "ss":
		return pad(strconv.Itoa(t.Second()), len(tok))
	case "SSS":
		return pad(strconv.Itoa(t.Nanosecond()/1e6), 3)
	case "Z":
		return zoneOffset(t)
	}
	return strings.Replace(zoneOffset(t), ":", "", 1) // ZZ
}

// pad is padStart, as dayjs's utilities write it: s with zeros before it, up to n
// characters, a sign included.
func pad(s string, n int) string {
	return strings.Repeat("0", max(n-len(s), 0)) + s
}

// zoneOffset is the offset of t's zone, as +08:00, rounded to a quarter of an hour as
// dayjs rounds getTimezoneOffset, the minutes the zone is behind UTC.
func zoneOffset(t time.Time) string {
	_, off := t.Zone()
	behind := int(jsRound(-float64(off)/60/15)) * 15
	sign := "+"
	if behind > 0 {
		sign = "-"
	}
	behind = max(behind, -behind)
	return sign + pad(strconv.Itoa(behind/60), 2) + ":" + pad(strconv.Itoa(behind%60), 2)
}

// parseFormat reads s in the layout, as dayjs(s, layout, true) does in strict mode:
// what customParseFormat takes from s, written back with the layout, must be s again.
// Its false is an invalid date; its error, one Mermaid fails with.
func parseFormat(s, layout string, env Env) (time.Time, bool, error) {
	if layout == "x" || layout == "X" {
		return parseTimestamp(s, layout, env)
	}
	t, ok := readFormat(s, layout, env)
	if !ok {
		return time.Time{}, false, nil
	}
	out, err := format(t, layout)
	return t, err == nil && out == s, err
}

var canonicalInt = regexp.MustCompile(`^(?:0|-?[1-9]\d*)$`)

// parseTimestamp reads the milliseconds, or seconds, of the x and X formats. Only an
// integer written as JavaScript writes one comes back the same, so only one is valid.
func parseTimestamp(s, layout string, env Env) (time.Time, bool, error) {
	n, err := strconv.ParseInt(s, 10, 64)
	if !canonicalInt.MatchString(s) || err != nil {
		return time.Time{}, false, nil
	}
	ms := float64(n)
	if layout == "X" {
		ms *= 1000
	}
	t, ok := fromMillis(ms, env.Zone)
	return t, ok, nil
}

var (
	parseTokens = regexp.MustCompile(`(\[[^\[]*\])|([-_:/.,()` + syntax.JSSpaces + `]+)|(A|a|Q|YYYY|YY?|ww?|MM?M?M?|Do|DD?|hh?|HH?|mm?|ss?|S{1,3}|z|ZZ?)`)
	// localized is a token of localizedFormat, which the English locale has no formats
	// for, so reading one fails.
	localized = regexp.MustCompile(`(\[[^\]]+])|(LTS?|l{1,4}|L{1,4})`)
)

// The values a format reads. A value of zero is one not read, as JavaScript reads
// either as false, but for an hour, an offset and a meridiem.
type reading struct {
	year, month, day, hour, minute, second, milli, week, offset float64
	hasHour, hasOffset, hasMeridiem, pm                         bool
}

// readFormat is customParseFormat's parser. A value's pattern is found anywhere past
// where the literals have stepped to, and the first copy of what it matched is cut
// from the input, wherever it is. Lengths are in UTF-16 code units, as JavaScript's.
func readFormat(s, layout string, env Env) (time.Time, bool) {
	for _, m := range localized.FindAllStringSubmatch(layout, -1) {
		if m[2] != "" {
			return time.Time{}, false
		}
	}
	toks := parseTokens.FindAllString(layout, -1)
	if toks == nil {
		return time.Time{}, false
	}
	in, start := utf16.Encode([]rune(s)), 0
	var r reading
	for _, tok := range toks {
		read, ok := readers[tok]
		if !ok {
			start += len(utf16.Encode([]rune(strings.TrimSuffix(strings.TrimPrefix(tok, "["), "]"))))
			continue
		}
		part := in[min(start, len(in)):]
		i, j := read.match(part)
		if i < 0 {
			return time.Time{}, false
		}
		value := string(utf16.Decode(part[i:j]))
		if !read.set(&r, value) {
			return time.Time{}, false
		}
		in = cutFirst(in, part[i:j])
	}
	return compose(r, env)
}

// cutFirst is s with the first copy of v cut out.
func cutFirst(s, v []uint16) []uint16 {
	for i := 0; i+len(v) <= len(s); i++ {
		if slices.Equal(s[i:i+len(v)], v) {
			return append(append([]uint16{}, s[:i]...), s[i+len(v):]...)
		}
	}
	return s
}

// compose makes the date of what was read, as parseFormattedInput does, taking what
// was not read from now.
func compose(r reading, env Env) (time.Time, bool) {
	if r.hasMeridiem {
		if r.pm && r.hasHour && r.hour < 12 {
			r.hour += 12
		} else if !r.pm && r.hour == 12 {
			r.hour = 0
		}
	}
	now := env.Now.In(env.Zone)
	d := r.day
	if d == 0 {
		d = 1
		if r.year == 0 && r.month == 0 {
			d = float64(now.Day())
		}
	}
	y := r.year
	if y == 0 {
		y = float64(now.Year())
	}
	month := 0.0
	if r.year == 0 || r.month != 0 {
		month = float64(now.Month() - 1)
		if r.month > 0 {
			month = r.month - 1
		}
	}
	if r.hasOffset {
		return fromMillis(dateUTC(y, month, d, r.hour, r.minute, r.second, r.milli+r.offset*60000), env.Zone)
	}
	if r.week != 0 { // the week plugin is not loaded: the call fails and the date is invalid
		return time.Time{}, false
	}
	return newDate(env.Zone, y, month, d, r.hour, r.minute, r.second, r.milli)
}

// A reader is how a token of a format reads its value: where its pattern matches first,
// as the start and end of the match or -1, and what it sets, false where the plugin
// fails.
type reader struct {
	match func(s []uint16) (int, int)
	set   func(r *reading, v string) bool
}

func number(v string) float64 {
	n, _ := strconv.ParseFloat(v, 64)
	return n
}

func setter(field func(r *reading) *float64) func(*reading, string) bool {
	return func(r *reading, v string) bool {
		*field(r) = number(v)
		return true
	}
}

func hour(r *reading, v string) bool {
	r.hour, r.hasHour = number(v), true
	return true
}

func meridiem(pm string) func(*reading, string) bool {
	return func(r *reading, v string) bool {
		r.hasMeridiem, r.pm = true, v == pm
		return true
	}
}

func month(names func(string) string) func(*reading, string) bool {
	return func(r *reading, v string) bool {
		for i, m := range months {
			if names(m) == v {
				r.month = float64(i + 1)
				return true
			}
		}
		return false
	}
}

var firstDigits = regexp.MustCompile(`\d+`)

// dayOrdinal is Do: the digits of the value. The plugin goes on to look the value up
// among the locale's ordinals, whose digits in English are the day already read, and
// keeps a day of 0 as the text "0", which Do never writes back: neither changes a date
// that reads back.
func dayOrdinal(r *reading, v string) bool {
	digits := firstDigits.FindString(v)
	r.day = number(digits)
	return digits != ""
}

// offset is the zone of a Z or ZZ, in minutes behind UTC, as offsetFromString keeps it.
func offset(r *reading, v string) bool {
	r.hasOffset = true
	if v == "Z" {
		return true
	}
	parts := zoneParts.FindAllString(v, -1)
	minutes := number(parts[1]) * 60
	if len(parts) > 2 {
		minutes += number(parts[2])
	}
	if parts[0] == "+" {
		minutes = -minutes
	}
	r.offset = minutes
	return true
}

var zoneParts = regexp.MustCompile(`[+-]|\d\d`)

var readers = map[string]reader{
	"A":    {matchWord, meridiem("PM")},
	"a":    {matchWord, meridiem("pm")},
	"Q":    {matchDigits(1, 1), func(r *reading, v string) bool { r.month = (number(v)-1)*3 + 1; return true }},
	"S":    {matchDigits(1, 1), func(r *reading, v string) bool { r.milli = number(v) * 100; return true }},
	"SS":   {matchDigits(2, 2), func(r *reading, v string) bool { r.milli = number(v) * 10; return true }},
	"SSS":  {matchDigits(3, 3), setter(func(r *reading) *float64 { return &r.milli })},
	"s":    {matchDigits(1, 2), setter(func(r *reading) *float64 { return &r.second })},
	"ss":   {matchDigits(1, 2), setter(func(r *reading) *float64 { return &r.second })},
	"m":    {matchDigits(1, 2), setter(func(r *reading) *float64 { return &r.minute })},
	"mm":   {matchDigits(1, 2), setter(func(r *reading) *float64 { return &r.minute })},
	"H":    {matchDigits(1, 2), hour},
	"h":    {matchDigits(1, 2), hour},
	"HH":   {matchDigits(1, 2), hour},
	"hh":   {matchDigits(1, 2), hour},
	"D":    {matchDigits(1, 2), setter(func(r *reading) *float64 { return &r.day })},
	"DD":   {matchDigits(2, 2), setter(func(r *reading) *float64 { return &r.day })},
	"Do":   {matchWord, dayOrdinal},
	"w":    {matchDigits(1, 2), setter(func(r *reading) *float64 { return &r.week })},
	"ww":   {matchDigits(2, 2), setter(func(r *reading) *float64 { return &r.week })},
	"M":    {matchDigits(1, 2), setter(func(r *reading) *float64 { return &r.month })},
	"MM":   {matchDigits(2, 2), setter(func(r *reading) *float64 { return &r.month })},
	"MMM":  {matchWord, month(func(m string) string { return m[:3] })},
	"MMMM": {matchWord, month(func(m string) string { return m })},
	"Y":    {matchSigned, setter(func(r *reading) *float64 { return &r.year })},
	"YY": {matchDigits(2, 2), func(r *reading, v string) bool {
		r.year = number(v) + 2000
		if number(v) > 68 {
			r.year -= 100
		}
		return true
	}},
	"YYYY": {matchDigits(4, 4), setter(func(r *reading) *float64 { return &r.year })},
	"Z":    {matchOffset, offset},
	"ZZ":   {matchOffset, offset},
}

func isDigit(c uint16) bool { return '0' <= c && c <= '9' }

// leftmost is the first match in s of a pattern that at matches at a position: the end
// of its match there, or -1.
func leftmost(s []uint16, at func(s []uint16, i int) int) (int, int) {
	for i := range s {
		if j := at(s, i); j >= 0 {
			return i, j
		}
	}
	return -1, -1
}

func digitsAt(s []uint16, i, most int) int {
	j := i
	for j < len(s) && j-i < most && isDigit(s[j]) {
		j++
	}
	return j
}

// matchDigits is \d{least,most}.
func matchDigits(least, most int) func([]uint16) (int, int) {
	return func(s []uint16) (int, int) {
		return leftmost(s, func(s []uint16, i int) int {
			if j := digitsAt(s, i, most); j-i >= least {
				return j
			}
			return -1
		})
	}
}

// matchSigned is [+-]?\d+.
func matchSigned(s []uint16) (int, int) {
	return leftmost(s, func(s []uint16, i int) int {
		j := i
		if s[j] == '+' || s[j] == '-' {
			j++
		}
		if k := digitsAt(s, j, len(s)); k > j {
			return k
		}
		return -1
	})
}

// matchOffset is [+-]\d\d:?(\d\d)?|Z.
func matchOffset(s []uint16) (int, int) {
	return leftmost(s, func(s []uint16, i int) int {
		if s[i] == 'Z' {
			return i + 1
		}
		if (s[i] != '+' && s[i] != '-') || digitsAt(s, i+1, 2) != i+3 {
			return -1
		}
		j := i + 3
		if j < len(s) && s[j] == ':' {
			j++
		}
		if digitsAt(s, j, 2) == j+2 {
			j += 2
		}
		return j
	})
}

// matchWord is \d*[^-_:/,()\s\d]+.
func matchWord(s []uint16) (int, int) {
	return leftmost(s, func(s []uint16, i int) int {
		j := digitsAt(s, i, len(s))
		k := j
		for k < len(s) && !isDigit(s[k]) && !strings.ContainsRune("-_:/,()", rune(s[k])) && !syntax.IsSpace(rune(s[k])) {
			k++
		}
		if k == j {
			return -1
		}
		return k
	})
}
