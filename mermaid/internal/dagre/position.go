package dagre

import (
	"math"
	"slices"
)

// position gives the nodes without children their centers (position/index.js
// line 7).
func position(g *layoutGraph, nodeSep, rankSep float64) {
	s := asNonCompound(g)
	positionY(s, rankSep)
	for v, x := range positionX(s, nodeSep) {
		s.node(v).x = x
	}
}

// positionY stacks the ranks, each as tall as its tallest node, with each
// node centered in its rank (position/index.js line 16).
func positionY(g *layoutGraph, rankSep float64) {
	prevY := 0.0
	for _, layer := range layerMatrix(g) {
		maxHeight := 0.0
		for _, v := range layer {
			maxHeight = max(maxHeight, g.node(v).height)
		}
		for _, v := range layer {
			g.node(v).y = prevY + maxHeight/2
		}
		prevY += maxHeight + rankSep
	}
}

// positionX is each node's x, by Brandes and Köpf: an alignment for each of
// up and down, left and right, in that order ul, ur, dl, dr, brought
// together on the narrowest and averaged (bk.js line 364). Going down and
// right turn the layering over; going right mirrors x.
func positionX(g *layoutGraph, nodeSep float64) map[string]float64 {
	layering := layerMatrix(g)
	c := conflicts{}
	findType1Conflicts(g, layering, c)
	findType2Conflicts(g, layering, c)
	var xss [4]map[string]float64
	for i := range xss {
		down, right := i >= 2, i%2 == 1
		adjusted, neighbors := slices.Clone(layering), g.predecessors
		if down {
			slices.Reverse(adjusted)
			neighbors = g.successors
		}
		if right {
			for j, layer := range adjusted {
				adjusted[j] = slices.Clone(layer)
				slices.Reverse(adjusted[j])
			}
		}
		root, align := verticalAlignment(adjusted, c, neighbors)
		xs := horizontalCompaction(g, adjusted, root, align, right, nodeSep)
		if right {
			for v, x := range xs {
				xs[v] = -x
			}
		}
		xss[i] = xs
	}
	alignCoordinates(&xss, smallestWidth(g, &xss))
	return balance(&xss)
}

// smallestWidth is which of xss is the narrowest, the first of those that
// are (bk.js line 306).
func smallestWidth(g *layoutGraph, xss *[4]map[string]float64) int {
	best, least := 0, math.Inf(1)
	for i, xs := range xss {
		lo, hi := math.Inf(1), math.Inf(-1)
		for v, x := range xs {
			half := g.node(v).width / 2
			lo, hi = min(lo, x-half), max(hi, x+half)
		}
		if hi-lo < least {
			best, least = i, hi-lo
		}
	}
	return best
}

// alignCoordinates moves the others onto xss[alignTo]: those going left to
// its least x, those going right to its greatest (bk.js line 329).
func alignCoordinates(xss *[4]map[string]float64, alignTo int) {
	lo, hi := extent(xss[alignTo])
	for i, xs := range xss {
		if i == alignTo {
			continue
		}
		l, h := extent(xs)
		delta := lo - l
		if i%2 == 1 {
			delta = hi - h
		}
		for v := range xs {
			xs[v] += delta
		}
	}
}

func extent(xs map[string]float64) (lo, hi float64) {
	lo, hi = math.Inf(1), math.Inf(-1)
	for _, x := range xs {
		lo, hi = min(lo, x), max(hi, x)
	}
	return lo, hi
}

// balance is each node's two middle xs averaged (bk.js line 353).
func balance(xss *[4]map[string]float64) map[string]float64 {
	out := make(map[string]float64, len(xss[0]))
	for v := range xss[0] {
		xs := [4]float64{xss[0][v], xss[1][v], xss[2][v], xss[3][v]}
		slices.Sort(xs[:])
		out[v] = (xs[1] + xs[2]) / 2
	}
	return out
}
