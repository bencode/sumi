package sequence

import (
	"slices"

	"github.com/bencode/sumi/mermaid/internal/syntax"
	"github.com/bencode/sumi/scene"
)

// A life is what a message does to its participants' lifelines: it creates
// To, or ends From's or To's.
type life struct{ creates, destroysFrom, destroysTo bool }

// lifecycles finds each create's and destroy's message, with Mermaid's
// checks (sequenceDb.ts lines 581-645): the next message must be to the
// participant created; a destroy waits for the next message that creates
// nothing, which must be from or to the participant destroyed. Mermaid
// quietly drops a create or destroy whose message is not the very next
// statement, and ends one lifeline of a message that ends two; sumi takes
// each to its message, whatever is in between. A create or destroy that no
// message follows fails, which Mermaid quietly drops too.
func lifecycles(body []Stmt) (map[*Message]life, error) {
	lives := map[*Message]life{}
	var created []*create
	var destroyed, fresh []*destroy // fresh: those no message follows yet
	var err error
	walk(body, func(s Stmt) {
		if err != nil {
			return
		}
		switch s := s.(type) {
		case *create:
			created = append(created, s)
		case *destroy:
			destroyed, fresh = append(destroyed, s), append(fresh, s)
		case *Message:
			lv := life{creates: len(created) > 0}
			for _, c := range created {
				if s.To != c.d.p.ID {
					err = syntax.Errorf(c.d.pos, "%q is created, but the next message is not to it", c.d.p.ID)
					return
				}
			}
			var waiting []*destroy
			for _, d := range destroyed {
				switch {
				case s.From == d.id || s.To == d.id:
					lv.destroysFrom = lv.destroysFrom || s.From == d.id
					lv.destroysTo = lv.destroysTo || s.To == d.id
				case lv.creates:
					waiting = append(waiting, d)
				default:
					err = syntax.Errorf(d.pos, "%q is destroyed, but the next message is neither from nor to it", d.id)
					return
				}
			}
			lives[s], created, destroyed, fresh = lv, nil, waiting, nil
		}
	})
	switch {
	case err != nil:
		return nil, err
	case len(created) > 0:
		return nil, syntax.Errorf(created[0].d.pos, "%q is created, but no message follows", created[0].d.p.ID)
	case len(fresh) > 0:
		return nil, syntax.Errorf(fresh[0].pos, "%q is destroyed, but no message follows", fresh[0].id)
	}
	return lives, nil
}

// created widens the gap before each participant made by create by half
// its width, or the room before the first participant (addActorRenderingData,
// sequenceRenderer.ts lines 785-788). Mermaid leaves this out for one made
// before any other statement, as it reads that one's place, 0, as false;
// sumi does not.
func created(ps []*Participant, w []float64, lead float64, g []float64) (float64, []float64) {
	out := slices.Clone(g)
	for i, p := range ps {
		switch {
		case !p.Created:
		case i == 0:
			lead += w[i] / 2
		default:
			out[i-1] += w[i] / 2
		}
	}
	return lead, out
}

// figured are the shapes whose drawn figure, not their width, a message to
// or from them stops at when it creates or destroys them: Mermaid's
// actorArray, which leaves out boundary.
var figured = map[Shape]bool{Figure: true, Control: true, Entity: true, Database: true}

// lifeEnds is a message that creates or destroys a participant as Mermaid
// adjusts it (adjustCreatedDestroyedData, sequenceRenderer.ts lines
// 946-1033): its line's ends moved in to the participant's box, which is
// drawn about the line, and the room Mermaid's bounds take for the box,
// down to noteMargin under it. ok is false for any other message. A
// message to itself keeps its loop: Mermaid moves one end of it, and draws
// it as a straight line then.
func lifeEnds(p *pMessage, e env) (mm msgModel, room scene.Rect, ok bool) {
	mm, ok = p.model, p.Creates || p.DestroysFrom || p.DestroysTo
	if !ok || p.From == p.To {
		return mm, room, ok
	}
	bottom := p.y + e.c.height/2 + e.c.noteMargin
	inset := func(id string) (float64, bool) { // how far id's end moves in, and whether id is left of the other end
		i, j := e.at[id], e.at[p.From]
		if id == p.From {
			j = e.at[p.To]
		}
		d := e.w[i] / 2
		if figured[e.ps[i].Shape] {
			d = 18 // ACTOR_TYPE_WIDTH / 2
		}
		return d, e.x[i]-e.w[i]/2 < e.x[j]-e.w[j]/2
	}
	var r scene.Rect
	if p.DestroysFrom {
		d, left := inset(p.From)
		if left {
			r, mm.startx = between(mm.startx-d, p.top, mm.stopx, bottom), mm.startx+d
		} else {
			r, mm.startx = between(mm.stopx, p.top, mm.startx+d, bottom), mm.startx-d
		}
		room = union(room, r)
	}
	if p.Creates || p.DestroysTo {
		d, left := inset(p.To)
		d += 3
		if left {
			r, mm.stopx = between(mm.stopx-d, p.top, mm.startx, bottom), mm.stopx+d
		} else {
			r, mm.stopx = between(mm.startx, p.top, mm.stopx+d, bottom), mm.stopx-d
		}
		room = union(room, r)
	}
	return mm, room, ok
}

// lifelines are where each participant's top and foot are drawn: at the
// top and the foot of the diagram, or, for one created or destroyed, about
// the line of the message that does it.
func lifelines(l layout) (tops, feet []float64) {
	n := len(l.d.Participants)
	tops, feet = make([]float64, n), make([]float64, n)
	for i := range n {
		tops[i], feet[i] = l.top, l.bottom
	}
	for _, it := range l.items {
		p, ok := it.(*pMessage)
		if !ok {
			continue
		}
		y := p.y - l.e.c.height/2
		if p.Creates {
			tops[l.e.at[p.To]] = y
		}
		if p.DestroysFrom {
			feet[l.e.at[p.From]] = y
		}
		if p.DestroysTo {
			feet[l.e.at[p.To]] = y
		}
	}
	return tops, feet
}
