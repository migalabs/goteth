# syntax=docker/dockerfile:1
FROM golang:1.17-alpine
RUN apk add --update git
RUN apk add --update gcc
RUN apk add --update g++
RUN apk add --update openssh-client
RUN apk add --update make

RUN mkdir /app
RUN mkdir /app/pkg
RUN mkdir /app/cmd

ADD ./pkg /app/pkg
ADD ./cmd /app/cmd
ADD ./main.go /app/
ADD ./go.mod /app/
ADD ./go.sum /app/

ADD ./Makefile /app/
ADD ./.env /app/


WORKDIR /app
RUN ls -la
RUN go mod tidy
RUN go get
RUN make build

ENTRYPOINT ["./build/eth-cl-state-analyzer"]
