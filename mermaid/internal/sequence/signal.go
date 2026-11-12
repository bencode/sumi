package sequence

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// parseParticipant reads a participant or actor statement. Its settings'
// type takes the keyword's place, and their alias is its label unless as
// gives another one than its name (addActor, sequenceDb.ts lines 157-181).
//
//	participant = ( "participant" | "actor" ) NAME [ SETTINGS ] [ "as" REST ] NEWLINE
func parseParticipant(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	shape := Box
	if ts[0].Kind == kActorKw {
		shape = Figure
	}
	n, ts, err := syntax.Expect(ts[1:], kName, "a participant's name")
	if err != nil {
		return nil, nil, err
	}
	id := trim(n.Text)
	alias := ""
	if t, rest, ok := syntax.Accept(ts, kConfig); ok {
		if shape, alias, err = parseSettings(t, shape); err != nil {
			return nil, nil, err
		}
		ts = rest
	}
	d := &declare{p: Participant{ID: id, Label: syntax.Decode(id), Shape: shape}, pos: n.Pos}
	if _, rest, ok := syntax.Accept(ts, kAs); ok {
		a, rest, err := syntax.Expect(rest, kRest, "an alias")
		if err != nil {
			return nil, nil, err
		}
		text, wrap := label(a.Text)
		if text == d.p.Label && alias != "" { // as names the participant itself
			text = syntax.Decode(alias)
		}
		d.p.Label, d.p.Wrap = text, wrap
		d.alias, ts = true, rest
	} else if alias != "" {
		d.p.Label, d.alias = syntax.Decode(alias), true
	}
	_, ts, err = syntax.Expect(ts, syntax.Newline, "the end of the line")
	return d, ts, err
}

// parseCreate reads a participant statement that makes a participant at
// the next message, rather than at the top.
//
//	create = "create" participant
func parseCreate(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	if k := ts[1].Kind; k != kParticipant && k != kActorKw {
		return nil, nil, syntax.Unexpected(ts[1], `"participant" or "actor"`)
	}
	d, ts, err := parseParticipant(ts[1:])
	if err != nil {
		return nil, nil, err
	}
	return &create{d.(*declare)}, ts, nil
}

// parseDestroy reads a destroy statement.
//
//	destroy = "destroy" NAME NEWLINE
func parseDestroy(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	n, ts, err := syntax.Expect(ts[1:], kName, "a participant's name")
	if err != nil {
		return nil, nil, err
	}
	_, ts, err = syntax.Expect(ts, syntax.Newline, "the end of the line")
	return &destroy{id: trim(n.Text), pos: n.Pos}, ts, err
}

// parseSettings reads a participant's @{...}: its type, which replaces shape
// unless it is null, and its alias. Mermaid reads them as YAML; sumi reads the JSON that
// Mermaid's own examples write, and refuses the rest as not supported yet.
func parseSettings(t syntax.Token, shape Shape) (Shape, string, error) {
	body, ok := strings.CutSuffix(strings.TrimPrefix(t.Text, "@{"), "}")
	if !ok {
		return 0, "", syntax.Errorf(t.Pos, `"@{" is not closed`)
	}
	if body == "" {
		return 0, "", syntax.Errorf(t.Pos, `unexpected "@{}", expected settings in it`)
	}
	if strings.Contains(body, "\n") {
		return 0, "", syntax.Unsupported(t.Pos, "@{...} over several lines")
	}
	var m map[string]any
	if err := json.Unmarshal([]byte("{"+body+"}"), &m); err != nil {
		return 0, "", syntax.Unsupported(t.Pos, "@{...} that is not JSON")
	}
	if v := m["type"]; v != nil {
		name, _ := v.(string)
		var known bool
		if shape, known = shapeOf[name]; !known {
			return 0, "", syntax.Errorf(t.Pos, "unknown participant type %q", fmt.Sprint(v))
		}
	}
	alias, _ := m["alias"].(string)
	return shape, alias, nil
}

var shapeOf = map[string]Shape{
	"participant": Box, "actor": Figure, "boundary": Boundary, "control": Control,
	"entity": Entity, "database": Database, "collections": Collections, "queue": Queue,
}

// parseSignal reads a message. A () before its arrow or after it is a
// central connection on the sender's or the receiver's lifeline; one after
// it takes the place of + or -.
//
//	signal = NAME [ "()" ] arrow [ "+" | "-" | "()" ] NAME TEXT NEWLINE
func parseSignal(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	from, ts := ts[0], ts[1:]
	_, ts, circleFrom := syntax.Accept(ts, kCentral)
	a, ts, err := syntax.Expect(ts, kArrow, "an arrow")
	if err != nil {
		return nil, nil, err
	}
	m := arrowOf[strings.ToLower(a.Text)]
	m.From, m.Line, m.CircleFrom = trim(from.Text), from.Pos.Line, circleFrom
	switch k := ts[0].Kind; {
	case k == kCentral:
		m.CircleTo, ts = true, ts[1:]
	case circleFrom: // ()->> takes no + or -
	case k == kPlus:
		m.Activate, ts = true, ts[1:]
	case k == kMinus:
		m.Deactivate, ts = true, ts[1:]
	}
	to, ts, err := syntax.Expect(ts, kName, "a participant's name")
	if err != nil {
		return nil, nil, err
	}
	m.To = trim(to.Text)
	if m.Text, m.Wrap, ts, err = parseText(ts); err != nil {
		return nil, nil, err
	}
	_, ts, err = syntax.Expect(ts, syntax.Newline, "the end of the line")
	return &m, ts, err
}

// arrowOf is the message each arrow draws.
var arrowOf = map[string]Message{
	"->>":    {Head: Arrow},
	"-->>":   {Head: Arrow, Dotted: true},
	"<<->>":  {Head: Arrow, Both: true},
	"<<-->>": {Head: Arrow, Both: true, Dotted: true},
	"->":     {Head: NoHead},
	"-->":    {Head: NoHead, Dotted: true},
	"-x":     {Head: Cross},
	"--x":    {Head: Cross, Dotted: true},
	"-)":     {Head: Open},
	"--)":    {Head: Open, Dotted: true},

	`-|\`: {Head: SolidTop}, `--|\`: {Head: SolidTop, Dotted: true},
	`-|/`: {Head: SolidBottom}, `--|/`: {Head: SolidBottom, Dotted: true},
	`-\\`: {Head: StickTop}, `--\\`: {Head: StickTop, Dotted: true},
	`-//`: {Head: StickBottom}, `--//`: {Head: StickBottom, Dotted: true},
	`/|-`: {Head: SolidTop, Back: true}, `/|--`: {Head: SolidTop, Back: true, Dotted: true},
	`\|-`: {Head: SolidBottom, Back: true}, `\|--`: {Head: SolidBottom, Back: true, Dotted: true},
	`//-`: {Head: StickTop, Back: true}, `//--`: {Head: StickTop, Back: true, Dotted: true},
	`\\-`: {Head: StickBottom, Back: true}, `\\--`: {Head: StickBottom, Back: true, Dotted: true},
}

// parseNote reads a note.
//
//	note = "note" ( ( "left of" | "right of" ) NAME
//	              | "over" NAME [ "," NAME ] ) TEXT NEWLINE
func parseNote(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	place, ok := placeOf[ts[1].Kind]
	if !ok {
		return nil, nil, syntax.Unexpected(ts[1], `"left of", "right of" or "over"`)
	}
	n := &Note{Place: place}
	ts = ts[2:]
	for {
		p, rest, err := syntax.Expect(ts, kName, "a participant's name")
		if err != nil {
			return nil, nil, err
		}
		n.Of, ts = append(n.Of, trim(p.Text)), rest
		_, rest, comma := syntax.Accept(ts, kComma)
		if place != Over || len(n.Of) == 2 || !comma {
			break
		}
		ts = rest
	}
	var err error
	if n.Text, n.Wrap, ts, err = parseText(ts); err != nil {
		return nil, nil, err
	}
	_, ts, err = syntax.Expect(ts, syntax.Newline, "the end of the line")
	return n, ts, err
}

var placeOf = map[syntax.Kind]Place{kLeftOf: LeftOf, kRightOf: RightOf, kOver: Over}

// parseActivation reads activate or deactivate.
//
//	activation = ( "activate" | "deactivate" ) NAME NEWLINE
func parseActivation(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	kw := ts[0]
	n, ts, err := syntax.Expect(ts[1:], kName, "a participant's name")
	if err != nil {
		return nil, nil, err
	}
	_, ts, err = syntax.Expect(ts, syntax.Newline, "the end of the line")
	return &Activation{ID: trim(n.Text), On: kw.Kind == kActivate, Line: kw.Pos.Line}, ts, err
}

// parseAutonumber reads an autonumber statement.
//
//	autonumber = "autonumber" [ "off" | NUM [ NUM ] ] NEWLINE
func parseAutonumber(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	a := &autonumber{}
	ts = ts[1:]
	if _, rest, ok := syntax.Accept(ts, kOff); ok {
		a.off, ts = true, rest
	} else if n, rest, ok := syntax.Accept(ts, kNum); ok {
		a.start, ts = parseNum(n.Text), rest
		if n, rest, ok := syntax.Accept(ts, kNum); ok {
			a.step, ts = parseNum(n.Text), rest
		}
	}
	_, ts, err := syntax.Expect(ts, syntax.Newline, "the end of the line")
	return a, ts, err
}

// parseNum is the value of a NUM token, which the lexer has checked.
func parseNum(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// parseText reads the text after a message's or a note's colon.
//
//	TEXT = ":" text
func parseText(ts []syntax.Token) (string, bool, []syntax.Token, error) {
	t, ts, err := syntax.Expect(ts, kText, `":" and a text`)
	if err != nil {
		return "", false, nil, err
	}
	s, wrap := label(trim(t.Text)[1:])
	return s, wrap, ts, nil
}

// label is a text as Mermaid's parseMessage and extractWrap (sequenceDb.ts
// lines 355-387) take it: trimmed, with its entity codes decoded and <br>
// kept, and without a wrap: or nowrap: before it, which may follow a colon.
// It tells whether the text asks to be wrapped: wrap: does; nowrap: does
// not, as no text is wrapped unless it asks.
func label(s string) (string, bool) {
	s = trim(s)
	t := strings.TrimPrefix(s, ":")
	for _, w := range []string{"wrap:", "nowrap:"} {
		if rest, ok := strings.CutPrefix(t, w); ok {
			return syntax.Decode(trim(rest)), w == "wrap:"
		}
	}
	return syntax.Decode(s), false
}

// trim takes off the spaces JavaScript's trim does, which U+0085 is not.
func trim(s string) string { return strings.TrimFunc(s, syntax.IsSpace) }
