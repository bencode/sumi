package syntax

import (
	"testing"

	"github.com/bencode/sumi/scene"
)

func TestColor(t *testing.T) {
	for _, c := range []struct {
		in   string
		want scene.Color
		ok   bool
	}{
		{"rgb(0, 128, 255)", scene.Color{R: 0, G: 128, B: 255, A: 255}, true},
		{"rgba(100%, 0%, 0%, 0.5)", scene.Color{R: 255, A: 128}, true},
		{"rgb(0 128 255 / 25%)", scene.Color{R: 0, G: 128, B: 255, A: 64}, true},
		{"RGB(300, -5, 0)", scene.Color{R: 255, A: 255}, true},
		{"hsl(120deg, 100%, 25%)", scene.Color{G: 128, A: 255}, true},
		{"hsl(120 100 25)", scene.Color{G: 128, A: 255}, true},
		{"hsl(0.25turn 100% 50% / none)", scene.Color{R: 128, G: 255}, true},
		{"RebeccaPurple", scene.Color{R: 0x66, G: 0x33, B: 0x99, A: 255}, true},
		{"transparent", scene.Color{}, true},
		{"#F9f", scene.Color{R: 0xff, G: 0x99, B: 0xff, A: 255}, true},
		{"#0a0f", scene.Color{G: 0xaa, A: 255}, true},
		{"#ff8800", scene.Color{R: 0xff, G: 0x88, A: 255}, true},
		{"#33669980", scene.Color{R: 0x33, G: 0x66, B: 0x99, A: 0x80}, true},
		{"#12345", scene.Color{}, false},
		{"#ggg", scene.Color{}, false},
		{"#+fff", scene.Color{}, false},
		{"#", scene.Color{}, false},
		{"rgb(1,2)", scene.Color{}, false},
		{"rgb(1, 2 3)", scene.Color{}, false},
		{"rgb(100%, 0, 0)", scene.Color{}, false},
		{"rgb(none, 0, 0)", scene.Color{}, false},
		{"hsl(120, 100, 25)", scene.Color{}, false},
		{"blu", scene.Color{}, false},
		{"", scene.Color{}, false},
	} {
		got, ok := Color(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("Color(%q) = %v, %v; want %v, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}
