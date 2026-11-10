// Package gantt reads and draws Mermaid gantt diagrams. A diagram goes through a
// pipeline, each step taking values and returning new ones:
//
//	Lex      source -> tokens       lex.go, rules.go
//	Parse    tokens -> statements   parse.go
//	collect  statements -> Diagram  collect.go, with the dates of dates.go,
//	                                datestring.go, datetokens.go and format.go
//	axisOf   Diagram, width -> Axis axis.go, with d3's times of timeinterval.go
//	                                and timeformat.go
//	Layout   Diagram, Axis -> Scene draw.go, grid.go, tasks.go
//
// The lexer follows the rules Jison compiles from gantt.jison, which are not always those
// written there, and the parser its grammar, which Parse gives.
//
// collect does what ganttDb does with the statements, then compiles the tasks as
// getTasks does, up to eleven times over so that a task can start after one written
// later. Its dates are those of JavaScript and dayjs, in the zone of an Env, made with
// Go's time: a date in the diagram's format is read and written with dayjs's tokens,
// months and years are added as dayjs adds them, keeping the day where the month has
// it, a local time a change of offset skips or repeats takes the offset from before the
// change, and a start in no format falls back to new Date, read as V8 reads it, its
// legacy rules too. A time Mermaid would make and JavaScript cannot hold is not
// supported.
//
// axisOf lays out the time axis as ganttRenderer does with d3: the scale from the
// earliest start to the latest end, and the ticks of the tickInterval or d3's own, with
// d3's local times and its labels in en-US. Its times are float64 milliseconds, NaN
// where JavaScript's Date is invalid, so that d3's arithmetic and the way its loops stop
// carry over as they are. Where d3 would look for a tick forever, axisOf is an error.
//
// Layout draws the diagram as ganttRenderer does, at Mermaid's default sizes: the
// excluded days, the grid, a row for each task, the bars and then their texts, the
// sections' titles, the line at the time now and the title, in that order, on a canvas
// the width given and as high as its rows. Where a task's text goes, in its bar or beside
// it, hangs on its width; the renderer measures it with its entity codes still set aside.
// Attributes that are NaN, of tasks with no times, are taken as SVG takes them.
package gantt
