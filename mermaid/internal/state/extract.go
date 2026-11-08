package state

import "strings"

// tables are what the statements at the root say of classes and styles
// (StateDB.extract): the ones in a composite state's body do nothing.
type tables struct {
	classDefs map[string][]string // the styles of a classDef
	classes   map[string][]string // a state's classes
	styles    map[string][]string // a state's style statement
}

func extract(doc []Stmt) tables {
	t := tables{map[string][]string{}, map[string][]string{}, map[string][]string{}}
	for _, s := range doc {
		switch v := s.(type) {
		case Relation:
			t.applyClasses(v.State1)
			t.applyClasses(v.State2)
		case ClassDef:
			t.classDefs[v.ID] = append(t.classDefs[v.ID], declarations(v.Styles)...)
		case Style:
			for _, id := range strings.Split(v.ID, ",") {
				t.styles[trim(id)] = styleEntries(v.Styles)
			}
		case ApplyClass:
			t.addClass(v.IDs, v.Class)
		}
	}
	return t
}

// applyClasses is what a transition's A:::c does; a statement of A:::c alone does
// nothing.
func (t tables) applyClasses(s State) {
	for _, c := range s.Classes {
		t.addClass(s.ID, c)
	}
}

func (t tables) addClass(ids, class string) {
	for _, id := range strings.Split(ids, ",") {
		t.classes[trim(id)] = append(t.classes[trim(id)], class)
	}
}

// declarations are addStyleClass's styles of a classDef: each loses its first ;.
func declarations(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	for i, p := range parts {
		parts[i] = trim(strings.Replace(p, ";", "", 1))
	}
	return parts
}

// styleEntries are handleStyleDef's: each loses all its ;s.
func styleEntries(s string) []string {
	parts := strings.Split(s, ",")
	for i, p := range parts {
		parts[i] = trim(strings.ReplaceAll(p, ";", ""))
	}
	return parts
}
