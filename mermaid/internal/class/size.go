package class

import (
	"math"
	"regexp"
	"strings"

	"github.com/bencode/sumi/mermaid/internal/label"
	"github.com/bencode/sumi/mermaid/internal/shape"
	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

// What getData and labelHelper give a note or an interface, from Mermaid's default config.
// A note does not wrap, as getData's white-space: nowrap outweighs addHtmlSpan's.
const (
	notePadding   = 6   // getData's, as class.padding is unset
	interfaceWrap = 120 // flowchart.wrappingWidth
)

// paragraphs finds the blank line of a title in two paragraphs, which addText counts as
// one line for raising it.
var paragraphs = regexp.MustCompile(`\n[ \t]*\n`)

// Placed is a text in a class box.
type Placed struct {
	Box label.Box
	At  scene.Point // the box's top left, from the node's center
}

// A ClassBox is what classBox draws inside its outline.
type ClassBox struct {
	Annotation       *Placed // nil without one
	Title            Placed
	Members, Methods []Placed
	Dividers         [2]float64 // the y of the two lines, from the node's center
}

// Sized is a node as insertMeasuredNode leaves it. A namespace's is left empty.
type Sized struct {
	Shape shape.Node
	Label label.Box // of a note or an interface
	Class *ClassBox // of a class
}

// Size sizes the nodes of d, in d.Nodes's order.
func Size(d *Diagram, m scene.Measurer) ([]Sized, error) {
	out := make([]Sized, len(d.Nodes))
	for i, n := range d.Nodes {
		if n.Kind == Namespace {
			continue
		}
		s, err := sizeNode(n, m)
		if err != nil {
			return nil, err
		}
		out[i] = s
	}
	return out, nil
}

func sizeNode(n Node, m scene.Measurer) (Sized, error) {
	switch n.Kind {
	case Note:
		return labelled(n, "markdown", shape.Note, notePadding, math.Inf(1), m)
	case Interface:
		return labelled(n, "string", shape.Rect, 0, interfaceWrap, m)
	}
	return sizeClass(n, m)
}

// labelled sets n's label as labelHelper does and k around it.
func labelled(n Node, kind string, k shape.Kind, padding, wrap float64, m scene.Measurer) (Sized, error) {
	lines, err := label.Read(n.Label, kind, n.At)
	if err != nil {
		return Sized{}, err
	}
	b := label.Set(m, lines, wrap, 0)
	return Sized{Shape: shape.Set(k, b.W, b.H, padding), Label: b}, nil
}

func sizeClass(n Node, m scene.Measurer) (Sized, error) {
	var annotation []piece
	if len(n.Annotations) > 0 {
		p, err := set(m, "«"+n.Annotations[0]+"»", n.At, plain)
		if err != nil {
			return Sized{}, err
		}
		annotation = []piece{p}
	}
	text, err := titleText(n)
	if err != nil {
		return Sized{}, err
	}
	title, err := set(m, text, n.At, bold)
	if err != nil {
		return Sized{}, err
	}
	members, err := setMembers(m, n.Members, n.At)
	if err != nil {
		return Sized{}, err
	}
	methods, err := setMembers(m, n.Methods, n.At)
	if err != nil {
		return Sized{}, err
	}
	box, w, h := layOut(annotation, title, members, methods)
	return Sized{Shape: shape.Box(w, h), Class: &box}, nil
}

// titleText is a class's text as a browser gets it. Its only < comes from setClassLabel,
// which adds the type to the label as it is, and DOMPurify takes that for a tag and drops
// it. A type that cannot start a tag stays in the text as DOMPurify writes it.
func titleText(n Node) (string, error) {
	if paragraphs.MatchString(n.Label) {
		return "", syntax.Unsupported(n.At, "a blank line in the label of a class")
	}
	name, typ, generic := strings.Cut(n.Label, "<")
	if !generic {
		return n.Label, nil
	}
	if c := typ[0] | 0x20; c < 'a' || c > 'z' { // typ ends in >, so it is not empty
		return "", syntax.Unsupported(n.At, "a generic type that does not start with a letter, beside a label")
	}
	return name, nil
}

func plain(s label.Span) label.Span { return s }

func bold(s label.Span) label.Span { s.Bold = true; return s }

func italic(s label.Span) label.Span { s.Italic = true; return s }

// set reads text as addText does and sets it. It wraps at an estimate made at 16px plus
// 50, which text at 14px, even bold, does not reach.
func set(m scene.Measurer, text string, at syntax.Pos, style func(label.Span) label.Span) (piece, error) {
	lines, err := label.Read(text, "markdown", at)
	if err != nil {
		return piece{}, err
	}
	return styled(m, lines, style), nil
}

func styled(m scene.Measurer, lines [][]label.Span, style func(label.Span) label.Span) piece {
	out := make([][]label.Span, len(lines))
	for i, l := range lines {
		out[i] = make([]label.Span, len(l))
		for j, s := range l {
			out[i][j] = style(s)
		}
	}
	return piece{label.Set(m, out, math.Inf(1), 0), max(len(lines), 1)}
}

func setMembers(m scene.Measurer, ms []Member, at syntax.Pos) ([]piece, error) {
	out := make([]piece, len(ms))
	for i, mem := range ms {
		lines, err := memberLines(mem, at)
		if err != nil {
			return nil, err
		}
		style := plain
		if mem.Classifier == "*" {
			style = italic
		}
		out[i] = styled(m, lines, style)
	}
	return out, nil
}

// memberLines reads a member's text. Its visibility is an escaped character, plain text
// to Markdown, so it stays out of what label reads: a [ after it is not Markdown beside
// an escape.
func memberLines(mem Member, at syntax.Pos) ([][]label.Span, error) {
	escaped := `\` + mem.Visibility
	if mem.Visibility == "" || !strings.HasPrefix(mem.Text, escaped) {
		return label.Read(mem.Text, "markdown", at)
	}
	lines, err := label.Read(strings.TrimPrefix(mem.Text, escaped), "markdown", at)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		lines = [][]label.Span{nil}
	}
	if first := lines[0]; len(first) > 0 && !first[0].Bold && !first[0].Italic {
		first[0].Text = mem.Visibility + first[0].Text
		return lines, nil
	}
	lines[0] = append([]label.Span{{Text: mem.Visibility}}, lines[0]...)
	return lines, nil
}
