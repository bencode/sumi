package er

import (
	"fmt"
	"slices"
	"strings"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

func trim(s string) string { return strings.TrimFunc(s, syntax.IsSpace) }

type cssClass struct{ styles, textStyles []string }

type subgraph struct {
	at                     syntax.Pos
	id, title, dir         string
	nodes, classes, styles []string
}

// A db is what ErDB holds while the statements of a diagram run.
type db struct {
	direction string
	entities  map[string]*Node
	names     []string // of the entities, in the order they came
	classes   map[string]*cssClass
	edges     []Edge
	subgraphs []*subgraph
}

func collect(stmts []Stmt) (*Diagram, error) {
	d := &db{direction: "TB", entities: map[string]*Node{}, classes: map[string]*cssClass{}}
	if _, _, err := d.run(stmts, false); err != nil {
		return nil, err
	}
	return d.diagram(), nil
}

// run runs stmts in order, as the parser's actions do, the inside of a subgraph before
// the subgraph. It gives what a subgraph holding them is told: the names of what they
// are, and the last direction among them. Only inside a subgraph is a direction not the
// diagram's.
func (d *db) run(stmts []Stmt, inside bool) (names []string, dir string, _ error) {
	for _, s := range stmts {
		switch s := s.(type) {
		case EntityStmt:
			n := d.addEntity(s.Name, s.Alias, s.At)
			n.Attributes = append(n.Attributes, s.Attributes...)
			d.setClass([]string{s.Name}, s.Classes)
			names = append(names, s.Name)
		case RelationStmt:
			d.addRelationship(s)
			names = append(names, s.A, s.B)
		case AccStmt:
			names = append(names, s.Value)
		case DirectionStmt:
			if inside {
				dir = s.Value
			} else {
				d.direction = s.Value
			}
		case ClassDefStmt:
			d.addClass(s.IDs, s.Styles)
			names = append(names, s.Word)
		case ClassStmt:
			d.setClass(s.IDs, s.Classes)
			names = append(names, s.Word)
		case StyleStmt:
			d.addStyles(s.IDs, s.Styles)
			names = append(names, s.Word)
		case SubgraphStmt:
			held, heldDir, err := d.run(s.Body, true)
			if err != nil {
				return nil, "", err
			}
			id, err := d.addSubgraph(s, held, heldDir)
			if err != nil {
				return nil, "", err
			}
			names = append(names, id)
		}
	}
	return names, dir, nil
}

func (d *db) addEntity(name, alias string, at syntax.Pos) *Node {
	n, ok := d.entities[name]
	if !ok {
		n = &Node{ID: fmt.Sprintf("entity-%s-%d", name, len(d.entities)), Label: name, Alias: alias, Classes: "default", At: at}
		d.entities[name] = n
		d.names = append(d.names, name)
	} else if n.Alias == "" {
		n.Alias = alias
	}
	return n
}

// subgraphNamed is the subgraph id, the last if several have it, or nil.
func (d *db) subgraphNamed(id string) *subgraph {
	for _, s := range slices.Backward(d.subgraphs) {
		if s.id == id {
			return s
		}
	}
	return nil
}

// endpoint is what a relationship's end joins: a subgraph of that name if one has closed
// by now, or else the entity.
func (d *db) endpoint(name string, at syntax.Pos) string {
	e := d.addEntity(name, "", at)
	if d.subgraphNamed(name) != nil {
		return name
	}
	return e.ID
}

func (d *db) addRelationship(r RelationStmt) {
	start, end := d.endpoint(r.A, r.At), d.endpoint(r.B, r.At)
	pattern := "dashed"
	if r.Identifying {
		pattern = "solid"
	}
	d.edges = append(d.edges, Edge{
		ID:          fmt.Sprintf("id_%s_%s_%d", start, end, len(d.edges)),
		Start:       start,
		End:         end,
		Label:       r.Role,
		StartMarker: r.Start,
		EndMarker:   r.End,
		Pattern:     pattern,
		At:          r.At,
	})
	d.setClass([]string{r.A}, r.ClassesA)
	d.setClass([]string{r.B}, r.ClassesB)
}

// setClass reaches only what exists by now, as does addStyles.
func (d *db) setClass(ids, classes []string) {
	for _, id := range ids {
		if n := d.entities[id]; n != nil {
			for _, c := range classes {
				n.Classes += " " + c
			}
		}
		if s := d.subgraphNamed(id); s != nil {
			s.classes = append(s.classes, classes...)
		}
	}
}

func (d *db) addStyles(ids, styles []string) {
	for _, id := range ids {
		if n := d.entities[id]; n != nil {
			n.Styles = append(n.Styles, styles...)
		}
		if s := d.subgraphNamed(id); s != nil {
			s.styles = append(s.styles, styles...)
		}
	}
}

// addClass is a classDef. A style about a color is kept a second time for the text.
func (d *db) addClass(ids, styles []string) {
	for _, id := range ids {
		c := d.classes[id]
		if c == nil {
			c = &cssClass{}
			d.classes[id] = c
		}
		for _, s := range styles {
			if strings.Contains(s, "color") {
				c.textStyles = append(c.textStyles, strings.Replace(s, "fill", "bgFill", 1))
			}
			c.styles = append(c.styles, s)
		}
	}
}

func (d *db) compiled(classes []string) []string {
	var out []string
	for _, name := range classes {
		if c := d.classes[name]; c != nil {
			out = append(append(out, c.styles...), c.textStyles...)
		}
	}
	for i, s := range out {
		out[i] = trim(s)
	}
	return out
}

// addSubgraph is addSubGraph. A title goes through sanitizeText, which leaves one without
// a < as it is. It returns the subgraph's id.
func (d *db) addSubgraph(s SubgraphStmt, names []string, dir string) (string, error) {
	if strings.Contains(s.Title, "<") {
		return "", syntax.Unsupported(s.At, "a < in the title of a subgraph")
	}
	sg := &subgraph{at: s.At, id: trim(s.ID), title: trim(s.Title), dir: dir, nodes: members(names)}
	d.subgraphs = append(d.subgraphs, sg)
	if d.inCycle(sg.id) {
		return "", syntax.Errorf(s.At, "subgraph %q is inside itself", sg.id)
	}
	return sg.id, nil
}

// members are names without blanks and repeats, compared trimmed. A name that an earlier
// subgraph holds as well stays that subgraph's, as holder finds the earliest.
func members(names []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, name := range names {
		key := trim(name)
		if key != "" && !seen[key] {
			out = append(out, name)
		}
		seen[key] = true
	}
	return out
}

// holder is the earliest subgraph that has name as a member, or nil.
func (d *db) holder(name string) *subgraph {
	i := slices.IndexFunc(d.subgraphs, func(s *subgraph) bool { return slices.Contains(s.nodes, name) })
	if i < 0 {
		return nil
	}
	return d.subgraphs[i]
}

// inCycle reports whether the subgraph id is inside itself. Mermaid's layout never
// returns from such a diagram when it has an edge, and draws nothing when it has none.
// Any cycle goes through id, as each subgraph is asked when it closes.
func (d *db) inCycle(id string) bool {
	for at := id; ; {
		s := d.holder(at)
		if s == nil {
			return false
		}
		if at = s.id; at == id {
			return true
		}
	}
}

// diagram is getData: the subgraphs last to first, then the entities, leaving out any a
// subgraph has the name of.
func (d *db) diagram() *Diagram {
	parent := func(name string) string {
		if s := d.holder(name); s != nil {
			return s.id
		}
		return ""
	}
	var nodes []Node
	for _, s := range slices.Backward(d.subgraphs) {
		nodes = append(nodes, Node{
			ID: s.id, Parent: parent(s.id), Label: s.title, Group: true, Classes: strings.Join(s.classes, " "),
			Dir: s.dir, Compiled: d.compiled(s.classes), Styles: s.styles, At: s.at,
		})
	}
	for _, name := range d.names {
		if d.subgraphNamed(name) != nil {
			continue
		}
		n := *d.entities[name]
		n.Parent = parent(name)
		n.Compiled = d.compiled(strings.Split(n.Classes, " "))
		nodes = append(nodes, n)
	}
	return &Diagram{Direction: d.direction, Nodes: nodes, Edges: d.edges}
}
