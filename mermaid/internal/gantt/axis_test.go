package gantt

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// The ticks are those Mermaid 12.3.0 drew in Chrome, in Shanghai, the diagram 740 wide;
// each is its x and label.
func TestAxis(t *testing.T) {
	const ymd = "gantt\ndateFormat YYYY-MM-DD\n"
	// January's odd days, two days a tick over 590 pixels
	odd := func(year int, suffix string) []string {
		var ticks []string
		for i, x := range []int{0, 39, 79, 118, 157, 197, 236, 275, 315, 354, 393, 433, 472, 511, 551, 590} {
			ticks = append(ticks, fmt.Sprintf("%d %d-01-%02d%s", x, year, 2*i+1, suffix))
		}
		return ticks
	}
	cases := []struct {
		name, src string
		want      []string
	}{
		{"30 days tick every two", ymd + "A :2014-01-01, 30d", odd(2014, "")},
		{"two days count again from the first of a month", ymd + "tickInterval 2day\naxisFormat %b %e\nA :2014-01-25, 14d",
			[]string{"0 Jan 25", "84 Jan 27", "169 Jan 29", "253 Jan 31", "295 Feb  1", "379 Feb  3", "464 Feb  5", "548 Feb  7"}},
		{"over 10000 ticks are d3's own", ymd + "tickInterval 1millisecond\nA :2014-01-01, 3d",
			[]string{"0 2014-01-01", "49 2014-01-01", "98 2014-01-01", "148 2014-01-01", "197 2014-01-02", "246 2014-01-02", "295 2014-01-02",
				"344 2014-01-02", "393 2014-01-03", "443 2014-01-03", "492 2014-01-03", "541 2014-01-03", "590 2014-01-04"}},
		{"an offset of local mean time keeps its fraction", ymd + "axisFormat %Y-%m-%d %Z\nA :1900-01-01, 30d", odd(1900, " +085.71666666666664")},
	}
	zone, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	env := Env{Now: time.Date(2026, 11, 9, 10, 30, 15, 0, zone), Zone: zone}
	for _, c := range cases {
		body, _ := syntax.Prepare(c.src)
		d, err := Read(body, env)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		a, err := axisOf(d, 740)
		var got []string
		for _, tk := range a.Ticks {
			got = append(got, fmt.Sprintf("%v %s", tk.X, tk.Label))
		}
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %v %q, want %q", c.name, err, got, c.want)
		}
	}
}

// The next thousandth week from 1970 is past what a Date holds, where d3 never stops
// looking for the tick. A time in milliseconds is the way to come this near the end.
func TestAxisNeverFinished(t *testing.T) {
	body, _ := syntax.Prepare("gantt\ndateFormat x\ntickInterval 1000week\nA :8639990000000000, 1d")
	d, err := Read(body, Env{Now: time.Now(), Zone: time.UTC})
	if err != nil {
		t.Fatal(err)
	}
	_, err = axisOf(d, 740)
	if se, ok := err.(*syntax.Error); !ok || se.Line != 3 || se.Col != 1 {
		t.Errorf("got %v, want an error at 3:1", err)
	}
}
