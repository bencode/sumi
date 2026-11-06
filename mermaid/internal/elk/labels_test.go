// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// fullLayout lays spec out in a direction, on nodes of the sizes routed gives
// them. spec lists edges like "a>b", with "=30x10" after one that has a label.
func fullLayout(d Direction, spec string) Result {
	widths := []float64{30, 52, 18, 70, 41}
	heights := []float64{45, 30, 5, 62, 14}
	in := Input{Direction: d}
	seen := map[string]bool{}
	addNode := func(id string) {
		if !seen[id] {
			seen[id] = true
			i := int(id[0]) % len(heights)
			in.Nodes = append(in.Nodes, InputNode{id, widths[i], heights[i]})
		}
	}
	for i, part := range strings.Fields(spec) {
		ends, size, hasLabel := strings.Cut(part, "=")
		source, target, _ := strings.Cut(ends, ">")
		addNode(source)
		addNode(target)
		e := InputEdge{ID: strconv.Itoa(i), Source: source, Target: target}
		if hasLabel {
			width, height, _ := strings.Cut(size, "x")
			w, _ := strconv.ParseFloat(width, 64)
			h, _ := strconv.ParseFloat(height, 64)
			e.Label = &Size{w, h}
		}
		in.Edges = append(in.Edges, e)
	}
	g := importGraph(in)
	toLTR(g)
	for _, step := range []func(*Graph){
		hideSelfLoops, breakCycles, insertLabelDummies, assignLayers, splitLongEdges,
		assignPortSides, sortPortLists, sortByModelOrder, minimizeCrossings, restoreLoopPorts,
		positionPorts, nodeMargins, routeSelfLoops, switchLabelDummies, sizeLabelDummies,
		placeNodes, measureLayers, routeEdges, joinLongEdges, restoreSelfLoops,
		removeLabelDummies, restoreReversed, collectNodes,
	} {
		step(g)
	}
	fromLTR(g)
	return export(g)
}

// labelDrawing is drawing with each edge's labels at the end, as x,y.
func labelDrawing(r Result) string {
	var labels []string
	for _, e := range r.Edges {
		for _, l := range e.Labels {
			labels = append(labels, fmt.Sprintf("%v,%v", l.X, l.Y))
		}
	}
	return drawing(r) + " @ " + strings.Join(labels, " ")
}

// These are the smallest graphs found to depend on a rule of labels and self
// loops. The drawings come from elkjs 0.12.1. The long fractions are sizes where
// the order of a sum shows in the last bit.
func TestLabelsAndLoops(t *testing.T) {
	for _, c := range []struct {
		direction Direction
		spec      string
		want      string
	}{
		{Down, "a>b=130x7",
			"154x178 77,17 77,104 @ 12,57"},
		{Right, "a>b=8x0",
			"200x86 30,43 118,43 @ 70,42.5"},
		{Up, "a>b=0.1x0.3",
			"94x171.3 47,154.3 47,74.00000000000001 @ 46.95,114"},
		{Right, "a>b=0.1x0.3",
			"192.1x86 30,43 110.1,43 @ 70,42"},
		{Up, "a>b=8x0",
			"94x171 47,154 47,74 @ 43,114"},
		{Right, "a>b a>c c>b",
			"253x86 30,45.5 70,45.5 70,27.5 110.5,27.5 151,27.5 151,36 171,36 | 30,40.5 50,40.5 50,58.5 90,58.5 | 131,58.5 151,58.5 151,50 171,50 @ "},
		{Right, "a>b b>c a>d d>e=0.1x0.3 e>b c>e=30x10",
			"415x134 30,71 70,71 70,43 90,43 | 160,36 180,36 180,30.25 200,30.25 | 30,66 50,66 50,99.5 125,99.5 205.5,99.5 | 235.5,99.5 331,99.5 331,58.25 351,58.25 | 351,55.75 296,55.75 220.5,55.75 180,55.75 180,50 160,50 | 241,30.25 331,30.25 331,53.25 351,53.25 @ 295.95,98.5 281,24.75"},
		{Down, "a>b=8x0 a>c c>b=30x10",
			"113x255 48.58333333333333,17 48.58333333333333,57 15.999999999999998,57 15.999999999999998,136 15.999999999999998,161 39.916666666666664,161 39.916666666666664,181 | 47.916666666666664,17 47.916666666666664,37 80.5,37 80.5,77 | 80.5,91 80.5,161 56.58333333333333,161 56.58333333333333,181 @ 11.999999999999998,84 65.5,131"},
		{Left, "a>b=30x0.0007696177887156945 a>c",
			"253x109.00076961778872 223,70.50038480889435 183,70.50038480889435 183,43 82,43 | 223,65.50038480889435 203,65.50038480889435 203,93.00076961778872 163,93.00076961778872 @ 127.5,42"},
		{Up, "a>a",
			"62x30.333333333333332 32,12 12,12 12,18.333333333333332 32,18.333333333333332 @ "},
		{Up, "a>a a>a",
			"82x35.400000000000006 52,15.800000000000004 32,15.800000000000004 32,19.600000000000005 52,19.600000000000005 | 52,12.000000000000004 12,12.000000000000004 12,23.400000000000006 52,23.400000000000006 @ "},
		{Up, "a>b a>a",
			"94x132.33333333333334 47,114.66666666666669 47,74 | 37,114.00000000000001 17,114.00000000000001 17,120.33333333333334 37,120.33333333333334 @ "},
		{Up, "a>b a>a a>a a>a",
			"129x139.57142857142858 82,118.28571428571429 82,74 | 72,119.42857142857144 52,119.42857142857144 52,122.14285714285715 72,122.14285714285715 | 72,116.71428571428572 32,116.71428571428572 32,124.85714285714286 72,124.85714285714286 | 72,114 12,114 12,127.57142857142858 72,127.57142857142858 @ "},
		{Left, "a>b a>c a>b a>a",
			"172x140 142,53.75 102,53.75 102,36 82,36 | 142,46.25 122,46.25 122,124 82,124 | 142,50 82,50 | 150,47.5 150,26.25 152,26.25 152,47.5 @ "},
		{Down, "a>a=8x0",
			"74x30.333333333333332 44,18.333333333333332 24,18.333333333333332 24,12 44,12 @ 12,15.166666666666664"},
		{Down, "a>a=130x7",
			"196x31 166,18.666666666666668 146,18.666666666666668 146,12.333333333333336 166,12.333333333333336 @ 12,12"},
		{Down, "a>a=8x0 a>a",
			"94x35.400000000000006 64,19.6 44,19.6 44,15.8 64,15.8 | 64,23.400000000000002 12,23.400000000000002 12,12 64,12 @ 32,17.700000000000003"},
		{Left, "a>a=8x0 a>a",
			"42x69 20.4,52 20.4,32 21.599999999999998,32 21.599999999999998,52 | 19.2,52 19.2,12 22.799999999999997,12 22.799999999999997,52 @ 17,28"},
		{Left, "a>a=44.3x31.3 a>a=17.7x13.1",
			"68.3x121.4 33.55,104.4 33.55,84.4 34.75,84.4 34.75,104.4 | 32.349999999999994,104.4 32.349999999999994,29.10000000000001 35.949999999999996,29.10000000000001 35.949999999999996,104.4 @ 12,49.10000000000001 25.299999999999997,12"},
		{Up, "a>a=18.35750407136546x10 a>a=8.02316900341696x10",
			"116.38067307478242x35.400000000000006 86.38067307478242,15.800000000000004 66.38067307478242,15.800000000000004 66.38067307478242,19.600000000000005 86.38067307478242,19.600000000000005 | 86.38067307478242,12.000000000000004 24.02316900341696,12.000000000000004 24.02316900341696,23.400000000000006 86.38067307478242,23.400000000000006 @ 44.02316900341696,12.700000000000003 11.999999999999986,12.700000000000003"},
		{Right, "a>a=30x17.808577802589333 a>a=30x36.8222147304997",
			"54x131.63079253308905 27.6,114.63079253308904 27.6,94.63079253308904 26.400000000000002,94.63079253308904 26.400000000000002,114.63079253308904 | 28.8,114.63079253308904 28.8,52.8222147304997 25.200000000000003,52.8222147304997 25.200000000000003,114.63079253308904 @ 12,72.8222147304997 12,12"},
	} {
		if got := labelDrawing(fullLayout(c.direction, c.spec)); got != c.want {
			t.Errorf("%s: got  %s\nwant %s", c.spec, got, c.want)
		}
	}
}
