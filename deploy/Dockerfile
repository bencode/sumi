FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd/sumi-demo ./cmd/sumi-demo
COPY internal/demo ./internal/demo
COPY mermaid ./mermaid
COPY scene ./scene
COPY svg ./svg
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/sumi-demo ./cmd/sumi-demo

FROM alpine:3.23
COPY --from=build /out/sumi-demo /usr/local/bin/sumi-demo
COPY LICENSE /usr/share/licenses/sumi/LICENSE
COPY mermaid/internal/elk/LICENSE mermaid/internal/elk/NOTICE /usr/share/licenses/sumi/elk/
USER 65534:65534
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/sumi-demo"]
CMD ["-listen", "0.0.0.0:8080"]
