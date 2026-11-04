package dagre

// top is graphlib's GRAPH_NODE, the parent of the top-level nodes, and
// its DEFAULT_EDGE_NAME, the name in the id of an edge without one
// (graph.js lines 3-4).
const top = "\x00"

// A graph is graphlib's Graph (graphlib/graph.js), with the methods dagre's
// steps so far use. Its nodes carry an *N and its edges an *E, which the
// steps change in place, as dagre's do. Everything it lists, it lists in
// the order graphlib does: see keys.
type graph[N, E any] struct {
	directed, compound bool
	nodes              keys[*N]
	parent             map[string]string          // compound: top for none
	children           map[string]*keys[struct{}] // compound, top included
	in, out            map[string]*keys[edgeKey]  // by edge id
	preds, sucs        map[string]*keys[int]      // how many edges each
	edges              keys[edgeKey]              // by edge id
	labels             map[string]*E              // by edge id
}

func newGraph[N, E any](directed, compound bool) *graph[N, E] {
	g := &graph[N, E]{
		directed: directed, compound: compound,
		in: map[string]*keys[edgeKey]{}, out: map[string]*keys[edgeKey]{},
		preds: map[string]*keys[int]{}, sucs: map[string]*keys[int]{},
		labels: map[string]*E{},
	}
	if compound {
		g.parent = map[string]string{}
		g.children = map[string]*keys[struct{}]{top: {}}
	}
	return g
}

// setNode adds v, labeled label, or a new N when label is nil. When v is
// there already, a label replaces its label and nil leaves it.
func (g *graph[N, E]) setNode(v string, label *N) {
	if g.nodes.has(v) {
		if label != nil {
			g.nodes.set(v, label)
		}
		return
	}
	if label == nil {
		label = new(N)
	}
	g.nodes.set(v, label)
	if g.compound {
		g.parent[v] = top
		g.children[v] = &keys[struct{}]{}
		g.children[top].set(v, struct{}{})
	}
	g.in[v], g.out[v] = &keys[edgeKey]{}, &keys[edgeKey]{}
	g.preds[v], g.sucs[v] = &keys[int]{}, &keys[int]{}
}

func (g *graph[N, E]) node(v string) *N {
	n, _ := g.nodes.get(v)
	return n
}

func (g *graph[N, E]) hasNode(v string) bool { return g.nodes.has(v) }
func (g *graph[N, E]) nodeNames() []string   { return g.nodes.names() }
func (g *graph[N, E]) nodeCount() int        { return g.nodes.len() }

// removeNode removes v and its edges; its children move to the top level
// (graph.js line 526).
func (g *graph[N, E]) removeNode(v string) {
	if !g.nodes.has(v) {
		return
	}
	g.nodes.del(v)
	if g.compound {
		g.children[g.parent[v]].del(v)
		delete(g.parent, v)
		for _, c := range g.childrenOf(v) {
			g.setParent(c, "")
		}
		delete(g.children, v)
	}
	for _, e := range g.in[v].values() {
		g.removeEdgeKey(e)
	}
	delete(g.in, v)
	delete(g.preds, v)
	for _, e := range g.out[v].values() {
		g.removeEdgeKey(e)
	}
	delete(g.out, v)
	delete(g.sucs, v)
}

// setParent makes parent, or the top level when it is "", v's parent,
// adding both when they are not there. v moves to the end of its new
// parent's children (graph.js line 560).
func (g *graph[N, E]) setParent(v, parent string) {
	if parent == "" {
		parent = top
	} else {
		g.setNode(parent, nil)
	}
	g.setNode(v, nil)
	g.children[g.parent[v]].del(v)
	g.parent[v] = parent
	g.children[parent].set(v, struct{}{})
}

// parentOf is v's parent, "" at the top level.
func (g *graph[N, E]) parentOf(v string) string {
	if p := g.parent[v]; p != top {
		return p
	}
	return ""
}

// childrenOf are v's children, or the top-level nodes when v is ""
// (graph.js line 624). A graph that is not compound has only top-level
// nodes.
func (g *graph[N, E]) childrenOf(v string) []string {
	if v == "" {
		v = top
	}
	if g.compound {
		if c, ok := g.children[v]; ok {
			return c.names()
		}
		return nil
	}
	if v == top {
		return g.nodes.names()
	}
	return nil
}

// sources are the nodes no edge goes into.
func (g *graph[N, E]) sources() []string {
	var out []string
	for _, v := range g.nodes.names() {
		if g.in[v].len() == 0 {
			out = append(out, v)
		}
	}
	return out
}

func (g *graph[N, E]) predecessors(v string) []string { return names(g.preds[v]) }
func (g *graph[N, E]) successors(v string) []string   { return names(g.sucs[v]) }

// neighbors are v's predecessors, then those of its successors that are
// not among them (graph.js line 678).
func (g *graph[N, E]) neighbors(v string) []string {
	out := g.predecessors(v)
	for _, w := range g.successors(v) {
		if !g.preds[v].has(w) {
			out = append(out, w)
		}
	}
	return out
}

func names[V any](k *keys[V]) []string {
	if k == nil {
		return nil
	}
	return k.names()
}
