FROM golang:1.24 AS build

WORKDIR /src

COPY go.mod go.sum ./
COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/epics ./cmd/epics
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/epicsd ./cmd/epicsd

FROM debian:bookworm-slim

COPY --from=build /out/epics /usr/local/bin/epics
COPY --from=build /out/epicsd /usr/local/bin/epicsd

WORKDIR /workspace
