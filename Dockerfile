# syntax=docker/dockerfile:1
FROM golang:1.20-alpine as builder
RUN apk add --update git gcc g++ openssh-client make
WORKDIR /app
COPY go.mod go.sum ./
COPY go-relay-client/ go-relay-client/
RUN go mod download
COPY . .
RUN go get
RUN go build -o ./build/goteth


FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /
COPY --from=builder /app/build/goteth ./
COPY --from=builder /app/pkg/db/migrations ./pkg/db/migrations
ENTRYPOINT ["/goteth"]
