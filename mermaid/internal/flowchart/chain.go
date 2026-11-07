package flowchart

import (
	"slices"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// parseChain reads a vertexStatement and the separator after it. The steps come out in
// the order the grammar reduces them: a node's vertices and classes, the metadata on
// its last vertex, then the link that reaches it.
func parseChain(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	first, ts, err := parseNode(ts)
	if err != nil {
		return nil, ts, err
	}
	c := &Chain{Steps: first.steps, Members: slices.Clone(first.ids)}
	from := first.ids
	for is(ts, kLink, kStartLink, kLinkID) {
		l, rest, err := parseLink(ts)
		if err != nil {
			return nil, ts, err
		}
		n, rest, err := parseNode(rest)
		if err != nil {
			return nil, rest, err
		}
		l.From, l.To = from, n.ids
		c.Steps = append(append(c.Steps, n.steps...), l)
		c.Members = append(c.Members, n.ids...)
		from, ts = n.ids, rest
	}
	ts, err = separator(ts)
	return c, ts, err
}

// A node is the grammar's node with what may follow it: vertices joined by &, and
// metadata and spaces after the last.
type node struct {
	ids   []string
	steps []Stmt
}

// parseNode reads node [ shapeData ] [ spaceList ].
func parseNode(ts []syntax.Token) (node, []syntax.Token, error) {
	var n node
	var err error
	if ts, err = n.styled(ts); err != nil {
		return n, ts, err
	}
	for {
		var data *Data
		if is(ts, kData) {
			data, ts = parseData(ts, n.ids[len(n.ids)-1])
		}
		spaced := is(ts, kSpace)
		ts = skipSpaces(ts)
		more := spaced && is(ts, kAmp)
		if more {
			if ts, err = spaceList(ts[1:]); err != nil {
				return n, ts, err
			}
			if ts, err = n.styled(ts); err != nil {
				return n, ts, err
			}
		}
		if data != nil {
			n.steps = append(n.steps, data)
		}
		if !more {
			return n, ts, nil
		}
	}
}

// styled reads a styledVertex into n.
func (n *node) styled(ts []syntax.Token) ([]syntax.Token, error) {
	v, ts, err := parseVertex(ts)
	if err != nil {
		return ts, err
	}
	n.ids = append(n.ids, v.ID)
	n.steps = append(n.steps, v)
	if !is(ts, kStyleSep) {
		return ts, nil
	}
	class, ts, err := parseID(ts[1:])
	if err != nil {
		return ts, err
	}
	n.steps = append(n.steps, &Class{v.ID, class})
	return ts, nil
}

// parseData reads shapeData, which is about the node id.
func parseData(ts []syntax.Token, id string) (*Data, []syntax.Token) {
	d := &Data{Pos: ts[0].Pos, ID: id}
	for ; is(ts, kData); ts = ts[1:] {
		d.Text += ts[0].Text
	}
	return d, ts
}

// A bracket is a way to write a node's shape.
type bracket struct {
	open, close []syntax.Kind
	shape       Shape
	want        string
}

// brackets are the shapes of the vertex rule, those that start alike longest first.
var brackets = []bracket{
	{[]syntax.Kind{kSQS}, []syntax.Kind{kSQE}, Rect, `"]"`},
	{[]syntax.Kind{kDoubleCircleStart}, []syntax.Kind{kDoubleCircleEnd}, DoubleCircle, `")))"`},
	{[]syntax.Kind{kPS, kPS}, []syntax.Kind{kPE, kPE}, Circle, `"))"`},
	{[]syntax.Kind{kEllipseStart}, []syntax.Kind{kEllipseEnd}, ellipse, `"-)"`},
	{[]syntax.Kind{kStadiumStart}, []syntax.Kind{kStadiumEnd}, Stadium, `"])"`},
	{[]syntax.Kind{kSubroutineStart}, []syntax.Kind{kSubroutineEnd}, Subroutine, `"]]"`},
	{[]syntax.Kind{kCylinderStart}, []syntax.Kind{kCylinderEnd}, Cylinder, `")]"`},
	{[]syntax.Kind{kPS}, []syntax.Kind{kPE}, Round, `")"`},
	{[]syntax.Kind{kDiamondStart, kDiamondStart}, []syntax.Kind{kDiamondStop, kDiamondStop}, Hexagon, `"}}"`},
	{[]syntax.Kind{kDiamondStart}, []syntax.Kind{kDiamondStop}, Diamond, `"}"`},
	{[]syntax.Kind{kTagEnd}, []syntax.Kind{kSQE}, Odd, `"]"`},
}

// parseVertex reads a vertex.
func parseVertex(ts []syntax.Token) (*Vertex, []syntax.Token, error) {
	at := ts[0].Pos
	id, ts, err := parseID(ts)
	if err != nil {
		return nil, ts, err
	}
	v := &Vertex{Pos: at, ID: id}
	if is(ts, kTrapStart, kInvTrapStart) {
		return parseTrapezoid(ts, v)
	}
	if is(ts, kVertexProps) {
		return parseProps(ts[1:], v)
	}
	for _, b := range brackets {
		if rest, ok := prefix(ts, b.open); ok {
			return parseBracket(rest, v, b)
		}
	}
	return v, ts, nil
}

// prefix takes ks from the start of ts, if ts starts with them.
func prefix(ts []syntax.Token, ks []syntax.Kind) ([]syntax.Token, bool) {
	for i, k := range ks {
		if i >= len(ts) || ts[i].Kind != k {
			return ts, false
		}
	}
	return ts[len(ks):], true
}

func parseBracket(ts []syntax.Token, v *Vertex, b bracket) (*Vertex, []syntax.Token, error) {
	text, ts, err := parseText(ts)
	if err != nil {
		return nil, ts, err
	}
	rest, ok := prefix(ts, b.close)
	if !ok {
		return nil, ts, syntax.Unexpected(ts[0], b.want)
	}
	v.Text, v.Shape = text, b.shape
	return v, rest, nil
}

// parseTrapezoid reads [/text/], [\text\], [/text\] or [\text/].
func parseTrapezoid(ts []syntax.Token, v *Vertex) (*Vertex, []syntax.Token, error) {
	open := ts[0].Kind
	text, ts, err := parseText(ts[1:])
	if err != nil {
		return nil, ts, err
	}
	if !is(ts, kTrapEnd, kInvTrapEnd) {
		return nil, ts, syntax.Unexpected(ts[0], "the end of a trapezoid")
	}
	v.Text = text
	switch {
	case open == kTrapStart && ts[0].Kind == kTrapEnd:
		v.Shape = Trapezoid
	case open == kTrapStart:
		v.Shape = LeanRight
	case ts[0].Kind == kInvTrapEnd:
		v.Shape = InvTrapezoid
	default:
		v.Shape = LeanLeft
	}
	return v, ts[1:], nil
}

// parseProps reads [|field:value|text], which draws as a plain rectangle: nothing reads
// the field.
func parseProps(ts []syntax.Token, v *Vertex) (*Vertex, []syntax.Token, error) {
	for _, k := range []syntax.Kind{kNodeString, kColon, kNodeString, kPipe} {
		_, rest, err := syntax.Expect(ts, k, "field:value|")
		if err != nil {
			return nil, ts, err
		}
		ts = rest
	}
	return parseBracket(ts, v, bracket{close: []syntax.Kind{kSQE}, shape: Rect, want: `"]"`})
}

// parseLink reads a link.
func parseLink(ts []syntax.Token) (*Link, []syntax.Token, error) {
	l := &Link{Pos: ts[0].Pos}
	if is(ts, kLinkID) {
		l.ID, ts = ts[0].Text, ts[1:]
	}
	if is(ts, kStartLink) {
		return parseLinkText(ts, l)
	}
	t, ts, err := syntax.Expect(ts, kLink, "a link")
	if err != nil {
		return nil, ts, err
	}
	l.Type, l.Stroke = destructLink(t.Text, "")
	if !is(ts, kPipe) {
		return l, ts, nil
	}
	if l.Text, ts, err = parseText(ts[1:]); err != nil {
		return nil, ts, err
	}
	if _, ts, err = syntax.Expect(ts, kPipe, `"|"`); err != nil {
		return nil, ts, err
	}
	if is(ts, kSpace) {
		ts = ts[1:]
	}
	return l, ts, nil
}

// parseLinkText reads START_LINK edgeText LINK.
func parseLinkText(ts []syntax.Token, l *Link) (*Link, []syntax.Token, error) {
	start := ts[0].Text
	text, ts, err := parseTextOf(ts[1:], kEdgeText)
	if err != nil {
		return nil, ts, err
	}
	end, ts, err := syntax.Expect(ts, kLink, "the end of a link")
	if err != nil {
		return nil, ts, err
	}
	l.Text = text
	l.Type, l.Stroke = destructLink(end.Text, start)
	return l, ts, nil
}

// parseText reads text. The grammar takes more tokens in it than TEXT, but the modes text
// is in make no others.
func parseText(ts []syntax.Token) (*Label, []syntax.Token, error) {
	return parseTextOf(ts, kText)
}

// parseTextOf reads a string or a token of kinds, then more of them. What it reads takes
// its kind from how it starts.
func parseTextOf(ts []syntax.Token, kinds ...syntax.Kind) (*Label, []syntax.Token, error) {
	l := &Label{Kind: "text"}
	switch {
	case is(ts, kString):
		l.Text, l.Kind = ts[0].Text, "string"
	case is(ts, kMarkdown):
		l.Text, l.Kind = ts[0].Text, "markdown"
	case is(ts, kinds...):
		l.Text = ts[0].Text
	default:
		return nil, ts, syntax.Unexpected(ts[0], "text")
	}
	for ts = ts[1:]; is(ts, kinds...); ts = ts[1:] {
		l.Text += ts[0].Text
	}
	return l, ts, nil
}

// idTokens are idStringToken.
var idTokens = []syntax.Kind{kNum, kNodeString, kDown, kMinus, kDefault, kComma, kColon, kAmp, kBrkt, kMult, kUnicode}

// parseID reads an idString.
func parseID(ts []syntax.Token) (string, []syntax.Token, error) {
	return parseRun(ts, idTokens, "an id")
}

// alphaNumTokens are alphaNumToken, without DIR, which comes only after the first GRAPH.
var alphaNumTokens = []syntax.Kind{kNum, kUnicode, kNodeString, kDown, kMinus, kComma, kColon, kAmp, kBrkt, kMult}

// parseAlphaNum reads an alphaNum.
func parseAlphaNum(ts []syntax.Token) (string, []syntax.Token, error) {
	return parseRun(ts, alphaNumTokens, "a name")
}

// parseRun reads one or more tokens of kinds as one string.
func parseRun(ts []syntax.Token, kinds []syntax.Kind, want string) (string, []syntax.Token, error) {
	if !is(ts, kinds...) {
		return "", ts, syntax.Unexpected(ts[0], want)
	}
	s := ""
	for ; is(ts, kinds...); ts = ts[1:] {
		s += ts[0].Text
	}
	return s, ts, nil
}
