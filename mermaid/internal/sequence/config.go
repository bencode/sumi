package sequence

// config holds Mermaid's sequence diagram defaults (config.schema.yaml,
// SequenceDiagramConfig) and the fixed sizes its renderer uses
// (sequenceRenderer.ts, svgDraw.js). There is no margin around the diagram:
// a scene's content starts at (0, 0), and the caller adds what room it
// wants.
type config struct {
	width, height   float64 // a participant's least width, and its height
	actorMargin     float64 // the least gap between participants
	wrapPadding     float64 // room on each side of a message's text
	noteMargin      float64 // a note's padding
	boxMargin       float64 // spacing around blocks and notes
	boxTextMargin   float64
	labelBoxWidth   float64 // a block's keyword tab
	labelBoxHeight  float64
	activationWidth float64
	actorFont       float64 // font sizes
	noteFont        float64
	messageFont     float64
	numberFont      float64 // an autonumber's digits, when there are 3 or fewer
	numberRadius    float64 // the number marker's radius, before the 2px stroke scales it
	titleFont       float64
	footerGap       float64 // between the last statement and the bottom participants
}

var defaults = config{
	width:           150,
	height:          65,
	actorMargin:     50,
	wrapPadding:     10,
	noteMargin:      10,
	boxMargin:       10,
	boxTextMargin:   5,
	labelBoxWidth:   50,
	labelBoxHeight:  20,
	activationWidth: 10,
	actorFont:       14,
	noteFont:        14,
	messageFont:     16,
	numberFont:      12,
	numberRadius:    6,
	titleFont:       16,
	footerGap:       20,
}
