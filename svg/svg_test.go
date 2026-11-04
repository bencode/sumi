package svg

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/bencode/sumi/scene"
)

type doc struct {
	Width  string  `xml:"width,attr"`
	Height string  `xml:"height,attr"`
	Groups []group `xml:"g"`
}

type group struct {
	Class  string   `xml:"class,attr"`
	ID     string   `xml:"id,attr"`
	Parent string   `xml:"data-parent,attr"`
	From   string   `xml:"data-from,attr"`
	To     string   `xml:"data-to,attr"`
	Paths  []pathEl `xml:"path"`
	Texts  []textEl `xml:"text"`
}

type pathEl struct {
	D      string `xml:"d,attr"`
	Fill   string `xml:"fill,attr"`
	Stroke string `xml:"stroke,attr"`
	Dash   string `xml:"stroke-dasharray,attr"`
}

type textEl struct {
	X    string `xml:"x,attr"`
	Y    string `xml:"y,attr"`
	Fill string `xml:"fill,attr"`
	S    string `xml:",chardata"`
}

func sample() (*scene.Scene, scene.Text) {
	f := scene.Font{Size: 14}
	box := func(r scene.Rect, fill, stroke scene.Token) scene.Path {
		return scene.Path{Segs: scene.RoundRect(r, 4), Fill: scene.Use(fill),
			Stroke: scene.Stroke{Paint: scene.Use(stroke), Width: 1}}
	}
	nodeText, _ := scene.SetText(scene.Arial, "节点 A", f, 0, scene.Center, scene.Point{X: 60, Y: 40})
	head, end := scene.Arrowhead(scene.Point{X: 100, Y: 40}, scene.Point{X: 160, Y: 40}, 8)
	line := scene.Path{Stroke: scene.Stroke{Paint: scene.Use(scene.LineColor), Width: 1.5, Dash: []float64{4, 3}}}
	line.MoveTo(100, 40).LineTo(end.X, end.Y)
	label, _ := scene.SetText(scene.Arial, "第一行<br>第二行", f, 0, scene.Left, scene.Point{X: 10, Y: 100})
	s := &scene.Scene{Width: 200, Height: 120, Items: []scene.Item{
		{Kind: scene.Group, ID: "G", Paths: []scene.Path{box(scene.Rect{X: 5, Y: 5, W: 110, H: 70}, scene.GroupFill, scene.GroupStroke)}},
		{Kind: scene.Edge, From: "A", To: "B", Paths: []scene.Path{line, {Segs: head, Fill: scene.Use(scene.LineColor)}}},
		{Kind: scene.Node, ID: "A", Parent: "G", Paths: []scene.Path{box(scene.Rect{X: 20, Y: 20, W: 80, H: 40}, scene.NodeFill, scene.NodeStroke)},
			Texts: []scene.Text{nodeText}},
		{Kind: scene.Label, Texts: []scene.Text{label}},
	}}
	return s, label
}

func hex(c scene.Color) string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }

var commands = regexp.MustCompile(`[MLQCZ]`)

func TestRender(t *testing.T) {
	s, label := sample()
	var d doc
	if err := xml.Unmarshal(Render(s, scene.Light), &d); err != nil {
		t.Fatalf("output is not XML: %v", err)
	}
	if d.Width != "200" || d.Height != "120" {
		t.Errorf("size = %s×%s, want 200×120", d.Width, d.Height)
	}
	classes := make([]string, len(d.Groups))
	for i, g := range d.Groups {
		classes[i] = g.Class
	}
	if want := []string{"group", "edge", "node", "label"}; !slices.Equal(classes, want) {
		t.Fatalf("groups = %v, want %v in drawing order", classes, want)
	}
	group, edge, node, lbl := d.Groups[0], d.Groups[1], d.Groups[2], d.Groups[3]

	if node.ID != "A" || node.Parent != "G" || edge.From != "A" || edge.To != "B" || group.ID != "G" {
		t.Errorf("ids: node %q in %q, edge %q→%q, group %q", node.ID, node.Parent, edge.From, edge.To, group.ID)
	}
	if got, want := node.Paths[0].Fill, hex(scene.Light[scene.NodeFill]); got != want {
		t.Errorf("node fill = %s, want the palette's %s", got, want)
	}
	if len(edge.Paths) != 2 {
		t.Fatalf("edge has %d paths, want a line and a head", len(edge.Paths))
	}
	if l := edge.Paths[0]; l.Dash != "4 3" || l.Fill != "none" || l.Stroke != hex(scene.Light[scene.LineColor]) {
		t.Errorf("edge line: dash %q, fill %q, stroke %q", l.Dash, l.Fill, l.Stroke)
	}
	if got := strings.Join(commands.FindAllString(edge.Paths[1].D, -1), ""); got != "MLLZ" {
		t.Errorf("arrowhead commands = %s, want MLLZ", got)
	}
	if len(lbl.Texts) != 2 {
		t.Fatalf("label has %d lines, want 2", len(lbl.Texts))
	}
	for i, txt := range lbl.Texts {
		l := label.Lines[i]
		if txt.S != l.S || txt.X != num(l.X) || txt.Y != num(l.Baseline) {
			t.Errorf("line %d = %q at (%s, %s), want %q at (%s, %s)", i, txt.S, txt.X, txt.Y, l.S, num(l.X), num(l.Baseline))
		}
	}
}
