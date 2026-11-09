package class

import (
	"slices"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// A Stmt is a statement of a diagram.
type Stmt any

// The kinds of end a relation's line has, by addRelation's numbers, and none.
const (
	endNone = iota
	endAggregation
	endExtension
	endComposition
	endDependency
	endLollipop
)

type (
	// A ClassStmt is a class statement. Label, Style and Annotation are "" when unset,
	// save a Label that HasLabel marks.
	ClassStmt struct {
		At                syntax.Pos
		Name, Label       string
		HasLabel          bool
		Style, Annotation string
		Members           []string
	}
	// A MemberStmt is `Foo : text`.
	MemberStmt struct {
		At          syntax.Pos
		Class, Text string
	}
	// An AnnotationStmt is `<<name>> Foo`.
	AnnotationStmt struct {
		At          syntax.Pos
		Class, Name string
	}
	// A RelationStmt is a relation. Title1 and Title2 are "none" when not written, as
	// addRelation's are.
	RelationStmt struct {
		At             syntax.Pos
		Class1, Class2 string
		End1, End2     int
		Dotted         bool
		Title1, Title2 string
		Label          string
		HasLabel       bool
	}
	// A NamespaceStmt is a namespace and what it holds, a Class, Note or Namespace.
	NamespaceStmt struct {
		At          syntax.Pos
		Name, Label string
		Items       []Stmt
	}
	// A NoteStmt is a note, on Class if HasClass.
	NoteStmt struct {
		At       syntax.Pos
		Class    string
		HasClass bool
		Text     string
	}
	// A ClickableStmt is a link, callback or click on Class, of which all that shows
	// is that the class is clickable.
	ClickableStmt struct {
		At    syntax.Pos
		Class string
	}
	// A StyleStmt is a style statement, a CSSClassStmt a cssClass one, and a
	// ClassDefStmt a classDef one.
	StyleStmt struct {
		ID     string
		Styles []string
	}
	CSSClassStmt struct{ IDs, Class string }
	ClassDefStmt struct {
		IDs, Styles []string
	}
	// A DirectionStmt is a direction statement.
	DirectionStmt struct{ Value string }
)

// directions are the values of the kinds of direction statement.
var directions = map[syntax.Kind]string{kDirectionTB: "TB", kDirectionBT: "BT", kDirectionRL: "RL", kDirectionLR: "LR"}

// ends are the kinds of a relation's end.
var ends = map[syntax.Kind]int{kAggregation: endAggregation, kExtension: endExtension, kComposition: endComposition, kDependency: endDependency, kLollipop: endLollipop}

func is(ts []syntax.Token, ks ...syntax.Kind) bool { return slices.Contains(ks, ts[0].Kind) }

func isAlnum(ts []syntax.Token) bool { return is(ts, kUnicode, kAlpha, kMinus) }

func isLine(ts []syntax.Token) bool { return is(ts, kLine, kDottedLine) }

func isEnd(ts []syntax.Token) bool {
	_, ok := ends[ts[0].Kind]
	return ok
}

// Parse reads the tokens of a class diagram.
func Parse(ts []syntax.Token) ([]Stmt, error) {
	_, ts, err := syntax.Expect(ts, kClassDiagram, "classDiagram")
	if err != nil {
		return nil, err
	}
	if _, ts, err = syntax.Expect(ts, syntax.Newline, "a new line"); err != nil {
		return nil, err
	}
	var stmts []Stmt
	for {
		s, rest, err := parseStatement(ts)
		if err != nil {
			return nil, err
		}
		if s != nil {
			stmts = append(stmts, s)
		}
		ts = rest
		if !is(ts, syntax.Newline) {
			break
		}
		ts = ts[1:]
		if is(ts, kEnd) {
			break
		}
	}
	_, _, err = syntax.Expect(ts, kEnd, "the end of the diagram")
	return stmts, err
}

// parseStatement reads a statement. A statement that sets nothing is nil.
func parseStatement(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	t := ts[0]
	if d, ok := directions[t.Kind]; ok {
		return DirectionStmt{d}, ts[1:], nil
	}
	switch t.Kind {
	case kAccTitle:
		_, ts, err := syntax.Expect(ts[1:], kAccTitleValue, "a title")
		return nil, ts, err
	case kAccDescr:
		_, ts, err := syntax.Expect(ts[1:], kAccDescrValue, "a description")
		return nil, ts, err
	case kAccBlockValue:
		return nil, ts[1:], nil
	case kNamespace:
		return parseNamespace(ts)
	case kClass:
		return parseClass(ts)
	case kNote, kNoteFor:
		return parseNote(ts)
	case kAnnotationStart:
		return parseAnnotation(ts)
	case kCallback, kLink, kClick:
		return parseClickable(ts)
	case kStyle:
		return parseStyle(ts)
	case kCSSClass:
		return parseCSSClass(ts)
	case kClassDef:
		return parseClassDef(ts)
	}
	return parseNamed(ts)
}

// parseNamed reads the statements that start with a class's name.
func parseNamed(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	at := ts[0].Pos
	name, ts, err := parseClassName(ts)
	if err != nil {
		return nil, nil, err
	}
	if is(ts, kLabel) {
		return MemberStmt{At: at, Class: name, Text: ts[0].Text}, ts[1:], nil
	}
	if !is(ts, kString) && !isLine(ts) && !isEnd(ts) {
		return nil, ts, nil // a bare name: memberStatement does nothing with it
	}
	return parseRelation(at, name, ts)
}

func parseRelation(at syntax.Pos, name string, ts []syntax.Token) (Stmt, []syntax.Token, error) {
	r := RelationStmt{At: at, Class1: name, Title1: "none", Title2: "none"}
	if is(ts, kString) {
		r.Title1, ts = ts[0].Text, ts[1:]
	}
	if isEnd(ts) {
		r.End1, ts = ends[ts[0].Kind], ts[1:]
	}
	if !isLine(ts) {
		return nil, nil, syntax.Unexpected(ts[0], "a line")
	}
	r.Dotted, ts = ts[0].Kind == kDottedLine, ts[1:]
	if isEnd(ts) {
		r.End2, ts = ends[ts[0].Kind], ts[1:]
	}
	if is(ts, kString) {
		r.Title2, ts = ts[0].Text, ts[1:]
	}
	var err error
	if r.Class2, ts, err = parseClassName(ts); err != nil {
		return nil, nil, err
	}
	if is(ts, kLabel) {
		r.Label, r.HasLabel, ts = ts[0].Text, true, ts[1:]
	}
	return r, ts, nil
}

// parseClassName reads a className, whose alnum parts join with nothing between them.
func parseClassName(ts []syntax.Token) (string, []syntax.Token, error) {
	if is(ts, kBackquote) {
		name, ts := ts[0].Text, ts[1:]
		name, ts = withGeneric(name, ts)
		return name, ts, nil
	}
	if !isAlnum(ts) {
		return "", nil, syntax.Unexpected(ts[0], "a class name")
	}
	name, ts := ts[0].Text, ts[1:]
	switch {
	case is(ts, kDot):
		rest, ts, err := parseClassName(ts[1:])
		return name + "." + rest, ts, err
	case is(ts, kGeneric):
		name, ts = withGeneric(name, ts)
		return name, ts, nil
	case isAlnum(ts) || is(ts, kBackquote):
		rest, ts, err := parseClassName(ts)
		return name + rest, ts, err
	}
	return name, ts, nil
}

func withGeneric(name string, ts []syntax.Token) (string, []syntax.Token) {
	if is(ts, kGeneric) {
		return name + "~" + ts[0].Text + "~", ts[1:]
	}
	return name, ts
}

func parseNamespaceName(ts []syntax.Token) (string, []syntax.Token, error) {
	if is(ts, kBackquote) {
		return ts[0].Text, ts[1:], nil
	}
	if !isAlnum(ts) {
		return "", nil, syntax.Unexpected(ts[0], "a namespace's name")
	}
	name, ts := ts[0].Text, ts[1:]
	switch {
	case is(ts, kDot):
		rest, ts, err := parseNamespaceName(ts[1:])
		return name + "." + rest, ts, err
	case isAlnum(ts) || is(ts, kBackquote):
		rest, ts, err := parseNamespaceName(ts)
		return name + rest, ts, err
	}
	return name, ts, nil
}

// parseClassLabel reads a ["label"], if there is one.
func parseClassLabel(ts []syntax.Token) (string, bool, []syntax.Token, error) {
	if !is(ts, kSquareOpen) {
		return "", false, ts, nil
	}
	s, ts, err := syntax.Expect(ts[1:], kString, "a label")
	if err != nil {
		return "", false, nil, err
	}
	_, ts, err = syntax.Expect(ts, kSquareClose, "]")
	return s.Text, true, ts, err
}

func parseNamespace(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	n := NamespaceStmt{At: ts[0].Pos}
	var err error
	if n.Name, ts, err = parseNamespaceName(ts[1:]); err != nil {
		return nil, nil, err
	}
	if n.Label, _, ts, err = parseClassLabel(ts); err != nil {
		return nil, nil, err
	}
	if _, ts, err = syntax.Expect(ts, kStructStart, "{"); err != nil {
		return nil, nil, err
	}
	if is(ts, syntax.Newline) {
		ts = ts[1:]
	}
	for {
		item, rest, err := parseItem(ts)
		if err != nil {
			return nil, nil, err
		}
		n.Items, ts = append(n.Items, item), rest
		if !is(ts, syntax.Newline) {
			break
		}
		ts = ts[1:]
		if is(ts, kStructStop) {
			break
		}
	}
	_, ts, err = syntax.Expect(ts, kStructStop, "}")
	return n, ts, err
}

func parseItem(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	switch ts[0].Kind {
	case kClass:
		return parseClass(ts)
	case kNote, kNoteFor:
		return parseNote(ts)
	case kNamespace:
		return parseNamespace(ts)
	}
	return nil, nil, syntax.Unexpected(ts[0], "a class, a note or a namespace")
}

func parseClass(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	c := ClassStmt{At: ts[0].Pos}
	var err error
	if c.Name, ts, err = parseClassName(ts[1:]); err != nil {
		return nil, nil, err
	}
	if c.Label, c.HasLabel, ts, err = parseClassLabel(ts); err != nil {
		return nil, nil, err
	}
	switch {
	case is(ts, kStyleSeparator):
		if !isAlnum(ts[1:]) {
			return nil, nil, syntax.Unexpected(ts[1], "a class to apply")
		}
		c.Style, ts = ts[1].Text, ts[2:]
		if is(ts, kStructStart) && is(ts[1:], kStructStop) {
			return nil, nil, syntax.Unexpected(ts[1], "a member")
		}
	case is(ts, kAnnotationStart):
		if c.Annotation, ts, err = parseAnnotationName(ts); err != nil {
			return nil, nil, err
		}
	}
	if !is(ts, kStructStart) {
		return c, ts, nil
	}
	for ts = ts[1:]; is(ts, kMember); ts = ts[1:] {
		c.Members = append(c.Members, ts[0].Text)
	}
	_, ts, err = syntax.Expect(ts, kStructStop, "}")
	return c, ts, err
}

// parseAnnotationName reads <<name>>.
func parseAnnotationName(ts []syntax.Token) (string, []syntax.Token, error) {
	if !isAlnum(ts[1:]) {
		return "", nil, syntax.Unexpected(ts[1], "an annotation")
	}
	name := ts[1].Text
	_, ts, err := syntax.Expect(ts[2:], kAnnotationEnd, ">>")
	return name, ts, err
}

func parseAnnotation(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	at := ts[0].Pos
	name, ts, err := parseAnnotationName(ts)
	if err != nil {
		return nil, nil, err
	}
	class, ts, err := parseClassName(ts)
	if err != nil {
		return nil, nil, err
	}
	return AnnotationStmt{At: at, Class: class, Name: name}, ts, nil
}

func parseNote(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	n := NoteStmt{At: ts[0].Pos}
	var err error
	if forClass := is(ts, kNoteFor); forClass {
		if n.Class, ts, err = parseClassName(ts[1:]); err != nil {
			return nil, nil, err
		}
		n.HasClass = true
	} else {
		ts = ts[1:]
	}
	text, ts, err := syntax.Expect(ts, kString, "the text of a note")
	if err != nil {
		return nil, nil, err
	}
	n.Text = text.Text
	return n, ts, nil
}

// parseClickable reads a callback, link or click statement.
func parseClickable(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	c := ClickableStmt{At: ts[0].Pos}
	kind := ts[0].Kind
	var err error
	if c.Class, ts, err = parseClassName(ts[1:]); err != nil {
		return nil, nil, err
	}
	switch {
	case kind == kCallback:
		if _, ts, err = syntax.Expect(ts, kString, "a function's name"); err != nil {
			return nil, nil, err
		}
		if is(ts, kString) {
			ts = ts[1:]
		}
		return c, ts, nil
	case kind == kLink:
		return linkTail(c, ts)
	case is(ts, kHref):
		return linkTail(c, ts[1:])
	}
	if _, ts, err = syntax.Expect(ts, kCallbackName, "call or href"); err != nil {
		return nil, nil, err
	}
	if is(ts, kCallbackArgs) {
		ts = ts[1:]
	}
	if is(ts, kString) {
		ts = ts[1:]
	}
	return c, ts, nil
}

// linkTail reads a link's URL, tooltip and target.
func linkTail(c ClickableStmt, ts []syntax.Token) (Stmt, []syntax.Token, error) {
	_, ts, err := syntax.Expect(ts, kString, "a link")
	if err != nil {
		return nil, nil, err
	}
	if is(ts, kString) {
		ts = ts[1:]
	}
	if is(ts, kLinkTarget) {
		ts = ts[1:]
	}
	return c, ts, nil
}

func parseStyle(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	id, ts, err := syntax.Expect(ts[1:], kAlpha, "a class")
	if err != nil {
		return nil, nil, err
	}
	styles, ts, err := parseStyles(ts)
	if err != nil {
		return nil, nil, err
	}
	return StyleStmt{ID: id.Text, Styles: styles}, ts, nil
}

func parseCSSClass(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	ids, ts, err := syntax.Expect(ts[1:], kString, "class names")
	if err != nil {
		return nil, nil, err
	}
	class, ts, err := syntax.Expect(ts, kAlpha, "a class to apply")
	if err != nil {
		return nil, nil, err
	}
	return CSSClassStmt{IDs: ids.Text, Class: class.Text}, ts, nil
}

func parseClassDef(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	first, ts, err := syntax.Expect(ts[1:], kAlpha, "a class")
	if err != nil {
		return nil, nil, err
	}
	ids := []string{first.Text}
	for is(ts, kComma) {
		var id syntax.Token
		if id, ts, err = syntax.Expect(ts[1:], kAlpha, "a class"); err != nil {
			return nil, nil, err
		}
		ids = append(ids, id.Text)
	}
	styles, ts, err := parseStyles(ts)
	if err != nil {
		return nil, nil, err
	}
	return ClassDefStmt{IDs: ids, Styles: styles}, ts, nil
}

func isStylePart(ts []syntax.Token) bool {
	return is(ts, kAlpha, kColon, kBrkt, kStyle, kPct, kLabel)
}

func parseStyles(ts []syntax.Token) ([]string, []syntax.Token, error) {
	var styles []string
	for {
		if !isStylePart(ts) {
			return nil, nil, syntax.Unexpected(ts[0], "a style")
		}
		s := ""
		for ; isStylePart(ts); ts = ts[1:] {
			s += ts[0].Text
		}
		styles = append(styles, s)
		if !is(ts, kComma) {
			return styles, ts, nil
		}
		ts = ts[1:]
	}
}
