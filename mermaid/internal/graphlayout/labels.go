package graphlayout

import (
	"math"
	"slices"
)

// labelGap is what stays between the texts of two opposite edges
// (OPPOSITE_LABEL_MIN_GAP, render.ts).
const labelGap = 4

// separateLabels pushes apart the texts of the two edges between a pair of nodes
// where they overlap. A pair with more edges stays (separateOppositeEdgeLabels,
// render.ts).
func separateLabels(routes []route) []route {
	separate := slices.Clone(routes)
	pairs := map[[2]string][]int{}
	for i, r := range separate {
		if r.label != nil {
			key := [2]string{min(r.start, r.end), max(r.start, r.end)}
			pairs[key] = append(pairs[key], i)
		}
	}
	for _, pair := range pairs {
		if len(pair) != 2 {
			continue
		}
		a, b := *separate[pair[0]].label, *separate[pair[1]].label
		if math.Abs(a.center.Y-b.center.Y) > max(a.size.Height, b.size.Height) {
			continue
		}
		required := a.size.Width/2 + b.size.Width/2 + labelGap
		current := b.center.X - a.center.X
		deficit := required - math.Abs(current)
		if deficit <= 0 {
			continue
		}
		direction := sign(current)
		if direction == 0 {
			direction = 1
		}
		a.center.X -= direction * deficit / 2
		b.center.X += direction * deficit / 2
		separate[pair[0]].label, separate[pair[1]].label = &a, &b
	}
	return separate
}
