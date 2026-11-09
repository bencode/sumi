package state

import (
	"fmt"
	"slices"
	"strings"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

const (
	noteSpacer   = "----note"
	parentSpacer = "----parent"
)

// An entry is what nodeDb keeps of a state between its mentions.
type entry struct {
	shape Shape
	text  string     // its description while there is one
	list  []string   // its descriptions once there are several
	at    syntax.Pos // where the last description was written, or else its first mention
	group bool
	dir   string
	alt   bool
}

// A fetcher gathers nodes and edges in the order dataFetcher does.
type fetcher struct {
	t     tables
	seen  map[string]*entry
	index map[string]int
	at    map[string]syntax.Pos
	nodes []Node
	edges []Edge
	count int // graphItemCount
}

// fetch is the diagram of doc: dataFetcher, the labels StateDB.extract takes from
// lists, and getData.
func fetch(doc []Stmt, t tables) (*Diagram, error) {
	f := &fetcher{t: t, seen: map[string]*entry{}, index: map[string]int{}, at: map[string]syntax.Pos{}}
	f.state("", State{ID: "root", Doc: doc}, true)
	for _, n := range f.nodes {
		if n.Group && len(n.Descriptions) > 0 {
			return nil, syntax.Errorf(f.at[n.ID], "a group can only have a label, and %q has more", n.ID)
		}
	}
	if err := f.checkParents(); err != nil {
		return nil, err
	}
	return &Diagram{Direction: dirOf(doc), Nodes: f.nodes, Edges: f.edges}, nil
}

// checkParents refuses a state inside itself. Mermaid's layout never comes back from
// one.
func (f *fetcher) checkParents() error {
	for _, n := range f.nodes {
		at := n
		for range f.nodes {
			i, ok := f.index[at.Parent]
			if !ok {
				break
			}
			at = f.nodes[i]
			if at.ID == n.ID {
				return syntax.Errorf(f.at[n.ID], "state %q is inside itself", n.ID)
			}
		}
	}
	return nil
}

// dirOf is getDir: the last direction of doc, or TB.
func dirOf(doc []Stmt) string {
	dir := "TB"
	for _, s := range doc {
		if d, ok := s.(Direction); ok {
			dir = d.Value
		}
	}
	return dir
}

func (f *fetcher) doc(parent string, doc []Stmt, alt bool) {
	for _, s := range doc {
		switch v := s.(type) {
		case State:
			f.state(parent, v, alt)
		case Relation:
			f.state(parent, v.State1, alt)
			f.state(parent, v.State2, alt)
			f.edges = append(f.edges, Edge{ID: fmt.Sprintf("edge%d", f.count), Start: v.State1.ID, End: v.State2.ID, Label: v.Desc, At: v.State1.At})
			f.count++
		}
	}
}

// shapeOf is the shape a state's first mention gives it.
func shapeOf(s State) Shape {
	if s.Type != "default" {
		return map[string]Shape{"fork": Fork, "join": Join, "choice": Choice, "divider": Divider}[s.Type]
	}
	return map[int]Shape{notPseudo: Rect, pseudoStart: Start, pseudoEnd: End}[s.Pseudo]
}

// describe adds s's descriptions to e. A second description makes a state's first
// its title, unless the first was only its id.
func (e *entry) describe(s State) {
	if len(s.Desc) == 0 {
		return
	}
	e.at = s.At
	switch {
	case e.list != nil:
		e.shape, e.list = RectWithTitle, append(e.list, s.Desc...)
	case e.text == s.ID && e.text != "":
		e.shape, e.list = RectWithTitle, slices.Clone(s.Desc)
	case e.text != "":
		e.shape, e.list = RectWithTitle, append([]string{e.text}, s.Desc...)
	case len(s.Desc) > 1:
		e.shape, e.list = Rect, slices.Clone(s.Desc)
	default:
		e.shape, e.text = Rect, s.Desc[0]
	}
}

// state is dataFetcher for s, inside the state parent, "" at the root. A state of
// the id root is the root, so it is left out and its body is at the root too.
func (f *fetcher) state(parent string, s State, alt bool) {
	inner := s.ID
	if s.ID == "root" {
		inner = ""
	} else {
		f.insertState(parent, s, alt)
	}
	if s.Doc != nil {
		f.doc(inner, s.Doc, !alt)
	}
}

func (f *fetcher) insertState(parent string, s State, alt bool) {
	e := f.seen[s.ID]
	if e == nil {
		e = &entry{shape: shapeOf(s), text: s.ID, at: s.At}
		f.seen[s.ID] = e
	}
	e.describe(s)
	if len(e.list) == 1 && e.shape == RectWithTitle {
		e.shape = Rect
		if e.group {
			e.shape = Composite
		}
	}
	if !e.group && s.Doc != nil {
		e.group, e.dir, e.alt, e.shape = true, dirOf(s.Doc), alt, Composite
		if s.Type == "divider" {
			e.shape = Divider
		}
	}
	n := Node{
		ID: s.ID, Parent: parent, Shape: e.shape, Group: e.group, Dir: e.dir, Alt: e.alt,
		Styles: f.t.styles[s.ID], Compiled: f.compiled(s.ID), At: e.at,
	}
	n.Label, n.Descriptions = e.label()
	if n.Shape == Divider {
		n.Label = ""
	}
	if f.at[s.ID] == (syntax.Pos{}) {
		f.at[s.ID] = s.At
	}
	if s.Note != nil {
		f.insertNote(s, n)
		return
	}
	f.insert(n)
}

func (e *entry) label() (string, []string) {
	if e.list == nil {
		return e.text, nil
	}
	return e.list[0], e.list[1:]
}

// compiled are the styles of the classes of the state id (cssCompiledStyles).
func (f *fetcher) compiled(id string) []string {
	var out []string
	for _, c := range strings.Split(strings.Join(f.t.classes[id], " "), " ") {
		out = append(out, f.t.classDefs[c]...)
	}
	return out
}

// insertNote puts n, the state s notes, after the note and the group that holds it,
// and joins the note to the state with a line that goes the way the note is.
func (f *fetcher) insertNote(s State, n Node) {
	note := fmt.Sprintf("%s%s-%d", s.ID, noteSpacer, f.count)
	group := s.ID + parentSpacer
	f.count++
	f.insert(Node{ID: group, Label: s.Note.Text, Shape: NoteGroup, Group: true, Position: s.Note.Position, Compiled: n.Compiled, At: s.At})
	f.insert(Node{ID: note, Parent: group, Label: s.Note.Text, Shape: Note, Position: s.Note.Position, At: s.At})
	f.insert(n)
	from, to := s.ID, note
	if s.Note.Position == "left of" {
		from, to = note, s.ID
	}
	f.edges = append(f.edges, Edge{ID: from + "-" + to, Start: from, End: to, Note: true, At: s.At})
}

// insert is insertOrUpdateNode: a node already there takes the new one's fields where
// it stands, all but a parent the new one does not set.
func (f *fetcher) insert(n Node) {
	if n.ID == "" || n.ID == "</join></fork>" || n.ID == "</choice>" {
		return
	}
	i, ok := f.index[n.ID]
	if !ok {
		f.index[n.ID] = len(f.nodes)
		f.nodes = append(f.nodes, n)
		return
	}
	if n.Parent == "" {
		n.Parent = f.nodes[i].Parent
	}
	f.nodes[i] = n
}
