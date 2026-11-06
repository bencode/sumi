// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import (
	"cmp"
	"slices"
)

// crossEdge is an edge across subgraph borders. ELK lays out edges within one
// graph only, so such an edge is cut at each border into segments, one per graph,
// and comes back together when all graphs are done (CrossHierarchyEdge and
// InternalProperties.CROSS_HIERARCHY_MAP).
type crossEdge struct {
	edge           *Edge
	source, target *Port
	segments       []crossSegment
	placed         PlacedEdge
}

// crossSegment is the part of a crossEdge in one graph
// (compound/CrossHierarchyEdge.java). output tells that it runs out of the
// subgraphs, towards the edge's top, and not back into them.
type crossSegment struct {
	edge   *Edge
	graph  *Graph
	output bool
}

// externalPort is a place where an edge crosses a border (ExternalPort,
// compound/CompoundGraphPreprocessor.java line 1005). port sits on the subgraph,
// and a dummy node in its graph stands for it. A port holds one edge, since Mermaid
// sets mergeEdges to false, so ELK's merging of external ports never happens.
type externalPort struct {
	edge     *Edge
	port     *Port
	dummy    *Node
	output   bool
	exported bool // the edge goes on, past the subgraph
}

// splitter cuts the edges of one layout (compound/CompoundGraphPreprocessor.java).
type splitter struct {
	crossing map[*Edge]*crossEdge
	found    []*crossEdge
	dummies  map[*Port]*Node // dummyNodeMap
}

// splitCrossEdges cuts every edge of root's layout that crosses a border, and leaves
// each subgraph with its ports on fixed sides (process and
// setSidesOfPortsToSidesOfDummyNodes, CompoundGraphPreprocessor.java lines 103 and
// 128). A cross edge leaves its ports and is kept in root.Crossing.
func splitCrossEdges(root *Graph) {
	s := &splitter{crossing: map[*Edge]*crossEdge{}, dummies: map[*Port]*Node{}}
	s.transform(root, nil)
	for _, c := range s.found {
		slices.SortStableFunc(c.segments, compareSegments)
		moveLabels(c)
		setSource(c.edge, nil)
		setTarget(c.edge, nil)
	}
	for p, dummy := range s.dummies {
		p.Side = dummy.ExtSide
		p.Owner.FixedPorts = true
	}
	root.Crossing = s.found
}

// compareSegments puts segments in order from the edge's source to its target:
// out of the subgraphs from deep to shallow, then back in
// (compound/CrossHierarchyEdgeComparator.java line 43).
func compareSegments(a, b crossSegment) int {
	switch {
	case a.output && !b.output:
		return -1
	case !a.output && b.output:
		return 1
	case a.output:
		return cmp.Compare(depth(b.graph), depth(a.graph))
	}
	return cmp.Compare(depth(a.graph), depth(b.graph))
}

func depth(g *Graph) int {
	level := 0
	for ; included(g); g = g.Parent.Graph {
		level++
	}
	return level
}

// moveLabels gives c's labels to its shallowest segment
// (moveLabelsAndRemoveOriginalEdges, CompoundGraphPreprocessor.java line 249).
func moveLabels(c *crossEdge) {
	if len(c.edge.Labels) > 0 {
		segment := c.segments[shallowestSegment(c.segments)].edge
		segment.Labels, c.edge.Labels = c.edge.Labels, nil
	}
}

// shallowestSegment gives the index of the last segment running out of the
// subgraphs, or else the first (getShallowestEdgeSegment,
// CompoundGraphPreprocessor.java line 315).
func shallowestSegment(all []crossSegment) int {
	i := slices.IndexFunc(all, func(s crossSegment) bool { return !s.output })
	switch i {
	case -1:
		return len(all) - 1
	case 0:
		return 0
	}
	return i - 1
}

// transform cuts the edges at g's borders, those of the graphs below it first,
// and gives the external ports that lead on above parent (transformHierarchyEdges,
// CompoundGraphPreprocessor.java line 154). Once a subgraph has external ports, all
// its ports need dummy nodes, for that is how they get their positions, even those
// with edges only outside.
func (s *splitter) transform(g *Graph, parent *Node) []*externalPort {
	var contained []*externalPort
	for _, n := range g.Layerless {
		if !n.Included {
			continue
		}
		contained = append(contained, s.transform(n.Nested, n)...)
		if n.Nested.External {
			s.addMissingDummies(n)
		}
	}
	exported := s.inner(g, parent, contained)
	if parent != nil {
		exported = append(exported, s.outer(g, parent)...)
	}
	return exported
}

func (s *splitter) addMissingDummies(n *Node) {
	for _, p := range n.Ports {
		if s.dummies[p] == nil {
			flow := len(p.Out) - len(p.In)
			dummy := newExternalDummy(n.Nested, flow, 0)
			s.link(dummy, p)
			n.Nested.Layerless = append(n.Nested.Layerless, dummy)
		}
	}
}

func (s *splitter) link(dummy *Node, p *Port) {
	dummy.Origin = p
	s.dummies[p] = dummy
}

// inner goes on with the edges coming out of g's subgraphs at the external ports
// contained: to a child of g, to another subgraph of g, or on past parent
// (processInnerHierarchicalEdgeSegments, CompoundGraphPreprocessor.java line 356).
func (s *splitter) inner(g *Graph, parent *Node, contained []*externalPort) []*externalPort {
	var exported []*externalPort
	for _, ext := range contained {
		far := ext.edge.Source
		if ext.output {
			far = ext.edge.Target
		}
		switch {
		case far.Owner.Graph == g:
			s.connectChild(g, ext, far)
		case parent == nil || isDescendant(far.Owner, parent):
			if ext.output {
				s.connectSiblings(g, ext, contained)
			}
		default:
			exported = s.keep(g, exported, s.introduce(g, parent, ext.edge, ext.port, ext.output))
		}
	}
	return exported
}

// connectChild joins ext to child, the port at the edge's end in g (line 474).
func (s *splitter) connectChild(g *Graph, ext *externalPort, child *Port) {
	if ext.output {
		s.addSegment(g, ext.edge, ext.port, child, true)
	} else {
		s.addSegment(g, ext.edge, child, ext.port, false)
	}
}

// connectSiblings joins out to the other external port of its edge (line 502).
func (s *splitter) connectSiblings(g *Graph, out *externalPort, contained []*externalPort) {
	in := contained[slices.IndexFunc(contained, func(ext *externalPort) bool {
		return ext != out && ext.edge == out.edge
	})]
	s.addSegment(g, out.edge, out.port, in.port, true)
}

// outer cuts the edges of g's children that leave parent (processOuterHierarchicalEdgeSegments,
// CompoundGraphPreprocessor.java line 541). What inner added runs within g, and
// the original edges stay on their ports until all is cut.
func (s *splitter) outer(g *Graph, parent *Node) []*externalPort {
	var exported, created []*externalPort
	for _, n := range g.Layerless {
		for _, p := range n.Ports {
			for _, e := range slices.Clone(p.Out) {
				if !isDescendant(e.Target.Owner, parent) {
					created = append(created, s.introduce(g, parent, e, e.Source, true))
				}
			}
			for _, e := range slices.Clone(p.In) {
				if !isDescendant(e.Source.Owner, parent) {
					created = append(created, s.introduce(g, parent, e, e.Target, false))
				}
			}
		}
	}
	for _, ext := range created {
		exported = s.keep(g, exported, ext)
	}
	return exported
}

// keep adds ext's dummy node to g and lists ext if it is exported.
func (s *splitter) keep(g *Graph, exported []*externalPort, ext *externalPort) []*externalPort {
	g.Layerless = append(g.Layerless, ext.dummy)
	if ext.exported {
		return append(exported, ext)
	}
	return exported
}

// introduce cuts e at g's border: a dummy node for the place it crosses, and a
// segment from there to opposite. The edge either ends at parent, on a port it
// already has, or goes on, on a new port of parent (introduceHierarchicalEdgeSegment
// and createExternalPortDummy, CompoundGraphPreprocessor.java lines 731 and 889).
func (s *splitter) introduce(g *Graph, parent *Node, e *Edge, opposite *Port, output bool) *externalPort {
	outside, inside := e.Source, e.Target
	if output {
		outside, inside = e.Target, e.Source
	}
	endsAtParent := outside.Owner == parent
	var dummy *Node
	if endsAtParent {
		dummy = newExternalDummy(g, parentFlow(inside, g, output), 0)
		s.link(dummy, outside)
	} else {
		flow := -1
		if output {
			flow = 1
		}
		// ELK's own comment here reads "FIXME No idea why ..." (line 959).
		dummy = newExternalDummy(g, flow, g.Settings.edgeEdge/2)
		s.link(dummy, newPort(parent, dummy.ExtSide))
	}
	g.External = true
	if output {
		s.addSegment(g, e, opposite, dummy.Ports[0], true)
	} else {
		s.addSegment(g, e, dummy.Ports[0], opposite, false)
	}
	return &externalPort{edge: e, port: dummy.Origin, dummy: dummy, output: output, exported: !endsAtParent}
}

// parentFlow is the net flow of a port on g's parent whose one edge comes from
// inside, at the port given. Only a direct child counts as inside, so an edge from
// deeper down votes for the other side (calculateNetFlow,
// CompoundGraphPreprocessor.java line 829).
func parentFlow(inside *Port, g *Graph, output bool) int {
	if output == (inside.Owner.Graph == g) {
		return 1
	}
	return -1
}

// newExternalDummy makes the dummy node of a port on g's parent, on the side edges
// leave g by if flow is not negative (createExternalPortDummy, graph/LGraphUtil.java
// line 800). Mermaid's subgraphs leave their ports free, and these ports have no
// size, so of ELK's cases only this one comes up.
func newExternalDummy(g *Graph, flow int, offset float64) *Node {
	side := sideOf[g.Direction]
	if flow < 0 {
		side = opposed[side]
	}
	dummy := &Node{Type: typeExternalPort, Graph: g, ExtSide: side, BorderOffset: offset}
	newPort(dummy, opposed[side])
	return dummy
}

// addSegment joins from to to with a new segment of e in g (createDummyEdge,
// CompoundGraphPreprocessor.java line 813).
func (s *splitter) addSegment(g *Graph, e *Edge, from, to *Port, output bool) {
	segment := &Edge{ID: e.ID, ModelOrder: e.ModelOrder}
	setSource(segment, from)
	setTarget(segment, to)
	c := s.crossing[e]
	if c == nil {
		c = &crossEdge{edge: e, source: e.Source, target: e.Target}
		s.crossing[e] = c
		s.found = append(s.found, c)
	}
	c.segments = append(c.segments, crossSegment{segment, g, output})
}
