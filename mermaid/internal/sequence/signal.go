package sequence

import (
	"strconv"
	"strings"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// parseParticipant reads a participant or actor statement.
//
//	participant = ( "participant" | "actor" ) NAME [ "as" REST ] NEWLINE
func parseParticipant(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	actor := ts[0].Kind == kActorKw
	n, ts, err := syntax.Expect(ts[1:], kName, "a participant's name")
	if err != nil {
		return nil, nil, err
	}
	if ts[0].Kind == kConfig {
		return nil, nil, syntax.Unsupported(ts[0].Pos, "@{...}")
	}
	id := strings.TrimSpace(n.Text)
	d := &declare{p: Participant{ID: id, Label: syntax.Decode(id), Actor: actor}}
	if _, rest, ok := syntax.Accept(ts, kAs); ok {
		a, rest, err := syntax.Expect(rest, kRest, "an alias")
		if err != nil {
			return nil, nil, err
		}
		if d.p.Label, err = label(a.Pos, a.Text); err != nil {
			return nil, nil, err
		}
		d.alias, ts = true, rest
	}
	_, ts, err = syntax.Expect(ts, syntax.Newline, "the end of the line")
	return d, ts, err
}

// parseSignal reads a message.
//
//	signal = NAME arrow [ "+" | "-" ] NAME TEXT NEWLINE
func parseSignal(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	from, ts := ts[0], ts[1:]
	if k := ts[0].Kind; k == kHalfArrow || k == kCentral {
		return nil, nil, syntax.Unsupported(ts[0].Pos, syntax.Describe(ts[0]))
	}
	a, ts, err := syntax.Expect(ts, kArrow, "an arrow")
	if err != nil {
		return nil, nil, err
	}
	m := arrowOf[strings.ToLower(a.Text)]
	m.From, m.Line = strings.TrimSpace(from.Text), from.Pos.Line
	switch ts[0].Kind {
	case kPlus:
		m.Activate, ts = true, ts[1:]
	case kMinus:
		m.Deactivate, ts = true, ts[1:]
	case kCentral:
		return nil, nil, syntax.Unsupported(ts[0].Pos, syntax.Describe(ts[0]))
	}
	to, ts, err := syntax.Expect(ts, kName, "a participant's name")
	if err != nil {
		return nil, nil, err
	}
	m.To = strings.TrimSpace(to.Text)
	if m.Text, ts, err = parseText(ts); err != nil {
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
		n.Of, ts = append(n.Of, strings.TrimSpace(p.Text)), rest
		_, rest, comma := syntax.Accept(ts, kComma)
		if place != Over || len(n.Of) == 2 || !comma {
			break
		}
		ts = rest
	}
	var err error
	if n.Text, ts, err = parseText(ts); err != nil {
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
	return &Activation{ID: strings.TrimSpace(n.Text), On: kw.Kind == kActivate, Line: kw.Pos.Line}, ts, err
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
func parseText(ts []syntax.Token) (string, []syntax.Token, error) {
	t, ts, err := syntax.Expect(ts, kText, `":" and a text`)
	if err != nil {
		return "", nil, err
	}
	s, err := label(t.Pos, strings.TrimSpace(t.Text)[1:])
	return s, ts, err
}

// label is a text as Mermaid's parseMessage (sequenceDb.ts line 380) takes
// it: trimmed, with its entity codes decoded and <br> kept for scene.Wrap. A
// wrap: or nowrap: before it, which asks for the text to be wrapped or not,
// is not supported yet.
func label(p syntax.Pos, s string) (string, error) {
	s = strings.TrimSpace(s)
	t := strings.TrimPrefix(s, ":")
	for _, w := range []string{"wrap:", "nowrap:"} {
		if strings.HasPrefix(t, w) {
			return "", syntax.Unsupported(p, `"`+w+`"`)
		}
	}
	return syntax.Decode(s), nil
}
