# ---------- Build Stage ----------
FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1

COPY . .

RUN sqlc generate && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/cinna \
    ./cmd/cinna

# ---------- Runtime stage ----------
FROM alpine:latest

RUN apk add --no-cache ca-certificates

COPY --from=builder /out/cinna /cinna

EXPOSE 8080

ENTRYPOINT [ "/cinna" ]
