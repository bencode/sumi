package flowchart

import (
	"regexp"
	"slices"
	"strings"

	"github.com/bencode/sumi/mermaid/internal/style"
	"github.com/bencode/sumi/scene"
)

// edgeStyling is an edge's styling: its classes' styles, then its own, as its path's
// style attribute has them.
func edgeStyling(e Edge) style.Styling {
	return style.Of(style.Declarations(slices.Concat(e.Compiled, e.Styles)))
}

// markerStroke is what edges.js looks for in a path's style, in just these letters.
var markerStroke = regexp.MustCompile(`stroke:([^;]+)`)

// markerColor is the color an edge's markers take: the first stroke in its path's
// style, or none.
func markerColor(e Edge) *scene.Color {
	m := markerStroke.FindStringSubmatch(strings.Join(slices.Concat(e.Compiled, e.Styles), ";"))
	if m == nil {
		return nil
	}
	var c *scene.Color
	style.ReadColor(&c, strings.TrimSpace(m[1]))
	return c
}
