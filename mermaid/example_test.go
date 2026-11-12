package mermaid_test

import (
	"fmt"
	"log"
	"unicode/utf8"

	"github.com/bencode/sumi/mermaid"
	"github.com/bencode/sumi/scene"
)

// canvas stands for your drawing backend: a window, an image or a PDF. This one
// prints the text it is asked to draw and paints nothing.
type canvas struct{}

func (canvas) Fill(segs []scene.Seg, c scene.Color) {}

func (canvas) Stroke(segs []scene.Seg, width float64, dash []float64, c scene.Color) {}

func (canvas) Text(s string, f scene.Font, x, baseline float64, c scene.Color) { fmt.Println(s) }

// fixedMeasurer measures text as canvas would set it, here every character 0.6
// em wide. Ascent and descent are the font's, the same for every string, the
// empty one too.
type fixedMeasurer struct{}

func (fixedMeasurer) Measure(s string, f scene.Font) scene.Metrics {
	return scene.Metrics{Width: 0.6 * f.Size * float64(utf8.RuneCountInString(s)), Ascent: 0.8 * f.Size, Descent: 0.2 * f.Size}
}

// draw paints s on cv in pal's colors, item by item: each path filled and then
// stroked, then each line of text on its baseline.
func draw(cv canvas, s *scene.Scene, pal scene.Palette) {
	for _, it := range s.Items {
		for _, p := range it.Paths {
			if c, ok := pal.Resolve(p.Fill); ok {
				cv.Fill(p.Segs, c)
			}
			if c, ok := pal.Resolve(p.Stroke.Paint); ok && p.Stroke.Width > 0 {
				cv.Stroke(p.Segs, p.Stroke.Width, p.Stroke.Dash, c)
			}
		}
		for _, t := range it.Texts {
			c, ok := pal.Resolve(t.Paint)
			if !ok {
				continue
			}
			for _, l := range t.Lines {
				cv.Text(l.S, t.Font, l.X, l.Baseline, c)
			}
		}
	}
}

// A diagram laid out in the font of the backend that draws it, and drawn there.
func Example_nativeDrawing() {
	s, err := mermaid.Layout("flowchart LR\n    A[Write] --> B[Draw]", mermaid.Options{Measurer: fixedMeasurer{}})
	if err != nil {
		log.Fatal(err)
	}
	draw(canvas{}, s, scene.Light)
	// Output:
	// Write
	// Draw
}
