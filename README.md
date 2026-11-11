# sumi

sumi (墨, ink) lays out Mermaid diagrams in pure Go. It returns paths and positioned text that your application draws natively or exports as SVG.

- **[Feature checklist](docs/compatibility.md)** — supported features, partial support, and known limitations.
- **[Online playground](https://sumi.fmap.dev)** — 17 editable diagrams rendered by Go; also [run locally](#local-playground).
- **Integration** — use a custom text measurer and your native drawing backend, or export SVG.
- **Rendering** — no browser, JavaScript runtime, or external layout service required by the Go library.
- **Status** — 0.x; the API may change. LaTeX math is planned, not implemented.

## Supported diagrams

- [x] Sequence diagrams
- [x] Flowcharts (`flowchart`, `graph`, `flowchart-elk`)
- [x] State diagrams (`stateDiagram`, `stateDiagram-v2`)
- [x] Class diagrams (`classDiagram`, `classDiagram-v2`)
- [x] Entity relationship diagrams (`erDiagram`)
- [x] Gantt diagrams (`gantt`)
- [ ] Pie and other Mermaid diagram types

## Install

Requires Go 1.27 or later.

```sh
go get github.com/bencode/sumi
```

## Draw a diagram

```go
package main

import (
	"log"
	"os"

	"github.com/bencode/sumi/mermaid"
	"github.com/bencode/sumi/scene"
	"github.com/bencode/sumi/svg"
)

func main() {
	s, err := mermaid.Layout(`sequenceDiagram
    Alice->>Bob: Hello Bob
    Bob-->>Alice: Hello Alice`, mermaid.Options{})
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile("hello.svg", svg.Render(s, scene.Light), 0o644); err != nil {
		log.Fatal(err)
	}
}
```

- `mermaid.Layout` reports source errors as `*mermaid.Error`, with 1-based `Line` and `Col`; `IsUnsupported()` identifies explicitly unsupported features.
- `scene.Scene` contains items in drawing order: paint each item's paths, then its text lines at their baselines. Coordinates use DIPs, with the origin at the top left.
- `mermaid.Options.Measurer` accepts a `scene.Measurer`; nil uses built-in Arial measurements. Match the font metrics of your native renderer.
- `svg.Render` uses `scene.Light`, `scene.Dark`, or your own palette. Theme changes do not require layout again; explicit author colors remain fixed.

## Local playground

Requires Go 1.27+, Node 24, and pnpm 11.23.0. From the repository root, run the Go service in one terminal:

```sh
go run ./cmd/sumi-demo -listen 127.0.0.1:8080
```

Start the frontend in another:

```sh
pnpm --dir web install --frozen-lockfile
pnpm --dir web dev
```

Open **http://localhost:5173**. Each example is editable; previews update 400ms after you stop typing. Errors keep the last successful preview. You can reset an example, copy its current source, or download it. Edits stay in page memory and reset on reload. The theme selector changes both the editor and preview.

- The Vite frontend proxies `/api` and `/healthz` to Go at `127.0.0.1:8080`.
- `POST /api/render` accepts `{"source":"..."}` and returns `lightSVG`, `darkSVG`, `width`, and `height`. Requests are limited to 64 KiB.
- Source/request errors return 400, oversized requests 413, unsupported features or empty scenes 422, and internal failures 500. Error JSON has `message` and, when available, `line`, `column`, and `unsupported`.
- `GET /healthz` returns `{"status":"ok"}`. The service uses no database and stores no submitted source.
- Edit [`docs/examples/catalog.json`](docs/examples/catalog.json) to organize examples. Category IDs name subdirectories; example IDs name `.mmd` files. Add the source and a catalog entry to add an example.
- Frontend and service code live in `web/`, `cmd/sumi-demo/`, and `internal/demo/`. See the [Docker Compose deployment guide](deploy/README.md) for container builds and the shared Caddy setup.

## Checks

```sh
go test ./...
go vet ./...
pnpm --dir web check
pnpm --dir web build
pnpm --dir web preview
```

The build writes ignored files to `web/dist/`. Preview the build at `http://localhost:4173` with the Go service still running. `go run ./internal/preview` remains available for low-level scene and upstream sequence fixture previews in `out/`.

## Limits

- Targets the supported subset of Mermaid v12.3.0 with default layout settings.
- Front matter titles work; front matter configuration and `%%{init}%%` directives are accepted but ignored.
- HTML beyond supported line breaks, formulas, icons, and most CSS are not drawn. Links and callbacks do not become interactive.
- See the [feature checklist](docs/compatibility.md) for type-specific behavior and restrictions.

## License

MIT. See [LICENSE](LICENSE).

The exception is `mermaid/internal/elk`, written after the Eclipse Layout Kernel and licensed under `EPL-2.0 OR GPL-3.0-or-later`. See its [LICENSE](mermaid/internal/elk/LICENSE) and [NOTICE](mermaid/internal/elk/NOTICE).
