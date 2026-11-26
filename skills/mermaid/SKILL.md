---
name: mermaid
description: Check and look at Mermaid diagrams you write, with the sumi command. Use whenever you write or edit Mermaid, in a Markdown file's mermaid code blocks or a .mmd file - design docs, reviews, READMEs, PR descriptions - to find syntax errors by file line and column, and to render each diagram as a PNG you read to see its layout before handing it over.
---

# Mermaid

`sumi` checks Mermaid diagrams and renders them as images, laid out by [sumi](https://github.com/bencode/sumi), the Go library that draws Mermaid in Jun. It needs no browser, Node or network.

## Install

```sh
go install github.com/bencode/sumi/cmd/sumi@latest
```

It lands in `$(go env GOPATH)/bin/sumi`; call it by that path if `sumi` is not on `PATH`. If it is missing and you cannot install it, tell the user rather than reaching for `npx` or another renderer.

## The loop

After writing or editing diagrams, before you hand them over:

1. `sumi check doc.md` checks every mermaid code block of a Markdown file; any other file is read as one diagram. Exit status 0 means every diagram is good.
2. Fix each error it reports, and check again until it passes.
3. `sumi render doc.md -o <dir>` writes `<dir>/doc-01.png`, `doc-02.png`, ..., one per block in order, and prints a line for each:

   ```text
   01  doc.md:13  out/doc-01.png  851x649
   ```

   the number, the line its source starts on, the image, and its size in CSS pixels. Use a scratch directory, not the repository.
4. Read each PNG. Look for labels cut off or crowded, edges crossing or running through nodes, nodes in the wrong group, and a diagram far wider than tall (a long `LR` chain reads badly in a page: try `TD`, or split it). Adjust and render again.

`render` checks first and writes nothing if any diagram has an error. Options: `-scale N` (default 2) and `-format svg`.

## Reading errors

Errors go to standard error as `FILE:LINE:COL: message`, in the lines of the file, with the line and a caret under the column:

```text
doc.md:44:9: unexpected "(", expected "]"
      A[路径/a(b): c] --> B
              ^
```

Each block reports its first error. Exit status 1 means a diagram has an error; 2 means the command could not run (usage, a file, no font).

- `unexpected end of input, expected X` is reported at the end of the block, its closing fence: something opened earlier was never closed. Look back for the unclosed `X` - a bracket, a `|` of an edge label, a quote, or a `subgraph`/`alt`/`loop` without its `end`.
- `... is not supported yet` is Mermaid that sumi does not draw: a diagram type (sumi draws flowchart/graph, sequenceDiagram, stateDiagram(-v2), classDiagram, erDiagram, gantt and pie) or a feature of one. Jun will not draw it either. Rewrite it with what is supported, or tell the user.

## Known pitfalls

- Quote a label that holds parentheses: `A["run (once)"]`. `:` and `/` are fine unquoted.
- `end` in lower case cannot name a node; use `End` or `e[end]`.
- In a sequence diagram, `;` ends the statement, cutting a message short. Write it as `#59;`.
- `<br/>` breaks a line in a label.
- A passing check says the syntax is right, not the diagram. A node with the same id as a subgraph, for instance, checks fine, but there is no such node: its edges end at the subgraph's box. Look at the image.
