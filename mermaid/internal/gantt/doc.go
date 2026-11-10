// Package gantt reads Mermaid gantt diagrams. A diagram goes through a pipeline, each
// step taking values and returning new ones:
//
//	Lex      source -> tokens       lex.go, rules.go
//	Parse    tokens -> statements   parse.go
//	collect  statements -> Diagram  collect.go, with the dates of dates.go and format.go
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
// change, and a start in no format falls back to the date time strings V8 reads. A
// time Mermaid would make and JavaScript cannot hold is not supported, nor are the
// legacy dates of V8 yet.
package gantt
