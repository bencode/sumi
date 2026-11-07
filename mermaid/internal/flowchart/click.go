package flowchart

import "github.com/bencode/sumi/mermaid/internal/syntax"

// clickable is the class every click statement leaves on its ids (setClickEvent, setLink).
const clickable = "clickable"

// parseClick reads a click statement. A scene has no links or callbacks, and a callback
// never runs at Mermaid's default security level, so all that is kept is the class.
func parseClick(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	ids := ts[0].Text
	ts = ts[1:]
	var err error
	switch {
	case is(ts, kCallbackName):
		ts = ts[1:]
		if is(ts, kCallbackArgs) {
			ts = ts[1:]
		}
		ts, err = clickTip(ts)
	case is(ts, kHref):
		if _, ts, err = syntax.Expect(ts[1:], kString, "a link"); err == nil {
			ts, err = clickLink(ts)
		}
	case is(ts, kString):
		ts, err = clickLink(ts[1:])
	default:
		if _, ts, err = parseAlphaNum(ts); err == nil {
			ts, err = clickTip(ts)
		}
	}
	if err != nil {
		return nil, ts, err
	}
	ts, err = separator(ts)
	return &Class{ids, clickable}, ts, err
}

// clickTip reads [ SPACE STR ].
func clickTip(ts []syntax.Token) ([]syntax.Token, error) {
	if !is(ts, kSpace) {
		return ts, nil
	}
	_, ts, err := syntax.Expect(ts[1:], kString, "a tooltip")
	return ts, err
}

// clickLink reads [ SPACE STR ] [ SPACE LINK_TARGET ].
func clickLink(ts []syntax.Token) ([]syntax.Token, error) {
	if !is(ts, kSpace) {
		return ts, nil
	}
	if is(ts[1:], kLinkTarget) {
		return ts[2:], nil
	}
	_, ts, err := syntax.Expect(ts[1:], kString, "a tooltip")
	if err != nil || !is(ts, kSpace) {
		return ts, err
	}
	_, ts, err = syntax.Expect(ts[1:], kLinkTarget, "a link target")
	return ts, err
}
