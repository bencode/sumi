// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import "slices"

// activationThreshold is greedySwitch.activationThreshold's default: above this
// many nodes a graph goes without (Layered.melk line 402).
const activationThreshold = 40

// The steps of one graph, on either side of crossing minimization, which can
// span graphs. A graph's first half ends with its layers built, the second with
// its nodes placed and back in Layerless.
var (
	stepsBeforeOrdering = []func(*Graph){
		toLTR, reverseConstrainedEdges, hideSelfLoops, breakCycles, insertLabelDummies,
		hideSeparateNodes, assignLayers, placeConstrainedNodes, splitLongEdges, assignPortSides,
		invertPorts, sortPortLists,
	}
	stepsAfterOrdering = []func(*Graph){
		restoreLoopPorts, positionPorts, nodeMargins, routeSelfLoops, placeEndLabels,
		switchLabelDummies, sizeLabelDummies, placeNodes, measureLayers, routeEdges,
		joinLongEdges, restoreSelfLoops, removeLabelDummies, applyEndLabels,
		restoreReversed, collectNodes,
	}
)

// Layout places the nodes and edges of in. ELK leaves a graph without nodes alone,
// with no padding around nothing (layoutRecursively, core
// RecursiveGraphLayoutEngine.java line 151).
func Layout(in Input) Result {
	g := importGraph(in)
	if len(in.Nodes) == 0 {
		return Result{}
	}
	layoutGraph(g)
	return export(g)
}

// layoutGraph lays out g and what sits in it, those of its subgraphs that are
// laid out on their own first, so that they have a size as nodes
// (layoutRecursively, core RecursiveGraphLayoutEngine.java line 132).
func layoutGraph(g *Graph) {
	layoutOwnSubgraphs(g)
	if g.Hierarchical {
		layoutHierarchy(g)
	} else {
		layoutFlat(g)
	}
}

// layoutOwnSubgraphs finds the subgraphs below g that a layout of g treats as
// nodes. ELK sets a node's size when it applies the layout, which holds the minimum
// once more, now the right way round (applyParentNodeLayout,
// graph/transform/ElkGraphLayoutTransferrer.java line 357; resizeNode, core
// ElkUtil.java line 257).
func layoutOwnSubgraphs(g *Graph) {
	for _, n := range g.Layerless {
		if n.Nested == nil {
			continue
		}
		if n.Included {
			layoutOwnSubgraphs(n.Nested)
			continue
		}
		layoutGraph(n.Nested)
		size := actualSize(n.Nested)
		n.Size = Vec{max(size.X, n.Nested.MinSize.X), max(size.Y, n.Nested.MinSize.Y)}
	}
}

// layoutHierarchy runs the graphs of one layout side by side: each up to crossing
// minimization, bottom up, then that over them all, then each to the end
// (doCompoundLayout and hierarchicalLayout, ElkLayered.java lines 179 and 214).
// A subgraph ends with its size and ports on its node, for its parent's second
// half. Edges across borders go in pieces, one per graph, until all are done.
func layoutHierarchy(root *Graph) {
	splitCrossEdges(root)
	all := graphsBottomUp(root)
	for _, g := range all {
		run(g, stepsBeforeOrdering)
		if g.Settings.modelOrder {
			sortByModelOrder(g)
		}
	}
	minimizeCrossings(root)
	for _, g := range all {
		run(g, stepsAfterOrdering)
		fitGraph(g)
		if g.Parent != nil {
			g.Parent.Size = actualSize(g)
		}
		portsToNode(g)
		fromLTR(g)
	}
	stitchEdges(root)
}

// layoutFlat lays out g's connected components one by one and puts them side by
// side (doLayout, ElkLayered.java line 135).
func layoutFlat(g *Graph) {
	greedy := len(g.Layerless) < activationThreshold
	all := splitComponents(g)
	for _, part := range all {
		run(part, stepsBeforeOrdering)
		minimizeCrossings(part)
		if greedy {
			greedySwitch(part)
		}
		run(part, stepsAfterOrdering)
		fromLTR(part)
	}
	combineComponents(all, g)
	fitGraph(g)
}

func run(g *Graph, steps []func(*Graph)) {
	for _, step := range steps {
		step(g)
	}
}

// graphsBottomUp gives the graphs of a layout, each after those below it, the
// last found first (collectAllGraphsBottomUp, ElkLayered.java line 308).
func graphsBottomUp(root *Graph) []*Graph {
	collected, pending := []*Graph{root}, []*Graph{root}
	for len(pending) > 0 {
		g := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		for _, n := range nodes(g) {
			if n.Included {
				collected = append(collected, n.Nested)
				pending = append(pending, n.Nested)
			}
		}
	}
	slices.Reverse(collected)
	return collected
}

// fitGraph makes g at least its minimum size (resizeGraph and
// resizeGraphNoReallyIMeanIt, ElkLayered.java lines 675 and 720, or the copy in
// intermediate/HierarchicalNodeResizingProcessor.java line 132). The root has no
// minimum. ELK always adds the padding and takes it off again, which can change
// the last bit of the size. The minimum never turns with the graph, so where
// the graph is turned, as in a hierarchy going down, it holds the other axis.
// ELK also moves east port dummies by what the graph grew, which portsToNode
// overwrites.
func fitGraph(g *Graph) {
	size := actualSize(g)
	p := g.Padding
	g.Size.X = max(size.X, g.MinSize.X) - p.Left - p.Right
	g.Size.Y = max(size.Y, g.MinSize.Y) - p.Top - p.Bottom
}
