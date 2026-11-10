package gantt

import (
	"time"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// An Env is what Mermaid takes from where it runs: the time now, for a date written
// without its year, month or day and for a task after one that does not exist, and the
// time zone the browser reads and writes local times in.
type Env struct {
	Now  time.Time
	Zone *time.Location
}

// A Diagram is what ganttDb holds once a diagram is read and its tasks compiled.
type Diagram struct {
	Title, AccTitle, AccDescr string
	// DateFormat, AxisFormat, TickInterval and TodayMarker are as written, past the
	// keyword and the one space after it.
	DateFormat, AxisFormat, TickInterval, TodayMarker string
	// Includes and Excludes are the words of every includes and excludes, lower case,
	// each once, in the order first written.
	Includes, Excludes []string
	Weekday, Weekend   string
	InclusiveEndDates  bool
	Sections           []string
	Tasks              []Task // in the order written, as getTasks gives them
}

// A Task is a task as getTasks leaves it. Tasks may share their times, which nothing
// changes once made.
type Task struct {
	At                syntax.Pos
	ID, Text, Section string
	// Start and End are the task's times, nil while not known. Processed is whether
	// both were once: a task after one whose end is never known has neither.
	Start, End *time.Time
	Processed  bool
	// RenderEnd is where the bar ends when the diagram excludes days and the end is not
	// written as a date: End before the last run of excluded days moved it, or End
	// itself. It is nil otherwise, and when the task ends before the day after it starts.
	RenderEnd                           *time.Time
	Active, Done, Crit, Milestone, Vert bool
	Order                               int  // the place among tasks that are not Vert, -1 for those
	Clickable                           bool // a click statement named the task after it was written
}

// Read reads a diagram's body, prepared by syntax.Prepare, and compiles its tasks.
func Read(body string, env Env) (*Diagram, error) {
	ts, err := Lex(body)
	if err != nil {
		return nil, err
	}
	stmts, err := Parse(ts)
	if err != nil {
		return nil, err
	}
	return collect(stmts, env)
}
