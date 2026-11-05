// SPDX-License-Identifier: EPL-2.0 OR GPL-3.0-or-later

package elk

import (
	"slices"
	"strings"
	"testing"
)

// The values expected below come from elkjs 0.12.1.

// layered runs the steps up to splitLongEdges.
func layered(spec string) *Graph {
	g := graphOf(spec)
	breakCycles(g)
	assignLayers(g)
	splitLongEdges(g)
	assignPortSides(g)
	sortPortLists(g)
	return g
}

// portNames writes the ports of g's real nodes like "a:e0,e1 b:e0", each port
// by its edge.
func portNames(g *Graph) string {
	var all []string
	for _, n := range nodes(g) {
		if n.Type != typeNormal {
			continue
		}
		ids := make([]string, len(n.Ports))
		for i, p := range n.Ports {
			ids[i] = slices.Concat(p.In, p.Out)[0].ID
		}
		all = append(all, n.ID+":"+strings.Join(ids, ","))
	}
	return strings.Join(all, " ")
}

func TestRandom(t *testing.T) {
	tests := []struct {
		seed          int64
		high, low     int32
		boolean       bool
		double, float float64
	}{
		{1, -1155869325, 431529176, false, 0.4074398012118764, 0.03623533248901367},
		{42, -1170105035, 234785527, true, 0.047939305137387644, 0.9420735239982605},
		{-7, 1155869324, -423064701, true, 0.9458454776032215, 0.1464257836341858},
	}
	for _, tt := range tests {
		r := newRandom(tt.seed)
		if got, want := r.nextLong(), int64(tt.high)<<32+int64(tt.low); got != want {
			t.Errorf("seed %d: nextLong %d, want %d", tt.seed, got, want)
		}
		if got := r.nextBoolean(); got != tt.boolean {
			t.Errorf("seed %d: nextBoolean %v", tt.seed, got)
		}
		if got := r.nextDouble(); got != tt.double {
			t.Errorf("seed %d: nextDouble %v, want %v", tt.seed, got, tt.double)
		}
		if got := r.nextFloat(); got != tt.float {
			t.Errorf("seed %d: nextFloat %v, want %v", tt.seed, got, tt.float)
		}
	}
}

func TestPortLists(t *testing.T) {
	tests := []struct{ spec, ports string }{
		{"a>e b>e c>e", "a:e0 b:e1 c:e2 e:e0,e1,e2"},
		{"a>e b>e c>e d>e", "a:e0 b:e1 c:e2 d:e3 e:e2,e1,e0,e3"},
		{"a>e b>e c>e d>e f>e", "a:e0 b:e1 c:e2 d:e3 f:e4 e:e3,e2,e1,e0,e4"},
		{"x>a x>b a>c b>c x>c c>d", "x:e0,e1,e4 a:e2,e0 b:e3,e1 c:e5,e2,e3,e4 d:e5"},
	}
	for _, tt := range tests {
		g := layered(tt.spec)
		if got := portNames(g); got != tt.ports {
			t.Errorf("%s: ports %s, want %s", tt.spec, got, tt.ports)
		}
		for _, n := range nodes(g) {
			if n.Type == typeLongEdge && n.Ports[0].Side != sideEast {
				t.Errorf("%s: dummy node's ports start on side %d", tt.spec, n.Ports[0].Side)
			}
		}
	}
}

func TestOrder(t *testing.T) {
	tests := []struct{ spec, modelLayers, modelPorts, layers, ports string }{
		{
			"a>b a>c c>d",
			"a | b,c | d",
			"a:e0,e1 b:e0 c:e2,e1 d:e2",
			"a | b,c | d",
			"a:e0,e1 b:e0 c:e2,e1 d:e2",
		},
		{
			"a>b c>a b>d e>d b>f g>f g>h e>h",
			"c | a | b,e,g | d,f,h",
			"c:e1 a:e0,e1 b:e2,e4,e0 e:e3,e7 g:e5,e6 d:e3,e2 f:e5,e4 h:e6,e7",
			"c | a | b,e,g | d,f,h",
			"c:e1 a:e0,e1 b:e2,e4,e0 e:e3,e7 g:e5,e6 d:e3,e2 f:e5,e4 h:e6,e7",
		},
		{
			"a>b b>c c>d b>e a>f b>c a>c b>c b>e",
			"a | b,f,. | c,e | d",
			"a:e0,e4,e6 b:e1,e5,e7,e3,e8,e0 f:e4 c:e2,e6,e7,e5,e1 e:e8,e3 d:e2",
			"a | b,f,. | e,c | d",
			"a:e0,e4,e6 b:e3,e8,e1,e5,e7,e0 f:e4 e:e8,e3 c:e2,e6,e7,e5,e1 d:e2",
		},
		{
			"a>b b>c a>d d>e c>f a>g b>h g>i f>j b>k h>j d>f",
			"a | b,d,g | c,.,e,i,k,. | f,h | j",
			"a:e0,e2,e5 b:e1,e6,e9,e0 d:e3,e11,e2 g:e7,e5 c:e4,e1 e:e3 i:e7 k:e9 f:e8,e11,e4 h:e10,e6 j:e10,e8",
			"a | b,d,g | k,.,c,.,e,i | h,f | j",
			"a:e0,e2,e5 b:e9,e6,e1,e0 d:e11,e3,e2 g:e7,e5 k:e9 c:e4,e1 e:e3 i:e7 h:e10,e6 f:e8,e11,e4 j:e8,e10",
		},
		{
			"a>b b>c a>d d>e b>f b>g g>h b>i i>j d>f i>b c>d h>i b>f",
			"a | b,. | c,.,.,g,.,.,. | .,.,d,h,.,. | e,f,i | j",
			"a:e0,e2 b:e1,e4,e13,e5,e7,e10,e0 c:e11,e1 g:e6,e5 d:e3,e9,e2,e11 h:e12,e6 e:e3 f:e9,e13,e4 i:e8,e10,e7,e12 j:e8",
			"a | b,. | g,.,.,c,.,.,. | h,.,.,d,.,. | i,e,f | j",
			"a:e0,e2 b:e5,e7,e10,e1,e4,e13,e0 g:e6,e5 c:e11,e1 h:e12,e6 d:e3,e9,e2,e11 i:e8,e10,e7,e12 e:e3 f:e13,e4,e9 j:e8",
		},
		{
			"b>a a>c b>d e>c a>c f>g h>g g>i j>f i>k l>g m>k n>h g>h g>i f>i f>k",
			"b,j | a,d,e,f,l | c,.,.,g,n | .,h,i,m | k",
			"b:e0,e2 j:e8 a:e1,e4,e0 d:e2 e:e3 f:e5,e15,e16,e8 l:e10 c:e3,e4,e1 g:e7,e14,e6,e13,e10,e5 n:e12 h:e12,e13,e6 i:e9,e14,e7,e15 m:e11 k:e11,e9,e16",
			"j,b | f,l,a,d,e | .,.,g,n,c | .,m,i,h | k",
			"j:e8 b:e0,e2 f:e16,e15,e5,e8 l:e10 a:e1,e4,e0 d:e2 e:e3 g:e7,e14,e6,e13,e10,e5 n:e12 c:e3,e4,e1 m:e11 i:e9,e14,e7,e15 h:e12,e13,e6 k:e9,e11,e16",
		},
		{
			"a>b a>c b>d c>g a>e d>e c>e b>e m>s o>q r>s h>t k>u p>t n>r j>l h>l k>o r>t m>n h>q v>y y>n27 y>n28 n28>n29 n27>n31 y>n33 n26>n31 w>n29 x>n27 x>z y>n32 v>n31 x>n29",
			"a,m,v | b,c,.,.,k,n,y,.,x | d,.,g,.,.,.,o,r,h,u,p,j,n27,n28,n33,n26,w,z,n32,.,. | e,s,q,t,l,n29,n31",
			"a:e0,e1,e4 m:e8,e19 v:e21,e32 b:e2,e7,e0 c:e3,e6,e1 k:e12,e17 n:e14,e19 y:e22,e23,e26,e31,e21 x:e29,e30,e33 d:e5,e2 g:e3 o:e9,e17 r:e10,e18,e14 h:e11,e16,e20 u:e12 p:e13 j:e15 n27:e25,e29,e22 n28:e24,e23 n33:e26 n26:e27 w:e28 z:e30 n32:e31 e:e4,e6,e7,e5 s:e10,e8 q:e20,e9 t:e13,e11,e18 l:e15,e16 n29:e33,e28,e24 n31:e32,e27,e25",
			"a,m,v | c,b,.,k,.,n,x,y,. | .,g,d,.,.,j,h,o,u,.,r,p,w,.,z,n33,n28,n27,n32,n26,. | e,l,q,s,t,n29,n31",
			"a:e1,e0,e4 m:e8,e19 v:e21,e32 c:e6,e3,e1 b:e2,e7,e0 k:e17,e12 n:e14,e19 x:e33,e30,e29 y:e26,e23,e22,e31,e21 g:e3 d:e5,e2 j:e15 h:e16,e20,e11 o:e9,e17 u:e12 r:e10,e18,e14 p:e13 w:e28 z:e30 n33:e26 n28:e24,e23 n27:e25,e22,e29 n32:e31 n26:e27 e:e4,e7,e5,e6 l:e16,e15 q:e9,e20 s:e10,e8 t:e13,e18,e11 n29:e24,e33,e28 n31:e32,e27,e25",
		},
	}
	for _, tt := range tests {
		g := layered(tt.spec)
		sortByModelOrder(g)
		if got := layerNames(g); got != tt.modelLayers {
			t.Errorf("%s: model order %s, want %s", tt.spec, got, tt.modelLayers)
		}
		if got := portNames(g); got != tt.modelPorts {
			t.Errorf("%s: model ports %s, want %s", tt.spec, got, tt.modelPorts)
		}
		minimizeCrossings(g)
		if got := layerNames(g); got != tt.layers {
			t.Errorf("%s: layers %s, want %s", tt.spec, got, tt.layers)
		}
		if got := portNames(g); got != tt.ports {
			t.Errorf("%s: ports %s, want %s", tt.spec, got, tt.ports)
		}
	}
}
