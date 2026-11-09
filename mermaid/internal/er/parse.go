package er

import (
	"slices"
	"strings"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// A Stmt is a statement of a diagram.
type Stmt any

type (
	// An EntityStmt is an entity on its own, with what it says of it.
	EntityStmt struct {
		At          syntax.Pos
		Name, Alias string
		Classes     []string
		Attributes  []Attribute
	}
	// A RelationStmt is a relationship. Start and End are the cardinalities written
	// beside A and beside B.
	RelationStmt struct {
		At                 syntax.Pos
		A, B, Role         string
		ClassesA, ClassesB []string
		Start, End         string
		Identifying        bool
	}
	// A DirectionStmt is a direction statement.
	DirectionStmt struct{ Value string }
	// An AccStmt is an accessible title or description, which only a subgraph looks at.
	AccStmt struct{ Value string }
	// A ClassDefStmt, ClassStmt and StyleStmt keep Word, their keyword as written, which
	// a subgraph holding them takes for the name of something in it.
	ClassDefStmt struct {
		Word        string
		IDs, Styles []string
	}
	ClassStmt struct {
		Word         string
		IDs, Classes []string
	}
	StyleStmt struct {
		Word        string
		IDs, Styles []string
	}
	// A SubgraphStmt is a subgraph and the statements in it.
	SubgraphStmt struct {
		At        syntax.Pos
		ID, Title string
		Body      []Stmt
	}
)

// directions are the values of the kinds of direction statement.
var directions = map[syntax.Kind]string{kDirectionTB: "TB", kDirectionBT: "BT", kDirectionRL: "RL", kDirectionLR: "LR"}

// cardinalities are the kinds of cardinality, by the arrow type Mermaid makes of each.
var cardinalities = map[syntax.Kind]string{
	kOnlyOne:    "only_one",
	kZeroOrOne:  "zero_or_one",
	kZeroOrMore: "zero_or_more",
	kOneOrMore:  "one_or_more",
	kMdParent:   "md_parent",
}

func is(ts []syntax.Token, ks ...syntax.Kind) bool { return slices.Contains(ks, ts[0].Kind) }

func isName(ts []syntax.Token) bool {
	return is(ts, kEntityName, kUnicodeText, kNum, kDecimal, kEntityOne)
}

func unquote(s string) string { return strings.ReplaceAll(s, `"`, "") }

// Parse reads the tokens of a diagram.
func Parse(ts []syntax.Token) ([]Stmt, error) {
	_, ts, err := syntax.Expect(ts, kDiagram, "erDiagram")
	if err != nil {
		return nil, err
	}
	stmts, _, err := parseLines(ts, kDone)
	return stmts, err
}

// parseLines reads statements up to and past a token of kind stop.
func parseLines(ts []syntax.Token, stop syntax.Kind) ([]Stmt, []syntax.Token, error) {
	var stmts []Stmt
	for !is(ts, stop) {
		if is(ts, syntax.Newline) {
			ts = ts[1:]
			continue
		}
		s, rest, err := parseStatement(ts)
		if err != nil {
			return nil, nil, err
		}
		stmts, ts = append(stmts, s), rest
	}
	return stmts, ts[1:], nil
}

func parseStatement(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	t := ts[0]
	if d, ok := directions[t.Kind]; ok {
		return DirectionStmt{d}, ts[1:], nil
	}
	switch t.Kind {
	case kAccTitle, kAccDescr:
		return AccStmt{trim(ts[1].Text)}, ts[2:], nil
	case kAccMultiValue:
		return AccStmt{trim(t.Text)}, ts[1:], nil
	case kClassDef:
		ids, styles, rest, err := parseStyling(ts[1:])
		return ClassDefStmt{t.Text, ids, styles}, rest, err
	case kStyle:
		ids, styles, rest, err := parseStyling(ts[1:])
		return StyleStmt{t.Text, ids, styles}, rest, err
	case kClass:
		ids, rest, err := parseIDs(ts[1:])
		if err != nil {
			return nil, nil, err
		}
		classes, rest, err := parseIDs(rest)
		return ClassStmt{t.Text, ids, classes}, rest, err
	case kSubgraph:
		return parseSubgraph(ts)
	}
	if isName(ts) {
		return parseNamed(ts)
	}
	if t.Kind == kDone {
		t.Kind = syntax.EOF
	}
	return nil, nil, syntax.Unexpected(t, "a statement")
}

// parseName reads a name, called what in an error.
func parseName(ts []syntax.Token, what string) (string, []syntax.Token, error) {
	if !isName(ts) {
		return "", nil, syntax.Unexpected(ts[0], what)
	}
	return unquote(ts[0].Text), ts[1:], nil
}

// parseNamed reads the statements that start with a name.
func parseNamed(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	e := EntityStmt{At: ts[0].Pos, Name: unquote(ts[0].Text)}
	ts = ts[1:]
	var err error
	aliased := is(ts, kSquareOpen)
	if aliased {
		if e.Alias, ts, err = parseName(ts[1:], "an alias"); err != nil {
			return nil, nil, err
		}
		if _, ts, err = syntax.Expect(ts, kSquareClose, `"]"`); err != nil {
			return nil, nil, err
		}
	}
	if e.Classes, ts, err = parseClasses(ts); err != nil {
		return nil, nil, err
	}
	if _, ok := cardinalities[ts[0].Kind]; ok && !aliased {
		return parseRelation(e, ts)
	}
	if is(ts, kBlockStart) {
		e.Attributes, ts, err = parseAttributes(ts[1:])
		if err != nil {
			return nil, nil, err
		}
	}
	return e, ts, nil
}

func parseClasses(ts []syntax.Token) ([]string, []syntax.Token, error) {
	if !is(ts, kStyleSeparator) {
		return nil, ts, nil
	}
	return parseIDs(ts[1:])
}

// parseRelation reads the rest of a relationship from its first cardinality on.
func parseRelation(e EntityStmt, ts []syntax.Token) (Stmt, []syntax.Token, error) {
	r := RelationStmt{At: e.At, A: e.Name, ClassesA: e.Classes, Start: cardinalities[ts[0].Kind]}
	ts = ts[1:]
	if !is(ts, kIdentifying, kNonIdentifying) {
		return nil, nil, syntax.Unexpected(ts[0], "a line")
	}
	r.Identifying = is(ts, kIdentifying)
	ts = ts[1:]
	end, ok := cardinalities[ts[0].Kind]
	if !ok {
		return nil, nil, syntax.Unexpected(ts[0], "a cardinality")
	}
	r.End = end
	b, ts, err := parseName(ts[1:], "the name of an entity")
	if err != nil {
		return nil, nil, err
	}
	r.B = b
	if r.ClassesB, ts, err = parseClasses(ts); err != nil {
		return nil, nil, err
	}
	if _, ts, err = syntax.Expect(ts, kColon, `":"`); err != nil {
		return nil, nil, err
	}
	if !is(ts, kWord, kEntityName, kUnicodeText) {
		return nil, nil, syntax.Unexpected(ts[0], "the text of a relationship")
	}
	r.Role = unquote(ts[0].Text)
	return r, ts[1:], nil
}

// parseAttributes reads the attributes of a block and its end.
func parseAttributes(ts []syntax.Token) ([]Attribute, []syntax.Token, error) {
	var as []Attribute
	for !is(ts, kBlockStop) {
		a, rest, err := parseAttribute(ts)
		if err != nil {
			return nil, nil, err
		}
		as, ts = append(as, a), rest
	}
	return as, ts[1:], nil
}

func isChar(ts []syntax.Token, c string) bool { return is(ts, kChar) && ts[0].Text == c }

func parseAttribute(ts []syntax.Token) (Attribute, []syntax.Token, error) {
	t, ts, err := syntax.Expect(ts, kAttributeWord, "the type of an attribute")
	if err != nil {
		return Attribute{}, nil, err
	}
	a := Attribute{Type: t.Text}
	if isChar(ts, "?") {
		a.Type += "?"
		ts = ts[1:]
	}
	if t, ts, err = syntax.Expect(ts, kAttributeWord, "the name of an attribute"); err != nil {
		return Attribute{}, nil, err
	}
	a.Name = t.Text
	if is(ts, kAttributeKey) {
		a.Keys = append(a.Keys, ts[0].Text)
		ts = ts[1:]
		for isChar(ts, ",") {
			if t, ts, err = syntax.Expect(ts[1:], kAttributeKey, "PK, FK or UK"); err != nil {
				return Attribute{}, nil, err
			}
			a.Keys = append(a.Keys, t.Text)
		}
	}
	if is(ts, kComment) {
		a.Comment = unquote(ts[0].Text)
		ts = ts[1:]
	}
	return a, ts, nil
}

func parseIDs(ts []syntax.Token) ([]string, []syntax.Token, error) {
	var ids []string
	for {
		if !is(ts, kUnicodeText, kStyleText) {
			return nil, nil, syntax.Unexpected(ts[0], "a name")
		}
		ids = append(ids, ts[0].Text)
		ts = ts[1:]
		if !is(ts, kComma) {
			return ids, ts, nil
		}
		ts = ts[1:]
	}
}

// parseStyling reads the names and the styles of a style or classDef statement.
func parseStyling(ts []syntax.Token) (ids, styles []string, _ []syntax.Token, err error) {
	if ids, ts, err = parseIDs(ts); err != nil {
		return nil, nil, nil, err
	}
	for {
		if !is(ts, kStyleText, kColon, kBrkt) {
			return nil, nil, nil, syntax.Unexpected(ts[0], "a style")
		}
		var b strings.Builder
		for is(ts, kStyleText, kColon, kBrkt) {
			b.WriteString(ts[0].Text)
			ts = ts[1:]
		}
		styles = append(styles, b.String())
		if !is(ts, kComma) {
			break
		}
		ts = ts[1:]
	}
	ts, err = parseSeparator(ts)
	return ids, styles, ts, err
}

func parseSeparator(ts []syntax.Token) ([]syntax.Token, error) {
	if !is(ts, syntax.Newline, kSemi) {
		return nil, syntax.Unexpected(ts[0], "the end of the line")
	}
	return ts[1:], nil
}

func parseSubgraph(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	s := SubgraphStmt{At: ts[0].Pos}
	id, ts, err := parseName(ts[1:], "the name of a subgraph")
	if err != nil {
		return nil, nil, err
	}
	s.ID, s.Title = id, id
	if is(ts, kSquareOpen) {
		var words []string
		for ts = ts[1:]; len(words) == 0 || isName(ts); {
			var w string
			if w, ts, err = parseName(ts, "a title"); err != nil {
				return nil, nil, err
			}
			words = append(words, w)
		}
		if _, ts, err = syntax.Expect(ts, kSquareClose, `"]"`); err != nil {
			return nil, nil, err
		}
		s.Title = strings.Join(words, " ")
	}
	if ts, err = parseSeparator(ts); err != nil {
		return nil, nil, err
	}
	if s.Body, ts, err = parseLines(ts, kEnd); err != nil {
		return nil, nil, err
	}
	return s, ts, nil
}
