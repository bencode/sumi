package gantt

import (
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// The times are those Mermaid 12.3.0 gave in Chrome, in milliseconds.
func TestReadDates(t *testing.T) {
	const shanghai, newYork = "Asia/Shanghai", "America/New_York"
	const ymd = "gantt\ndateFormat YYYY-MM-DD\n"
	cases := []struct {
		name, zone, src       string
		start, end, renderEnd int64 // of the first task; no renderEnd is 0
	}{
		{"a date alone is UTC with no dateFormat", shanghai, "gantt\nA :2014-01-01, 1d", 1388534400000, 1388620800000, 0},
		{"and local with one", shanghai, ymd + "A :2014-01-01, 1d", 1388505600000, 1388592000000, 0},
		{"half a day rounds to a day", shanghai, ymd + "A :2014-01-31, 0.5d", 1391097600000, 1391184000000, 0},
		{"hours add their milliseconds", shanghai, ymd + "A :2014-01-31, 1.5h", 1391097600000, 1391103000000, 0},
		{"a month from January 31 ends February 28", shanghai, ymd + "A :2014-01-31, 1M", 1391097600000, 1393516800000, 0},
		{"two days into summer time are 47 hours", newYork, ymd + "A :2025-03-08, 2d", 1741410000000, 1741579200000, 0},
		{"a skipped local time takes the offset before", newYork, "gantt\nA :2025-03-09T02:30, 1d", 1741505400000, 1741591800000, 0},
		{"a repeated one takes it too", newYork, "gantt\nA :2025-11-02T01:30, 1d", 1762061400000, 1762151400000, 0},
		{"a time with an offset", shanghai, "gantt\nA :2014-01-01T10:00+05:30, 1d", 1388550600000, 1388637000000, 0},
		{"after a task written later", shanghai, ymd + "B :b1, after a1, 2d\nA :a1, 2014-01-01, 3d", 1388764800000, 1388937600000, 0},
		{"excluded weekends move the end", shanghai, ymd + "excludes weekends\nweekend friday\nA :2014-01-01, 7d", 1388505600000, 1389456000000, 1389283200000},
	}
	for _, c := range cases {
		zone, err := time.LoadLocation(c.zone)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := syntax.Prepare(c.src)
		d, err := Read(body, Env{Now: time.Date(2026, 11, 9, 10, 30, 15, 0, zone), Zone: zone})
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		task := d.Tasks[0]
		renderEnd := int64(0)
		if task.RenderEnd != nil {
			renderEnd = task.RenderEnd.UnixMilli()
		}
		if task.Start.UnixMilli() != c.start || task.End.UnixMilli() != c.end || renderEnd != c.renderEnd {
			t.Errorf("%s: %d, %d, %d, want %d, %d, %d", c.name, task.Start.UnixMilli(), task.End.UnixMilli(), renderEnd, c.start, c.end, c.renderEnd)
		}
	}
}

// A start in no format and not in ECMAScript's date time string format is an invalid
// date, which Mermaid throws. Mermaid hands such a string to the browser's Date, which
// reads some of them its own way: these Chrome reads.
func TestReadInvalidDate(t *testing.T) {
	for _, s := range []string{"x5 Jan 2014", "Jan 10 2014", "2014-02-30", "2014/01/08"} {
		body, _ := syntax.Prepare("gantt\nA :" + s + ", 3d")
		_, err := Read(body, Env{Now: time.Now(), Zone: time.UTC})
		if se, ok := err.(*syntax.Error); !ok || se.IsUnsupported() || se.Msg != "Invalid date:"+s {
			t.Errorf("%s: got %v, want Invalid date:%s", s, err, s)
		}
	}
}

// Tasks that start after each other never get their times, and stay unprocessed.
func TestReadCycle(t *testing.T) {
	body, _ := syntax.Prepare("gantt\ndateFormat YYYY-MM-DD\nA :b, after a, 1d\nB :a, after b, 1d")
	d, err := Read(body, Env{Now: time.Now(), Zone: time.UTC})
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range d.Tasks {
		if task.Processed || task.Start != nil || task.End != nil {
			t.Errorf("%s: processed %v, start %v, end %v", task.ID, task.Processed, task.Start, task.End)
		}
	}
}
