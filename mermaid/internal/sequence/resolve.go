package sequence

import (
	"math"
	"strconv"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// Resolve settles what a parsed diagram leaves open, in three walks over
// its statements, in order: it lists the participants, numbers the messages
// and checks activations. frontTitle is the title the front matter gave; a
// title statement takes its place. It returns a new diagram, without
// participant, autonumber and title statements.
func Resolve(d *Diagram, frontTitle string) (*Diagram, error) {
	ps := participants(d.Body)
	if err := checkActivations(d.Body, ps); err != nil {
		return nil, err
	}
	body, _ := number(d.Body, counter{next: 1, step: 1})
	return &Diagram{Title: titleOf(d.Body, frontTitle), Participants: ps, Body: body}, nil
}

// participants lists the participants in the order they first appear, in a
// participant statement, a message or a note, as Mermaid's addActor does
// (sequenceDb.ts): a later participant statement with an alias changes the
// label and the shape of one that has appeared, but not its place; one
// without an alias changes nothing.
func participants(body []Stmt) []*Participant {
	var ps []*Participant
	at := map[string]*Participant{}
	add := func(p Participant, alias bool) {
		old, ok := at[p.ID]
		if !ok {
			at[p.ID] = &p
			ps = append(ps, &p)
			return
		}
		if alias {
			old.Label, old.Shape = p.Label, p.Shape
		}
	}
	walk(body, func(s Stmt) {
		switch s := s.(type) {
		case *declare:
			add(s.p, s.alias)
		case *Message:
			add(implicit(s.From), false)
			add(implicit(s.To), false)
		case *Note:
			for _, id := range s.Of {
				add(implicit(id), false)
			}
		}
	})
	return ps
}

func implicit(id string) Participant { return Participant{ID: id, Label: syntax.Decode(id)} }

// walk calls f on each statement of body, in order, blocks before what they
// hold.
func walk(body []Stmt, f func(Stmt)) {
	for _, s := range body {
		f(s)
		switch s := s.(type) {
		case *Block:
			for _, sec := range s.Sections {
				walk(sec.Body, f)
			}
		case *Rect:
			walk(s.Body, f)
		}
	}
}

// checkActivations fails, as Mermaid does, on deactivating a participant
// that is not active (sequenceDb.ts line 284), and on activating one that
// never appears, which Mermaid cannot draw.
func checkActivations(body []Stmt, ps []*Participant) error {
	known := map[string]bool{}
	for _, p := range ps {
		known[p.ID] = true
	}
	active := map[string]int{}
	var err error
	walk(body, func(s Stmt) {
		if err != nil {
			return
		}
		switch s := s.(type) {
		case *Activation:
			switch {
			case !s.On:
				err = deactivate(active, s.ID, s.Line)
			case !known[s.ID]:
				err = &syntax.Error{Line: s.Line, Col: 1, Msg: "cannot activate " + strconv.Quote(s.ID) + ": no such participant"}
			default:
				active[s.ID]++
			}
		case *Message:
			if s.Activate {
				active[s.To]++
			}
			if s.Deactivate {
				err = deactivate(active, s.From, s.Line)
			}
		}
	})
	return err
}

func deactivate(active map[string]int, id string, line int) error {
	if active[id] < 1 {
		return &syntax.Error{Line: line, Col: 1, Msg: "cannot deactivate " + strconv.Quote(id) + ": it is not active"}
	}
	active[id]--
	return nil
}

// A counter is the state of autonumbering: the next number, the step to
// the one after, and whether numbers show.
type counter struct {
	next, step float64
	on         bool
}

// number returns body without its participant, autonumber and title
// statements, each message copied with its number when numbering is on, as
// Mermaid's renderer counts (sequenceRenderer.ts lines 1268-1277 and 1380):
// every message counts, numbered or not; autonumber's start and step,
// when not 0, replace the counter's.
func number(body []Stmt, c counter) ([]Stmt, counter) {
	var out []Stmt
	for _, s := range body {
		switch s := s.(type) {
		case *autonumber:
			c.on = !s.off
			if s.start != 0 {
				c.next = s.start
			}
			if s.step != 0 {
				c.step = s.step
			}
		case *Message:
			m := *s
			if c.on {
				m.Number = strconv.FormatFloat(c.next, 'f', -1, 64)
			}
			c.next = math.Round((c.next+c.step)*100) / 100
			out = append(out, &m)
		case *Block:
			b := &Block{Kind: s.Kind}
			for _, sec := range s.Sections {
				var sb []Stmt
				sb, c = number(sec.Body, c)
				b.Sections = append(b.Sections, Section{Label: sec.Label, Body: sb})
			}
			out = append(out, b)
		case *Rect:
			r := &Rect{Color: s.Color}
			r.Body, c = number(s.Body, c)
			out = append(out, r)
		case *declare, *title:
		default:
			out = append(out, s)
		}
	}
	return out, c
}

// titleOf is the last title statement's text, or else frontTitle.
func titleOf(body []Stmt, frontTitle string) string {
	t := frontTitle
	walk(body, func(s Stmt) {
		if s, ok := s.(*title); ok {
			t = s.text
		}
	})
	return t
}
