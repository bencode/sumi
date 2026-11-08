package state

import (
	"fmt"
	"strings"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// A Stmt is a statement of a document.
type Stmt any

// The way a [*] goes, from where it stands on its arrow.
const (
	notPseudo = iota
	pseudoStart
	pseudoEnd
)

// A State is a state statement. A Type of "" is a note's, which sets none.
type State struct {
	At      syntax.Pos
	ID      string
	Type    string   // default, fork, join, choice or divider
	Desc    []string // none, one description, or the two `state "a" as id:b` gives
	Doc     []Stmt   // nil unless it has a body
	Note    *StateNote
	Pseudo  int
	Classes []string
}

// A StateNote is a note's text and where it sits.
type StateNote struct{ Position, Text string }

// A Relation is a transition.
type Relation struct {
	State1, State2 State
	Desc           string
}

// A ClassDef is a classDef statement, a Style a style statement, and an ApplyClass
// a class statement.
type (
	ClassDef   struct{ ID, Styles string }
	Style      struct{ ID, Styles string }
	ApplyClass struct{ IDs, Class string }
)

// A Direction is a direction statement.
type Direction struct{ Value string }

// An Ignored is a statement that sets nothing but still fills a region of a `--`:
// `state a` without a body, a floating note, hide empty description, scale.
type Ignored struct{}

// Parse reads the tokens of a state diagram.
func Parse(ts []syntax.Token) ([]Stmt, error) {
	for is(ts, syntax.Newline) {
		ts = ts[1:]
	}
	_, ts, err := syntax.Expect(ts, kSD, "stateDiagram")
	if err != nil {
		return nil, err
	}
	doc, _, err := parseDocument(numberDividers(ts), syntax.EOF)
	return doc, err
}

// numberDividers names each -- as getDividerId does, in the order they come.
func numberDividers(ts []syntax.Token) []syntax.Token {
	out := make([]syntax.Token, len(ts))
	n := 0
	for i, t := range ts {
		if t.Kind == kConcurrent {
			n++
			t.Text = fmt.Sprintf("divider-id-%d", n)
		}
		out[i] = t
	}
	return out
}

func is(ts []syntax.Token, kinds ...syntax.Kind) bool {
	for _, k := range kinds {
		if ts[0].Kind == k {
			return true
		}
	}
	return false
}

// parseDocument reads statements up to a token of kind stop, which it leaves.
func parseDocument(ts []syntax.Token, stop syntax.Kind) ([]Stmt, []syntax.Token, error) {
	doc := []Stmt{}
	for !is(ts, stop) {
		if is(ts, syntax.Newline) {
			ts = ts[1:]
			continue
		}
		s, rest, err := parseStatement(ts)
		if err != nil {
			return nil, nil, err
		}
		doc, ts = append(doc, s), rest
	}
	return doc, ts, nil
}

func parseStruct(ts []syntax.Token) ([]Stmt, []syntax.Token, error) {
	doc, ts, err := parseDocument(ts[1:], kStructStop)
	if err != nil {
		return nil, nil, err
	}
	return doc, ts[1:], nil
}

var pseudoTypes = map[syntax.Kind]string{kFork: "fork", kJoin: "join", kChoice: "choice"}

func parseStatement(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	t := ts[0]
	switch t.Kind {
	case kClassDef:
		return parsePair(ts, kClassDefID, kClassDefStyles, func(a, b string) Stmt { return ClassDef{a, b} })
	case kStyle:
		return parsePair(ts, kStyleIDs, kStyleStyles, func(a, b string) Stmt { return Style{a, b} })
	case kClass:
		return parsePair(ts, kClassIDs, kClassName, func(a, b string) Stmt { return ApplyClass{a, b} })
	case kID, kEdgeState:
		return parseIDStatement(ts)
	case kHideEmpty, kAccBlockValue:
		return Ignored{}, ts[1:], nil
	case kScale:
		_, ts, err := syntax.Expect(ts[1:], kWidth, "a width")
		return Ignored{}, ts, err
	case kComposite:
		if !is(ts[1:], kStructStart) {
			return Ignored{}, ts[1:], nil
		}
		doc, rest, err := parseStruct(ts[1:])
		return State{At: t.Pos, ID: t.Text, Type: "default", Doc: doc}, rest, err
	case kStateDescr:
		return parseDescribed(ts)
	case kFork, kJoin, kChoice:
		return State{At: t.Pos, ID: t.Text, Type: pseudoTypes[t.Kind]}, ts[1:], nil
	case kConcurrent:
		return State{At: t.Pos, ID: t.Text, Type: "divider"}, ts[1:], nil
	case kNote:
		return parseNote(ts)
	case kDirection:
		return Direction{t.Text}, ts[1:], nil
	case kAccTitle, kAccDescr:
		// the rules that make their values match anything, so a value always follows
		return Ignored{}, ts[2:], nil
	case kClick:
		return parseClick(ts)
	}
	return nil, nil, syntax.Unexpected(t, "a statement")
}

// parsePair reads the two tokens after a keyword, trimmed, into a statement.
func parsePair(ts []syntax.Token, a, b syntax.Kind, stmt func(a, b string) Stmt) (Stmt, []syntax.Token, error) {
	x, ts, err := syntax.Expect(ts[1:], a, "a name")
	if err != nil {
		return nil, nil, err
	}
	y, ts, err := syntax.Expect(ts, b, "its value")
	if err != nil {
		return nil, nil, err
	}
	return stmt(trim(x.Text), trim(y.Text)), ts, nil
}

// parseID reads an idStatement.
func parseID(ts []syntax.Token) (State, []syntax.Token, error) {
	if !is(ts, kID, kEdgeState) {
		return State{}, nil, syntax.Unexpected(ts[0], "a state")
	}
	s := State{At: ts[0].Pos, ID: trim(ts[0].Text), Type: "default"}
	ts = ts[1:]
	if is(ts, kStyleSep) {
		c, rest, err := syntax.Expect(ts[1:], kID, "a class")
		if err != nil {
			return State{}, nil, err
		}
		s.Classes, ts = []string{trim(c.Text)}, rest
	}
	return s, ts, nil
}

func trimColon(s string) string { return trim(strings.TrimPrefix(s, ":")) }

func parseIDStatement(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	s, ts, err := parseID(ts)
	if err != nil {
		return nil, nil, err
	}
	if is(ts, kArrow) {
		to, rest, err := parseID(ts[1:])
		if err != nil {
			return nil, nil, err
		}
		r := Relation{State1: s, State2: to}
		if is(rest, kDescr) {
			r.Desc, rest = trimColon(rest[0].Text), rest[1:]
		}
		return r, rest, nil
	}
	if is(ts, kDescr) {
		s.Desc, ts = oneDesc(trimColon(ts[0].Text)), ts[1:]
	}
	return s, ts, nil
}

// oneDesc is a description, none when it is empty, as JavaScript's truth has it.
func oneDesc(d string) []string {
	if d == "" {
		return nil
	}
	return []string{d}
}

// parseDescribed reads `state "a" as id`, with a body or with :b after the id.
func parseDescribed(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	desc := ts[0]
	_, ts, err := syntax.Expect(ts[1:], kAs, "as")
	if err != nil {
		return nil, nil, err
	}
	id, ts, err := syntax.Expect(ts, kID, "a state")
	if err != nil {
		return nil, nil, err
	}
	if is(ts, kStructStart) {
		doc, rest, err := parseStruct(ts)
		return State{At: desc.Pos, ID: id.Text, Type: "default", Desc: oneDesc(desc.Text), Doc: doc}, rest, err
	}
	if name, rest, ok := strings.Cut(id.Text, ":"); ok {
		second, _, _ := strings.Cut(rest, ":")
		return State{At: desc.Pos, ID: trim(name), Type: "default", Desc: []string{trim(desc.Text), second}}, ts, nil
	}
	return State{At: desc.Pos, ID: trim(id.Text), Type: "default", Desc: oneDesc(trim(desc.Text))}, ts, nil
}

func parseNote(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	at := ts[0].Pos
	ts = ts[1:]
	if is(ts, kNoteText) {
		_, ts, err := syntax.Expect(ts[1:], kAs, "as")
		if err != nil {
			return nil, nil, err
		}
		_, ts, err = syntax.Expect(ts, kID, "a name")
		return Ignored{}, ts, err
	}
	if !is(ts, kLeftOf, kRightOf) {
		return nil, nil, syntax.Unexpected(ts[0], "left of or right of")
	}
	position := ts[0].Text
	id, ts, err := syntax.Expect(ts[1:], kID, "a state")
	if err != nil {
		return nil, nil, err
	}
	text, ts, err := syntax.Expect(ts, kNoteText, "the note's text")
	if err != nil {
		return nil, nil, err
	}
	return State{At: at, ID: trim(id.Text), Note: &StateNote{trim(position), trim(text.Text)}}, ts, nil
}

// parseClick reads a click statement. A scene has no links, so all of it goes.
func parseClick(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	_, ts, err := parseID(ts[1:])
	if err != nil {
		return nil, nil, err
	}
	first := kString
	if is(ts, kHref) {
		first = kHref
	}
	for _, k := range []syntax.Kind{first, kString, syntax.Newline} {
		if _, ts, err = syntax.Expect(ts, k, "a link"); err != nil {
			return nil, nil, err
		}
	}
	return Ignored{}, ts, nil
}
