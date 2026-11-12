package gantt

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

func trim(s string) string { return strings.TrimFunc(s, syntax.IsSpace) }

// undefined is the id of a task with none, and the task before the first: a key of an
// object in JavaScript, which reads undefined as this.
const undefined = "undefined"

// A rawTask is a task as addTask keeps it, before getTasks compiles it.
type rawTask struct {
	Task
	prevID             string // the id of the task added before it
	after              bool   // it starts where that task ends
	startData, endData string
	wellFormed         bool // its data had one to three parts
	manualEnd          bool // its end is written as a date
}

// db is ganttDb, what the statements of a diagram set, in the order they set it.
type db struct {
	env    Env
	d      *Diagram
	tasks  []*rawTask
	ids    map[string]int // taskDb: the last task with each id
	lastID string
	count  int // the tasks given an id of their own
	order  int
}

func (b *db) find(id string) *rawTask {
	if i, ok := b.ids[id]; ok {
		return b.tasks[i]
	}
	return nil
}

// collect runs a diagram's statements against ganttDb, then compiles its tasks as
// getTasks does.
func collect(stmts []Stmt, env Env) (*Diagram, error) {
	b := &db{env: env, d: &Diagram{Weekday: "sunday", Weekend: "saturday"}, ids: map[string]int{}, lastID: undefined}
	for _, s := range stmts {
		var err error
		switch s := s.(type) {
		case SetStmt:
			err = b.set(s)
		case TaskStmt:
			err = b.addTask(s)
		case ClickStmt:
			b.click(s)
		}
		if err != nil {
			return nil, err
		}
	}
	if err := b.compileAll(); err != nil {
		return nil, err
	}
	for _, t := range b.tasks {
		b.d.Tasks = append(b.d.Tasks, t.Task)
	}
	return b.d, nil
}

var (
	tokenSeparators = regexp.MustCompile(`[` + syntax.JSSpaces + `,]+`)
	spacesAfterLine = regexp.MustCompile(`\n[` + syntax.JSSpaces + `]+`)
)

func (b *db) set(s SetStmt) error {
	d := b.d
	switch s.Kind {
	case kDateFormat:
		d.DateFormat = s.Value
	case kAxisFormat:
		d.AxisFormat = s.Value
	case kTickInterval:
		d.TickInterval, d.TickIntervalAt = s.Value, s.At
	case kTodayMarker:
		d.TodayMarker, d.TodayMarkerAt = s.Value, s.At
	case kWeekday:
		d.Weekday = s.Value
	case kWeekend:
		d.Weekend = s.Value
	case kInclusiveEndDates:
		d.InclusiveEndDates = true
	case kSection:
		b.d.Sections = append(b.d.Sections, s.Value)
	case kIncludes, kExcludes:
		words, err := mergeWords(s)
		if err != nil {
			return err
		}
		if s.Kind == kIncludes {
			d.Includes = merge(d.Includes, words)
		} else {
			d.Excludes = merge(d.Excludes, words)
		}
	case kTitle, kAccTitle, kAccDescr, kAccMultiValue:
		// sanitizeText leaves a text without a < as it is.
		if strings.Contains(s.Value, "<") {
			return syntax.Unsupported(s.At, "a < in a title or a description")
		}
		if s.Kind == kTitle {
			d.Title = s.Value
		} else if s.Kind == kAccTitle {
			d.AccTitle = s.Value
		} else {
			d.AccDescr = spacesAfterLine.ReplaceAllString(s.Value, "\n")
		}
	}
	return nil
}

// mergeWords are the words of an includes or excludes, lower case. JavaScript's
// toLowerCase differs from Go's for two letters: İ, which it writes as i and a dot
// above, and Σ, which it writes as ς at the end of a word.
func mergeWords(s SetStmt) ([]string, error) {
	if strings.ContainsAny(s.Value, "İΣ") {
		return nil, syntax.Unsupported(s.At, "an İ or a Σ in the days a diagram includes or excludes")
	}
	return tokenSeparators.Split(strings.ToLower(s.Value), -1), nil
}

// merge is mergeTokens: the words not empty and not in to yet, added in order.
func merge(to, words []string) []string {
	for _, w := range words {
		if w != "" && !slices.Contains(to, w) {
			to = append(to, w)
		}
	}
	return to
}

// addTask is addTask and parseData: the tags at the front of a task's data, then its
// id, start and end by how many parts are left.
func (b *db) addTask(s TaskStmt) error {
	t := &rawTask{Task: Task{At: s.At, Text: s.Text, Section: b.currentSection()}, prevID: b.lastID}
	data := strings.Split(s.Data[1:], ",")
	tags := map[string]*bool{"active": &t.Active, "done": &t.Done, "crit": &t.Crit, "milestone": &t.Milestone, "vert": &t.Vert}
	for len(data) > 0 && tags[trim(data[0])] != nil {
		*tags[trim(data[0])] = true
		data = data[1:]
	}
	if len(data) == 0 { // getTaskTags reads past the end of data
		return syntax.Errorf(s.At, "Cannot read properties of undefined (reading 'match')")
	}
	for i := range data {
		data[i] = trim(data[i])
	}
	t.wellFormed = len(data) <= 3
	t.ID = undefined
	if len(data) == 3 {
		t.ID, data = data[0], data[1:]
	} else if len(data) < 3 {
		b.count++
		t.ID = "task" + strconv.Itoa(b.count)
	}
	if len(data) == 1 {
		t.after, t.endData = true, data[0]
	} else if len(data) == 2 {
		t.startData, t.endData = data[0], data[1]
	}
	t.Order = -1
	if !t.Vert {
		t.Order, b.order = b.order, b.order+1
	}
	b.tasks = append(b.tasks, t)
	b.lastID = t.ID
	if t.ID != "__proto__" { // setting it on an object sets the object's prototype
		b.ids[t.ID] = len(b.tasks) - 1
	}
	return nil
}

func (b *db) currentSection() string {
	if n := len(b.d.Sections); n > 0 {
		return b.d.Sections[n-1]
	}
	return ""
}

// click is setLink and setClickEvent: the tasks a click names, so far, are clickable.
func (b *db) click(s ClickStmt) {
	for _, id := range strings.Split(s.IDs, ",") {
		if t := b.find(id); t != nil {
			t.Clickable = true
		}
	}
}

// compileAll is getTasks: the tasks compiled once, then again while some have no end,
// ten more times at most, as a task may start after one written later.
func (b *db) compileAll() error {
	for i := 0; i <= 10; i++ {
		done := true
		for _, t := range b.tasks {
			if err := b.compile(t); err != nil {
				return err
			}
			done = done && t.Processed
		}
		if done {
			return nil
		}
	}
	return nil
}

// compile is compileTask.
func (b *db) compile(t *rawTask) error {
	if !t.wellFormed {
		return syntax.Errorf(t.At, "Cannot read properties of undefined (reading 'type')")
	}
	if t.after {
		prev := b.find(t.prevID)
		if prev == nil {
			return syntax.Errorf(t.At, "Cannot read properties of undefined (reading 'endTime')")
		}
		t.Start = prev.End
	} else {
		start, err := b.startDate(t.At, t.startData)
		if err != nil {
			return err
		}
		if start != nil {
			t.Start = start
		}
	}
	if t.Start == nil {
		return nil
	}
	end, err := b.endDate(t.At, *t.Start, t.endData)
	if err != nil {
		return err
	}
	if t.End = end; end == nil {
		return nil
	}
	t.Processed, t.manualEnd = true, isDay(t.endData, b.env)
	return b.checkDates(t)
}
