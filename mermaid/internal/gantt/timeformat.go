package gantt

import (
	"math"
	"strconv"
	"strings"
	"time"
)

// formatTime is timeFormat(spec)(t) of d3-time-format, in its default locale, en-US. A %
// may take a - (no padding), _ (spaces) or 0 (zeros) before its letter; a letter it does
// not know is written as it is, and the % is dropped.
func formatTime(spec string, t float64, zone *time.Location) string {
	rs := []rune(spec)
	at := func(i int) rune {
		if i < len(rs) {
			return rs[i]
		}
		return -1
	}
	l := dateOf(t, zone)
	var b strings.Builder
	for i := 0; i < len(rs); i++ {
		if rs[i] != '%' {
			b.WriteRune(rs[i])
			continue
		}
		i++
		c := at(i)
		pad, ok := pads[c]
		if ok {
			i++
			c = at(i)
		} else if c == 'e' {
			pad = " "
		} else {
			pad = "0"
		}
		if s, ok := localeFormats[c]; ok {
			b.WriteString(formatTime(s, t, zone))
		} else if f, ok := directives[c]; ok {
			b.WriteString(f(t, l, zone, pad))
		} else if c >= 0 {
			b.WriteRune(c)
		}
	}
	return b.String()
}

var pads = map[rune]string{'-': "", '_': " ", '0': "0"}

// The names are format.go's, en-US's too, their first three letters the short ones.

// localeFormats are the directives en-US gives as formats of their own, which take no
// padding.
var localeFormats = map[rune]string{'c': "%x, %X", 'x': "%-m/%-d/%Y", 'X': "%-I:%M:%S %p"}

type directive func(t float64, l localDate, zone *time.Location, pad string) string

// twoDigits pads a field of the local date to two digits.
func twoDigits(get func(localDate) float64) directive {
	return func(_ float64, l localDate, _ *time.Location, pad string) string { return padded(get(l), pad, 2) }
}

var directives = map[rune]directive{
	'a': func(_ float64, l localDate, _ *time.Location, _ string) string { return weekdays[int(l.weekday)][:3] },
	'A': func(_ float64, l localDate, _ *time.Location, _ string) string { return weekdays[int(l.weekday)] },
	'b': func(_ float64, l localDate, _ *time.Location, _ string) string { return months[int(l.m)][:3] },
	'B': func(_ float64, l localDate, _ *time.Location, _ string) string { return months[int(l.m)] },
	'd': twoDigits(func(l localDate) float64 { return l.d }),
	'e': twoDigits(func(l localDate) float64 { return l.d }),
	'f': func(_ float64, l localDate, _ *time.Location, pad string) string { return padded(l.ms, pad, 3) + "000" },
	'g': func(t float64, _ localDate, zone *time.Location, pad string) string {
		return padded(math.Mod(dateOf(isoThursday(t, zone), zone).y, 100), pad, 2)
	},
	'G': func(t float64, _ localDate, zone *time.Location, pad string) string {
		return padded(math.Mod(dateOf(isoThursday(t, zone), zone).y, 10000), pad, 4)
	},
	'H': twoDigits(func(l localDate) float64 { return l.h }),
	'I': twoDigits(func(l localDate) float64 {
		if h := math.Mod(l.h, 12); h != 0 {
			return h
		}
		return 12
	}),
	'j': func(t float64, _ localDate, zone *time.Location, pad string) string {
		return padded(1+timeDay(zone).countOf(timeYear(zone).floor(t), t), pad, 3)
	},
	'L': func(_ float64, l localDate, _ *time.Location, pad string) string { return padded(l.ms, pad, 3) },
	'm': twoDigits(func(l localDate) float64 { return l.m + 1 }),
	'M': twoDigits(func(l localDate) float64 { return l.mi }),
	'p': func(_ float64, l localDate, _ *time.Location, _ string) string {
		if l.h >= 12 {
			return "PM"
		}
		return "AM"
	},
	'q': func(_ float64, l localDate, _ *time.Location, _ string) string {
		return jsString(1 + math.Trunc(l.m/3))
	},
	'Q': func(t float64, _ localDate, _ *time.Location, _ string) string { return jsString(t) },
	's': func(t float64, _ localDate, _ *time.Location, _ string) string { return jsString(math.Floor(t / 1000)) },
	'S': twoDigits(func(l localDate) float64 { return l.s }),
	'u': func(_ float64, l localDate, _ *time.Location, _ string) string {
		if l.weekday == 0 {
			return "7"
		}
		return jsString(l.weekday)
	},
	'U': weekNumber(0),
	'V': func(t float64, _ localDate, zone *time.Location, pad string) string {
		d := isoThursday(t, zone)
		y := timeYear(zone).floor(d)
		n := timeWeek(4, zone).countOf(y, d)
		if dateOf(y, zone).weekday == 4 {
			n++
		}
		return padded(n, pad, 2)
	},
	'w': func(_ float64, l localDate, _ *time.Location, _ string) string { return jsString(l.weekday) },
	'W': weekNumber(1),
	'y': twoDigits(func(l localDate) float64 { return math.Mod(l.y, 100) }),
	'Y': func(_ float64, l localDate, _ *time.Location, pad string) string {
		return padded(math.Mod(l.y, 10000), pad, 4)
	},
	'Z': func(t float64, _ localDate, zone *time.Location, _ string) string {
		z, sign := tzOffset(t, zone), "-"
		if !(z > 0) {
			z, sign = -z, "+"
		}
		return sign + padded(math.Trunc(z/60), "0", 2) + padded(math.Mod(z, 60), "0", 2)
	},
	'%': func(float64, localDate, *time.Location, string) string { return "%" },
}

// weekNumber is %U for i of 0 and %W for 1: the weeks from weekday i since the last
// moment of the year before.
func weekNumber(i float64) directive {
	return func(t float64, _ localDate, zone *time.Location, pad string) string {
		return padded(timeWeek(i, zone).countOf(timeYear(zone).floor(t)-1, t), pad, 2)
	}
}

// isoThursday is the Thursday of t's ISO week: the one before it from Thursday to Sunday,
// else the next.
func isoThursday(t float64, zone *time.Location) float64 {
	thursday := timeWeek(4, zone)
	if wd := dateOf(t, zone).weekday; wd >= 4 || wd == 0 {
		return thursday.floor(clip(t))
	}
	return thursday.ceil(t)
}

// padded is d3's pad: the number filled on the left to width, a minus before the fill.
func padded(v float64, fill string, width int) string {
	sign := ""
	if v < 0 {
		sign, v = "-", -v
	}
	s := jsString(v)
	if len(s) < width {
		s = strings.Repeat(fill, width-len(s)) + s
	}
	return sign + s
}

// jsString is a number as JavaScript writes it, for the numbers a format meets: under
// 1e21, so never with an exponent, and -0 as 0.
func jsString(v float64) string {
	if v == 0 {
		return "0"
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}
