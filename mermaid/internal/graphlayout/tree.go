package graphlayout

// tree gives each node its parent, with no entry for one at the root, which is
// "" here (TreeData.parentById).
type tree map[string]string

func treeOf(nodes []Node) tree {
	t := tree{}
	for _, n := range nodes {
		if n.Parent != "" {
			t[n.ID] = n.Parent
		}
	}
	return t
}

// commonAncestor gives the lowest node holding both a and b, which may be one of
// them. For a node and itself that is the node, where Mermaid gives its parent
// (findCommonAncestor, find-common-ancestor.ts).
func (t tree) commonAncestor(a, b string) string {
	seen := map[string]bool{}
	for id := a; id != ""; id = t[id] {
		seen[id] = true
	}
	for id := b; id != ""; id = t[id] {
		if seen[id] {
			return id
		}
	}
	return ""
}

// childOnPath gives the child of ancestor that holds id, or id itself (childOnPath,
// subgraphFeedbackEdges.ts).
func (t tree) childOnPath(id, ancestor string) string {
	for t[id] != ancestor {
		id = t[id]
	}
	return id
}

// inside tells whether id sits in ancestor at any depth (isInside,
// nestedEdgeOrder.ts).
func (t tree) inside(id, ancestor string) bool {
	for p := t[id]; p != ""; p = t[p] {
		if p == ancestor {
			return true
		}
	}
	return false
}
