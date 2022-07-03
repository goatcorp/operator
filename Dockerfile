# Build stage
FROM golang:1.18-alpine AS builder
WORKDIR /go/src/app
RUN apk add --no-cache git

# Setup dependencies
COPY go.mod go.sum /go/src/app/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go mod download
COPY ./ /go/src/app/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -v -o /go/src/app/operator ./cmd/operator

# Run stage
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /go/src/app/operator /app
LABEL Name=operator
ENTRYPOINT /app