// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import (
	"cmp"
	"maps"
	"math"
	"slices"
)

// modelOrder is what sorting by the order of the input knows about g.
type modelOrder struct {
	layers   []*Layer
	layerOf  map[*Node]int
	previous []*Node
	targets  map[*Port]*Node // InternalProperties.LONG_EDGE_TARGET_NODE
}

// sortByModelOrder orders each layer and its ports as the input wrote them
// (intermediate/SortByInputModelProcessor.java line 59). Nodes sort again after
// their ports, since comparing nodes reads port order. A dummy node's two ports
// sit on different sides, which sortPortLists has seen to. ELK reads the index of
// later layers before writing it, but only where a dummy node's source in the same
// layer has already decided (ModelOrderNodeComparator.java line 338).
func sortByModelOrder(g *Graph) {
	o := &modelOrder{layers: g.Layers, layerOf: map[*Node]int{}, targets: map[*Port]*Node{}}
	for i, l := range g.Layers {
		for _, n := range l.Nodes {
			o.layerOf[n] = i
		}
	}
	for i, l := range g.Layers {
		o.previous = g.Layers[max(i-1, 0)].Nodes
		insertionSort(l.Nodes, o.newNodeOrder(true))
		for _, n := range l.Nodes {
			if n.Type == typeNormal {
				ports := o.newPortOrder(o.targetOrders(n))
				mergeSort(slices.Clone(n.Ports), n.Ports, 0, len(n.Ports), ports.compare)
			}
		}
		insertionSort(l.Nodes, o.newNodeOrder(false))
	}
}

// insertionSort sorts layer (intermediate/SortByInputModelProcessor.java line
// 158). The comparison of nodes is not transitive, so the sort has to be this one.
func insertionSort(layer []*Node, order *nodeOrder) {
	for i := 1; i < len(layer); i++ {
		n := layer[i]
		j := i
		for j > 0 && order.compare(layer[j-1], n) > 0 {
			layer[j] = layer[j-1]
			j--
		}
		layer[j] = n
	}
}

// mergeSort is GWT's sort, which elkjs gets for Collections.sort (Arrays.mergeSort
// in GWT's emulated library). The comparison of ports need not be consistent, so
// the sort has to be this one. temp starts as a copy of all.
func mergeSort[T any](temp, all []T, low, high int, compare func(a, b T) int) {
	if high-low < 7 {
		for i := low + 1; i < high; i++ {
			for j := i; j > low && compare(all[j-1], all[j]) > 0; j-- {
				all[j], all[j-1] = all[j-1], all[j]
			}
		}
		return
	}
	mid := low + (high-low)>>1
	mergeSort(all, temp, low, mid, compare)
	mergeSort(all, temp, mid, high, compare)
	if compare(temp[mid-1], temp[mid]) <= 0 {
		copy(all[low:high], temp[low:high])
		return
	}
	left, right := low, mid
	for i := low; i < high; i++ {
		if right >= high || left < mid && compare(temp[left], temp[right]) <= 0 {
			all[i] = temp[left]
			left++
		} else {
			all[i] = temp[right]
			right++
		}
	}
}

// decisions remembers what a comparison decided and what follows from that, which
// then goes before comparing again (biggerThan and smallerThan,
// intermediate/preserveorder/ModelOrderNodeComparator.java lines 117 and 309). The
// lookup is ELK's, whose last check repeats the second.
type decisions[T comparable] struct {
	bigger, smaller map[T]map[T]bool
}

func newDecisions[T comparable]() *decisions[T] {
	return &decisions[T]{map[T]map[T]bool{}, map[T]map[T]bool{}}
}

func (d *decisions[T]) lookup(a, b T) (int, bool) {
	switch {
	case d.bigger[a][b]:
		return 1, true
	case d.bigger[b][a], d.smaller[a][b]:
		return -1, true
	}
	return 0, false
}

func (d *decisions[T]) note(bigger, smaller T) {
	addTo(d.bigger, bigger, smaller)
	addTo(d.smaller, smaller, bigger)
	for _, least := range slices.Collect(maps.Keys(d.bigger[smaller])) {
		addTo(d.bigger, bigger, least)
		addTo(d.smaller, least, bigger, slices.Collect(maps.Keys(d.smaller[bigger]))...)
	}
	for _, most := range slices.Collect(maps.Keys(d.smaller[bigger])) {
		addTo(d.smaller, smaller, most)
		addTo(d.bigger, most, smaller, slices.Collect(maps.Keys(d.bigger[smaller]))...)
	}
}

func addTo[T comparable](sets map[T]map[T]bool, key, first T, rest ...T) {
	if sets[key] == nil {
		sets[key] = map[T]bool{}
	}
	sets[key][first] = true
	for _, v := range rest {
		sets[key][v] = true
	}
}

// settle notes the outcome of comparing a and b and gives it.
func (d *decisions[T]) settle(a, b T, outcome int) int {
	if outcome > 0 {
		d.note(a, b)
	} else {
		d.note(b, a)
	}
	return outcome
}

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

// greater gives 1 if a is greater than b, and -1 for less or equal.
func greater(a, b int) int {
	if a > b {
		return 1
	}
	return -1
}

// firstOf gives -1 if a comes first in all, 1 if b does, and 0 with neither.
func firstOf[T comparable](all []T, a, b T) int {
	for _, v := range all {
		switch v {
		case a:
			return -1
		case b:
			return 1
		}
	}
	return 0
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
