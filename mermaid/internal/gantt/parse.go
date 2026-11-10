package gantt

import (
	"strings"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// A Stmt is a statement of a diagram.
type Stmt any

type (
	// A SetStmt is a statement that sets one of ganttDb's values, Kind its token.
	SetStmt struct {
		At    syntax.Pos
		Kind  syntax.Kind
		Value string
	}
	// A TaskStmt is a task: its text and its data, the : that starts it included.
	TaskStmt struct {
		At         syntax.Pos
		Text, Data string
	}
	// A ClickStmt names tasks, by ids joined with commas, for a callback, a link or
	// both. A scene has neither, and a callback never runs at Mermaid's default security
	// level, so the ids are all that is kept.
	ClickStmt struct{ IDs string }
)

// cuts is how much of each kind of setting's token the grammar's action takes off the
// front: the keyword and one space. substr counts UTF-16 code units, and each of these
// characters is one.
var cuts = map[syntax.Kind]int{
	kDateFormat: 11, kAxisFormat: 11, kTickInterval: 13, kExcludes: 9, kIncludes: 9,
	kTodayMarker: 12, kTitle: 6, kSection: 8,
}

// values are the kinds of token that take a value after them, and the kind of the value.
var values = map[syntax.Kind]syntax.Kind{kAccTitle: kAccTitleValue, kAccDescr: kAccDescrValue}

// Parse reads the statements of a diagram's tokens, as the grammar of gantt.jison:
//
//	start     = GANTT { line } EOF ;
//	line      = statement | NL ;
//	statement = DATE_FORMAT | INCLUSIVE_END_DATES | TOP_AXIS | AXIS_FORMAT | TICK_INTERVAL
//	          | EXCLUDES | INCLUDES | TODAY_MARKER | WEEKDAY | WEEKEND | TITLE | SECTION
//	          | ACC_TITLE ACC_TITLE_VALUE | ACC_DESCR ACC_DESCR_VALUE | ACC_DESCR_MULTILINE_VALUE
//	          | click | TASK_TXT TASK_DATA ;
//	click     = CLICK ( CALLBACKNAME [ CALLBACKARGS ] [ HREF ] | HREF [ CALLBACKNAME [ CALLBACKARGS ] ] ) ;
//
// The grammar's line also takes an EOF, and a SPACE no rule makes, neither of which
// helps a source to be read.
func Parse(ts []syntax.Token) ([]Stmt, error) {
	_, ts, err := syntax.Expect(ts, kGantt, "gantt")
	if err != nil {
		return nil, err
	}
	var stmts []Stmt
	for ts[0].Kind != kDone {
		if ts[0].Kind == syntax.Newline {
			ts = ts[1:]
			continue
		}
		s, rest, err := parseStatement(ts)
		if err != nil {
			return nil, err
		}
		stmts, ts = append(stmts, s), rest
	}
	return stmts, nil
}

func parseStatement(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	t := ts[0]
	if n, ok := cuts[t.Kind]; ok {
		return SetStmt{t.Pos, t.Kind, string([]rune(t.Text)[n:])}, ts[1:], nil
	}
	if t.Kind == kWeekday || t.Kind == kWeekend {
		f := strings.FieldsFunc(t.Text, isJSSpace)
		return SetStmt{t.Pos, t.Kind, strings.ToLower(f[len(f)-1])}, ts[1:], nil
	}
	if t.Kind == kInclusiveEndDates || t.Kind == kAccMultiValue {
		return SetStmt{t.Pos, t.Kind, trim(t.Text)}, ts[1:], nil
	}
	if t.Kind == kTopAxis {
		// The action calls yy.TopAxis, which ganttDb does not have: its function is
		// enableTopAxis.
		return nil, nil, syntax.Errorf(t.Pos, "topAxis fails in Mermaid 12.3.0, whose parser calls a function ganttDb does not have")
	}
	if values[t.Kind] != 0 {
		v, rest, err := syntax.Expect(ts[1:], values[t.Kind], "a value")
		if err != nil {
			return nil, nil, err
		}
		return SetStmt{t.Pos, t.Kind, trim(v.Text)}, rest, nil
	}
	if t.Kind == kTaskTxt {
		data, rest, err := syntax.Expect(ts[1:], kTaskData, "the data of a task")
		if err != nil {
			return nil, nil, err
		}
		return TaskStmt{At: t.Pos, Text: t.Text, Data: data.Text}, rest, nil
	}
	if t.Kind == kClick {
		return parseClick(ts)
	}
	return nil, nil, syntax.Unexpected(t, "a statement")
}

func parseClick(ts []syntax.Token) (Stmt, []syntax.Token, error) {
	c := ClickStmt{IDs: ts[0].Text}
	ts = ts[1:]
	callback := func() {
		if _, rest, ok := syntax.Accept(ts, kCallbackName); ok {
			ts = rest
			if _, rest, ok := syntax.Accept(ts, kCallbackArgs); ok {
				ts = rest
			}
		}
	}
	link := func() {
		if _, rest, ok := syntax.Accept(ts, kHref); ok {
			ts = rest
		}
	}
	if ts[0].Kind == kCallbackName {
		callback()
		link()
		return c, ts, nil
	}
	if ts[0].Kind == kHref {
		link()
		callback()
		return c, ts, nil
	}
	return nil, nil, syntax.Unexpected(ts[0], "a callback or an href")
}
