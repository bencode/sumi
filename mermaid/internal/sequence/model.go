package sequence

import "github.com/bencode/sumi/scene"

// A Diagram is a sequence diagram as its source says it: who takes part, and
// what happens, in order, as a tree of statements.
type Diagram struct {
	Title        string
	Participants []*Participant // from left to right
	Body         []Stmt
}

// A Participant is a lifeline.
type Participant struct {
	ID    string // the name messages use for it
	Label string // what its box shows: its alias, or its ID; may hold <br>
	Shape Shape
}

// A Shape is how a participant is drawn: Mermaid's participant types
// (PARTICIPANT_TYPE, sequenceDb.ts).
type Shape uint8

const (
	Box    Shape = iota // participant
	Figure              // actor: a stick figure
	Boundary
	Control
	Entity
	Database
	Collections
	Queue
)

// A Stmt is one of *Message, *Note, *Activation, *Block and *Rect.
type Stmt interface{ stmt() }

// A Message is an arrow from one participant to another, or to itself.
type Message struct {
	From, To   string
	Dotted     bool // -- rather than -
	Head       Head
	Both       bool   // a head at each end: <<->> and <<-->>
	Text       string // may hold <br>
	Number     string // its autonumber, as "3" or "1.75"; empty when autonumber is off
	Activate   bool   // +: To is activated after it
	Deactivate bool   // -: From is deactivated after it
	CircleFrom bool   // () before the arrow: a central connection on From's lifeline
	CircleTo   bool   // () after the arrow: one on To's
	Back       bool   // the head is at From's end: /|-, \|-, //- and \\-
	Line       int
}

// A Head is the end of a message's arrow.
type Head uint8

const (
	NoHead      Head = iota // ->
	Arrow                   // ->>, a filled triangle
	Cross                   // -x
	Open                    // -), an open arrow, for an async message
	SolidTop                // -|\, half a filled triangle, above a line drawn left to right
	SolidBottom             // -|/, below it
	StickTop                // -\\, one stroke, above it
	StickBottom             // -//, below it
)

// A Note is a box of text beside a participant, or over one or two.
type Note struct {
	Place Place
	Of    []string // one participant; for Over, one or two
	Text  string
}

// A Place is where a note goes.
type Place uint8

const (
	LeftOf Place = iota
	RightOf
	Over
)

// An Activation turns a participant's activation on or off, on a line of
// its own: activate A, deactivate A.
type Activation struct {
	ID   string
	On   bool
	Line int
}

// A Block is a box around statements: a loop, or the sections of an alt.
type Block struct {
	Kind     BlockKind
	Sections []Section // one for loop, opt and break; one per else, and or option
}

// A BlockKind is a block's keyword.
type BlockKind uint8

const (
	Loop BlockKind = iota
	Alt
	Opt
	Par
	Critical
	Break
)

// A Section is part of a block: its label, the block's title for the first
// section and the text after else, and or option for the others.
type Section struct {
	Label string
	Body  []Stmt
}

// A Rect is a background behind statements (rect).
type Rect struct {
	Color *scene.Color // the author's color, nil when the line names none
	Body  []Stmt
}

func (*Message) stmt()    {}
func (*Note) stmt()       {}
func (*Activation) stmt() {}
func (*Block) stmt()      {}
func (*Rect) stmt()       {}

// The statements below live only between Parse and Resolve, which reads them
// and leaves them out of the diagram.

// A declare is a participant or actor statement.
type declare struct {
	p     Participant
	alias bool // it gave a label, with as or in its settings
}

// An autonumber turns numbering on, from start by step when they are not 0,
// or off.
type autonumber struct {
	start, step float64
	off         bool
}

// A title is the diagram's title, from a title statement.
type title struct{ text string }

func (*declare) stmt()    {}
func (*autonumber) stmt() {}
func (*title) stmt()      {}
