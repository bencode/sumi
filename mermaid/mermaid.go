// Package mermaid lays out diagrams written in Mermaid's syntax as scenes,
// drawing what Mermaid v12.3.0 draws. So far it knows sequence diagrams, flowcharts,
// state diagrams, class diagrams, entity relationship diagrams, gantt diagrams and pie
// charts. A gantt diagram is drawn as wide as Options.Width says, in the time zone of
// time.Local, its line at the time now where time.Now falls. The styles of
// flowcharts, state, class and ER diagrams draw their fills,
// strokes, widths, dashes and text colors; fonts and the rest of CSS that changes how
// Chrome draws are not supported.
//
//	s, err := mermaid.Layout("sequenceDiagram\nAlice->>Bob: Hello", mermaid.Options{})
//	if err != nil {
//		var e *mermaid.Error
//		if errors.As(err, &e) {
//			// e.Line and e.Col say where in the source it went wrong.
//		}
//	}
//	out := svg.Render(s, scene.Light)
//
// Front matter's title is used; its config, and %%{init}%% directives, are
// not yet.
package mermaid

import (
	"strings"
	"time"

	"github.com/bencode/sumi/mermaid/internal/class"
	"github.com/bencode/sumi/mermaid/internal/er"
	"github.com/bencode/sumi/mermaid/internal/flowchart"
	"github.com/bencode/sumi/mermaid/internal/gantt"
	"github.com/bencode/sumi/mermaid/internal/pie"
	"github.com/bencode/sumi/mermaid/internal/sequence"
	"github.com/bencode/sumi/mermaid/internal/state"
	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

// Options are how a diagram is laid out.
type Options struct {
	// Measurer measures text as the backend that draws the scene will set
	// it. Nil measures it as Arial.
	Measurer scene.Measurer
	// Width is how wide a gantt diagram is drawn, as Mermaid's gantt.useWidth
	// sets it. 0 is 1200, Mermaid's where it has no element to fill. The other
	// diagrams are as wide as what they draw.
	Width float64
}

// An Error is a mistake in a diagram's source, at a line and column of it,
// counting from 1. Syntax that Mermaid accepts and sumi does not yet is an
// Error too, and its IsUnsupported says so.
type Error = syntax.Error

// kinds are the diagrams Layout knows, by the word a diagram starts with,
// as Mermaid's detectors match it (sequenceDetector.ts, flowDetector-v2.ts).
// "flowchart" takes flowchart-elk too, which Mermaid 12 lays out with ELK as well.
var kinds = map[string]func(body, title string, o Options) (*scene.Scene, error){
	"sequenceDiagram": measured(sequence.Layout),
	"graph":           measured(flowchart.Layout),
	"flowchart":       measured(flowchart.Layout),
	"stateDiagram":    measured(state.Layout), // stateDiagram-v2 too, as stateDetector-V2 matches
	"classDiagram":    measured(class.Layout), // classDiagram-v2 too, as classDetector-V2 matches
	"erDiagram":       measured(er.Layout),
	"pie":             measured(pie.Layout),
	"gantt": func(body, title string, o Options) (*scene.Scene, error) {
		return gantt.Layout(body, title, o.Measurer, gantt.Env{Now: time.Now(), Zone: time.Local}, o.Width)
	},
}

// measured is the layout of a diagram that needs of the options only its measurer.
func measured(layout func(body, title string, m scene.Measurer) (*scene.Scene, error)) func(string, string, Options) (*scene.Scene, error) {
	return func(body, title string, o Options) (*scene.Scene, error) { return layout(body, title, o.Measurer) }
}

// Layout lays out the diagram src, picking its kind by its first word.
func Layout(src string, o Options) (*scene.Scene, error) {
	if o.Measurer == nil {
		o.Measurer = scene.Arial
	}
	if o.Width == 0 {
		o.Width = 1200
	}
	body, title := syntax.Prepare(src)
	i := len(body) - len(strings.TrimLeftFunc(body, syntax.IsSpace))
	for word, layout := range kinds {
		if strings.HasPrefix(body[i:], word) {
			return layout(body, title, o)
		}
	}
	at, rest := syntax.LinesOf(body).Pos(i), body[i:]
	if rest == "" {
		return nil, syntax.Errorf(at, "no diagram")
	}
	end := strings.IndexFunc(rest, func(r rune) bool { return syntax.IsSpace(r) || r == ';' })
	if end < 0 {
		end = len(rest)
	}
	return nil, syntax.Unsupported(at, `diagram type "`+rest[:end]+`"`)
}
