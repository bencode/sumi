// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import (
	"maps"
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
