package sequence

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

// The cases follow Mermaid's own parser tests, "when parsing a
// sequenceDiagram" in sequenceDiagram.spec.js (v12.3.0, lines 438-1783),
// for the syntax sumi supports.

// read takes src through every step up to a resolved diagram.
func read(src string) (*Diagram, error) {
	body, front := syntax.Prepare(src)
	ts, err := Lex(body)
	if err != nil {
		return nil, err
	}
	d, err := Parse(ts)
	if err != nil {
		return nil, err
	}
	return Resolve(d, front)
}

func ps(ids ...string) []*Participant {
	out := make([]*Participant, len(ids))
	for i, id := range ids {
		out[i] = &Participant{ID: id, Label: id}
	}
	return out
}

func msg(from, to, text string, m Message) *Message {
	m.From, m.To, m.Text = from, to, text
	return &m
}

func TestParse(t *testing.T) {
	tests := []struct {
		name, src string
		want      *Diagram
	}{
		{"messages and a note", `sequenceDiagram
Alice->Bob:Hello Bob, how are you?
Note right of Bob: Bob thinks
Bob-->Alice: I am good thanks!`, &Diagram{
			Participants: ps("Alice", "Bob"),
			Body: []Stmt{
				msg("Alice", "Bob", "Hello Bob, how are you?", Message{Line: 2}),
				&Note{Place: RightOf, Of: []string{"Bob"}, Text: "Bob thinks"},
				msg("Bob", "Alice", "I am good thanks!", Message{Dotted: true, Line: 4}),
			}}},
		{"title, accessibility, semicolons and comments", `---
title: Front
---
sequenceDiagram;title: Diagram Title;accTitle: A; B
accDescr {
x
}
  %% Comment
A->>B: a #9829; b # comment`, &Diagram{
			Title:        "Diagram Title",
			Participants: ps("A", "B"),
			Body:         []Stmt{msg("A", "B", "a \u2665 b", Message{Head: Arrow, Line: 9})}}},
		{"names with spaces, dashes and equals", `sequenceDiagram
participant Alice=Wonderland
Alice-in-Wonderland ->>Bob Smith: Hello - you
Bob Smith-x Alice=Wonderland: x`, &Diagram{
			Participants: ps("Alice=Wonderland", "Alice-in-Wonderland", "Bob Smith"),
			Body: []Stmt{
				msg("Alice-in-Wonderland", "Bob Smith", "Hello - you", Message{Head: Arrow, Line: 3}),
				msg("Bob Smith", "Alice=Wonderland", "x", Message{Head: Cross, Line: 4}),
			}}},
		{"aliases and actors", `sequenceDiagram
A->>B: hi
actor B as Bob<br>Smith
participant A
participant C as Carol`, &Diagram{
			Participants: []*Participant{
				{ID: "A", Label: "A"}, {ID: "B", Label: "Bob<br>Smith", Shape: Figure}, {ID: "C", Label: "Carol"},
			},
			Body: []Stmt{msg("A", "B", "hi", Message{Head: Arrow, Line: 2})}}},
		{"central connections", `sequenceDiagram
A->>()B: 1
A()-->>B: 2
A()->>()B: 3
B()<<->>()A: 4`, &Diagram{
			Participants: ps("A", "B"),
			Body: []Stmt{
				msg("A", "B", "1", Message{Head: Arrow, CircleTo: true, Line: 2}),
				msg("A", "B", "2", Message{Head: Arrow, Dotted: true, CircleFrom: true, Line: 3}),
				msg("A", "B", "3", Message{Head: Arrow, CircleFrom: true, CircleTo: true, Line: 4}),
				msg("B", "A", "4", Message{Head: Arrow, Both: true, CircleFrom: true, CircleTo: true, Line: 5}),
			}}},
		{"half arrows", `sequenceDiagram
A-|\B: 1
A--//B: 2
A/|-B: 3
A\\--B: 4`, &Diagram{
			Participants: ps("A", "B"),
			Body: []Stmt{
				msg("A", "B", "1", Message{Head: SolidTop, Line: 2}),
				msg("A", "B", "2", Message{Head: StickBottom, Dotted: true, Line: 3}),
				msg("A", "B", "3", Message{Head: SolidTop, Back: true, Line: 4}),
				msg("A", "B", "4", Message{Head: StickBottom, Back: true, Dotted: true, Line: 5}),
			}}},
		{"rect", `sequenceDiagram
rect rgb(0, 128, 255)
A->>B: 1
rect
B->>A: 2
end
end`, &Diagram{
			Participants: ps("A", "B"),
			Body: []Stmt{&Rect{Color: &scene.Color{R: 0, G: 128, B: 255, A: 255}, Body: []Stmt{
				msg("A", "B", "1", Message{Head: Arrow, Line: 3}),
				&Rect{Body: []Stmt{msg("B", "A", "2", Message{Head: Arrow, Line: 5})}},
			}}}}},
		{"boxes", `sequenceDiagram
box rgb(0,0,0) Phase (1)
participant A
actor B
end
box Group 1
participant C
end
box
actor D
end
participant A as Alpha
A->>C: x`, func() *Diagram {
			g1, g2, g3 := &Group{Title: "Phase (1)", Fill: scene.Color{A: 255}}, &Group{Title: "Group 1"}, &Group{}
			return &Diagram{
				Participants: []*Participant{
					{ID: "A", Label: "Alpha", Group: g1}, {ID: "B", Label: "B", Shape: Figure, Group: g1},
					{ID: "C", Label: "C", Group: g2}, {ID: "D", Label: "D", Shape: Figure, Group: g3},
				},
				Groups: []*Group{g1, g2, g3},
				Body:   []Stmt{msg("A", "C", "x", Message{Head: Arrow, Line: 13})},
			}
		}()},
		{"wrap", `sequenceDiagram
participant C as wrap:Z
A->>B:wrap: x
Note over A:nowrap: y
loop wrap: t
end`, &Diagram{
			Participants: []*Participant{{ID: "C", Label: "Z", Wrap: true}, {ID: "A", Label: "A"}, {ID: "B", Label: "B"}},
			Body: []Stmt{
				msg("A", "B", "x", Message{Head: Arrow, Wrap: true, Line: 3}),
				&Note{Place: Over, Of: []string{"A"}, Text: "y"},
				&Block{Kind: Loop, Sections: []Section{{Label: "t"}}},
			}}},
		{"participant types", `sequenceDiagram
participant D@{ "type": "database" }
actor G@{ "type" : "boundary" }
participant Q@{ "type": "queue", "alias": "Jobs" }
participant S@{ "alias": "X" } as Y
participant P@{ "type": "control", "alias": "Ctl" } as P
actor N@{ "type": null }`, &Diagram{
			Participants: []*Participant{
				{ID: "D", Label: "D", Shape: Database}, {ID: "G", Label: "G", Shape: Boundary},
				{ID: "Q", Label: "Jobs", Shape: Queue}, {ID: "S", Label: "Y"}, {ID: "P", Label: "Ctl", Shape: Control},
				{ID: "N", Label: "N", Shape: Figure},
			}}},
		{"arrows", `sequenceDiagram
A->B: 1
A-->>B: 2
A<<->>B: 3
A<<-->>B: 4
A--xB: 5
A-)B: 6
A--)B: 7`, &Diagram{
			Participants: ps("A", "B"),
			Body: []Stmt{
				msg("A", "B", "1", Message{Line: 2}),
				msg("A", "B", "2", Message{Head: Arrow, Dotted: true, Line: 3}),
				msg("A", "B", "3", Message{Head: Arrow, Both: true, Line: 4}),
				msg("A", "B", "4", Message{Head: Arrow, Both: true, Dotted: true, Line: 5}),
				msg("A", "B", "5", Message{Head: Cross, Dotted: true, Line: 6}),
				msg("A", "B", "6", Message{Head: Open, Line: 7}),
				msg("A", "B", "7", Message{Head: Open, Dotted: true, Line: 8}),
			}}},
		{"activations", `sequenceDiagram
A->>+B: a
activate B
B->>-A: b
deactivate B`, &Diagram{
			Participants: ps("A", "B"),
			Body: []Stmt{
				msg("A", "B", "a", Message{Head: Arrow, Activate: true, Line: 2}),
				&Activation{ID: "B", On: true, Line: 3},
				msg("B", "A", "b", Message{Head: Arrow, Deactivate: true, Line: 4}),
				&Activation{ID: "B", Line: 5},
			}}},
		{"blocks, with special characters and no labels", `sequenceDiagram
loop -:<>,;# comment
  alt yes
    A->>B: 1
  else ,<>:-#; comment
    Note over A,B: 2
  end
end
opt
end`, &Diagram{
			Participants: ps("A", "B"),
			Body: []Stmt{
				&Block{Kind: Loop, Sections: []Section{{Label: "-:<>,", Body: []Stmt{
					&Block{Kind: Alt, Sections: []Section{
						{Label: "yes", Body: []Stmt{msg("A", "B", "1", Message{Head: Arrow, Line: 4})}},
						{Label: ",<>:-", Body: []Stmt{&Note{Place: Over, Of: []string{"A", "B"}, Text: "2"}}},
					}},
				}}}},
				&Block{Kind: Opt, Sections: []Section{{}}},
			}}},
		{"autonumber", `sequenceDiagram
A->>B: 1
autonumber
A->>B: 2
loop
autonumber 10 .5
A->>B: 3
end
A->>B: 4
autonumber off
A->>B: 5`, &Diagram{
			Participants: ps("A", "B"),
			Body: []Stmt{
				msg("A", "B", "1", Message{Head: Arrow, Line: 2}),
				msg("A", "B", "2", Message{Head: Arrow, Number: "2", Line: 4}),
				&Block{Kind: Loop, Sections: []Section{{Body: []Stmt{
					msg("A", "B", "3", Message{Head: Arrow, Number: "10", Line: 7}),
				}}}},
				msg("A", "B", "4", Message{Head: Arrow, Number: "10.5", Line: 9}),
				msg("A", "B", "5", Message{Head: Arrow, Line: 11}),
			}}},
		{"%% in a text, and a number on the last line", "sequenceDiagram\nA->>B: 100%% sure\nautonumber 5", &Diagram{
			Participants: ps("A", "B"),
			Body:         []Stmt{msg("A", "B", "100%% sure", Message{Head: Arrow, Line: 2})},
		}},
	}
	for _, tt := range tests {
		d, err := read(tt.src)
		if err != nil {
			t.Errorf("%s: %v", tt.name, err)
			continue
		}
		if !reflect.DeepEqual(d, tt.want) {
			t.Errorf("%s:\n got %s\nwant %s", tt.name, dump(d), dump(tt.want))
		}
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct{ name, src, want string }{
		{"no header", "A->>B: hi", `line 1:1: unexpected "A", expected "sequenceDiagram"`},
		{"unclosed block", "sequenceDiagram\nloop x\nA->>B: hi", `line 4:1: unexpected end of input, expected "end"`},
		{"else outside alt", "sequenceDiagram\nloop\nelse\nend", `line 3:1: unexpected "else", expected a statement`},
		{"stray end", "sequenceDiagram\nend", `line 2:1: unexpected "end", expected a statement`},
		{"no text", "sequenceDiagram\nA->>B", `line 2:6: unexpected end of line, expected ":" and a text`},
		{"inactive", "sequenceDiagram\nA->>B: x\nB->>-A: y", `line 3:1: cannot deactivate "B": it is not active`},
		{"unknown", "sequenceDiagram\nactivate C", `line 2:1: cannot activate "C": no such participant`},
		{"deactivating an unknown participant", "sequenceDiagram\nA->>B: x\ndeactivate C", `line 3:1: cannot deactivate "C": it is not active`},
		{"thousandths", "sequenceDiagram\nautonumber 10.001\nA->>B: x", `line 2:12: unexpected "10.001", expected the end of the line`},
		{"two boxes", "sequenceDiagram\nbox X\nparticipant A\nend\nbox Y\nparticipant A\nend", `line 6:13: "A" is already in another box`},
		{"outside a box first", "sequenceDiagram\nA->>B: x\nbox X\nparticipant A\nend", `line 4:13: "A" has already appeared outside a box`},
		{"a message in a box", "sequenceDiagram\nbox X\nA->>B: x\nend", `line 3:1: unexpected "A", expected "end"`},
		{"unknown participant type", `sequenceDiagram
participant A@{ "type": "databse" }`, `line 2:14: unknown participant type "databse"`},
		{"empty settings", "sequenceDiagram\nparticipant A@{}", `line 2:14: unexpected "@{}", expected settings in it`},
		{"unclosed settings", "sequenceDiagram\nparticipant A@{ \"type\": \"queue\"", `line 2:14: "@{" is not closed`},
		{"settings not JSON", `sequenceDiagram
participant A@{ type: database }`, `line 2:14: @{...} that is not JSON is not supported yet`},
		{"() and +", "sequenceDiagram\nA()->>+B: x", `line 2:7: unexpected "+", expected a participant's name`},
		{"not a color", "sequenceDiagram\nrect rgb(1,2)\nend", `line 2:6: "rgb(1,2)" is not a color`},
	}
	for _, tt := range tests {
		_, err := read(tt.src)
		var e *syntax.Error
		if !errors.As(err, &e) || err.Error() != tt.want {
			t.Errorf("%s: error %v, want %s", tt.name, err, tt.want)
		}
	}
}

// dump shows d with its pointers followed, for a failing test.
func dump(d *Diagram) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%q |", d.Title)
	for _, p := range d.Participants {
		q := *p
		q.Group = nil
		fmt.Fprintf(&b, " %+v", q)
		if p.Group != nil {
			fmt.Fprintf(&b, " in %+v", *p.Group)
		}
	}
	dumpBody(&b, d.Body)
	return b.String()
}

func dumpBody(b *strings.Builder, body []Stmt) {
	for _, s := range body {
		switch s := s.(type) {
		case *Block:
			fmt.Fprintf(b, " [%d", s.Kind)
			for _, sec := range s.Sections {
				fmt.Fprintf(b, " %q:", sec.Label)
				dumpBody(b, sec.Body)
			}
			b.WriteString(" ]")
		case *Rect:
			b.WriteString(" [rect")
			if s.Color != nil {
				fmt.Fprintf(b, " %v", *s.Color)
			}
			dumpBody(b, s.Body)
			b.WriteString(" ]")
		default:
			fmt.Fprintf(b, " %+v", s)
		}
	}
}
