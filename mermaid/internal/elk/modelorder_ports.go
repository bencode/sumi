// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import "cmp"

// portOrder compares the ports of a node
// (intermediate/preserveorder/ModelOrderPortComparator.java). Mermaid's nodes have
// no ports of their own, so ports have no model order.
type portOrder struct {
	*modelOrder
	known  *decisions[*Port]
	orders map[*Node]int
}

func (o *modelOrder) newPortOrder(orders map[*Node]int) *portOrder {
	return &portOrder{o, newDecisions[*Port](), orders}
}

// compare puts east ports before west ones. On one side, ports coming in go by where
// they come from, ports going out by where they lead, going out goes first, and
// what is left keeps its order. West sides run the other way (compare,
// ModelOrderPortComparator.java line 107).
func (o *portOrder) compare(a, b *Port) int {
	if outcome, ok := o.known.lookup(a, b); ok {
		return outcome
	}
	if a.Side != b.Side {
		return o.known.settle(a, b, cmp.Compare(a.Side, b.Side))
	}
	turn := 1
	if a.Side == sideWest {
		turn = -1
	}
	switch {
	case len(a.In) > 0 && len(b.In) > 0:
		if outcome := o.compareSources(a, b, turn); outcome != 0 {
			return o.known.settle(a, b, outcome)
		}
		return o.known.settle(a, b, -turn)
	case len(a.Out) > 0 && len(b.Out) > 0:
		return o.known.settle(a, b, turn*o.compareTargets(a, b))
	case len(a.In) > 0 && len(b.Out) > 0:
		return o.known.settle(a, b, 1)
	case len(a.Out) > 0 && len(b.In) > 0:
		return o.known.settle(a, b, -1)
	}
	return o.known.settle(a, b, -1)
}

// compareSources goes by the ports of a common source, by long edge dummies of the
// node's own layer, or by the previous layer, and gives 0 with none of them.
func (o *portOrder) compareSources(a, b *Port, turn int) int {
	from, other := a.In[0].Source, b.In[0].Source
	if from.Owner == other.Owner {
		return turn * firstOf(from.Owner.Ports, from, other)
	}
	own := o.layerOf[a.Owner]
	if from.Owner.Type == typeLongEdge && other.Owner.Type == typeLongEdge &&
		o.layerOf[from.Owner] == own && o.layerOf[other.Owner] == own {
		if outcome := firstOf(o.layers[own].Nodes, from.Owner, other.Owner); outcome != 0 {
			if a.Side == sideEast {
				turn = -turn
			}
			return turn * outcome
		}
	}
	return turn * firstOf(o.previous, from.Owner, other.Owner)
}

// compareTargets goes by edges' model order, which ports leading to one node share
// unless they lead there together.
func (o *portOrder) compareTargets(a, b *Port) int {
	orderA, orderB := a.Out[0].ModelOrder, b.Out[0].ModelOrder
	targetA, targetB := o.targets[a], o.targets[b]
	if targetA != nil && targetA == targetB {
		return greater(orderA, orderB)
	}
	if order, ok := o.orders[targetA]; ok {
		orderA = order
	}
	if order, ok := o.orders[targetB]; ok {
		orderB = order
	}
	return greater(orderA, orderB)
}

// targetOrders notes where n's ports lead and gives, for each real node there, the
// first model order among its edges that are not reversed
// (longEdgeTargetNodePreprocessing, intermediate/SortByInputModelProcessor.java line
// 109).
func (o *modelOrder) targetOrders(n *Node) map[*Node]int {
	orders := map[*Node]int{}
	for _, p := range n.Ports {
		if len(p.Out) == 0 {
			continue
		}
		target := o.targetOf(p)
		o.targets[p] = target
		if target == nil || p.Out[0].Reversed {
			continue
		}
		if order, ok := orders[target]; !ok || p.Out[0].ModelOrder < order {
			orders[target] = p.Out[0].ModelOrder
		}
	}
	return orders
}

// targetOf follows p's edge through dummy nodes to a real node, if it gets to one
// (getTargetNode, intermediate/SortByInputModelProcessor.java line 139). An inverted
// port's dummy knows only the next node as its target, which may be a label dummy
// (setLongEdgeSourceAndTarget, InvertedPortProcessor.java line 322). ELK may stop at
// an external port dummy, which only one edge reaches, so that its order comes to
// the edge's own all the same.
func (o *modelOrder) targetOf(p *Port) *Node {
	n := p.Out[0].Target.Owner
	if n.Type == typeLongEdge && o.layerOf[n] == o.layerOf[p.Owner] {
		if next := outgoing(n)[0].Target.Owner; next.Type == typeLabel {
			return next
		}
	}
	for n.Type != typeNormal {
		next := outgoing(n)
		if len(next) == 0 {
			return nil
		}
		n = next[0].Target.Owner
	}
	return n
}
