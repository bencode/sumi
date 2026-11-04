// Command preview writes hand-made scenes as SVG, in the light and dark
// palettes, with an index.html showing them side by side: for looking at what
// scene and svg draw.
//
//	go run ./internal/preview [-o out]
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/bencode/sumi/scene"
	"github.com/bencode/sumi/svg"
)

var (
	m    = scene.Arial
	body = scene.Font{Size: 14}
)

func pt(x, y float64) scene.Point { return scene.Point{X: x, Y: y} }

func stroke(t scene.Token, w float64) scene.Stroke {
	return scene.Stroke{Paint: scene.Use(t), Width: w}
}

// node is a rounded box with a label centered in it, wrapped 8 inside its
// sides.
func node(id, parent string, r scene.Rect, label string, fill scene.Paint) scene.Item {
	text, _ := scene.SetText(m, label, body, r.W-16, scene.Center, pt(r.X+r.W/2, r.Y+r.H/2))
	if fill.Token == scene.None {
		text.Paint = scene.Fixed(scene.TextOn(fill.Color))
	}
	return scene.Item{Kind: scene.Node, ID: id, Parent: parent, Box: r,
		Paths: []scene.Path{{Segs: scene.RoundRect(r, 6), Fill: fill, Stroke: stroke(scene.NodeStroke, 1)}},
		Texts: []scene.Text{text}}
}

// edge is a line from a to b with an arrowhead at b, and a label above its
// middle.
func edge(from, to string, a, b scene.Point, dash []float64, label string) scene.Item {
	head, end := scene.Arrowhead(a, b, 9)
	line := scene.Path{Stroke: scene.Stroke{Paint: scene.Use(scene.LineColor), Width: 1.5, Dash: dash}}
	line.MoveTo(a.X, a.Y).LineTo(end.X, end.Y)
	it := scene.Item{Kind: scene.Edge, From: from, To: to,
		Paths: []scene.Path{line, {Segs: head, Fill: scene.Use(scene.LineColor)}}}
	if label != "" {
		text, _ := scene.SetText(m, label, scene.Font{Size: 12}, 0, scene.Center, pt((a.X+b.X)/2, (a.Y+b.Y)/2-12))
		it.Texts = []scene.Text{text}
	}
	return it
}

func shapes() *scene.Scene {
	yellow := scene.Fixed(scene.Color{R: 0xff, G: 0xcc, B: 0x00, A: 255})
	blue := scene.Fixed(scene.Color{R: 0x1f, G: 0x6f, B: 0xeb, A: 255})
	ct, _ := scene.SetText(m, "圆 circle", body, 0, scene.Center, pt(330, 60))
	circle := scene.Item{Kind: scene.Node, ID: "C", Texts: []scene.Text{ct},
		Paths: []scene.Path{{Segs: scene.Ellipse(pt(330, 60), 40, 40), Fill: scene.Use(scene.NodeFill), Stroke: stroke(scene.NodeStroke, 1)}}}
	square := scene.Path{Segs: scene.RoundRect(scene.Rect{X: 420, Y: 30, W: 90, H: 60}, 0),
		Fill: scene.Use(scene.NoteFill), Stroke: stroke(scene.NoteStroke, 1)}
	return &scene.Scene{Width: 540, Height: 230, Items: []scene.Item{
		node("A", "", scene.Rect{X: 20, Y: 30, W: 120, H: 60}, "圆角矩形", scene.Use(scene.NodeFill)),
		edge("A", "C", pt(140, 60), pt(290, 60), nil, "实线"),
		circle,
		{Kind: scene.Mark, Paths: []scene.Path{square}},
		edge("A", "Y", pt(80, 90), pt(80, 150), []float64{5, 4}, ""),
		node("Y", "", scene.Rect{X: 20, Y: 150, W: 120, H: 50}, "作者底色（黄）", yellow),
		node("B", "", scene.Rect{X: 270, Y: 150, W: 120, H: 50}, "作者底色（蓝）", blue),
		edge("Y", "B", pt(140, 175), pt(270, 175), []float64{5, 4}, "虚线 dashed"),
	}}
}

// text shows each alignment against a guide at the point it is set at, and
// lines broken at a width and at <br>.
func text() *scene.Scene {
	var items []scene.Item
	guide := func(x float64) scene.Item {
		p := scene.Path{Stroke: scene.Stroke{Paint: scene.Use(scene.GroupStroke), Width: 1, Dash: []float64{2, 2}}}
		p.MoveTo(x, 10).LineTo(x, 110)
		return scene.Item{Kind: scene.Mark, Paths: []scene.Path{p}}
	}
	label := func(s string, f scene.Font, width float64, a scene.Align, at scene.Point) scene.Item {
		t, box := scene.SetText(m, s, f, width, a, at)
		frame := scene.Path{Segs: scene.RoundRect(box, 0), Stroke: stroke(scene.NoteStroke, 0.5)}
		return scene.Item{Kind: scene.Label, Box: box, Paths: []scene.Path{frame}, Texts: []scene.Text{t}}
	}
	for i, a := range []scene.Align{scene.Left, scene.Center, scene.Right} {
		x := 110 + float64(i)*200
		items = append(items, guide(x), label("左 Left<br>居中 Center<br>右 Right", body, 0, a, pt(x, 60)))
	}
	items = append(items,
		label("Mermaid 的标签超过最大宽度时自动换行：英文在空格处断开，中文可以在任意两个字之间断开，标点不放在行首。", body, 220, scene.Left, pt(20, 190)),
		label("Bold 粗体 and a very long English sentence that wraps at spaces only.", scene.Font{Size: 14, Bold: true}, 200, scene.Center, pt(400, 190)),
		label("mono: go run ./internal/preview", scene.Font{Size: 13, Mono: true}, 0, scene.Left, pt(20, 270)),
	)
	return &scene.Scene{Width: 620, Height: 300, Items: items}
}

// diagram is a small flowchart laid out by hand: a group with two nodes, a
// node outside it, and a note.
func diagram() *scene.Scene {
	g := scene.Rect{X: 20, Y: 20, W: 380, H: 120}
	gl, _ := scene.SetText(m, "子图 G", scene.Font{Size: 12, Bold: true}, 0, scene.Left, pt(g.X+10, g.Y+14))
	note := node("N", "", scene.Rect{X: 430, Y: 50, W: 150, H: 60}, "注释：节点压在连线上面", scene.Use(scene.NoteFill))
	note.Paths[0].Stroke = stroke(scene.NoteStroke, 1)
	return &scene.Scene{Width: 600, Height: 260, Items: []scene.Item{
		{Kind: scene.Group, ID: "G", Box: g, Texts: []scene.Text{gl},
			Paths: []scene.Path{{Segs: scene.RoundRect(g, 8), Fill: scene.Use(scene.GroupFill), Stroke: stroke(scene.GroupStroke, 1)}}},
		edge("A", "B", pt(160, 80), pt(240, 80), nil, "下一步"),
		edge("B", "C", pt(310, 110), pt(310, 190), []float64{5, 4}, ""),
		node("A", "G", scene.Rect{X: 40, Y: 50, W: 120, H: 60}, "开始", scene.Use(scene.NodeFill)),
		node("B", "G", scene.Rect{X: 240, Y: 50, W: 140, H: 60}, "处理数据 Process", scene.Use(scene.NodeFill)),
		node("C", "", scene.Rect{X: 250, Y: 190, W: 120, H: 50}, "结束", scene.Use(scene.NodeFill)),
		note,
	}}
}

func main() {
	out := flag.String("o", "out", "directory to write into")
	flag.Parse()
	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}
	scenes := []struct {
		name string
		s    *scene.Scene
	}{{"shapes", shapes()}, {"text", text()}, {"diagram", diagram()}}
	var rows strings.Builder
	for _, sc := range scenes {
		rows.WriteString("<tr>")
		for _, th := range []struct {
			name string
			p    scene.Palette
			bg   string
		}{{"light", scene.Light, "#ffffff"}, {"dark", scene.Dark, "#0d1117"}} {
			file := sc.name + "-" + th.name + ".svg"
			if err := os.WriteFile(filepath.Join(*out, file), svg.Render(sc.s, th.p), 0o644); err != nil {
				log.Fatal(err)
			}
			fmt.Fprintf(&rows, `<td style="background:%s"><img src="%s"></td>`, th.bg, file)
		}
		rows.WriteString("</tr>\n")
	}
	page := "<!doctype html><meta charset=utf-8><title>sumi preview</title>\n" +
		"<style>body{margin:16px;font:14px sans-serif}td{padding:16px;vertical-align:top}</style>\n" +
		"<table>\n" + rows.String() + "</table>\n"
	if err := os.WriteFile(filepath.Join(*out, "index.html"), []byte(page), 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Println("wrote", filepath.Join(*out, "index.html"))
}
