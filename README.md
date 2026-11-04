# sumi

sumi (墨, ink) lays out the diagrams and formulas people write in Markdown, Mermaid and LaTeX math, in pure Go. It stops at a scene: the laid-out picture as data, paths and lines of text at their final places. Your program draws the scene, natively on any backend or as SVG.

> sumi is at 0.x and has only its base so far: the `scene` package and SVG output. Mermaid comes next, LaTeX after it. Its API may change in any 0.x release.

## Example

```go
text, _ := scene.SetText(scene.Arial, "Hello, 墨", scene.Font{Size: 14}, 0, scene.Center, scene.Point{X: 60, Y: 30})
box := scene.Path{
	Segs:   scene.RoundRect(scene.Rect{X: 10, Y: 10, W: 100, H: 40}, 6),
	Fill:   scene.Use(scene.NodeFill),
	Stroke: scene.Stroke{Paint: scene.Use(scene.NodeStroke), Width: 1},
}
s := &scene.Scene{Width: 120, Height: 60, Items: []scene.Item{
	{Kind: scene.Node, ID: "hello", Paths: []scene.Path{box}, Texts: []scene.Text{text}},
}}
out := svg.Render(s, scene.Light)
```

`go run ./internal/preview` writes sample scenes as SVG, in the light and dark palettes, to `out/`.

## License

MIT. See [LICENSE](LICENSE).
