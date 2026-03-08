FROM golang:1.24 AS build

WORKDIR /src

COPY go.mod go.sum ./
COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/epics ./cmd/epics

FROM debian:bookworm-slim

COPY --from=build /out/epics /usr/local/bin/epics

WORKDIR /workspace

ENTRYPOINT ["epics"]
