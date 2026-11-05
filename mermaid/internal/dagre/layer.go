package dagre

// A layerNode is a node of a layer graph. A node or dummy shares g's label
// through n, so the order set here is g's; a subgraph has only its border
// nodes in this rank, and the root nothing.
type layerNode struct {
	n           *node
	left, right string
}

type layerEdge struct{ weight int }

type layerGraph = graph[layerNode, layerEdge]

// constraints holds which of two subgraphs goes left of the other.
type constraints = graph[struct{}, struct{}]

// A layer is what it takes to sort one rank: a graph of the rank's nodes,
// under root and their subgraphs, and of their neighbors in the rank they
// are sorted by.
type layer struct {
	g    *layerGraph
	root string
}

// buildLayerGraph is rank's layer. adjacent lists the edges to the rank
// sorted by; they all go from the neighbor to the node in the layer, with
// the edges between two nodes as one (build-layer-graph.js line 36). The
// root is there even in a rank with no nodes.
func buildLayerGraph(g *layoutGraph, c *counter, rank int, adjacent func(string) []edgeKey) layer {
	root := c.next("_root")
	for g.hasNode(root) {
		root = c.next("_root")
	}
	lg := newGraph[layerNode, layerEdge](true, true)
	lg.setNode(root, nil)
	for _, v := range g.nodeNames() {
		label := layerLabel(g.node(v), rank)
		if label == nil {
			continue
		}
		lg.setNode(v, label)
		parent := g.parentOf(v)
		if parent == "" {
			parent = root
		}
		lg.setParent(v, parent)
		for _, e := range adjacent(v) {
			u := e.v
			if u == v {
				u = e.w
			}
			if !lg.hasNode(u) {
				lg.setNode(u, &layerNode{n: g.node(u)})
			}
			weight := g.edgeOf(e).weight
			if l := lg.edge(u, v, ""); l != nil {
				weight += l.weight
			}
			lg.setEdge(u, v, "", &layerEdge{weight})
		}
	}
	return layer{lg, root}
}

// layerLabel is n's label in rank's layer, nil when n is not in rank. A
// subgraph has no rank, and other nodes no minRank or maxRank, which JS
// compares as undefined and Go would as 0 (build-layer-graph.js lines 48
// and 60).
func layerLabel(n *node, rank int) *layerNode {
	if n.borderTop != "" {
		if rank < n.minRank || n.maxRank < rank {
			return nil
		}
		i := rank - n.minRank
		return &layerNode{left: n.borderLeft[i], right: n.borderRight[i]}
	}
	if n.rank != rank {
		return nil
	}
	return &layerNode{n: n}
}

// addSubgraphConstraints notes in cg the order vs gives subgraphs. From
// each node it goes up through the subgraphs around it, as far as the
// first that comes after a sibling, and has that sibling stay left of it
// (add-subgraph-constraints.js line 5).
func addSubgraphConstraints(lg *layerGraph, cg *constraints, vs []string) {
	prev := map[string]string{} // the last child seen, by parent
next:
	for _, v := range vs {
		for child := lg.parentOf(v); child != ""; {
			parent := lg.parentOf(child)
			last := prev[parent]
			prev[parent] = child
			if last != "" && last != child {
				cg.setEdge(last, child, "", nil)
				continue next
			}
			child = parent
		}
	}
}
