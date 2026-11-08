package state

import (
	"fmt"

	"github.com/bencode/sumi/mermaid/internal/syntax"
)

// translateDoc is docTranslator for the document of the state id, written at at: a
// [*] becomes its parent's start or end, and each -- closes a region of what comes
// before it.
func translateDoc(id string, at syntax.Pos, doc []Stmt) []Stmt {
	regioned := regions(at, doc)
	out := make([]Stmt, len(regioned))
	for i, s := range regioned {
		out[i] = translate(id, s)
	}
	return out
}

func translate(parent string, s Stmt) Stmt {
	switch v := s.(type) {
	case Relation:
		v.State1, v.State2 = translateState(parent, v.State1, true), translateState(parent, v.State2, false)
		return v
	case State:
		return translateState(parent, v, true)
	}
	return s
}

func translateState(parent string, s State, first bool) State {
	if s.ID == "[*]" {
		s.ID, s.Pseudo = parent+"_end", pseudoEnd
		if first {
			s.ID, s.Pseudo = parent+"_start", pseudoStart
		}
	} else {
		s.ID = trim(s.ID)
	}
	if s.Doc != nil {
		s.Doc = translateDoc(s.ID, s.At, s.Doc)
	}
	return s
}

// regions are doc's statements gathered in the regions its --s close, the last by a
// region of its own, which Mermaid gives a random id and sumi one no state can have.
// With nothing after the last --, Mermaid leaves doc as it was.
func regions(at syntax.Pos, doc []Stmt) []Stmt {
	var out []Stmt
	current := []Stmt{}
	for _, s := range doc {
		if d, ok := s.(State); ok && d.Type == "divider" {
			d.Doc = current
			out, current = append(out, d), []Stmt{}
			continue
		}
		current = append(current, s)
	}
	if len(out) == 0 || len(current) == 0 {
		return doc
	}
	return append(out, State{At: at, ID: fmt.Sprintf("divider\n%d:%d", at.Line, at.Col), Type: "divider", Doc: current})
}
