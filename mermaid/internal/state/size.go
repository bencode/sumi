package state

import (
	"math"
	"strings"

	"github.com/bencode/sumi/mermaid/internal/label"
	"github.com/bencode/sumi/mermaid/internal/shape"
	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

// What getData and dataFetcher give every node, from Mermaid's default config.
const (
	statePadding = 8   // dataFetcher's padding
	notePadding  = 15  // flowchart.padding, a note's
	wrapWidth    = 120 // state.wrappingWidth
	leastWidth   = 120 // state.minNodeWidth
)

// Sized is a node as insertMeasuredNode leaves it. A group's is left empty.
type Sized struct {
	Shape              shape.Node
	Label              label.Box     // of a Rect or a Note
	Title, Description label.Box     // of a RectWithTitle
	Titled             *shape.Titled // of a RectWithTitle, whose Node is Shape too
}

// Size sizes the nodes of d, in d.Nodes's order.
func Size(d *Diagram, m scene.Measurer) ([]Sized, error) {
	byID := make(map[string]Node, len(d.Nodes))
	for _, n := range d.Nodes {
		byID[n.ID] = n
	}
	out := make([]Sized, len(d.Nodes))
	for i, n := range d.Nodes {
		if strings.Contains(n.ID, "<") { // DOMPurify rewrites a text with one, so a label that is the id stops being it
			return nil, syntax.Unsupported(n.At, "a < in the name of a state")
		}
		if n.Group {
			continue
		}
		s, err := sizeNode(n, flowOf(n, byID, d.Direction), m)
		if err != nil {
			return nil, err
		}
		out[i] = s
	}
	return out, nil
}

// flowOf is resolveNodeDir: the direction of the nearest group around n that has one,
// or the diagram's.
func flowOf(n Node, byID map[string]Node, diagram string) string {
	for at, ok := n, true; ok; at, ok = byID[at.Parent] {
		if at.Dir != "" {
			return at.Dir
		}
	}
	return diagram
}

// sizeNode is the shape insertNode draws n in. Mermaid has no shape for a state only a
// note names, or for a region with nothing in it.
func sizeNode(n Node, flow string, m scene.Measurer) (Sized, error) {
	switch n.Shape {
	case Rect:
		return labelled(n, shape.Rounded, statePadding, m)
	case Note:
		return labelled(n, shape.Note, notePadding, m)
	case RectWithTitle:
		return titled(n, m)
	case Start:
		return bare(shape.StateStart), nil
	case End:
		return bare(shape.StateEnd), nil
	case Choice:
		return bare(shape.Choice), nil
	case Fork, Join:
		if flow == "LR" || flow == "RL" {
			return bare(shape.ForkSide), nil
		}
		return bare(shape.Fork), nil
	case Divider:
		return Sized{}, syntax.Errorf(n.At, "a region (--) with nothing in it cannot be drawn")
	}
	return Sized{}, syntax.Errorf(n.At, "state %q is named only by a note, which leaves it no shape", n.ID)
}

func bare(k shape.Kind) Sized { return Sized{Shape: shape.Set(k, 0, 0, 0)} }

// labelled sets n's label as labelHelper does and k around it.
func labelled(n Node, k shape.Kind, padding float64, m scene.Measurer) (Sized, error) {
	lines, err := label.Read(n.Label, "markdown", n.At)
	if err != nil {
		return Sized{}, err
	}
	least := 0.0
	if n.Label != "" {
		least = leastWidth
	}
	b := label.Set(m, lines, wrapWidth, least)
	return Sized{Shape: shape.Set(k, b.W, b.H, padding), Label: b}, nil
}

// titled sets n's title and descriptions as createLabel does: plain text that never wraps.
func titled(n Node, m scene.Measurer) (Sized, error) {
	set := func(text string) (label.Box, error) {
		lines, err := label.Read(text, "string", n.At)
		if err != nil {
			return label.Box{}, err
		}
		return label.Set(m, lines, math.Inf(1), 0), nil
	}
	title, err := set(n.Label)
	if err != nil {
		return Sized{}, err
	}
	desc, err := set(strings.Join(n.Descriptions, "<br/>"))
	if err != nil {
		return Sized{}, err
	}
	if title.H == 0 || desc.H == 0 { // getBBox leaves out an empty box, and the node's box is then not its rect's
		return Sized{}, syntax.Unsupported(n.At, "an empty title or description beside the other")
	}
	t := shape.RectWithTitle(title.W, title.H, desc.W, desc.H, statePadding)
	return Sized{Shape: t.Node, Title: title, Description: desc, Titled: &t}, nil
}
