package sequence

import (
	"reflect"
	"strings"
	"testing"

	"github.com/bencode/sumi/scene"
)

// tenPx sets every character 10 wide, so widths are easy to work out.
type tenPx struct{}

func (tenPx) Measure(s string, f scene.Font) scene.Metrics {
	return scene.Metrics{Width: 10 * float64(len([]rune(s))), Ascent: 0.9 * f.Size, Descent: 0.2 * f.Size}
}

// TestGaps has a case for each way a message or note widens a gap
// (getMaxMessageWidthPerActor). Three participants 150 wide are 200 apart;
// a text 30 wide in characters needs 300 + 2 * 10 of padding + 50.
func TestGaps(t *testing.T) {
	wide := strings.Repeat("x", 30)
	tests := []struct {
		name, stmt string
		want       []float64
	}{
		{"neighbors", "A->>B: " + wide, []float64{370, 200}},
		{"neighbors, leftward", "B->>A: " + wide, []float64{370, 200}},
		{"to itself", "B->>B: " + wide, []float64{200, 210}},
		{"right of", "Note right of B: " + wide, []float64{200, 370}},
		{"left of", "Note left of B: " + wide, []float64{370, 200}},
		{"over one", "Note over B: " + wide, []float64{210, 210}},
		{"over two", "Note over A,B: " + wide, []float64{210, 200}},
		{"past a participant", "A->>C: " + wide, []float64{200, 200}},
		{"left of the first", "Note left of A: " + wide, []float64{200, 200}},
		{"right of the last", "Note right of C: " + wide, []float64{200, 200}},
		{"a wide label", "participant B as " + strings.Repeat("y", 20), []float64{235, 235}},
	}
	for _, tt := range tests {
		d, err := read("sequenceDiagram\nparticipant A\nparticipant B\nparticipant C\n" + tt.stmt)
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		x := arrange(d, tenPx{}).e.x
		got := []float64{x[1] - x[0], x[2] - x[1]}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: gaps %v, want %v", tt.name, got, tt.want)
		}
	}
}

// TestWrap checks what wrap: changes across and down. Wrapped to 130, a
// word of 30 or 60 characters breaks into hyphenated lines of 13 and a
// hyphen, 140 wide, as wrapLabel keeps the character that overflows: a
// message's gap needs its widest line, 140, and padding, not 300; a
// participant's five lines make every participant 75 tall, not 65.
func TestWrap(t *testing.T) {
	tests := []struct {
		name, src string
		got       func(layout) float64
		want      float64
	}{
		{"a message's gap", "A->>B:wrap: " + strings.Repeat("x", 30),
			func(l layout) float64 { return l.e.x[1] - l.e.x[0] }, 210},
		{"a participant's height", "participant A as wrap: " + strings.Repeat("x", 60) + "\nparticipant B",
			func(l layout) float64 { return l.e.c.height }, 75},
	}
	for _, tt := range tests {
		d, err := read("sequenceDiagram\n" + tt.src)
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		if got := tt.got(arrange(d, tenPx{})); got != tt.want {
			t.Errorf("%s: %v, want %v", tt.name, got, tt.want)
		}
	}
}
