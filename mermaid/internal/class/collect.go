package class

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// markers are getArrowMarker's names, by a relation's end.
var markers = map[int]string{
	endNone: "none", endAggregation: "aggregation", endExtension: "extension",
	endComposition: "composition", endDependency: "dependency", endLollipop: "lollipop",
}

// startsWithDigit is the test setCssClass makes of an id, which then looks up another.
var startsWithDigit = regexp.MustCompile(`^\d`)

// A db is the ClassDB of one diagram, filled statement by statement. Maps of ids keep
// the place in the slice that holds each, as a Map's order is insertion order.
type db struct {
	namespaces  []Node
	namespaceOf map[string]int
	classes     []Node
	classOf     map[string]int
	types       map[string]string // a class's generic type, for its label
	notes       []Node
	noteClass   []*string
	interfaces  []Node
	relations   []RelationStmt
	direction   string
}

// collect runs the statements through a db, and gives what getData would of it.
func collect(stmts []Stmt) (*Diagram, error) {
	d := &db{namespaceOf: map[string]int{}, classOf: map[string]int{}, types: map[string]string{}, direction: "TB"}
	for _, s := range stmts {
		if err := d.statement(s); err != nil {
			return nil, err
		}
	}
	return d.diagram(), nil
}

func (d *db) statement(s Stmt) error {
	switch v := s.(type) {
	case ClassStmt:
		_, err := d.class(v)
		return err
	case MemberStmt:
		return d.memberStatement(v)
	case AnnotationStmt:
		return d.annotation(v)
	case RelationStmt:
		return d.addRelation(v)
	case NamespaceStmt:
		return d.namespace(v, "")
	case NoteStmt:
		d.addNote(v)
	case ClickableStmt:
		d.setCSSClass(v.Class, "clickable")
	case StyleStmt:
		d.setCSSStyle(v)
	case CSSClassStmt:
		d.setCSSClass(v.IDs, v.Class)
	case ClassDefStmt:
		d.defineClass(v)
	case DirectionStmt:
		d.direction = v.Value
	}
	return nil
}

// splitName is splitClassNameAndType.
func splitName(id string) (name, typ string) {
	if strings.Index(id, "~") > 0 {
		parts := strings.Split(id, "~")
		return parts[0], parts[1]
	}
	return id, ""
}

// refuseAngle stops text with a < in it, where DOMPurify would take a tag from it, and
// so change what is read of it. Without a <, sanitizeText leaves a text as it is.
func refuseAngle(at syntax.Pos, what, s string) error {
	if strings.Contains(s, "<") {
		return syntax.Unsupported(at, "a < in "+what)
	}
	return nil
}

func (d *db) addClass(id string, at syntax.Pos) error {
	if err := refuseAngle(at, "the name of a class", id); err != nil {
		return err
	}
	name, typ := splitName(id)
	if _, ok := d.classOf[name]; ok {
		return nil
	}
	label := name
	if typ != "" {
		label += "&lt;" + typ + "&gt;"
	}
	d.classOf[name] = len(d.classes)
	d.classes = append(d.classes, Node{ID: name, Kind: Class, Label: label, Classes: "default", At: at})
	d.types[name] = typ
	return nil
}

// byID is the class of a class name, its generic type aside, or nil.
func (d *db) byID(id string) *Node {
	name, _ := splitName(id)
	if i, ok := d.classOf[name]; ok {
		return &d.classes[i]
	}
	return nil
}

// class runs a class statement, and gives the name it was written with.
func (d *db) class(s ClassStmt) (string, error) {
	if err := d.addClass(s.Name, s.At); err != nil {
		return "", err
	}
	if s.HasLabel {
		if err := refuseAngle(s.At, "the label of a class", s.Label); err != nil {
			return "", err
		}
		c := d.byID(s.Name)
		c.Label = s.Label
		if t := d.types[c.ID]; t != "" { // setClassLabel writes these as they are, unlike addClass
			c.Label += "<" + t + ">"
		}
	}
	if s.Style != "" {
		d.setCSSClass(s.Name, s.Style)
	}
	if s.Annotation != "" {
		c := d.byID(s.Name)
		c.Annotations = append(c.Annotations, s.Annotation)
	}
	for _, m := range s.Members {
		if err := d.addMember(s.Name, m, s.At); err != nil {
			return "", err
		}
	}
	return s.Name, nil
}

// memberStatement is `Foo : text`, whose text cleanupLabel has sanitized, unlike a
// class body's lines, so that a < is refused even in an <<annotation>>.
func (d *db) memberStatement(s MemberStmt) error {
	if err := refuseAngle(s.At, "a member", s.Text); err != nil {
		return err
	}
	return d.addMember(s.Class, cleanupLabel(s.Text), s.At)
}

func cleanupLabel(s string) string { return trim(strings.TrimPrefix(s, ":")) }

// addMember is addMember: an <<annotation>>, a method if a ) stands after its start,
// or an attribute.
func (d *db) addMember(class, member string, at syntax.Pos) error {
	if err := d.addClass(class, at); err != nil {
		return err
	}
	c := d.byID(class)
	s := trim(member)
	if strings.HasPrefix(s, "<<") && strings.HasSuffix(s, ">>") {
		inner := s[2 : len(s)-2]
		if err := refuseAngle(at, "an annotation", inner); err != nil {
			return err
		}
		c.Annotations = append(c.Annotations, inner)
		return nil
	}
	if s == "" {
		return nil
	}
	if err := refuseAngle(at, "a member", s); err != nil {
		return err
	}
	method := strings.Index(s, ")") > 0
	m, err := newMember(s, method, at)
	if err != nil {
		return err
	}
	if method {
		c.Methods = append(c.Methods, m)
	} else {
		c.Members = append(c.Members, m)
	}
	return nil
}

// annotation is addAnnotation, which throws on a class that does not exist yet.
func (d *db) annotation(s AnnotationStmt) error {
	c := d.byID(s.Class)
	if c == nil {
		return syntax.Errorf(s.At, "no class %s to annotate", s.Class)
	}
	c.Annotations = append(c.Annotations, s.Name)
	return nil
}

// addRelation is addRelation. A lollipop with nothing at its other end has an
// interface there in place of a class.
func (d *db) addRelation(r RelationStmt) error {
	switch {
	case r.End1 == endLollipop && r.End2 == endNone:
		if err := d.addClass(r.Class2, r.At); err != nil {
			return err
		}
		r.Class1 = d.addInterface(r.Class1, r.At)
	case r.End2 == endLollipop && r.End1 == endNone:
		if err := d.addClass(r.Class1, r.At); err != nil {
			return err
		}
		r.Class2 = d.addInterface(r.Class2, r.At)
	default:
		if err := d.addClass(r.Class1, r.At); err != nil {
			return err
		}
		if err := d.addClass(r.Class2, r.At); err != nil {
			return err
		}
	}
	r.Class1, _ = splitName(r.Class1)
	r.Class2, _ = splitName(r.Class2)
	r.Title1, r.Title2 = trim(r.Title1), trim(r.Title2)
	d.relations = append(d.relations, r)
	return nil
}

func (d *db) addInterface(label string, at syntax.Pos) string {
	id := fmt.Sprintf("interface%d", len(d.interfaces))
	d.interfaces = append(d.interfaces, Node{ID: id, Kind: Interface, Label: label, At: at})
	return id
}

// addNote is addNote, and gives the note's place in notes.
func (d *db) addNote(n NoteStmt) int {
	i := len(d.notes)
	d.notes = append(d.notes, Node{ID: fmt.Sprintf("note%d", i), Kind: Note, Label: n.Text, At: n.At})
	var class *string
	if n.HasClass {
		class = &n.Class
	}
	d.noteClass = append(d.noteClass, class)
	return i
}

// setCSSClass is setCssClass. An id that starts with a digit is looked up with a
// prefix, which no class has.
func (d *db) setCSSClass(ids, class string) {
	for _, id := range strings.Split(ids, ",") {
		if startsWithDigit.MatchString(id) {
			continue
		}
		if c := d.byID(id); c != nil {
			c.Classes += " " + class
		}
	}
}

func (d *db) setCSSStyle(s StyleStmt) {
	i, ok := d.classOf[s.ID]
	if !ok {
		return
	}
	for _, style := range s.Styles {
		d.classes[i].Styles = append(d.classes[i].Styles, strings.Split(style, ",")...)
	}
}

// defineClass is defineClass: it copies the styles to the classes that have the name
// by then, and to any whose cssClasses contain it as part of a word.
func (d *db) defineClass(s ClassDefStmt) {
	var styles []string
	for _, style := range s.Styles {
		styles = append(styles, strings.Split(style, ",")...)
	}
	for _, id := range s.IDs {
		for i := range d.classes {
			if strings.Contains(d.classes[i].Classes, id) {
				d.classes[i].Styles = append(d.classes[i].Styles, styles...)
			}
		}
	}
}

// namespace is a namespace and what it holds; enclosing is the id of the one around it.
func (d *db) namespace(s NamespaceStmt, enclosing string) error {
	id := s.Name
	if enclosing != "" {
		id = enclosing + "." + s.Name
	}
	d.addNamespace(id, s.Label, s.At)
	var classes []string
	var notes []int
	for _, item := range s.Items {
		switch v := item.(type) {
		case ClassStmt:
			name, err := d.class(v)
			if err != nil {
				return err
			}
			classes = append(classes, name)
		case NoteStmt:
			notes = append(notes, d.addNote(v))
		case NamespaceStmt:
			if err := d.namespace(v, id); err != nil {
				return err
			}
		}
	}
	for _, name := range classes {
		d.byID(name).Parent = id
	}
	for _, i := range notes {
		d.notes[i].Parent = id
	}
	return nil
}

// addNamespace is addNamespace: every dotted prefix of id is a namespace, named by its
// last part but for label on id itself, which a later statement may set.
func (d *db) addNamespace(id, label string, at syntax.Pos) {
	if i, ok := d.namespaceOf[id]; ok {
		if label != "" {
			d.namespaces[i].Label = label
		}
		return
	}
	parts := strings.Split(id, ".")
	current := ""
	for i, part := range parts {
		parent := current
		if i > 0 {
			current += "."
		}
		current += part
		name := part
		if i == len(parts)-1 && label != "" {
			name = label
		}
		if _, ok := d.namespaceOf[current]; !ok {
			d.namespaceOf[current] = len(d.namespaces)
			d.namespaces = append(d.namespaces, Node{ID: current, Parent: parent, Kind: Namespace, Label: name, At: at})
		}
	}
}

// diagram is getData.
func (d *db) diagram() *Diagram {
	out := &Diagram{Direction: d.direction}
	out.Nodes = append(append(append(append(out.Nodes, d.namespaces...), d.classes...), d.notes...), d.interfaces...)
	for i, n := range d.notes {
		if c := d.noteClass[i]; c != nil {
			if j, ok := d.classOf[*c]; ok {
				out.Edges = append(out.Edges, Edge{ID: fmt.Sprintf("edgeNote%d", i), Start: n.ID, End: d.classes[j].ID,
					StartMarker: "none", EndMarker: "none", Pattern: "dotted", At: n.At})
			}
		}
	}
	for i, r := range d.relations {
		e := Edge{ID: fmt.Sprintf("id_%s_%s_%d", r.Class1, r.Class2, i+1), Start: r.Class1, End: r.Class2,
			StartMarker: markers[r.End1], EndMarker: markers[r.End2], Pattern: "solid", At: r.At}
		if r.Dotted {
			e.Pattern = "dashed"
		}
		if r.HasLabel {
			e.Label = cleanupLabel(r.Label)
		}
		if r.Title1 != "none" {
			e.StartLabel = r.Title1
		}
		if r.Title2 != "none" {
			e.EndLabel = r.Title2
		}
		out.Edges = append(out.Edges, e)
	}
	return out
}
