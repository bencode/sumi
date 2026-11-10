package gantt

import (
	"math"
	"slices"
	"sort"
	"time"
)

// The local time intervals of d3-time, on the time values of JavaScript: milliseconds as
// a float64, NaN for an Invalid Date. Every change a Date setter makes goes through local,
// as V8 makes it. d3 only ever moves an interval on by one, so next is offset(t, 1).

// localDate is the local fields of a time as the getters of a Date give them, all NaN
// for an Invalid Date: the month from 0, and the day of the week, Sunday 0.
type localDate struct{ y, m, d, h, mi, s, ms, weekday float64 }

func dateOf(t float64, zone *time.Location) localDate {
	if math.IsNaN(t) {
		n := math.NaN()
		return localDate{n, n, n, n, n, n, n, n}
	}
	at := time.UnixMilli(int64(t)).In(zone)
	y, m, d, h, mi, s, ms := fields(at)
	return localDate{y, m, d, h, mi, s, ms, float64(at.Weekday())}
}

// set is the time of the fields, as a setter makes it: NaN where V8 gives no time.
func (l localDate) set(zone *time.Location) float64 {
	t, ok := local(zone, l.y, l.m, l.d, l.h, l.mi, l.s, l.ms)
	if !ok {
		return math.NaN()
	}
	return float64(t.UnixMilli())
}

// zeroHours is setHours(0, 0, 0, 0).
func zeroHours(t float64, zone *time.Location) float64 {
	l := dateOf(t, zone)
	l.h, l.mi, l.s, l.ms = 0, 0, 0, 0
	return l.set(zone)
}

// tzOffset is getTimezoneOffset: the minutes from local time to UTC, a fraction where the
// offset has seconds.
func tzOffset(t float64, zone *time.Location) float64 {
	if math.IsNaN(t) {
		return t
	}
	_, off := time.UnixMilli(int64(t)).In(zone).Zone()
	return float64(-off) / 60
}

// An interval is one of d3-time's: floor takes a time back to one of its times, next on
// to the next, count is how many lie between two of them, and field is what every tests,
// nil where every counts from 1970 instead. Only days and weeks are counted, for every
// and the numbers of a format, and a year's every is yearEvery, so others leave out what
// nothing asks of them.
type interval struct {
	floor, next func(t float64) float64
	count       func(a, b float64) float64
	field       func(t float64) float64
}

// countOf is interval.count: from the times a and b are floored to, rounded down.
func (iv interval) countOf(a, b float64) float64 {
	return math.Floor(iv.count(iv.floor(clip(a)), iv.floor(clip(b))))
}

func (iv interval) ceil(t float64) float64 { return iv.floor(iv.next(iv.floor(clip(t - 1)))) }

// every is interval.every(k): the interval itself for k of 1, else its times whose field,
// or count of them from 1970, k divides; none for a k that is not a finite number of at
// least 1.
func (iv interval) every(k float64) stepper {
	k = math.Floor(k)
	if math.IsInf(k, 0) || !(k > 0) {
		return nil
	}
	if !(k > 1) {
		return iv
	}
	if iv.field != nil {
		return filter{iv, func(t float64) bool { return math.Mod(iv.field(t), k) == 0 }}
	}
	return filter{iv, func(t float64) bool { return math.Mod(iv.countOf(0, t), k) == 0 }}
}

// A stepper is what d3 asks of an interval to range over it. ok is false where d3 searches
// for ever.
type stepper interface {
	floorOf(t float64) (float64, bool)
	nextOf(t float64) (float64, bool)
}

func (iv interval) floorOf(t float64) (float64, bool) { return iv.floor(t), true }
func (iv interval) nextOf(t float64) (float64, bool)  { return iv.next(t), true }

// A filter is interval.filter(test): the times of an interval that pass test, which d3
// looks for a step at a time. A step past the times a Date holds gives NaN, which never
// passes, and the search never ends.
type filter struct {
	iv   interval
	test func(t float64) bool
}

func (f filter) floorOf(t float64) (float64, bool) {
	if math.IsNaN(t) {
		return t, true
	}
	for {
		if t = f.iv.floor(t); math.IsNaN(t) {
			return t, false
		}
		if f.test(t) {
			return t, true
		}
		t = clip(t - 1)
	}
}

func (f filter) nextOf(t float64) (float64, bool) {
	if math.IsNaN(t) {
		return t, true
	}
	for {
		if t = f.iv.next(t); math.IsNaN(t) {
			return t, false
		}
		if f.test(t) {
			return t, true
		}
	}
}

// times is interval.range(start, stop): its times from start to before stop, ending too
// where a step does not move on.
func times(s stepper, start, stop float64) ([]float64, bool) {
	t, ok := s.floorOf(clip(start - 1))
	if ok {
		if t, ok = s.nextOf(t); ok {
			t, ok = s.floorOf(t)
		}
	}
	if !ok || !(t < stop) {
		return nil, ok
	}
	var ts []float64
	for {
		prev := t
		ts = append(ts, prev)
		if t, ok = s.nextOf(t); ok {
			t, ok = s.floorOf(t)
		}
		if !ok {
			return nil, false
		}
		if !(prev < t && t < stop) {
			return ts, true
		}
	}
}

const (
	msPerSecond = 1000
	msPerMinute = 60 * msPerSecond
	msPerHour   = 60 * msPerMinute
	msPerWeek   = 7 * msPerDay
)

var timeMillisecond = interval{
	floor: func(t float64) float64 { return t },
	next:  func(t float64) float64 { return clip(t + 1) },
}

// millisecondEvery is millisecond.every, which d3 writes on its own: multiples of k.
func millisecondEvery(k float64) stepper {
	k = math.Floor(k)
	if math.IsInf(k, 0) || !(k > 0) {
		return nil
	}
	if !(k > 1) {
		return timeMillisecond
	}
	return interval{
		floor: func(t float64) float64 { return clip(math.Floor(t/k) * k) },
		next:  func(t float64) float64 { return clip(t + k) },
	}
}

// fixed is an interval of a fixed length, which floors by subtracting the local fields
// below it: second, minute and hour. Its field is what get gives.
func fixed(length float64, below func(l localDate) float64, get func(t float64, l localDate) float64, zone *time.Location) interval {
	return interval{
		floor: func(t float64) float64 { return clip(t - below(dateOf(t, zone))) },
		next:  func(t float64) float64 { return clip(t + length) },
		field: func(t float64) float64 { return get(t, dateOf(t, zone)) },
	}
}

func timeSecond(zone *time.Location) interval {
	return fixed(msPerSecond, func(l localDate) float64 { return l.ms }, func(t float64, _ localDate) float64 {
		return math.Mod(math.Mod(math.Floor(t/msPerSecond), 60)+60, 60) // getUTCSeconds
	}, zone)
}

func timeMinute(zone *time.Location) interval {
	return fixed(msPerMinute, func(l localDate) float64 { return l.ms + float64(l.s*msPerSecond) },
		func(_ float64, l localDate) float64 { return l.mi }, zone)
}

func timeHour(zone *time.Location) interval {
	return fixed(msPerHour, func(l localDate) float64 { return l.ms + float64(l.s*msPerSecond) + float64(l.mi*msPerMinute) },
		func(_ float64, l localDate) float64 { return l.h }, zone)
}

// calendarCount is the count of days or weeks: the time between, less the change of
// offset, in lengths.
func calendarCount(length float64, zone *time.Location) func(a, b float64) float64 {
	return func(a, b float64) float64 {
		return (b - a - float64((tzOffset(b, zone)-tzOffset(a, zone))*msPerMinute)) / length
	}
}

func timeDay(zone *time.Location) interval {
	return interval{
		floor: func(t float64) float64 { return zeroHours(t, zone) },
		next: func(t float64) float64 {
			l := dateOf(t, zone)
			l.d++
			return l.set(zone)
		},
		count: calendarCount(msPerDay, zone),
		field: func(t float64) float64 { return dateOf(t, zone).d - 1 },
	}
}

// week is the weeks from weekday i, Sunday 0.
func timeWeek(i float64, zone *time.Location) interval {
	return interval{
		floor: func(t float64) float64 {
			l := dateOf(t, zone)
			l.d -= math.Mod(l.weekday+7-i, 7)
			return zeroHours(l.set(zone), zone)
		},
		next: func(t float64) float64 {
			l := dateOf(t, zone)
			l.d += 7
			return l.set(zone)
		},
		count: calendarCount(msPerWeek, zone),
	}
}

func timeMonth(zone *time.Location) interval {
	return interval{
		floor: func(t float64) float64 {
			l := dateOf(t, zone)
			l.d = 1
			return zeroHours(l.set(zone), zone)
		},
		next: func(t float64) float64 {
			l := dateOf(t, zone)
			l.m++
			return l.set(zone)
		},
		field: func(t float64) float64 { return dateOf(t, zone).m },
	}
}

// januaryFirst is setMonth(0, 1) and then setHours(0, 0, 0, 0).
func januaryFirst(t float64, zone *time.Location) float64 {
	l := dateOf(t, zone)
	l.m, l.d = 0, 1
	return zeroHours(l.set(zone), zone)
}

func timeYear(zone *time.Location) interval {
	return interval{
		floor: func(t float64) float64 { return januaryFirst(t, zone) },
		next: func(t float64) float64 {
			l := dateOf(t, zone)
			l.y++
			return l.set(zone)
		},
	}
}

// yearEvery is year.every, which d3 writes on its own: the years k divides, made with
// setFullYear before the year is floored, even for k of 1.
func yearEvery(k float64, zone *time.Location) stepper {
	k = math.Floor(k)
	if math.IsInf(k, 0) || !(k > 0) {
		return nil
	}
	return interval{
		floor: func(t float64) float64 {
			l := dateOf(t, zone)
			l.y = math.Floor(l.y/k) * k
			return januaryFirst(l.set(zone), zone)
		},
		next: func(t float64) float64 {
			l := dateOf(t, zone)
			l.y += k
			return l.set(zone)
		},
	}
}

// autoSteps are the intervals d3 chooses ticks from, by their lengths.
var autoSteps = []struct {
	unit   func(*time.Location) interval
	k, len float64
}{
	{timeSecond, 1, msPerSecond}, {timeSecond, 5, 5 * msPerSecond}, {timeSecond, 15, 15 * msPerSecond}, {timeSecond, 30, 30 * msPerSecond},
	{timeMinute, 1, msPerMinute}, {timeMinute, 5, 5 * msPerMinute}, {timeMinute, 15, 15 * msPerMinute}, {timeMinute, 30, 30 * msPerMinute},
	{timeHour, 1, msPerHour}, {timeHour, 3, 3 * msPerHour}, {timeHour, 6, 6 * msPerHour}, {timeHour, 12, 12 * msPerHour},
	{timeDay, 1, msPerDay}, {timeDay, 2, 2 * msPerDay},
	{timeSunday, 1, msPerWeek},
	{timeMonth, 1, 30 * msPerDay}, {timeMonth, 3, 90 * msPerDay},
	{timeYear, 1, 365 * msPerDay},
}

func timeSunday(zone *time.Location) interval { return timeWeek(0, zone) }

// autoInterval is timeTickInterval for ten ticks from start to stop, start the earlier:
// the interval nearest a tenth of the time between, or multiples of milliseconds or
// years past either end of the list.
func autoInterval(start, stop float64, zone *time.Location) stepper {
	target := math.Abs(stop-start) / 10
	i := sort.Search(len(autoSteps), func(i int) bool { return autoSteps[i].len > target })
	if i == len(autoSteps) {
		return yearEvery(tickStep(start/(365*msPerDay), stop/(365*msPerDay)), zone)
	}
	if i == 0 {
		return millisecondEvery(math.Max(tickStep(start, stop), 1))
	}
	if target/autoSteps[i-1].len < autoSteps[i].len/target {
		i--
	}
	return autoSteps[i].unit(zone).every(autoSteps[i].k)
}

// tickStep is d3-array's tickStep for ten ticks, start the earlier: a power of ten times
// 1, 2, 5 or 10 near a tenth of the way between, its reciprocal kept exact below 1.
func tickStep(start, stop float64) float64 {
	step := (stop - start) / 10
	power := math.Floor(math.Log10(step))
	e := step / math.Pow(10, power)
	factor := 1.0
	switch {
	case e >= math.Sqrt(50):
		factor = 10
	case e >= math.Sqrt(10):
		factor = 5
	case e >= math.Sqrt(2):
		factor = 2
	}
	if power < 0 {
		return 1 / (math.Pow(10, -power) / factor)
	}
	return math.Pow(10, power) * factor
}

// timeTicks is the ticks of scaleTime from start to stop, in the order of the domain: the
// times of s, or of autoInterval where s is nil. ok is false where d3 never finishes.
func timeTicks(start, stop float64, s stepper, zone *time.Location) ([]float64, bool) {
	reverse := stop < start
	if reverse {
		start, stop = stop, start
	}
	if s == nil {
		if s = autoInterval(start, stop, zone); s == nil {
			return nil, true
		}
	}
	ts, ok := times(s, start, stop+1)
	if reverse {
		slices.Reverse(ts)
	}
	return ts, ok
}
