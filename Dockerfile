FROM node:24-alpine AS web
WORKDIR /src/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26-alpine AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/nestrr ./cmd/nestrr
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/mocktmdb ./cmd/mocktmdb

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata && addgroup -S nestrr && adduser -S -G nestrr nestrr
WORKDIR /app
COPY --from=backend /out/nestrr /usr/local/bin/nestrr
COPY --from=backend /out/mocktmdb /usr/local/bin/mocktmdb
COPY --from=web /src/web/dist /app/web/dist
RUN mkdir -p /data /cache/images && chown -R nestrr:nestrr /data /cache /app
USER nestrr
EXPOSE 8080
ENTRYPOINT ["nestrr"]
