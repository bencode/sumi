package gantt

import (
	"math"
	"time"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// parseDate is new Date(s) for a string, as V8 reads one: first as a date time string of
// ECMAScript, then on with its legacy rules where that stops at the date. An ISO date
// alone is UTC, other dates local unless they give an offset. A string V8 cannot read is
// an invalid date, an error in Mermaid.
func parseDate(at syntax.Pos, s string, zone *time.Location) (time.Time, error) {
	if f, ok := readDate(s); ok {
		if t, ok := f.in(zone); ok {
			return t, nil
		}
	}
	return time.Time{}, syntax.Errorf(at, "Invalid date:%s", s)
}

// dateFields are the fields of a date V8 has read: the month from 0, and the offset in
// seconds, nil for a local time.
type dateFields struct {
	year, month, day, hour, minute, second, milli int
	offset                                        *int
}

// in is the time of f in zone, as ParseDateTimeString makes it.
func (f dateFields) in(zone *time.Location) (time.Time, bool) {
	y, m, d := float64(f.year), float64(f.month), float64(f.day)
	h, mi, s, ms := float64(f.hour), float64(f.minute), float64(f.second), float64(f.milli)
	if f.offset == nil {
		return local(zone, y, m, d, h, mi, s, ms)
	}
	return fromMillis(makeDay(y, m, d)*msPerDay+makeTime(h, mi, s, ms)-float64(*f.offset)*1000, zone)
}

// readDate is DateParser::Parse.
func readDate(s string) (dateFields, bool) {
	d, t, z := dayParts{month: none}, timeParts{ampm: none}, offsetParts{none, none, none}
	rest, ok := readISO(dateTokens(s), &d, &t, &z)
	if !ok || !readLegacy(rest, &d, &t, &z) {
		return dateFields{}, false
	}
	var f dateFields
	var dayOK, timeOK, offsetOK bool
	f.year, f.month, f.day, dayOK = d.write()
	f.hour, f.minute, f.second, f.milli, timeOK = t.write()
	f.offset, offsetOK = z.write()
	return f, dayOK && timeOK && offsetOK
}

// tokens are what is left to read of a string, its end last, which stays.
type tokens []dateToken

func (ts tokens) peek() dateToken { return ts[0] }

func (ts *tokens) next() dateToken {
	t := (*ts)[0]
	if len(*ts) > 1 {
		*ts = (*ts)[1:]
	}
	return t
}

func (ts *tokens) skip(c rune) bool {
	if ts.peek().isSymbol(c) {
		ts.next()
		return true
	}
	return false
}

// readISO is ParseES5DateTime: [+-YYYYYY|YYYY][-MM[-DD]][THH:mm[:ss[.s+]][Z|+-hh:mm|+-hhmm]],
// all of the string. Where it stops at the date, it gives the legacy rules the tokens to
// go on with, those it took being lost; where it stops in the time, the date is invalid.
func readISO(ts tokens, d *dayParts, t *timeParts, z *offsetParts) (tokens, bool) {
	if ts.peek().isSign() {
		sign := ts.next()
		if !ts.peek().isNumber(6) {
			return append(tokens{sign}, ts...), true
		}
		year := ts.next().value
		if sign.isSymbol('-') {
			if year == 0 {
				return append(tokens{sign}, ts...), true
			}
			year = -year
		}
		d.add(year)
	} else if ts.peek().isNumber(4) {
		d.add(ts.next().value)
	} else {
		return ts, true
	}
	if ts.skip('-') {
		if !ts.peek().isNumber(2) || !isMonth(ts.peek().value) {
			return ts, true
		}
		d.add(ts.next().value)
		if ts.skip('-') {
			if !ts.peek().isNumber(2) || !isMonthDay(ts.peek().value) {
				return ts, true
			}
			d.add(ts.next().value)
		}
	}
	if ts.peek().word != wTimeSep {
		if ts.peek().kind != tEnd {
			return ts, true
		}
	} else if !readISOTime(&ts, t, z) {
		return nil, false
	}
	if z.isEmpty() && t.isEmpty() {
		z.set(0) // a date alone is UTC
	}
	d.iso = true
	return ts, true
}

// readISOTime reads the time of an ISO date from its T to the end.
func readISOTime(ts *tokens, t *timeParts, z *offsetParts) bool {
	ts.next()
	two := func(lo, hi int) bool { p := ts.peek(); return p.isNumber(2) && between(p.value, lo, hi) }
	if !two(0, 24) {
		return false
	}
	h24 := ts.peek().value == 24
	t.add(ts.next().value)
	if !ts.skip(':') || !two(0, 59) || h24 && ts.peek().value > 0 {
		return false
	}
	t.add(ts.next().value)
	if ts.skip(':') {
		if !two(0, 59) || h24 && ts.peek().value > 0 {
			return false
		}
		t.add(ts.next().value)
		if ts.skip('.') {
			if ts.peek().kind != tNumber || h24 && ts.peek().value > 0 {
				return false
			}
			t.add(millis(ts.next()))
		}
	}
	if ts.peek().isZ() {
		ts.next()
		z.set(0)
	} else if ts.peek().isSign() {
		z.setSign(ts.next().value)
		if ts.peek().isNumber(4) {
			hm := ts.next().value
			if !isHour(hm/100) || !isMinute(hm%100) {
				return false
			}
			z.hour, z.minute = hm/100, hm%100
		} else {
			if !two(0, 23) {
				return false
			}
			z.hour = ts.next().value
			if !ts.skip(':') || !two(0, 59) {
				return false
			}
			z.minute = ts.next().value
		}
	}
	return ts.peek().kind == tEnd
}

// readLegacy is the loop of DateParser::Parse after ParseES5DateTime: words before the
// first number are ignored, and so is any character that is no number, word or sign.
func readLegacy(ts tokens, d *dayParts, t *timeParts, z *offsetParts) bool {
	read := len(d.parts) > 0 // a number has been read
	for tok := ts.next(); tok.kind != tEnd; tok = ts.next() {
		switch {
		case tok.kind == tNumber:
			read = true
			if !readLegacyNumber(&ts, tok.value, d, t, z) {
				return false
			}
		case tok.kind == tWord:
			switch {
			case tok.word == wAmPm && !t.isEmpty():
				t.ampm = tok.value
			case tok.word == wMonth:
				d.month = tok.value
				ts.skip('-')
			case tok.word == wZone && read:
				z.set(tok.value)
			case read || ts.peek().kind == tNumber:
				// A word after a number, or one right before the first.
				return false
			}
		case tok.isSign() && (z.isUTC() || !t.isEmpty()):
			// An offset, after UTC or a time: +hh:, +h or +hh, +hmm or +hhmm.
			z.setSign(tok.value)
			n, length := 0, 0
			if ts.peek().kind == tNumber {
				p := ts.next()
				n, length = p.value, p.length
			}
			read = true
			switch {
			case ts.peek().isSymbol(':'):
				z.hour, z.minute = n, none
			case length == 1 || length == 2:
				z.hour, z.minute = n, 0
			case length == 3 || length == 4:
				z.hour, z.minute = n/100, n%100
			default:
				return false
			}
		case (tok.isSign() || tok.isSymbol(')')) && read:
			return false
		}
	}
	return true
}

// readLegacyNumber places a number: a time's before a colon, a time's and its
// milliseconds before a point, the minutes of an offset or the next of a time where they
// fit, else a part of the date.
func readLegacyNumber(ts *tokens, n int, d *dayParts, t *timeParts, z *offsetParts) bool {
	switch {
	case ts.skip(':'):
		if ts.skip(':') {
			if !t.isEmpty() {
				return false
			}
			t.add(n)
			t.add(0)
		} else {
			if !t.add(n) {
				return false
			}
			ts.skip('.')
		}
	case ts.skip('.') && t.expecting(n): // the point is taken even where the time cannot go on
		t.add(n)
		if ts.peek().kind != tNumber {
			return false
		}
		t.addFinal(millis(ts.next()))
	case z.expecting(n):
		z.minute = n
	case t.expecting(n):
		t.addFinal(n)
		p := ts.peek()
		return p.kind == tEnd || p.kind == tSpace || p.isZ() || p.isSign()
	default:
		if !d.add(n) {
			return false
		}
		ts.skip('-')
	}
	return true
}

// none is a part not read, V8's kNone.
const none = math.MaxInt32

func between(x, lo, hi int) bool { return lo <= x && x <= hi }
func isMonth(x int) bool         { return between(x, 1, 12) }
func isMonthDay(x int) bool      { return between(x, 1, 31) }
func isHour(x int) bool          { return between(x, 0, 23) }
func isMinute(x int) bool        { return between(x, 0, 59) }

// dayParts is DayComposer: up to three numbers of a date, and a month named.
type dayParts struct {
	parts []int
	month int
	iso   bool
}

func (d *dayParts) add(n int) bool {
	if len(d.parts) == 3 {
		return false
	}
	d.parts = append(d.parts, n)
	return true
}

// write is the year, month from 0 and day. The parts are filled to three with ones before
// their order is chosen, so a day and a month alone are of the year 1, and so 2001.
func (d dayParts) write() (year, month, day int, ok bool) {
	if len(d.parts) == 0 {
		return 0, 0, 0, false
	}
	p := [3]int{1, 1, 1}
	copy(p[:], d.parts)
	switch {
	case d.month != none && isMonthDay(p[0]):
		month, day, year = d.month, p[0], p[1]
	case d.month != none:
		month, year, day = d.month, p[0], p[1]
	case d.iso || !isMonthDay(p[0]):
		year, month, day = p[0], p[1], p[2]
	default:
		month, day, year = p[0], p[1], p[2]
	}
	if !d.iso && between(year, 0, 49) {
		year += 2000
	} else if !d.iso && between(year, 50, 99) {
		year += 1900
	}
	return year, month - 1, day, isMonth(month) && isMonthDay(day)
}

// timeParts is TimeComposer: up to four numbers of a time, and the hours am or pm add.
type timeParts struct {
	parts []int
	ampm  int
}

func (t timeParts) isEmpty() bool { return len(t.parts) == 0 }

func (t *timeParts) add(n int) bool {
	if len(t.parts) == 4 {
		return false
	}
	t.parts = append(t.parts, n)
	return true
}

// addFinal adds n as the last part.
func (t *timeParts) addFinal(n int) bool {
	if !t.add(n) {
		return false
	}
	for len(t.parts) < 4 {
		t.parts = append(t.parts, 0)
	}
	return true
}

// expecting is whether n fits as the next part after the hour.
func (t timeParts) expecting(n int) bool {
	i := len(t.parts)
	return i == 1 && isMinute(n) || i == 2 && isMinute(n) || i == 3 && between(n, 0, 999)
}

// write is the hour, minute, second and millisecond: 24:00 is allowed, and am or pm needs
// an hour up to 12.
func (t timeParts) write() (h, m, s, ms int, ok bool) {
	p := [4]int{}
	copy(p[:], t.parts)
	h, m, s, ms = p[0], p[1], p[2], p[3]
	if t.ampm != none {
		if !between(h, 0, 12) {
			return 0, 0, 0, 0, false
		}
		h = h%12 + t.ampm
	}
	valid := isHour(h) && isMinute(m) && isMinute(s) && between(ms, 0, 999)
	return h, m, s, ms, valid || h == 24 && m == 0 && s == 0 && ms == 0
}

// offsetParts is TimeZoneComposer: the sign, hours and minutes of an offset.
type offsetParts struct {
	sign, hour, minute int
}

func (z *offsetParts) set(hours int) {
	z.sign = 1
	if hours < 0 {
		z.sign = -1
	}
	z.hour, z.minute = hours*z.sign, 0
}

// setSign takes the sign of a symbol, + or -.
func (z *offsetParts) setSign(c int) {
	z.sign = 1
	if c == '-' {
		z.sign = -1
	}
}

func (z offsetParts) isEmpty() bool        { return z.hour == none }
func (z offsetParts) isUTC() bool          { return z.hour == 0 && z.minute == 0 }
func (z offsetParts) expecting(n int) bool { return z.hour != none && z.minute == none && isMinute(n) }

// write is the offset in seconds, nil for none. V8 counts them in 32 unsigned bits, which
// wrap, and takes no more than a small integer of its own holds, 2^30 - 1.
func (z offsetParts) write() (*int, bool) {
	if z.sign == none {
		return nil, true
	}
	h, m := z.hour, z.minute
	if h == none {
		h = 0
	}
	if m == none {
		m = 0
	}
	total := uint32(h)*3600 + uint32(m)*60
	if total > 1<<30-1 {
		return nil, false
	}
	s := int(total) * z.sign
	return &s, true
}

// millis is ReadMilliseconds: the first three significant digits of a number, as V8
// infers them from the value it read and the number's length.
func millis(t dateToken) int {
	n, length := t.value, t.length
	if length == 1 {
		return n * 100
	}
	if length == 2 {
		return n * 10
	}
	for length = min(length, 9); length > 3; length-- {
		n /= 10
	}
	return n
}
