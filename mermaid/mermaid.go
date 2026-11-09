// Package mermaid lays out diagrams written in Mermaid's syntax as scenes,
// drawing what Mermaid v12.3.0 draws. So far it knows sequence diagrams, flowcharts
// and state diagrams. A flowchart's or a state diagram's styles draw their fills,
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

	"github.com/bencode/sumi/mermaid/internal/flowchart"
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
}

// An Error is a mistake in a diagram's source, at a line and column of it,
// counting from 1. Syntax that Mermaid accepts and sumi does not yet is an
// Error too, and its IsUnsupported says so.
type Error = syntax.Error

// kinds are the diagrams Layout knows, by the word a diagram starts with,
// as Mermaid's detectors match it (sequenceDetector.ts, flowDetector-v2.ts).
// "flowchart" takes flowchart-elk too, which Mermaid 12 lays out with ELK as well.
var kinds = map[string]func(body, title string, m scene.Measurer) (*scene.Scene, error){
	"sequenceDiagram": sequence.Layout,
	"graph":           flowchart.Layout,
	"flowchart":       flowchart.Layout,
	"stateDiagram":    state.Layout, // stateDiagram-v2 too, as stateDetector-V2 matches
}

// Layout lays out the diagram src, picking its kind by its first word.
func Layout(src string, o Options) (*scene.Scene, error) {
	m := o.Measurer
	if m == nil {
		m = scene.Arial
	}
	body, title := syntax.Prepare(src)
	i := len(body) - len(strings.TrimLeftFunc(body, syntax.IsSpace))
	for word, layout := range kinds {
		if strings.HasPrefix(body[i:], word) {
			return layout(body, title, m)
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
