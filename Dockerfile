# syntax=docker/dockerfile:1
FROM golang:1.17-alpine as builder
RUN apk add --update git
RUN apk add --update gcc
RUN apk add --update g++
RUN apk add --update openssh-client
RUN apk add --update make

RUN mkdir /app
WORKDIR /app
ADD . .

RUN go get
RUN make build


FROM alpine:latest  
RUN apk --no-cache add ca-certificates
WORKDIR /
COPY --from=builder /app/build/goteth ./
ENTRYPOINT ["/goteth"]

