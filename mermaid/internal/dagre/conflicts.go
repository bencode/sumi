package dagre

import "slices"

// A conflict is an entry on its way through resolveConflicts: its
// constraints in the constraint graph, and whether it went into another.
type conflict struct {
	entry
	indegree int
	in, out  []*conflict
	merged   bool
}

// resolveConflicts goes through entries in the order of cg's constraints.
// An entry that must stay left of another, but whose barycenter would not
// keep it there, goes into that other, in front, with their barycenters
// merged. The entries left can then be sorted by barycenter alone
// (resolve-conflicts.js line 30, after Forster 2004).
func resolveConflicts(entries []*entry, cg *constraints) []*entry {
	var mapped keys[*conflict]
	for _, e := range entries {
		mapped.set(e.vs[0], &conflict{entry: *e})
	}
	for _, k := range cg.edgeList() {
		v, okV := mapped.get(k.v)
		w, okW := mapped.get(k.w)
		if okV && okW {
			w.indegree++
			v.out = append(v.out, w)
		}
	}
	var sources []*conflict
	for _, c := range mapped.values() {
		if c.indegree == 0 {
			sources = append(sources, c)
		}
	}
	return doResolveConflicts(sources)
}

// doResolveConflicts takes sources as a stack (resolve-conflicts.js line
// 65).
func doResolveConflicts(sources []*conflict) []*entry {
	var done []*conflict
	for len(sources) > 0 {
		c := sources[len(sources)-1]
		sources = sources[:len(sources)-1]
		done = append(done, c)
		slices.Reverse(c.in)
		for _, u := range c.in {
			if !u.merged && (!u.ok || !c.ok || u.value >= c.value) {
				mergeEntries(c, u)
			}
		}
		for _, w := range c.out {
			w.in = append(w.in, c)
			w.indegree--
			if w.indegree == 0 {
				sources = append(sources, w)
			}
		}
	}
	var out []*entry
	for _, c := range done {
		if !c.merged {
			out = append(out, &c.entry)
		}
	}
	return out
}

// mergeEntries puts source in front of target (resolve-conflicts.js line
// 109).
func mergeEntries(target, source *conflict) {
	sum, weight := 0.0, 0
	if target.weight != 0 {
		sum += float64(target.value * float64(target.weight))
		weight += target.weight
	}
	if source.weight != 0 {
		sum += float64(source.value * float64(source.weight))
		weight += source.weight
	}
	target.vs = slices.Concat(source.vs, target.vs)
	target.center = center{sum / float64(weight), weight, true}
	target.i = min(source.i, target.i)
	source.merged = true
}
