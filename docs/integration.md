# Drawing scenes natively

`svg.Render` is one way to draw a diagram. This guide is for the other way: drawing a `scene.Scene` with your own renderer, in a window, an image or a PDF, in your own fonts and theme.

The steps are:

1. Lay the source out with `mermaid.Layout`, measuring text with your renderer's fonts.
2. Draw the scene's items in order: each item's paths, then its lines of text.
3. Settle colors with a `scene.Palette` when you draw, not when you lay out.

[`mermaid/example_test.go`](../mermaid/example_test.go) does all three in one short file, and `go test` runs it.

## Measure text as you draw it

A diagram's boxes are sized to their labels, so text must be measured the way your renderer will set it. Pass a `scene.Measurer` in `mermaid.Options`:

```go
type Measurer interface {
	Measure(s string, f scene.Font) scene.Metrics
}
```

- `Metrics.Width` is the advance width of `s`. Measure it with the same font, size, weight and fallback fonts you draw with; a measurer that disagrees with the renderer gives text that overflows or floats in its box.
- `Metrics.Ascent` and `Descent` are the font's, not the string's: return the same values for every string of one `scene.Font`, the empty string included. sumi asks for them with `""`. If they followed the glyphs of each string, labels of the same font would differ in height.
- A `scene.Font` names no typeface. Map it to yours: `Size` in DIPs, `Bold` to a bold weight, `Italic` to an italic style, and `Mono` to a monospace family.
- A nil measurer measures as Arial does, which suits SVG viewed in a browser, not a native renderer with another font.

Layout calls `Measure` for every label and for every word it wraps, in a few fonts. If measuring is slow in your renderer, cache what stays the same for a font, such as its ascent and descent.

## Draw paths and text

A scene's coordinates are DIPs (one CSS pixel), with the origin at the top left and y pointing down. Its content lies between (0, 0) and (`Width`, `Height`).

Draw `Scene.Items` in order. For each item, draw its `Paths` and then its `Texts`, so labels lie on their shapes.

### Paths

A path is a list of segments:

| `Op` | Points | Draw |
|---|---|---|
| `MoveTo` | `Pts[0]` | start a subpath |
| `LineTo` | `Pts[0]` | a line |
| `QuadTo` | `Pts[0]` control, `Pts[1]` end | a quadratic curve |
| `CubeTo` | `Pts[0]`, `Pts[1]` controls, `Pts[2]` end | a cubic curve |
| `Close` | none | a line back to the subpath's start |

Fill the path, then stroke it. The stroke's ends and corners are round. `Stroke.Dash` lists dash and gap lengths in turn, starting at the path's start; nil is a solid line. Circles and rounded corners are already cubic curves, so a renderer needs no arc command.

### Text

Each `scene.Line` in a `scene.Text` is placed: `X` is where it starts and `Baseline` is its baseline, in the text's `Font`. Lines are already broken and aligned; draw each one as it is, without wrapping or aligning it again.

Many renderers place text by its top. Move such a line up by your font's ascent, the same ascent your measurer returns.

## Colors and themes

A paint is either a theme token (`Paint.Token`) or a color the diagram's author wrote (`Paint.Color`, when the token is `scene.None`). Resolve paints when you draw:

```go
if c, ok := palette.Resolve(path.Fill); ok {
	// fill with c
}
```

`Resolve` returns false when there is nothing to paint: the zero paint, a transparent color, or a token missing from the palette.

- `scene.Light` and `scene.Dark` are sumi's palettes. Pick one when you draw; switching themes needs no new layout. Author colors stay as written in both.
- To match your app's theme, clone a palette (`maps.Clone`) and change the colors you want. The tokens are listed in [`scene/paint.go`](../scene/paint.go).
- `scene.Background` is the canvas under the diagram. Edge labels sit on patches of it, and a sequence diagram writes its message numbers in it. Paint it in your view's background color, so the patches blend into the page.

## Fit the diagram to its place

Most diagrams are as wide as what they draw. To show one in a narrower view, scale it down uniformly: scale = view width / `Scene.Width`, and the height follows. Scaling it up past its size enlarges the text past the font size it was measured in, so cap the scale at 1.

A gantt diagram is the exception: it fills the width you give it in `Options.Width`, as Mermaid fills its element. Pass the width of the place it will be shown in; 0 lays it out 1200 wide. Other diagrams ignore `Width`.

## Lay out once

Layout costs much more than drawing; flowcharts, state, class and ER diagrams run a full graph layout. Lay a diagram out when its source changes and draw the cached scene on every frame. A cache key needs everything the layout depends on:

- the source;
- the fonts the measurer uses;
- `Options.Width`, for gantt diagrams.

A theme change is not on the list. A gantt diagram also reads `time.Now` for its today line and `time.Local` for its dates, so the same source laid out on another day, or in another time zone, can differ.

## Handle errors

`mermaid.Layout` returns a `*mermaid.Error` for a source it cannot read, with the 1-based `Line` and `Col` of the problem:

```go
s, err := mermaid.Layout(src, opts)
var e *mermaid.Error
if errors.As(err, &e) {
	// A problem at e.Line and e.Col. e.IsUnsupported() reports Mermaid
	// that sumi does not draw yet, rather than a mistake in the source.
} else if err != nil {
	// Some other failure; show err.
}
```

Show the user something useful instead of an empty space. For example, show the source as code and put the error under it. A scene can also be empty, such as a sequence diagram with no messages; check that `Width` and `Height` are above 0 before you scale by them.

## A complete integration

[Jun](https://github.com/bencode/jun), a terminal workspace for macOS, draws Mermaid code blocks in its Markdown view with sumi. [`internal/app/render_mermaid.go`](https://github.com/bencode/jun/blob/main/internal/app/render_mermaid.go) covers each part of this guide: a measurer that uses the window's font shaping, paths and text compiled once into paint calls, the background token painted in the window's background, scaling to the page, and a gantt diagram laid out at the reading width.
