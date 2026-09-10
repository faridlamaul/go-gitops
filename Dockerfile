FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /go-gitops ./cmd/github

FROM alpine:3.20
RUN apk add --no-cache ca-certificates git curl bash
COPY --from=builder /go-gitops /usr/local/bin/go-gitops
ENTRYPOINT ["go-gitops"]
