FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/mocktmdb ./cmd/mocktmdb
COPY cmd/mockjackett ./cmd/mockjackett
COPY cmd/mockqbittorrent ./cmd/mockqbittorrent
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/mocktmdb ./cmd/mocktmdb \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/mockjackett ./cmd/mockjackett \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/mockqbittorrent ./cmd/mockqbittorrent

FROM alpine:3.22
RUN apk add --no-cache ca-certificates ffmpeg
COPY --from=builder /out/* /usr/local/bin/
