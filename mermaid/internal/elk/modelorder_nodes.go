// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import "math"

// nodeOrder compares the nodes of a layer
// (intermediate/preserveorder/ModelOrderNodeComparator.java).
type nodeOrder struct {
	*modelOrder
	known       *decisions[*Node]
	beforePorts bool
}

func (o *modelOrder) newNodeOrder(beforePorts bool) *nodeOrder {
	return &nodeOrder{o, newDecisions[*Node](), beforePorts}
}

// compare takes real nodes in model order. With a dummy node among them it goes by
// where their edges come from in the previous layer. A node with no such edge goes
// by the dummy node it stands for and the first edge coming in, and last with
// neither (compare, ModelOrderNodeComparator.java line 117).
func (o *nodeOrder) compare(a, b *Node) int {
	if outcome, ok := o.known.lookup(a, b); ok {
		return outcome
	}
	if a.Type == typeNormal && b.Type == typeNormal {
		return o.known.settle(a, b, greater(a.ModelOrder, b.ModelOrder))
	}
	from, other := o.firstSource(a), o.firstSource(b)
	if from != nil && other != nil {
		if from.Owner == other.Owner {
			return o.known.settle(a, b, firstOf(from.Owner.Ports, from, other))
		}
		if outcome := firstOf(o.previous, from.Owner, other.Owner); outcome != 0 {
			return o.known.settle(a, b, outcome)
		}
		return o.known.settle(a, b, -1)
	}
	if outcome := o.compareHelpers(a, b); outcome != 0 {
		return o.known.settle(a, b, outcome)
	}
	if from == nil && other == nil {
		return o.known.settle(a, b, -1)
	}
	return o.known.settle(a, b, greater(edgeOrder(a), edgeOrder(b)))
}

// firstSource gives the port the first edge from the previous layer comes from.
func (o *modelOrder) firstSource(n *Node) *Port {
	for _, p := range n.Ports {
		if len(p.In) > 0 && o.layerOf[p.In[0].Source.Owner] == o.layerOf[n]-1 {
			return p.In[0].Source
		}
	}
	return nil
}

// edgeOrder is the model order of n's first incoming edge, or the largest there is
// for a dummy node to go over (getModelOrderFromConnectedEdges,
// ModelOrderNodeComparator.java line 292, with LongEdgeOrderingStrategy's default).
func edgeOrder(n *Node) int {
	for _, p := range n.Ports {
		if len(p.In) > 0 {
			return p.In[0].ModelOrder
		}
	}
	return math.MaxInt
}

// compareHelpers compares long edge dummies by the nodes of their layer they lead
// to or come from, those of inverted ports, or gives 0 where there are none
// (handleHelperDummyNodes, ModelOrderNodeComparator.java line 330).
func (o *nodeOrder) compareHelpers(a, b *Node) int {
	switch {
	case a.Type == typeLongEdge && b.Type == typeNormal:
		return o.dummyAgainst(a, b, o.layerOf[a], 1)
	case a.Type == typeNormal && b.Type == typeLongEdge:
		return o.dummyAgainst(b, a, o.layerOf[a], -1)
	case a.Type == typeLongEdge && b.Type == typeLongEdge:
		return o.compareDummies(a, b)
	}
	return 0
}

// dummyAgainst compares dummy with n, sign giving which came first in the call, and
// layer that one's layer: a dummy goes below the node it belongs to.
func (o *nodeOrder) dummyAgainst(dummy, n *Node, layer, sign int) int {
	source, target := dummyEnds(dummy)
	if o.layerOf[source.Owner] != layer && o.layerOf[target.Owner] != layer {
		return 0
	}
	if source.Owner == n || target.Owner == n {
		o.known.note(dummy, n)
		return sign
	}
	if sign > 0 {
		return o.compare(source.Owner, n)
	}
	return o.compare(n, source.Owner)
}

// dummyEnds gives the ports around a long edge dummy.
func dummyEnds(n *Node) (source, target *Port) {
	return incoming(n)[0].Source, outgoing(n)[0].Target
}

// compareDummies compares two long edge dummies, at least one of them an inverted
// port's, by the node it belongs to (ModelOrderNodeComparator.java line 385). The
// outcomes are ELK's, which note the opposite of what they return in places.
func (o *nodeOrder) compareDummies(a, b *Node) int {
	sourceA, targetA := dummyEnds(a)
	sourceB, targetB := dummyEnds(b)
	nodeA, fromA, toA := o.reference(a, sourceA, targetA)
	nodeB, fromB, toB := o.reference(b, sourceB, targetB)
	if nodeA == nodeB {
		switch {
		case !o.beforePorts:
			if outcome := firstOf(nodeA.Ports, sourceA, sourceB); outcome != 0 {
				return o.known.settle(a, b, outcome)
			}
		case fromA && fromB:
			ports := o.newPortOrder(nil)
			if ports.compare(sourceA, sourceB) > 0 {
				o.known.note(b, a)
				return 1
			}
			o.known.note(a, b)
			return -1
		case fromA && toB:
			o.known.note(b, a)
			return 1
		case toA && fromB:
			o.known.note(a, b)
			return -1
		case toA && toB:
			return 0
		}
	}
	return o.compare(nodeA, nodeB)
}

// reference gives the node of n's layer that n belongs to, and whether that is n's
// source or target, or n itself if it is neither.
func (o *modelOrder) reference(n *Node, source, target *Port) (node *Node, isSource, isTarget bool) {
	switch o.layerOf[n] {
	case o.layerOf[source.Owner]:
		return source.Owner, true, false
	case o.layerOf[target.Owner]:
		return target.Owner, false, true
	}
	return n, false, false
}
