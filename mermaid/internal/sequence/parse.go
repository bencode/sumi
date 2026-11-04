package sequence

import (
	"regexp"
	"slices"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// Parse reads the tokens of a sequence diagram into a tree, by the grammar of
// sequenceDiagram.jison (see the package comment). The tree still holds
// participant, autonumber and title statements, which Resolve settles.
//
//	diagram = { NEWLINE } "sequenceDiagram" document EOF
func Parse(ts []syntax.Token) (*Diagram, error) {
	for ts[0].Kind == syntax.Newline {
		ts = ts[1:]
	}
	_, ts, err := syntax.Expect(ts, kSD, `"sequenceDiagram"`)
	if err != nil {
		return nil, err
	}
	body, ts, err := parseDocument(ts)
	if err != nil {
		return nil, err
	}
	if _, _, err := syntax.Expect(ts, syntax.EOF, "a statement"); err != nil {
		return nil, err
	}
	return &Diagram{Body: body}, nil
}

// parseDocument reads statements up to a token of one of the kinds in ends,
// or the end of input. Mermaid passes over a token it cannot read where a
// statement starts.
//
//	document = { NEWLINE | INVALID | statement }
func parseDocument(ts []syntax.Token, ends ...syntax.Kind) ([]Stmt, []syntax.Token, error) {
	var body []Stmt
	for {
		k := ts[0].Kind
		switch {
		case k == syntax.EOF || slices.Contains(ends, k):
			return body, ts, nil
		case k == syntax.Newline || k == kInvalid:
			ts = ts[1:]
		default:
			s, rest, err := parseStatement(ts)
			if err != nil {
				return nil, nil, err
			}
			if s != nil {
				body = append(body, s)
			}
			ts = rest
		}
	}
}

// parseStatement reads a statement by its first token; for an accessible
// title or description, which a scene has no place for yet, it returns no
// statement.
func parseStatement(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	t := ts[0]
	switch t.Kind {
	case kParticipant, kActorKw:
		return parseParticipant(ts)
	case kName:
		return parseSignal(ts)
	case kNote:
		return parseNote(ts)
	case kActivate, kDeactivate:
		return parseActivation(ts)
	case kAutonumber:
		return parseAutonumber(ts)
	case syntax.Title:
		return &title{syntax.Decode(t.Text)}, ts[1:], nil
	case syntax.AccTitle, syntax.AccDescr:
		return nil, ts[1:], nil
	case kRect:
		return parseRect(ts)
	case kBox:
		return parseBox(ts)
	case kCreate:
		return parseCreate(ts)
	case kDestroy:
		return parseDestroy(ts)
	case kParOver, kMenu:
		return nil, nil, syntax.Unsupported(t.Pos, syntax.Describe(t))
	}
	if kind, ok := blockKinds[t.Kind]; ok {
		return parseBlock(ts, kind)
	}
	return nil, nil, syntax.Unexpected(t, "a statement")
}

var blockKinds = map[syntax.Kind]BlockKind{
	kLoop: Loop, kAlt: Alt, kOpt: Opt, kPar: Par, kCritical: Critical, kBreak: Break,
}

// separators are the keywords that start a block's next section.
var separators = map[BlockKind]syntax.Kind{Alt: kElse, Par: kAnd, Critical: kOption}

// parseBlock reads a block and its sections.
//
//	block = ( "loop" | "opt" | "break" ) REST document "end"
//	      | "alt" REST document { "else" REST document } "end"
//	      | "par" REST document { "and" REST document } "end"
//	      | "critical" REST document { "option" REST document } "end"
func parseBlock(ts []syntax.Token, kind BlockKind) (Stmt, []syntax.Token, error) {
	ends, want := []syntax.Kind{kEnd}, `"end"`
	sep, ok := separators[kind]
	if ok {
		ends = append(ends, sep)
		want = `"` + sepWords[sep] + `" or "end"`
	}
	b := &Block{Kind: kind}
	for ts = ts[1:]; ; {
		rest, after, err := syntax.Expect(ts, kRest, "the rest of the line")
		if err != nil {
			return nil, nil, err
		}
		lbl, _ := label(rest.Text) // a block's title is always wrapped
		body, after, err := parseDocument(after, ends...)
		if err != nil {
			return nil, nil, err
		}
		b.Sections = append(b.Sections, Section{Label: lbl, Body: body})
		if _, next, more := syntax.Accept(after, sep); ok && more {
			ts = next
			continue
		}
		_, after, err = syntax.Expect(after, kEnd, want)
		return b, after, err
	}
}

// parseRect reads a rect, its line a color or nothing.
//
//	rect = "rect" REST document "end"
func parseRect(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	rest, ts, err := syntax.Expect(ts[1:], kRest, "the rest of the line")
	if err != nil {
		return nil, nil, err
	}
	text, _ := label(rest.Text) // a rect's line is its color
	r := &Rect{}
	if text != "" {
		c, ok := syntax.Color(text)
		if !ok {
			return nil, nil, syntax.Errorf(rest.Pos, "%q is not a color", text)
		}
		r.Color = &c
	}
	if r.Body, ts, err = parseDocument(ts, kEnd); err != nil {
		return nil, nil, err
	}
	_, ts, err = syntax.Expect(ts, kEnd, `"end"`)
	return r, ts, err
}

// boxColor is the start of a box's line that may be a color: a color
// function up to its first ")", or a word (parseBoxData, sequenceDb.ts,
// which takes a function up to the last ")").
var boxColor = regexp.MustCompile(`^(?:(?:rgba?|hsla?)\s*\(.*?\)|\w*)`)

// parseBox reads a box: its line, a color and a title, either of which may
// be missing, and the participant statements in it. A line that does not
// start with a color is all title.
//
//	box = "box" REST { NEWLINE | participant } "end"
func parseBox(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	rest, ts, err := syntax.Expect(ts[1:], kRest, "the rest of the line")
	if err != nil {
		return nil, nil, err
	}
	g, title := &Group{}, rest.Text
	head := boxColor.FindString(title)
	if c, ok := syntax.Color(head); ok {
		g.Fill, title = c, title[len(head):]
	}
	g.Title, g.Wrap = label(title)
	s := &boxed{group: g}
	for {
		switch ts[0].Kind {
		case syntax.Newline:
			ts = ts[1:]
		case kParticipant, kActorKw:
			d, rest, err := parseParticipant(ts)
			if err != nil {
				return nil, nil, err
			}
			s.decls, ts = append(s.decls, d.(*declare)), rest
		default:
			_, ts, err = syntax.Expect(ts, kEnd, `"end"`)
			return s, ts, err
		}
	}
}

var sepWords = map[syntax.Kind]string{kElse: "else", kAnd: "and", kOption: "option"}
