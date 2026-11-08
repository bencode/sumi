package flowchart

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/bencode/sumi/mermaid/internal/style"
	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// maxEdges is Mermaid's default limit.
const maxEdges = 500

type vertex struct {
	at      syntax.Pos // where its label or shape was last given
	id      string
	label   Label
	shape   Shape
	styles  []string
	classes []string
}

type edge struct {
	at             syntax.Pos
	id, start, end string
	label          Label
	kind, stroke   string
	styles         []string
	classes        []string
}

type class struct{ styles, textStyles []string }

// tables are what FlowDB keeps while the parser reduces. They live only inside Resolve.
type tables struct {
	vertices     []*vertex // in the order they were first named
	byID         map[string]*vertex
	edges        []*edge
	defaultStyle []string
	classes      map[string]*class
	subgraphs    []*subgraph // in the order they closed
	subCount     int
}

// Resolve folds the statements of a diagram in the direction dir, as written, into its
// nodes and edges, as the actions of flow.jison fold them into a FlowDB.
func Resolve(dir string, body []Stmt) (*Diagram, error) {
	t := &tables{byID: map[string]*vertex{}, classes: map[string]*class{}}
	if _, _, err := t.document(body); err != nil {
		return nil, err
	}
	return t.data(direction(dir))
}

// direction is what setDirection makes of dir.
func direction(dir string) string {
	switch dir = trim(dir); dir {
	case "<":
		return "RL"
	case "^":
		return "BT"
	case ">":
		return "LR"
	case "v", "TD":
		return "TB"
	}
	return dir
}

// document applies body, and returns the values a subgraph reads from its statements:
// the ids of its members and the direction it asks for.
func (t *tables) document(body []Stmt) (members []string, dir string, err error) {
	for _, s := range body {
		switch s := s.(type) {
		case *Chain:
			members = append(members, s.Members...)
			err = t.steps(s.Steps)
		case *Subgraph:
			var id string
			id, err = t.addSubgraph(s)
			members = append(members, id)
		case *Member:
			members = append(members, s.Text)
		case *Direction:
			dir = s.Value
		case *Style:
			if err = style.Check(s.Pos, s.Styles); err != nil {
				break
			}
			if v := t.vertex(s.ID, s.Pos); v != nil {
				v.styles = append(v.styles, s.Styles...)
			}
		case *LinkStyle:
			if err = style.Check(s.Pos, s.Styles); err == nil {
				err = t.updateLink(s)
			}
		case *ClassDef:
			if err = style.Check(s.Pos, classStyles(s)); err == nil {
				t.addClass(s)
			}
		case *Class:
			t.setClass(s)
		}
		if err != nil {
			return nil, "", err
		}
	}
	return members, dir, nil
}

// steps applies the steps of a chain.
func (t *tables) steps(steps []Stmt) error {
	for _, s := range steps {
		var err error
		switch s := s.(type) {
		case *Vertex:
			t.addVertex(s)
		case *Class:
			t.setClass(s)
		case *Data:
			err = t.addData(s)
		case *Link:
			err = t.addLink(s)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// edge is the first edge with the id, or nil.
func (t *tables) edge(id string) *edge {
	if i := slices.IndexFunc(t.edges, func(e *edge) bool { return e.id == id }); i >= 0 {
		return t.edges[i]
	}
	return nil
}

// vertex is the vertex id, new if need be, first named at at, or nil if id is an edge's:
// addVertex then makes no vertex.
func (t *tables) vertex(id string, at syntax.Pos) *vertex {
	if t.edge(id) != nil {
		return nil
	}
	v := t.byID[id]
	if v == nil {
		v = &vertex{at: at, id: id, label: Label{Text: id, Kind: "text"}}
		t.byID[id] = v
		t.vertices = append(t.vertices, v)
	}
	return v
}

// addVertex is addVertex with what the grammar's vertex rule gives it.
func (t *tables) addVertex(s *Vertex) {
	v := t.vertex(s.ID, s.Pos)
	if v == nil || s.Text == nil {
		return
	}
	v.label = Label{Text: unquote(trim(s.Text.Text)), Kind: s.Text.Kind}
	v.shape, v.at = s.Shape, s.Pos
}

// unquote takes the quotes off s, if it has one at each end. Only the text of a trapezoid
// can: there a " after a blank is not a string.
func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// addData is addVertex with metadata, which goes to the subgraph or the edge of that id
// if there is one.
func (t *tables) addData(d *Data) error {
	m, err := readMetadata(d)
	if err != nil {
		return err
	}
	if t.subgraph(d.ID) != nil {
		return refuse(d, m, "view", "algorithm")
	}
	if t.edge(d.ID) != nil {
		return refuse(d, m, "animate", "animation", "step")
	}
	if err := refuse(d, m, "icon", "img", "form", "pos", "constraint", "w", "h"); err != nil {
		return err
	}
	v := t.byID[d.ID]
	if name := m["shape"]; name != "" {
		shape, ok := shapes[name]
		if !ok {
			return syntax.Unsupported(d.Pos, "shape "+name)
		}
		v.shape = shape
	}
	if text := m["label"]; text != "" {
		v.label, v.at = Label{Text: text, Kind: labelKind(m["labelType"])}, d.Pos
	}
	return nil
}

// refuse is the error for the first of keys that m has, or nil.
func refuse(d *Data, m metadata, keys ...string) error {
	for _, key := range keys {
		if _, ok := m[key]; ok {
			return syntax.Unsupported(d.Pos, key+" in @{...}")
		}
	}
	return nil
}

// labelKind is sanitizeNodeLabelType.
func labelKind(kind string) string {
	if kind == "string" || kind == "text" {
		return kind
	}
	return "markdown"
}

// addLink is addLink: only the last start and first end get the link's own id.
func (t *tables) addLink(l *Link) error {
	id := strings.Replace(l.ID, "@", "", 1)
	for _, start := range l.From {
		for _, end := range l.To {
			mine := ""
			if start == l.From[len(l.From)-1] && end == l.To[0] {
				mine = id
			}
			if err := t.addSingleLink(l, start, end, mine); err != nil {
				return err
			}
		}
	}
	return nil
}

// addSingleLink adds an edge from start to end, with the id if it is free.
func (t *tables) addSingleLink(l *Link, start, end, id string) error {
	e := &edge{at: l.Pos, id: id, start: start, end: end, label: Label{Kind: "text"}, kind: l.Type, stroke: l.Stroke}
	if l.Text != nil {
		e.label = Label{Text: trim(l.Text.Text), Kind: l.Text.Kind}
	}
	if id == "" || t.edge(id) != nil {
		e.id = t.edgeID(start, end)
	}
	if len(t.edges) >= maxEdges {
		return syntax.Errorf(l.Pos, "more than %d links", maxEdges)
	}
	t.edges = append(t.edges, e)
	return nil
}

// edgeID is the id of a new edge from start to end. The second of them is numbered 2.
func (t *tables) edgeID(start, end string) string {
	n := 0
	for _, e := range t.edges {
		if e.start == start && e.end == end {
			n++
		}
	}
	if n > 0 {
		n++
	}
	return fmt.Sprintf("L_%s_%s_%d", start, end, n)
}

// updateLink is updateLinkInterpolate and updateLink.
func (t *tables) updateLink(s *LinkStyle) error {
	var edges []*edge
	for _, p := range s.Positions {
		n, err := strconv.Atoi(p.Text)
		if err != nil || strconv.Itoa(n) != p.Text || n >= len(t.edges) {
			return syntax.Errorf(p.Pos, "no link %s", p.Text)
		}
		edges = append(edges, t.edges[n])
	}
	if s.Styles == nil {
		return nil
	}
	if s.Positions == nil {
		t.defaultStyle = s.Styles
		return nil
	}
	styles := s.Styles
	if !slices.ContainsFunc(styles, func(style string) bool { return strings.HasPrefix(style, "fill") }) {
		styles = append(styles, "fill:none")
	}
	for _, e := range edges {
		e.styles = styles
	}
	return nil
}

// classStyles are s's styles as addClass splits them: a , with a \ before it stays
// in the style it is in.
func classStyles(s *ClassDef) []string {
	joined := strings.ReplaceAll(strings.Join(s.Styles, ","), `\,`, "§§§")
	joined = strings.ReplaceAll(joined, ",", ";")
	return strings.Split(strings.ReplaceAll(joined, "§§§", ","), ";")
}

// addClass is addClass.
func (t *tables) addClass(s *ClassDef) {
	styles := classStyles(s)
	for _, id := range strings.Split(s.IDs, ",") {
		c := t.classes[id]
		if c == nil {
			c = &class{}
			t.classes[id] = c
		}
		for _, style := range styles {
			if strings.Contains(style, "color") {
				c.textStyles = append(c.textStyles, strings.Replace(style, "fill", "bgFill", 1))
			}
			c.styles = append(c.styles, style)
		}
	}
}

// setClass is setClass: an id may be a vertex, an edge and a subgraph at once.
func (t *tables) setClass(s *Class) {
	for _, id := range strings.Split(s.IDs, ",") {
		if v := t.byID[id]; v != nil {
			v.classes = append(v.classes, s.Class)
		}
		if e := t.edge(id); e != nil {
			e.classes = append(e.classes, s.Class)
		}
		if sg := t.subgraph(id); sg != nil {
			sg.classes = append(sg.classes, s.Class)
		}
	}
}
