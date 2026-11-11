# Feature checklist

Checked against sumi's implementation at [`806adec`](https://github.com/bencode/sumi/commit/806adecd7f85e60e05d9d5a2a0fdf764d8b31023). The reference is **Mermaid v12.3.0 with default layout settings**, including its ELK-based graph layout and neo shapes. Open the [online playground](https://sumi.fmap.dev/) to edit and render the examples using the Go service.

- `[x]` means the public `mermaid.Layout` → `scene.Scene` → `svg.Render` path draws the feature.
- `[ ]` identifies a limitation; its description says whether it is **rejected**, **accepted but ignored**, **partial**, or **internal only**.
- A parser accepting syntax is not enough to mark a feature supported. Invalid source can also fail with an ordinary syntax or semantic error.

## Diagram types

- [x] [Sequence diagrams](#sequence-diagrams): `sequenceDiagram`.
- [x] [Flowcharts](#flowcharts): `flowchart`, `graph`, and `flowchart-elk`.
- [x] [State diagrams](#state-diagrams): `stateDiagram` and `stateDiagram-v2`.
- [x] [Class diagrams](#class-diagrams): `classDiagram` and `classDiagram-v2`.
- [x] [Entity relationship diagrams](#entity-relationship-diagrams): `erDiagram`.
- [x] [Gantt diagrams](#gantt-diagrams): `gantt`.
- [ ] Other diagram types, including pie, Git graphs, mind maps, timelines, and user journeys — **rejected** by public diagram dispatch.
- [ ] LaTeX math — **not implemented**; neither a public math API nor formula rendering in diagram labels is available.

Source: [public dispatch](../mermaid/mermaid.go).

## Sequence diagrams

- [x] Explicit and implicit participants, actors, and aliases. [Request example](https://sumi.fmap.dev/#sequence-request-response)
- [x] Request, return, open/asynchronous, and cross-ended messages; self-messages. [Request](https://sumi.fmap.dev/#sequence-request-response), [parallel](https://sumi.fmap.dev/#sequence-parallel-processing), [lifecycle](https://sumi.fmap.dev/#sequence-participant-lifecycle)
- [x] Explicit `activate` / `deactivate`, `+` / `-` message activations, and nested activation bars. [Request](https://sumi.fmap.dev/#sequence-request-response), [lifecycle](https://sumi.fmap.dev/#sequence-participant-lifecycle)
- [x] Notes left/right of a participant or over one/two participants; `<br>` line breaks. [Request example](https://sumi.fmap.dev/#sequence-request-response)
- [x] Automatic message numbering. [Request example](https://sumi.fmap.dev/#sequence-request-response)
- [x] `loop`, `alt` / `else`, `opt`, and `break` blocks, including nesting. [Branches example](https://sumi.fmap.dev/#sequence-branches-retries)
- [x] `par` / `and`, and `critical` / `option` blocks. [Parallel example](https://sumi.fmap.dev/#sequence-parallel-processing)
- [x] Participant `box` groups and `rect` background regions. [Lifecycle example](https://sumi.fmap.dev/#sequence-participant-lifecycle)
- [x] Participant creation and destruction. [Lifecycle example](https://sumi.fmap.dev/#sequence-participant-lifecycle)
- [x] Typed participants via single-line JSON `@{...}`; the gallery shows `queue` and `database`. [Lifecycle example](https://sumi.fmap.dev/#sequence-participant-lifecycle)
- [x] Diagram titles from front matter. [Request example](https://sumi.fmap.dev/#sequence-request-response)
- [ ] YAML or multiline participant settings — **rejected**; settings must be single-line JSON. All eight supported types (`participant`, `actor`, `boundary`, `control`, `entity`, `database`, `collections`, `queue`) are listed in [signal.go](../mermaid/internal/sequence/signal.go).
- [ ] Participant links and menus — **rejected**.
- [ ] Distinct overlapping `par_over` geometry — **partial**: accepted and drawn as `par`, following the referenced Mermaid implementation's default behavior.

Additional supported variants with fixture coverage: bidirectional and half arrows, central connections, numbering offsets/increments and `off`, and `wrap:` / `nowrap:` labels. See [sequence tests](../mermaid/internal/sequence/fixtures_test.go), [parser tests](../mermaid/internal/sequence/parse_test.go), and [grammar](../mermaid/internal/sequence/doc.go).

## Flowcharts

- [x] Direction declarations: `TB` / `TD`, `BT`, `LR`, and `RL`. [Decision](https://sumi.fmap.dev/#flowchart-decision-flow), [shapes](https://sumi.fmap.dev/#flowchart-shapes-connections)
- [x] Nodes, chained/fan-out links, branch labels, and cycles. [Decision](https://sumi.fmap.dev/#flowchart-decision-flow), [nested system](https://sumi.fmap.dev/#flowchart-nested-system)
- [x] Traditional shapes: rectangle, rounded rectangle, stadium, subroutine, cylinder, circle, double circle, diamond, hexagon, asymmetric, two parallelograms, and two trapezoids. [Shapes example](https://sumi.fmap.dev/#flowchart-shapes-connections)
- [x] Solid, dotted, thick, open, bidirectional, circle-ended, and cross-ended connections. [Shapes example](https://sumi.fmap.dev/#flowchart-shapes-connections)
- [x] Nested subgraphs, group titles, direction declarations, and edges crossing boundaries. [Nested system example](https://sumi.fmap.dev/#flowchart-nested-system)
- [x] Quoted text, entities, `<br>` breaks, Markdown bold/italic, and wrapped labels. [Labels example](https://sumi.fmap.dev/#flowchart-labels-styles)
- [x] `classDef`, `class`, `style`, and indexed `linkStyle` for the supported style properties. [Labels example](https://sumi.fmap.dev/#flowchart-labels-styles)
- [x] `@{ shape, label }` metadata for supported shapes; `labelType` selects plain/Markdown label handling. [Labels example](https://sumi.fmap.dev/#flowchart-labels-styles)
- [ ] Every modern Mermaid shape — **partial**: the traditional shape set and aliases in [metadata.go](../mermaid/internal/flowchart/metadata.go) work; other shape names are **rejected**.
- [ ] Icon/image nodes and metadata keys `icon`, `img`, `form`, `pos`, `constraint`, `w`, `h` — **rejected**.
- [ ] Edge animation (`animate`, `animation`, `step`) and subgraph `view` / `algorithm` metadata — **rejected**.
- [ ] Complete YAML in shape metadata — **partial**: the supported scalar mapping subset is read; unsupported YAML forms are **rejected**.
- [ ] Click URLs, callbacks, and tooltips as interactions — **accepted but not interactive**; statements retain a `clickable` style class only.
- [ ] Requested edge curves and extra link length — **accepted but ignored** by the default ELK route; `linkStyle`'s `interpolate` and metadata `curve` do not choose a different path.

Additional supported variants with fixture coverage: named edges, invisible links, `:::` class shorthand, and default link styles. A blank subgraph ID is rejected; more than 500 edges exceeds the default limit. See [fixtures](../mermaid/internal/flowchart/fixtures_test.go), [grammar](../mermaid/internal/flowchart/doc.go), and [metadata restrictions](../mermaid/internal/flowchart/resolve.go).

## State diagrams

- [x] Initial/final states, aliases, state descriptions, and labeled transitions. [Lifecycle example](https://sumi.fmap.dev/#state-lifecycle)
- [x] Composite states with local start/end nodes and direction declarations. [Composite example](https://sumi.fmap.dev/#state-composite-workflow)
- [x] Concurrent regions separated by `--`. [Composite example](https://sumi.fmap.dev/#state-composite-workflow)
- [x] `<<choice>>`, `<<fork>>`, and `<<join>>`. [Choices example](https://sumi.fmap.dev/#state-choices-parallelism)
- [x] Attached left/right notes, including multiline notes. [Lifecycle example](https://sumi.fmap.dev/#state-lifecycle)
- [x] Class definitions, class assignments, and inline supported styles. [Choices example](https://sumi.fmap.dev/#state-choices-parallelism)
- [ ] Click links and tooltips — **accepted but ignored**.
- [ ] `scale ... width`, `hide empty description`, floating `note ... as ...`, and a bare `state ID` without a body — **accepted but ignored** as drawing commands.
- [ ] A literal `<` in a state name, or an empty title/description alongside its non-empty counterpart — **rejected** for the affected forms.

Source: [grammar](../mermaid/internal/state/doc.go), [ignored statements](../mermaid/internal/state/parse.go), [size restrictions](../mermaid/internal/state/size.go).

## Class diagrams

- [x] Classes, attributes, methods, parameters, and return types. [Members example](https://sumi.fmap.dev/#class-members)
- [x] `+`, `-`, `#`, and `~` visibility; static `$` and abstract `*` members; annotations such as `<<interface>>`. [Members example](https://sumi.fmap.dev/#class-members)
- [x] Generic notation using `~T~`. [Members example](https://sumi.fmap.dev/#class-members)
- [x] Inheritance, realization, association, dependency, aggregation, and composition. [Relationships example](https://sumi.fmap.dev/#class-relationships)
- [x] Relation labels, endpoint multiplicities, solid/dashed lines, and direction declarations. [Relationships example](https://sumi.fmap.dev/#class-relationships)
- [x] Nested namespaces and relations between their classes. [Namespaces example](https://sumi.fmap.dev/#class-namespaces)
- [x] Attached and standalone notes. [Namespaces example](https://sumi.fmap.dev/#class-namespaces)
- [x] `classDef`, class assignment, and inline supported styles. [Namespaces example](https://sumi.fmap.dev/#class-namespaces)
- [ ] Links, callbacks, and click statements as interactions — **accepted but not interactive**; a `clickable` style class may be retained.
- [ ] A literal `<` in class/member text — **rejected**; use Mermaid's `~T~` generic notation instead of angle brackets.
- [ ] Blank lines in a class label, or a generic type not beginning with a letter beside a custom label — **rejected** for the affected forms.
- [ ] A class sharing its identifier with a namespace or note — **rejected**.

Lollipop interfaces and class aliases also have [parser coverage](../mermaid/internal/class/read_test.go). Styles follow Mermaid's statement order: a class definition can affect only classes that already exist at that point. See [grammar](../mermaid/internal/class/doc.go), [size restrictions](../mermaid/internal/class/size.go), and [layout restrictions](../mermaid/internal/class/layout.go).

## Entity relationship diagrams

- [x] Exactly-one, zero-or-one, zero-or-more, and one-or-more cardinalities. [Cardinalities example](https://sumi.fmap.dev/#er-cardinalities)
- [x] Identifying/non-identifying relationships and role labels. [Cardinalities example](https://sumi.fmap.dev/#er-cardinalities)
- [x] Entities, display aliases, attribute names, and types. [Attributes example](https://sumi.fmap.dev/#er-attributes)
- [x] `PK`, `FK`, `UK`, combined keys, nullable `?` types, and quoted attribute comments. [Attributes example](https://sumi.fmap.dev/#er-attributes)
- [x] Direction declarations, subgraph grouping, and cross-group relationships. [Business model example](https://sumi.fmap.dev/#er-business-model)
- [x] Class definitions/assignments and supported entity/group styles. [Business model example](https://sumi.fmap.dev/#er-business-model)
- [ ] Duplicate/blank subgraph names, or a literal `<` in a subgraph title — **rejected**.

Source: [grammar](../mermaid/internal/er/doc.go), [collection](../mermaid/internal/er/collect.go), [placement restrictions](../mermaid/internal/er/place.go), [drawing tests](../mermaid/internal/er/draw_test.go).

## Gantt diagrams

- [x] Sections, tasks with ids, start dates, `after` and `until` dependencies, and durations; tasks with no start follow the previous one. [Reading tests](../mermaid/internal/gantt/read_test.go)
- [x] `dateFormat` with dayjs's tokens, and start dates in no format read as JavaScript's `new Date` reads them. [Reading tests](../mermaid/internal/gantt/read_test.go)
- [x] `done`, `active`, and `crit` tasks, milestones, and `vert` markers, each colored by its status. [Drawing tests](../mermaid/internal/gantt/draw_test.go)
- [x] `excludes` and `includes` with dates, weekdays, and `weekends`; `weekday` and `weekend` choices; `inclusiveEndDates`.
- [x] `axisFormat` and `tickInterval`, with d3's ticks and en-US labels. [Axis tests](../mermaid/internal/gantt/axis_test.go)
- [x] `title`, front matter titles, and multi-line section titles with `<br>`.
- [x] The line at the time now, `todayMarker off`, and `todayMarker` styles with supported properties. The time now is `time.Now()` in `time.Local`.
- [x] The chart width from `mermaid.Options.Width`; 0 is 1200.
- [ ] Compact mode (`displayMode: compact`), `topAxis`, and other `gantt` configuration — **accepted but ignored** in front matter and directives. The `topAxis` statement fails, as in Mermaid 12.3.0.
- [ ] An `opacity` in `todayMarker` — **rejected**, as opacity is in other styles.
- [ ] Click links and callbacks — **accepted but not interactive**; a clickable task's text is bold and in the link color.
- [ ] Times beyond what a JavaScript `Date` holds — **rejected**.

Source: [reading and drawing](../mermaid/internal/gantt/doc.go).

## Shared rendering and limitations

- [x] Top-level front matter `title`; source comments and normalized line endings. [Request source](examples/sequence/request-response.mmd), [preprocessing tests](../mermaid/internal/syntax/front_test.go)
- [x] Light/dark semantic palettes and custom `scene.Palette` values. Author-specified colors remain fixed across themes. [Gallery](https://sumi.fmap.dev/), [paint contract](../scene/paint.go)
- [x] Pluggable `scene.Measurer` for native backends; nil uses built-in Arial measurements. [Text contract](../scene/text.go)
- [x] Positioned paths/text, semantic item IDs, and drawing order in `scene.Scene`; SVG export. [Scene contract](../scene/scene.go), [SVG tests](../svg/svg_test.go)
- [x] Supported styles: `fill`, `stroke`, `color`, `stroke-width`, and `stroke-dasharray`, with the color/value forms the parser reads. [Styled labels example](https://sumi.fmap.dev/#flowchart-labels-styles)
- [x] Line/column errors and `IsUnsupported()` for explicitly recognized unsupported features. Not every unrecognized Mermaid construct is classified as unsupported. [API tests](../mermaid/mermaid_test.go)
- [ ] Front matter `config`, including layout/theme configuration, and `%%{init}%%` directives — **accepted but ignored**. [Preprocessor](../mermaid/internal/syntax/front.go)
- [ ] Mermaid theme names, custom CSS, and browser font selection through configuration — **not applied**. Select a sumi palette/measurer in Go instead.
- [ ] Diagram-provided accessibility title/description metadata — **accepted without accessible scene/SVG metadata**; `scene.Scene` has no corresponding fields. The playground provides its own image descriptions.
- [ ] Arbitrary HTML, label formulas, and Font Awesome icons in labels handled by the shared label reader — **rejected**; `<br>` line breaks are supported. [Label reader](../mermaid/internal/label/label.go)
- [ ] Complete Markdown in those labels — **partial**: paragraph text, emphasis, escapes, and supported breaks work; links, images, code, lists, headings, tables, and other block/token forms can be **rejected**. [Markdown reader](../mermaid/internal/label/markdown.go)
- [ ] Browser-dependent CSS such as font properties, opacity properties, transforms, filters, animation, and text-layout properties — **rejected** when the shared style checker recognizes them. CSS with no effect on the referenced SVG shapes can be ignored; this is not a general CSS engine. [Exact property checks](../mermaid/internal/style/style.go)
- [ ] Browser-dependent paint values such as `currentColor` or CSS variables — **rejected** in supported paint properties when not understood by the color parser.
- [ ] Pixel-identical text on every machine — **partial**: default Arial metrics are deterministic, but SVG uses the viewer's installed Arial/Helvetica/sans-serif fonts; non-ASCII and italic measurements are approximations. [Arial metrics](../scene/arial.go)
- [ ] SVG hyperlinks, callbacks, animation, or embedded fonts — **not emitted** by `svg.Render`. [SVG writer](../svg/svg.go)

## Example coverage

The 17 examples cover every checked common-feature group in the first five diagram sections above; Gantt diagrams have no example yet and link their tests instead. They combine related features rather than repeat one diagram for each syntax spelling. Additional variants and unusual inputs are linked to existing tests or source instead of expanding the gallery.

This is coverage of this checklist's common scenarios, **not** a measured share of user workloads, a Mermaid syntax compatibility percentage, or a guarantee that every combination works. When adding a supported feature, update its checklist entry and either extend a relevant example or link the behavior test that demonstrates it.
