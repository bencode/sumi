// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import (
	"cmp"
	"slices"
)

// sortByModelOrder orders each layer and its ports as the input wrote them
// (intermediate/SortByInputModelProcessor.java line 59). Nodes sort again after
// their ports, since comparing nodes reads port order.
func sortByModelOrder(g *Graph) {
	for i, l := range g.Layers {
		previous := g.Layers[max(i-1, 0)].Nodes
		insertionSort(l.Nodes, previous)
		for _, n := range l.Nodes {
			if n.Type == typeNormal {
				sortPortsByModel(n, previous)
			}
		}
		insertionSort(l.Nodes, previous)
	}
}

// insertionSort sorts layer (intermediate/SortByInputModelProcessor.java line
// 158). compareNodes is not transitive, so the sort has to be this one. ELK also
// caches what it has compared, but an insertion sort never asks about a pair
// the cache could answer.
func insertionSort(layer, previous []*Node) {
	for i := 1; i < len(layer); i++ {
		n := layer[i]
		j := i
		for j > 0 && compareNodes(layer[j-1], n, previous) > 0 {
			layer[j] = layer[j-1]
			j--
		}
		layer[j] = n
	}
}

// compareNodes takes real nodes in model order. With a dummy node among them it
// goes by where their edges come from in the previous layer, and a node with no
// incoming edge goes last
// (compare, intermediate/preserveorder/ModelOrderNodeComparator.java line 117).
func compareNodes(a, b *Node, previous []*Node) int {
	if a.Type == typeNormal && b.Type == typeNormal {
		return cmp.Compare(a.ModelOrder, b.ModelOrder)
	}
	from, other := firstSource(a), firstSource(b)
	if from == nil {
		return 1
	}
	if other == nil {
		return -1
	}
	if from.Owner == other.Owner {
		ports := from.Owner.Ports
		return slices.Index(ports, from) - slices.Index(ports, other)
	}
	return slices.Index(previous, from.Owner) - slices.Index(previous, other.Owner)
}

// firstSource gives the port the first incoming edge of n comes from.
func firstSource(n *Node) *Port {
	for _, p := range n.Ports {
		if len(p.In) > 0 {
			return p.In[0].Source
		}
	}
	return nil
}

// sortPortsByModel puts east ports first, by the order of the nodes they lead
// to, and west ports after, from the last source to the first
// (compare, intermediate/preserveorder/ModelOrderPortComparator.java line 107).
// These keys order the ports strictly, so any sort gives ELK's result and its
// cache of comparisons has nothing to add.
func sortPortsByModel(n *Node, previous []*Node) {
	orders := targetOrders(n)
	key := func(p *Port) []int {
		if p.Side == sideWest {
			source := p.In[0].Source
			return []int{1, -slices.Index(previous, source.Owner), -slices.Index(source.Owner.Ports, source)}
		}
		edge := p.Out[0]
		order, ok := orders[targetOf(p)]
		if !ok {
			order = edge.ModelOrder
		}
		return []int{0, order, edge.ModelOrder}
	}
	slices.SortFunc(n.Ports, func(a, b *Port) int { return slices.Compare(key(a), key(b)) })
}

// targetOrders gives, for each real node n leads to, the first model order among
// its edges there that are not reversed (longEdgeTargetNodePreprocessing,
// intermediate/SortByInputModelProcessor.java line 109).
func targetOrders(n *Node) map[*Node]int {
	orders := map[*Node]int{}
	for _, p := range n.Ports {
		if len(p.Out) == 0 || p.Out[0].Reversed {
			continue
		}
		target := targetOf(p)
		if order, ok := orders[target]; !ok || p.Out[0].ModelOrder < order {
			orders[target] = p.Out[0].ModelOrder
		}
	}
	return orders
}

// targetOf follows p's edge through dummy nodes to a real node (getTargetNode,
// intermediate/SortByInputModelProcessor.java line 139).
func targetOf(p *Port) *Node {
	n := p.Out[0].Target.Owner
	for n.Type != typeNormal {
		n = outgoing(n)[0].Target.Owner
	}
	return n
}
