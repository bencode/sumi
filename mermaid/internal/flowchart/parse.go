package flowchart

import (
	"strings"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// Parse reads the tokens of a flowchart into its direction, as written, and its
// statements.
func Parse(ts []syntax.Token) (string, []Stmt, error) {
	for is(ts, kSpace, syntax.Newline) {
		ts = ts[1:]
	}
	_, ts, err := syntax.Expect(ts, kGraph, `"flowchart"`)
	if err != nil {
		return "", nil, err
	}
	dir, ts, err := parseHeader(ts)
	if err != nil {
		return "", nil, err
	}
	body, ts, err := parseDocument(ts)
	if err != nil {
		return "", nil, err
	}
	if _, _, err := syntax.Expect(ts, syntax.EOF, "a statement"); err != nil {
		return "", nil, err
	}
	return dir, body, nil
}

// is reports whether the first token of ts is of one of kinds.
func is(ts []syntax.Token, kinds ...syntax.Kind) bool {
	for _, k := range kinds {
		if ts[0].Kind == k {
			return true
		}
	}
	return false
}

// skipSpaces takes the SPACE tokens at the start of ts.
func skipSpaces(ts []syntax.Token) []syntax.Token {
	for is(ts, kSpace) {
		ts = ts[1:]
	}
	return ts
}

// spaceList takes one or more SPACE tokens.
func spaceList(ts []syntax.Token) ([]syntax.Token, error) {
	if !is(ts, kSpace) {
		return ts, syntax.Unexpected(ts[0], "a space")
	}
	return skipSpaces(ts), nil
}

// parseHeader reads what follows GRAPH.
func parseHeader(ts []syntax.Token) (string, []syntax.Token, error) {
	if is(ts, kNoDir) {
		return "TB", ts[1:], nil
	}
	dir, ts, err := syntax.Expect(ts, kDir, "a direction")
	if err != nil {
		return "", ts, err
	}
	if is(ts, kSemi) {
		return dir.Text, ts[1:], nil
	}
	ts = skipSpaces(ts)
	_, ts, err = syntax.Expect(ts, syntax.Newline, "end of line")
	return dir.Text, ts, err
}

// parseDocument reads statements up to "end" or the end of the tokens, which it
// leaves for the caller.
func parseDocument(ts []syntax.Token) ([]Stmt, []syntax.Token, error) {
	var body []Stmt
	for {
		if is(ts, kSemi, syntax.Newline, kSpace) {
			ts = ts[1:]
			continue
		}
		if is(ts, kEnd, syntax.EOF) {
			return body, ts, nil
		}
		s, rest, err := parseStatement(ts)
		if err != nil {
			return nil, ts, err
		}
		body, ts = append(body, s), rest
	}
}

// parseStatement reads one statement.
func parseStatement(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	t := ts[0]
	switch t.Kind {
	case kStyle:
		id, styles, rest, err := parseIDStyles(ts[1:])
		return &Style{id, styles}, rest, err
	case kLinkStyle:
		return parseLinkStyle(ts[1:])
	case kClassDef:
		ids, styles, rest, err := parseIDStyles(ts[1:])
		return &ClassDef{ids, styles}, rest, err
	case kClass:
		return parseClass(ts[1:])
	case kClick:
		return parseClick(ts)
	case kSubgraph:
		return parseSubgraph(ts)
	case kDirection:
		return &Direction{t.Text}, ts[1:], nil
	case kAccTitle:
		v, rest, err := syntax.Expect(ts[1:], kAccTitleValue, "a title")
		return &Member{trim(v.Text)}, rest, err
	case kAccDescr:
		v, rest, err := syntax.Expect(ts[1:], kAccDescrValue, "a description")
		return &Member{trim(v.Text)}, rest, err
	case kAccBlock:
		return &Member{trim(t.Text)}, ts[1:], nil
	}
	return parseChain(ts)
}

// trim is String.prototype.trim.
func trim(s string) string { return strings.TrimFunc(s, isSpace) }

// separator takes the end of a statement.
func separator(ts []syntax.Token) ([]syntax.Token, error) {
	if !is(ts, syntax.Newline, kSemi) {
		return ts, syntax.Unexpected(ts[0], "end of statement")
	}
	return ts[1:], nil
}

// parseIDStyles reads SPACE idString SPACE stylesOpt separator.
func parseIDStyles(ts []syntax.Token) (string, []string, []syntax.Token, error) {
	_, ts, err := syntax.Expect(ts, kSpace, "a space")
	if err != nil {
		return "", nil, ts, err
	}
	id, ts, err := parseID(ts)
	if err != nil {
		return "", nil, ts, err
	}
	if _, ts, err = syntax.Expect(ts, kSpace, "a space"); err != nil {
		return "", nil, ts, err
	}
	styles, ts, err := parseStyles(ts)
	if err != nil {
		return "", nil, ts, err
	}
	ts, err = separator(ts)
	return id, styles, ts, err
}

func parseClass(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	_, ts, err := syntax.Expect(ts, kSpace, "a space")
	if err != nil {
		return nil, ts, err
	}
	ids, ts, err := parseID(ts)
	if err != nil {
		return nil, ts, err
	}
	if _, ts, err = syntax.Expect(ts, kSpace, "a space"); err != nil {
		return nil, ts, err
	}
	class, ts, err := parseID(ts)
	if err != nil {
		return nil, ts, err
	}
	ts, err = separator(ts)
	return &Class{ids, class}, ts, err
}

// parseLinkStyle reads a linkStyle statement past its keyword.
func parseLinkStyle(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	_, ts, err := syntax.Expect(ts, kSpace, "a space")
	if err != nil {
		return nil, ts, err
	}
	s := &LinkStyle{}
	if is(ts, kDefault) {
		ts = ts[1:]
	} else if s.Positions, ts, err = parseNumbers(ts); err != nil {
		return nil, ts, err
	}
	if _, ts, err = syntax.Expect(ts, kSpace, "a space"); err != nil {
		return nil, ts, err
	}
	if !is(ts, kInterpolate) {
		s.Styles, ts, err = parseStyles(ts)
	} else {
		ts, err = parseInterpolate(ts[1:], s)
	}
	if err != nil {
		return nil, ts, err
	}
	ts, err = separator(ts)
	return s, ts, err
}

// parseInterpolate reads SPACE alphaNum [ SPACE stylesOpt ] into s.
func parseInterpolate(ts []syntax.Token, s *LinkStyle) ([]syntax.Token, error) {
	_, ts, err := syntax.Expect(ts, kSpace, "a space")
	if err != nil {
		return ts, err
	}
	if _, ts, err = parseAlphaNum(ts); err != nil {
		return ts, err
	}
	if is(ts, kSpace) {
		s.Styles, ts, err = parseStyles(ts[1:])
	}
	return ts, err
}

// parseNumbers reads NUM { COMMA NUM }.
func parseNumbers(ts []syntax.Token) ([]syntax.Token, []syntax.Token, error) {
	n, ts, err := syntax.Expect(ts, kNum, "a number")
	if err != nil {
		return nil, ts, err
	}
	ns := []syntax.Token{n}
	for is(ts, kComma) {
		if n, ts, err = syntax.Expect(ts[1:], kNum, "a number"); err != nil {
			return nil, ts, err
		}
		ns = append(ns, n)
	}
	return ns, ts, nil
}

// parseStyles reads style { COMMA style }.
func parseStyles(ts []syntax.Token) ([]string, []syntax.Token, error) {
	var styles []string
	for {
		style := ""
		for is(ts, kNum, kNodeString, kColon, kSpace, kBrkt, kStyle) {
			style += ts[0].Text
			ts = ts[1:]
		}
		if style == "" {
			return nil, ts, syntax.Unexpected(ts[0], "a style")
		}
		styles = append(styles, style)
		if !is(ts, kComma) {
			return styles, ts, nil
		}
		ts = ts[1:]
	}
}

// parseSubgraph reads a subgraph up to its "end".
func parseSubgraph(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	at := ts[0].Pos
	ts = ts[1:]
	if !is(ts, kSpace) {
		// Mermaid accepts it, then addSubGraph reads the text of a title it has not got.
		if is(ts, syntax.Newline, kSemi) {
			return nil, ts, syntax.Errorf(at, "a subgraph needs a title")
		}
		return nil, ts, syntax.Unexpected(ts[0], "a title")
	}
	id, ts, err := parseTitle(ts[1:])
	if err != nil {
		return nil, ts, err
	}
	s := &Subgraph{Pos: at, ID: id, Title: id}
	if is(ts, kSQS) {
		if s.Title, ts, err = parseText(ts[1:]); err != nil {
			return nil, ts, err
		}
		if _, ts, err = syntax.Expect(ts, kSQE, `"]"`); err != nil {
			return nil, ts, err
		}
	}
	if ts, err = separator(ts); err != nil {
		return nil, ts, err
	}
	if s.Body, ts, err = parseDocument(ts); err != nil {
		return nil, ts, err
	}
	_, ts, err = syntax.Expect(ts, kEnd, `"end"`)
	return s, ts, err
}

// titleTokens are textNoTagsToken and keywords, without DIR, which comes only after the
// first GRAPH.
var titleTokens = []syntax.Kind{
	kNum, kNodeString, kSpace, kMinus, kAmp, kUnicode, kColon, kMult, kBrkt, kStartLink,
	kStyle, kLinkStyle, kClassDef, kClass, kClick, kGraph, kSubgraph, kEnd, kDown, kUp,
}

// parseTitle reads textNoTags. A string in it is of the kind "text", where the same
// string in a node is "string".
func parseTitle(ts []syntax.Token) (*Label, []syntax.Token, error) {
	l, ts, err := parseTextOf(ts, titleTokens...)
	if err != nil {
		return nil, ts, err
	}
	if l.Kind == "string" {
		l.Kind = "text"
	}
	return l, ts, nil
}
