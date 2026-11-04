package dagre

// An edgeKey is graphlib's edge object: an edge's ends and its name, ""
// for none. In an undirected graph v is the end that compares smaller.
type edgeKey struct{ v, w, name string }

// id is graphlib's edge id (edgeArgsToId, graph.js line 1125).
func (e edgeKey) id() string {
	name := e.name
	if name == "" {
		name = top
	}
	return e.v + "\x01" + e.w + "\x01" + name
}

// key is the edge object of the edge from v to w named name
// (edgeArgsToObj, graph.js line 1146).
func (g *graph[N, E]) key(v, w, name string) edgeKey {
	if !g.directed && jsLess(w, v) {
		v, w = w, v
	}
	return edgeKey{v, w, name}
}

// setEdge adds the edge from v to w named name, labeled label, adding its
// ends when they are not there. When the edge is there already, label
// replaces its label (graph.js line 852).
func (g *graph[N, E]) setEdge(v, w, name string, label *E) {
	k := g.key(v, w, name)
	id := k.id()
	if g.edges.has(id) {
		g.labels[id] = label
		return
	}
	g.setNode(v, nil)
	g.setNode(w, nil)
	g.labels[id] = label
	g.edges.set(id, k)
	increment(g.preds[k.w], k.v)
	increment(g.sucs[k.v], k.w)
	g.in[k.w].set(id, k)
	g.out[k.v].set(id, k)
}

func (g *graph[N, E]) edge(v, w, name string) *E { return g.labels[g.key(v, w, name).id()] }
func (g *graph[N, E]) edgeOf(e edgeKey) *E       { return g.labels[e.id()] }
func (g *graph[N, E]) hasEdge(v, w, name string) bool {
	return g.edges.has(g.key(v, w, name).id())
}

func (g *graph[N, E]) removeEdge(v, w, name string) { g.removeEdgeKey(g.key(v, w, name)) }

func (g *graph[N, E]) removeEdgeKey(e edgeKey) {
	id := e.id()
	if !g.edges.has(id) {
		return
	}
	g.edges.del(id)
	delete(g.labels, id)
	decrement(g.preds[e.w], e.v)
	decrement(g.sucs[e.v], e.w)
	g.in[e.w].del(id)
	g.out[e.v].del(id)
}

func (g *graph[N, E]) edgeList() []edgeKey { return g.edges.values() }

func (g *graph[N, E]) inEdges(v string) []edgeKey  { return g.in[v].values() }
func (g *graph[N, E]) outEdges(v string) []edgeKey { return g.out[v].values() }

// nodeEdges are v's edges in, then its edges out.
func (g *graph[N, E]) nodeEdges(v string) []edgeKey {
	return append(g.inEdges(v), g.outEdges(v)...)
}

func increment(k *keys[int], v string) {
	n, _ := k.get(v)
	k.set(v, n+1)
}

func decrement(k *keys[int], v string) {
	if n, _ := k.get(v); n > 1 {
		k.set(v, n-1)
	} else {
		k.del(v)
	}
}
