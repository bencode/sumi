package flowchart

import (
	"fmt"
	"slices"
	"strings"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

type subgraph struct {
	at      syntax.Pos
	id      string
	label   Label
	dir     string
	nodes   []string
	classes []string
}

// subgraph is the subgraph id, or nil.
func (t *tables) subgraph(id string) *subgraph {
	if i := slices.IndexFunc(t.subgraphs, func(sg *subgraph) bool { return sg.id == id }); i >= 0 {
		return t.subgraphs[i]
	}
	return nil
}

// addSubgraph applies the body of s and then s itself, as a subgraph takes effect when it
// closes, and returns its id (addSubGraph).
func (t *tables) addSubgraph(s *Subgraph) (string, error) {
	members, dir, err := t.document(s.Body)
	if err != nil {
		return "", err
	}
	id := trim(s.ID.Text)
	if s.ID == s.Title && strings.IndexFunc(s.Title.Text, isSpace) >= 0 {
		id = fmt.Sprintf("subGraph%d", t.subCount)
	}
	t.subCount++
	nodes := t.newMembers(id, members)
	if sg := t.subgraph(id); sg != nil {
		sg.nodes = append(sg.nodes, nodes...)
	} else {
		t.subgraphs = append(t.subgraphs, &subgraph{
			at: s.Pos, id: id, label: Label{Text: trimTitle(s.Title.Text), Kind: s.Title.Kind}, dir: dir, nodes: nodes,
		})
	}
	if t.inCycle(id) {
		return "", syntax.Errorf(s.Pos, "subgraph %q is inside itself", id)
	}
	return id, nil
}

// newMembers are members without blanks and repeats, without those an earlier subgraph
// has taken, which keeps them, and without id itself.
func (t *tables) newMembers(id string, members []string) []string {
	seen := map[string]bool{}
	var nodes []string
	for _, m := range members {
		if trim(m) == "" || seen[m] || m == id || t.taken(m) {
			seen[m] = true
			continue
		}
		seen[m] = true
		nodes = append(nodes, m)
	}
	return nodes
}

// taken reports whether a subgraph has the member (exists).
func (t *tables) taken(member string) bool {
	return slices.ContainsFunc(t.subgraphs, func(sg *subgraph) bool { return slices.Contains(sg.nodes, member) })
}

// inCycle reports whether the subgraph id is inside itself. Mermaid's layout never
// returns from such a diagram when it has an edge, and draws nothing when it has none.
// Any cycle goes through id, as each subgraph is asked when it closes.
func (t *tables) inCycle(id string) bool {
	for at := id; ; {
		i := slices.IndexFunc(t.subgraphs, func(sg *subgraph) bool { return slices.Contains(sg.nodes, at) })
		if i < 0 {
			return false
		}
		if at = t.subgraphs[i].id; at == id {
			return true
		}
	}
}

// trimTitle is a title trimmed after sanitizeText, as addSubGraph does: beside a <, that
// has written a no-break space as &nbsp;.
func trimTitle(s string) string {
	if !strings.Contains(s, "<") {
		return trim(s)
	}
	return strings.TrimFunc(s, func(c rune) bool { return c != '\u00a0' && isSpace(c) })
}
