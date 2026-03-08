FROM golang:1.24 AS build

WORKDIR /src

COPY go.mod go.sum ./
COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/epics ./cmd/epics

FROM node:20-bookworm

RUN apt-get update \
	&& apt-get install -y --no-install-recommends python3 python3-pip git ca-certificates \
	&& rm -rf /var/lib/apt/lists/*

RUN npm install -g @anthropic-ai/claude-code

COPY --from=build /out/epics /usr/local/bin/epics

WORKDIR /workspace
