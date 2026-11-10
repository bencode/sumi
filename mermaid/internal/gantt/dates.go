package gantt

import (
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// Times here are JavaScript's: whole milliseconds, made from the fields of a calendar as
// V8 makes them, where a month or a day past its end runs into the next. A time that
// would be NaN comes back as false.

const msPerDay = 86400000

// maxTime is the largest time a Date holds, either side of 1970.
const maxTime = 8.64e15

// makeDay is MakeDay, as V8 bounds it: the day of a year, month and date, NaN past a
// million years or ten million months.
func makeDay(y, m, d float64) float64 {
	if !(-1e6 <= y && y <= 1e6 && -1e7 <= m && m <= 1e7) || math.IsInf(d, 0) || math.IsNaN(d) {
		return math.NaN()
	}
	year, month := int(y)+int(m)/12, int(m)%12
	if month < 0 {
		month, year = month+12, year-1
	}
	first := time.Date(year, time.Month(month+1), 1, 0, 0, 0, 0, time.UTC)
	return float64(first.Unix()/86400) + math.Trunc(d) - 1
}

func makeTime(h, m, s, ms float64) float64 {
	return math.Trunc(h)*3600000 + math.Trunc(m)*60000 + math.Trunc(s)*1000 + math.Trunc(ms)
}

// clip is TimeClip.
func clip(t float64) float64 {
	if math.IsNaN(t) || math.Abs(t) > maxTime {
		return math.NaN()
	}
	return math.Trunc(t)
}

// fromMillis is the time of a time value, in zone, false for NaN.
func fromMillis(t float64, zone *time.Location) (time.Time, bool) {
	t = clip(t)
	if math.IsNaN(t) {
		return time.Time{}, false
	}
	return time.UnixMilli(int64(t)).In(zone), true
}

// fullYear is a year as the Date constructor and Date.UTC read one: 0 to 99 are years of
// the 1900s.
func fullYear(y float64) float64 {
	if t := math.Trunc(y); 0 <= t && t <= 99 {
		return 1900 + t
	}
	return y
}

// dateUTC is Date.UTC.
func dateUTC(y, m, d, h, mi, s, ms float64) float64 {
	return makeDay(fullYear(y), m, d)*msPerDay + makeTime(h, mi, s, ms)
}

// newDate is new Date(y, m, d, h, mi, s, ms): the fields of a local time.
func newDate(zone *time.Location, y, m, d, h, mi, s, ms float64) (time.Time, bool) {
	return local(zone, fullYear(y), m, d, h, mi, s, ms)
}

// local is the time of a local time's fields, as a setter of a Date, the Date constructor
// and Date.parse make it. V8 asks for no offset past a month beyond the times a Date
// holds.
func local(zone *time.Location, y, m, d, h, mi, s, ms float64) (time.Time, bool) {
	wall := makeDay(y, m, d)*msPerDay + makeTime(h, mi, s, ms)
	if math.IsNaN(wall) || math.Abs(wall) > maxTime+30*msPerDay {
		return time.Time{}, false
	}
	return fromMillis(wall-float64(offsetAt(zone, wall))*1000, zone)
}

// offsetAt is the offset of zone, in seconds, at the local time wall, as V8 asks ICU for
// it: a time a change of offset skips, or one it repeats, takes the offset from before
// the change. Go's time.Date leaves which one unsaid.
func offsetAt(zone *time.Location, wall float64) int {
	sec := int64(math.Floor(wall / 1000))
	offset := func(at int64) int {
		_, off := time.Unix(at, 0).In(zone).Zone()
		return off
	}
	before, after := offset(sec-86400), offset(sec+86400)
	if before != after && offset(sec-int64(before)) != before && offset(sec-int64(after)) == after {
		return after
	}
	return before
}

// fields are the local fields of t, as numbers to make another time from.
func fields(t time.Time) (y, m, d, h, mi, s, ms float64) {
	return float64(t.Year()), float64(t.Month() - 1), float64(t.Day()),
		float64(t.Hour()), float64(t.Minute()), float64(t.Second()), float64(t.Nanosecond() / 1e6)
}

// jsRound is Math.round: halves go up.
func jsRound(x float64) float64 {
	r := math.Floor(x)
	if x-r >= 0.5 {
		r++
	}
	return r
}

// daysInMonth is the number of days in t's month, as dayjs finds it: the last day of
// new Date(year, month+1, 0), a year of 0 to 99 there being one of the 1900s.
func daysInMonth(t time.Time) float64 {
	y := int(fullYear(float64(t.Year())))
	return float64(time.Date(y, t.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day())
}

// add is dayjs's add(n, unit) for the units of a duration. Days and weeks move the date
// of the month by n rounded, months and years move from the first of the month and then
// back to the day, or the month's last, and the rest add milliseconds.
func add(t time.Time, n float64, unit string) (time.Time, bool) {
	zone := t.Location()
	y, m, d, h, mi, s, ms := fields(t)
	if unit == "d" || unit == "w" {
		k := 1.0
		if unit == "w" {
			k = 7
		}
		return local(zone, y, m, d+jsRound(n*k), h, mi, s, ms)
	}
	if unit == "M" || unit == "y" {
		first, ok := local(zone, y, m, 1, h, mi, s, ms)
		if !ok {
			return time.Time{}, false
		}
		fy, fm, fd, fh, fmi, fs, fms := fields(first)
		if unit == "M" {
			fm = m + n
		} else {
			fy = y + n
		}
		moved, ok := local(zone, fy, fm, fd, fh, fmi, fs, fms)
		if !ok {
			return time.Time{}, false
		}
		my, mm, _, mh, mmi, ms2, mms := fields(moved)
		return local(zone, my, mm, math.Min(d, daysInMonth(moved)), mh, mmi, ms2, mms)
	}
	return fromMillis(float64(t.UnixMilli())+n*steps[unit], zone)
}

// steps are the milliseconds of the units add adds as milliseconds.
var steps = map[string]float64{"h": 3600000, "m": 60000, "s": 1000, "ms": 1}

// midnight is today at 00:00, local, as new Date() with setHours(0, 0, 0, 0) makes it.
func midnight(env Env) (time.Time, bool) {
	y, m, d, _, _, _, _ := fields(env.Now.In(env.Zone))
	return local(env.Zone, y, m, d, 0, 0, 0, 0)
}

var (
	afterIDs  = regexp.MustCompile(`^after[` + syntax.JSSpaces + `]+([\d\w\- ]+)`)
	untilIDs  = regexp.MustCompile(`^until[` + syntax.JSSpaces + `]+([\d\w\- ]+)`)
	duration  = regexp.MustCompile(`^(\d+(?:\.\d+)?)([Mdhmswy]|ms)$`)
	allDigits = regexp.MustCompile(`^\d+$`)
)

// startDate is getStartDate: the end of the latest task of an after, a date in the
// diagram's format, or else what new Date makes of the text. It is nil after tasks
// whose ends are not known yet.
func (b *db) startDate(at syntax.Pos, s string) (*time.Time, error) {
	s, layout := trim(s), trim(b.d.DateFormat)
	if (layout == "x" || layout == "X") && allDigits.MatchString(s) {
		n, _ := strconv.ParseFloat(s, 64)
		t, ok := fromMillis(n, b.env.Zone)
		if !ok {
			return nil, syntax.Unsupported(at, "a timestamp past the times a Date holds")
		}
		return &t, nil
	}
	if m := afterIDs.FindStringSubmatch(s); m != nil {
		var latest *rawTask
		for _, id := range strings.Fields(m[1]) {
			if t := b.find(id); t != nil && (latest == nil || later(t.End, latest.End)) {
				latest = t
			}
		}
		if latest != nil {
			return latest.End, nil
		}
		return b.today(at)
	}
	t, ok, err := parseFormat(s, layout, b.env)
	if err != nil {
		return nil, syntax.Errorf(at, "%s", err)
	}
	if ok {
		return &t, nil
	}
	t, err = parseDate(at, s, b.env.Zone)
	if err != nil {
		return nil, err
	}
	if y := t.Year(); y < -10000 || y > 10000 {
		return nil, syntax.Errorf(at, "Invalid date:%s", s)
	}
	return &t, nil
}

// endDate is getEndDate: the start of the earliest task of an until, a date in the
// diagram's format, a day later if end dates are inclusive, or start and a duration.
// Text that is none of these leaves the task with no length.
func (b *db) endDate(at syntax.Pos, start time.Time, s string) (*time.Time, error) {
	s = trim(s)
	if m := untilIDs.FindStringSubmatch(s); m != nil {
		var earliest *rawTask
		for _, id := range strings.Fields(m[1]) {
			if t := b.find(id); t != nil && (earliest == nil || later(earliest.Start, t.Start)) {
				earliest = t
			}
		}
		if earliest != nil {
			return earliest.Start, nil
		}
		return b.today(at)
	}
	t, ok, err := parseFormat(s, trim(b.d.DateFormat), b.env)
	if err != nil {
		return nil, syntax.Errorf(at, "%s", err)
	}
	if ok && b.d.InclusiveEndDates {
		if t, ok = add(t, 1, "d"); !ok {
			return nil, syntax.Unsupported(at, "an end date past the times a Date holds")
		}
	}
	if ok {
		return &t, nil
	}
	if m := duration.FindStringSubmatch(s); m != nil {
		n, _ := strconv.ParseFloat(m[1], 64)
		if t, ok := add(start, n, m[2]); ok {
			return &t, nil
		}
	}
	return &start, nil
}

func (b *db) today(at syntax.Pos) (*time.Time, error) {
	t, ok := midnight(b.env)
	if !ok {
		return nil, syntax.Unsupported(at, "a time now past the times a Date holds")
	}
	return &t, nil
}

// later is a > b for two times a task may not have yet, which JavaScript compares as
// NaN, so as false.
func later(a, b *time.Time) bool { return a != nil && b != nil && a.After(*b) }

// isDay is whether s is a date written YYYY-MM-DD, as dayjs reads one strictly.
func isDay(s string, env Env) bool {
	_, ok, _ := parseFormat(s, "YYYY-MM-DD", env)
	return ok
}

// weekendStart is the ISO day a weekend starts on.
var weekendStart = map[string]int{"friday": 5, "saturday": 6}

// excluded is isInvalidDate: whether t is a day the diagram excludes and does not
// include, named by its date in the diagram's format or as YYYY-MM-DD, by the name of
// its day, or as one of the weekends.
func (d *Diagram) excluded(t time.Time) (bool, error) {
	formatted, err := format(t, trim(d.DateFormat))
	if err != nil {
		return false, err
	}
	day, _ := format(t, "YYYY-MM-DD")
	in, ex := d.Includes, d.Excludes
	if slices.Contains(in, formatted) || slices.Contains(in, day) {
		return false, nil
	}
	iso := (int(t.Weekday())+6)%7 + 1
	if start := weekendStart[d.Weekend]; slices.Contains(ex, "weekends") && (iso == start || iso == start+1) {
		return true, nil
	}
	return slices.Contains(ex, strings.ToLower(weekdays[t.Weekday()])) || slices.Contains(ex, formatted) || slices.Contains(ex, day), nil
}

// checkDates is checkTaskDates: with days excluded, and an end not written as a date,
// the end moves a day later for each excluded day from the day after the start, and
// RenderEnd keeps where it was before the last run of them.
func (b *db) checkDates(t *rawTask) error {
	if len(b.d.Excludes) == 0 || t.manualEnd {
		return nil
	}
	past := syntax.Unsupported(t.At, "excluded days past the times a Date holds")
	start, ok := add(*t.Start, 1, "d")
	end := *t.End
	limit, ok2 := add(end, 10000, "d")
	if !ok || !ok2 {
		return past
	}
	invalid := false
	var render *time.Time
	for !start.After(end) {
		if !invalid {
			e := end
			render = &e
		}
		var err error
		if invalid, err = b.d.excluded(start); err != nil {
			return syntax.Errorf(t.At, "%s", err)
		}
		if invalid {
			if end, ok = add(end, 1, "d"); !ok {
				return past
			}
			if end.After(limit) {
				return syntax.Errorf(t.At, "Failed to find a valid date that was not excluded by `excludes` after 10,000 iterations.")
			}
		}
		if start, ok = add(start, 1, "d"); !ok {
			return past
		}
	}
	t.End, t.RenderEnd = &end, render
	return nil
}
