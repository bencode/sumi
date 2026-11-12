package gantt

import (
	"regexp"
	"strconv"
	"time"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// dateTime is the date time string format of ECMAScript, the one form new Date must
// read: a year of four digits or a signed one of six, then perhaps the month and the
// day, a time of hours and minutes, perhaps seconds and milliseconds, and an offset
// after the time.
var dateTime = regexp.MustCompile(`^(\d{4}|[+-]\d{6})(?:-(\d\d)(?:-(\d\d))?)?` +
	`(?:T(\d\d):(\d\d)(?::(\d\d)(?:\.(\d{3}))?)?(Z|([+-])(\d\d):(\d\d))?)?$`)

// parseDate is new Date(s) for a string in the date time string format. A date alone
// is UTC, a time local unless it gives an offset. Mermaid hands any other string to the
// browser's Date, which each browser reads its own way; here it is an invalid date, an
// error in Mermaid.
func parseDate(at syntax.Pos, s string, zone *time.Location) (time.Time, error) {
	if t, ok := readDate(s, zone); ok {
		return t, nil
	}
	return time.Time{}, syntax.Errorf(at, "Invalid date:%s", s)
}

func readDate(s string, zone *time.Location) (time.Time, bool) {
	m := dateTime.FindStringSubmatch(s)
	if m == nil || m[1] == "-000000" {
		return time.Time{}, false
	}
	num := func(i, absent int) int {
		if m[i] == "" {
			return absent
		}
		n, _ := strconv.Atoi(m[i])
		return n
	}
	y, mo, d := num(1, 0), num(2, 1), num(3, 1)
	h, mi, sec, ms := num(4, 0), num(5, 0), num(6, 0), num(7, 0)
	offH, offM := num(10, 0), num(11, 0)
	lastDay := time.Date(y, time.Month(mo+1), 0, 0, 0, 0, 0, time.UTC).Day()
	if mo < 1 || mo > 12 || d < 1 || d > lastDay || h > 24 || h == 24 && mi+sec+ms > 0 || mi > 59 || sec > 59 ||
		offH > 23 || offM > 59 {
		return time.Time{}, false
	}
	fy, fm, fd := float64(y), float64(mo-1), float64(d)
	fh, fmi, fs, fms := float64(h), float64(mi), float64(sec), float64(ms)
	if m[4] != "" && m[8] == "" {
		return local(zone, fy, fm, fd, fh, fmi, fs, fms)
	}
	offset := float64(offH*60+offM) * 60000
	if m[9] == "-" {
		offset = -offset
	}
	return fromMillis(makeDay(fy, fm, fd)*msPerDay+makeTime(fh, fmi, fs, fms)-offset, zone)
}
